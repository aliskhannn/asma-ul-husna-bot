.PHONY: help run build test migrate-up migrate-down migrate-create docker-up docker-down docker-down-v

help:
	@echo "Available commands:"
	@echo "  make run           - Run the bot"
	@echo "  make build         - Build the binary"
	@echo "  make test          - Run tests"
	@echo "  make migrate-up    - Apply database migrations"
	@echo "  make migrate-down  - Rollback last migration"
	@echo "  make docker-up     - Start docker-compose"
	@echo "  make docker-down   - Stop docker-compose"
	@echo "  make docker-down-v - Stop docker-compose and remove volumes"

run:
	go run cmd/bot/main.go

build:
	go build -o bin/asma-ul-husna-bot cmd/bot/main.go

test:
	go test -v ./...

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-create:
	goose -dir migrations create $(name) sql

docker-up:
	docker compose up --build

docker-down:
	docker compose down

docker-down-v:
	docker compose down -v

docker-logs:
	docker compose logs -f bot