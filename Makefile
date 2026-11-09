.PHONY: all build test up down clean help

help:
	@echo "API Key & Rate Limiting Gateway Commands:"
	@echo "  make build       - Build all Go binaries (gateway, control-plane, mock-backend)"
	@echo "  make test        - Run all Go unit and integration tests"
	@echo "  make up          - Start all containers via docker-compose"
	@echo "  make down        - Stop all containers"
	@echo "  make dashboard   - Start Next.js dashboard locally"

build:
	go build -o gateway/gateway.exe ./gateway/cmd/gateway
	go build -o control-plane/control-plane.exe ./control-plane/cmd/server
	go build -o mock-backend/mock-backend.exe ./mock-backend

test:
	go test -v -race ./gateway/... ./control-plane/...

up:
	docker compose up --build -d

down:
	docker compose down

clean:
	rm -f gateway/gateway.exe control-plane/control-plane.exe mock-backend/mock-backend.exe

