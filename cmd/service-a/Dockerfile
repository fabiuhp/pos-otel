FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/service-a ./cmd/service-a

FROM alpine:3.20
RUN adduser -D -H app
USER app
WORKDIR /home/app
COPY --from=build /out/service-a /usr/local/bin/service-a
EXPOSE 8080
ENV OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
ENTRYPOINT ["/usr/local/bin/service-a"]
