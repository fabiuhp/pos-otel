# Clima por CEP com OTEL & Zipkin

Dois serviços em Go com tracing distribuído.

- Serviço A (input): POST `http://localhost:8000/cep` com `{ "cep": "29902555" }`. Valida e encaminha para o Serviço B.
- Serviço B (orquestração): Recebe o CEP, consulta a cidade no ViaCEP, obtém temperatura atual no WeatherAPI e responde com cidade + temperaturas (C/F/K).

## Executar com Docker Compose

1) Build e sobe os serviços (a API key já está configurada no docker-compose):

```
docker compose up --build
```

Se houver conflito de porta (8000/8001):
- Liberar a porta: pare o processo ou container que usa a porta (ex.: `lsof -i :8000` / `kill <pid>` ou `docker ps` / `docker kill <id>`)
- Ou altere a porta de host ao subir: `SERVICE_A_PORT=18080 SERVICE_B_PORT=18081 docker compose up --build`

2) Testes rápidos:

```
curl -s -X POST http://localhost:8000/cep \
  -H 'Content-Type: application/json' \
  -d '{"cep":"01001000"}' | jq

curl -s -X POST http://localhost:8000/cep \
  -H 'Content-Type: application/json' \
  -d '{"cep":"123"}' | jq
```

Zipkin UI: http://localhost:9411

## Usando Makefile

- Subir com Docker: `make docker-up`
- Derrubar containers: `make docker-down`
- Testes: `make test` (gera `coverage.out`; veja resumo com `make cover`)
- Lint básico: `make lint` (fmt + vet)
- Build local: `make build` (binários em `.bin/`)
- Desenvolvimento local sem Docker:
  - Em um terminal: `make run-b WEATHER_API_KEY=e747012f0dff4b3990105613251107`
  - Em outro terminal: `make run-a`

Sobrescrever portas ao subir com Make/Compose:
- `SERVICE_A_PORT=18080 SERVICE_B_PORT=18081 make docker-up`

## Respostas

- Sucesso 200: `{ "city": "São Paulo", "temp_C": 28.5, "temp_F": 83.3, "temp_K": 301.6 }`
- CEP inválido 422: `{ "message": "invalid zipcode" }`
- CEP não encontrado 404: `{ "message": "can not find zipcode" }`

## Tracing

- Exportação via OTLP gRPC para o Collector, com envio ao Zipkin.
- Spans principais:
  - service-a: `service-a.cep`, `call.service-b`
  - service-b: `service-b.cep`, `viacep.lookup`, `weatherapi.current`

## Desenvolvimento local (sem Docker)

- Go 1.25+
- Variáveis: `WEATHER_API_KEY` e `OTEL_EXPORTER_OTLP_ENDPOINT` (ex.: `localhost:4317`).
- Execução:

```
go run ./cmd/service-b &
SERVICE_B_URL=http://localhost:8081 go run ./cmd/service-a
```

## Integrações

- ViaCEP: `https://viacep.com.br/ws/{CEP}/json/` para resolver cidade e UF.
- WeatherAPI: `http://api.weatherapi.com/v1/current.json?key=...&q=<cidade>,<UF>,BR&aqi=no`.
  - A chave já está configurada no `docker-compose.yml`.
  - Em desenvolvimento local sem Docker, passe `WEATHER_API_KEY` no `make run-b`.
