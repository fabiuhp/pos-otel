package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "math"
    "net/http"
    "net/url"
    "os"
    "regexp"
    "time"

    "pos-otel/internal/telemetry"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

type cepRequest struct {
    CEP string `json:"cep"`
}

type viaCEPResp struct {
    Localidade string `json:"localidade"`
    UF         string `json:"uf"`
    Erro       bool   `json:"erro"`
}

type weatherResp struct {
    City  string  `json:"city"`
    TempC float64 `json:"temp_C"`
    TempF float64 `json:"temp_F"`
    TempK float64 `json:"temp_K"`
}

var cepRe = regexp.MustCompile(`^\d{8}$`)

func main() {
    ctx := context.Background()
    shutdown, err := telemetry.InitProvider(ctx, "service-b")
    if err != nil {
        log.Fatalf("otel init: %v", err)
    }
    defer func() { _ = shutdown(context.Background()) }()

    mux := http.NewServeMux()
    mux.Handle("/cep", otelhttp.NewHandler(http.HandlerFunc(handleCEP), "service-b.cep"))

    addr := ":8081"
    log.Printf("service-b listening on %s", addr)
    if err := http.ListenAndServe(addr, mux); err != nil {
        log.Fatal(err)
    }
}

func handleCEP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var req cepRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !cepRe.MatchString(req.CEP) {
        writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid zipcode"})
        return
    }

    city, uf, err := lookupCity(r.Context(), req.CEP)
    if err != nil {
        if errors.Is(err, errNotFound) {
            writeJSON(w, http.StatusNotFound, map[string]string{"message": "can not find zipcode"})
            return
        }
        writeJSON(w, http.StatusBadGateway, map[string]string{"message": "zipcode lookup failed"})
        return
    }

    tempC, err := lookupTempC(r.Context(), city, uf)
    if err != nil {
        writeJSON(w, http.StatusBadGateway, map[string]string{"message": "weather lookup failed"})
        return
    }

    resp := weatherResp{
        City:  city,
        TempC: round1(tempC),
        TempF: round1(tempC*1.8 + 32),
        TempK: round1(tempC + 273),
    }

    writeJSON(w, http.StatusOK, resp)
}

var errNotFound = errors.New("not found")

func lookupCity(ctx context.Context, cep string) (string, string, error) {
    tracer := otel.Tracer("service-b")
    ctx, span := tracer.Start(ctx, "viacep.lookup")
    defer span.End()
    span.SetAttributes(attribute.String("cep", cep))

    u := url.URL{Scheme: "https", Host: "viacep.com.br", Path: fmt.Sprintf("/ws/%s/json/", cep)}
    urlStr := u.String()
    client := &http.Client{Timeout: 5 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)}
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
    res, err := client.Do(req)
    if err != nil {
        span.RecordError(err)
        return "", "", err
    }
    defer res.Body.Close()
    if res.StatusCode != http.StatusOK {
        if res.StatusCode == http.StatusBadRequest || res.StatusCode == http.StatusNotFound {
            return "", "", errNotFound
        }
        return "", "", fmt.Errorf("viacep status %d", res.StatusCode)
    }
    var v viaCEPResp
    if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
        span.RecordError(err)
        return "", "", err
    }
    if v.Erro || v.Localidade == "" {
        return "", "", errNotFound
    }
    span.SetAttributes(attribute.String("city", v.Localidade), attribute.String("uf", v.UF))
    return v.Localidade, v.UF, nil
}

func lookupTempC(ctx context.Context, city, uf string) (float64, error) {
    tracer := otel.Tracer("service-b")
    ctx, span := tracer.Start(ctx, "weatherapi.current")
    defer span.End()
    span.SetAttributes(attribute.String("city", city), attribute.String("uf", uf))

    key := os.Getenv("WEATHER_API_KEY")
    if key == "" {
        return 0, fmt.Errorf("missing WEATHER_API_KEY")
    }
    wurl := url.URL{Scheme: "http", Host: "api.weatherapi.com", Path: "/v1/current.json"}
    params := wurl.Query()
    params.Set("key", key)
    params.Set("q", fmt.Sprintf("%s,%s,BR", city, uf))
    params.Set("aqi", "no")
    wurl.RawQuery = params.Encode()
    urlStr := wurl.String()
    client := &http.Client{Timeout: 5 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)}
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
    res, err := client.Do(req)
    if err != nil {
        span.RecordError(err)
        return 0, err
    }
    defer res.Body.Close()
    if res.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(res.Body)
        span.RecordError(fmt.Errorf("weather status %d: %s", res.StatusCode, string(body)))
        return 0, fmt.Errorf("weather api status %d", res.StatusCode)
    }
    var parsed struct {
        Current struct {
            TempC float64 `json:"temp_c"`
        } `json:"current"`
    }
    if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
        span.RecordError(err)
        return 0, err
    }
    return parsed.Current.TempC, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }
