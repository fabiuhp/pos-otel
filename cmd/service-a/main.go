package main

import (
    "context"
    "encoding/json"
    "bytes"
    "io"
    "log"
    "net/http"
    "os"
    "regexp"
	"time"

	"pos-otel/internal/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

type cepRequest struct {
	CEP string `json:"cep"`
}

func main() {
	ctx := context.Background()
	shutdown, err := telemetry.InitProvider(ctx, "service-a")
	if err != nil {
		log.Fatalf("otel init: %v", err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	mux := http.NewServeMux()
	mux.Handle("/cep", otelhttp.NewHandler(http.HandlerFunc(handleCEP), "service-a.cep"))

	addr := ":8080"
	log.Printf("service-a listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

var cepRe = regexp.MustCompile(`^\d{8}$`)

func handleCEP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req cepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid zipcode"})
		return
	}
	if !cepRe.MatchString(req.CEP) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid zipcode"})
		return
	}

	svcB := os.Getenv("SERVICE_B_URL")
	if svcB == "" {
		svcB = "http://service-b:8081"
	}

    payload, _ := json.Marshal(req)
    client := &http.Client{Timeout: 5 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)}

    reqOut, err := http.NewRequestWithContext(r.Context(), http.MethodPost, svcB+"/cep", bytes.NewReader(payload))
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "forward error"})
        return
    }
    reqOut.Header.Set("Content-Type", "application/json")

	tracer := otel.Tracer("service-a")
	ctx, span := tracer.Start(r.Context(), "call.service-b")
	defer span.End()
	reqOut = reqOut.WithContext(ctx)

	resp, err := client.Do(reqOut)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"message": "service b unavailable"})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
    _, _ = io.Copy(w, resp.Body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
