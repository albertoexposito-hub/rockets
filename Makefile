# Makefile para Rockets

.PHONY: help
help:
	@echo "Comandos disponibles:"
	@echo "  make run          - Ejecutar directamente con go run"
	@echo "  make build        - Compilar binario"
	@echo "  make start        - Compilar y ejecutar binario"
	@echo "  make test         - Ejecutar tests"
	@echo "  make clean        - Limpiar binarios"
	@echo ""
	@echo "Variables:"
	@echo "  WORKER_COUNT=N    - Número de workers (default: 3)"
	@echo ""
	@echo "Ejemplos:"
	@echo "  make run"
	@echo "  make run WORKER_COUNT=5"

# Variables
WORKER_COUNT ?= 3

.PHONY: run
run:
	@echo "🚀 Starting Rockets Server (workers=$(WORKER_COUNT))..."
	WORKER_COUNT=$(WORKER_COUNT) go run cmd/server/main.go

.PHONY: build
build:
	@echo "🔨 Building..."
	go build -o bin/rockets-server cmd/server/main.go
	@echo "✅ Binary created at bin/rockets-server"

.PHONY: start
start: build
	@echo "🚀 Starting compiled server..."
	WORKER_COUNT=$(WORKER_COUNT) ./bin/rockets-server

.PHONY: test
test:
	@echo "🧪 Running tests..."
	go test -v ./...
	go test -v ./internal/application

.PHONY: test-integration
.PHONY: clean
clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/
	go clean
	@echo "✅ Clean done"

# Comandos de prueba rápida
.PHONY: curl-health
curl-health:
	@curl -s http://localhost:8088/health | jq .

.PHONY: curl-rockets
curl-rockets:
	@curl -s http://localhost:8088/rockets | jq .

.PHONY: curl-launch
curl-launch:
	@curl -s -X POST http://localhost:8088/messages \
		-H "Content-Type: application/json" \
		-d '{"metadata":{"channel":"test-rocket","messageNumber":1,"messageTime":"2026-01-22T10:00:00Z","messageType":"RocketLaunched"},"message":{"type":"Falcon-9","launchSpeed":25000,"mission":"Mars"}}' | jq .
