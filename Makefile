SHELL := /bin/sh

.PHONY: build test cover fmt vet lint tidy run-a run-b docker-up docker-down

build:
	go build -o ./.bin/service-a ./cmd/service-a
	go build -o ./.bin/service-b ./cmd/service-b

test:
	go test -race -coverprofile=coverage.out ./...

cover:
	@go tool cover -func=coverage.out | tail -n 1 || true

fmt:
	gofmt -s -w .

vet:
	go vet ./...

lint: fmt vet

tidy:
	go mod tidy

run-b:
	OTEL_EXPORTER_OTLP_ENDPOINT?=localhost:4317
	WEATHER_API_KEY?=
	@if [ -z "$$WEATHER_API_KEY" ]; then echo "Defina WEATHER_API_KEY ao rodar (ex.: make run-b WEATHER_API_KEY=...)"; exit 1; fi
	OTEL_EXPORTER_OTLP_ENDPOINT=$$OTEL_EXPORTER_OTLP_ENDPOINT WEATHER_API_KEY=$$WEATHER_API_KEY go run ./cmd/service-b

run-a:
	OTEL_EXPORTER_OTLP_ENDPOINT?=localhost:4317
	SERVICE_B_URL?=http://localhost:8081
	OTEL_EXPORTER_OTLP_ENDPOINT=$$OTEL_EXPORTER_OTLP_ENDPOINT SERVICE_B_URL=$$SERVICE_B_URL go run ./cmd/service-a

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v

