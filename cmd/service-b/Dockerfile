FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/service-b ./cmd/service-b

FROM alpine:3.20
# Install CA certificates for HTTPS (ViaCEP uses TLS)
RUN apk add --no-cache ca-certificates \
    && adduser -D -H app
USER app
WORKDIR /home/app
COPY --from=build /out/service-b /usr/local/bin/service-b
EXPOSE 8081
ENV OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
ENTRYPOINT ["/usr/local/bin/service-b"]
