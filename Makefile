.PHONY: run build build-worker sqlc migrate migrate-down migrate-create \
        test test-cover lint css css-watch seed \
        docker-up docker-down docker-reset docker-init-garage docker-build docker-push

run:
	@which air > /dev/null 2>&1 && air || go run ./cmd/api

build:
	go build -o bin/petrosync-api ./cmd/api

build-worker:
	go build -o bin/petrosync-worker ./cmd/worker

sqlc:
	sqlc generate

migrate:
	migrate -path sql/migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path sql/migrations -database "$$DATABASE_URL" down 1

migrate-create:
	migrate create -ext sql -dir sql/migrations -seq $(name)

test:
	go test ./... -v -race

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

css:
	bun run tailwindcss -i static/css/app.css -o static/css/dist.css --minify

css-watch:
	bun run tailwindcss -i static/css/app.css -o static/css/dist.css --watch

seed:
	psql "$$DATABASE_URL" -f sql/migrations/000015_seed.up.sql

# ── Docker Compose (local dev infrastructure) ──────────────────────────
docker-up:
	docker compose up -d
	@echo ""
	@echo "Services starting: postgres, valkey, garage"
	@echo "Run 'docker compose logs garage-init' for Garage S3 credentials."

docker-down:
	docker compose down

docker-reset:
	docker compose down -v

docker-init-garage:
	./docker/garage-init.sh

docker-build:
	docker build -t petrosync-api:$(shell git rev-parse --short HEAD) .

docker-push:
	docker tag petrosync-api:$(shell git rev-parse --short HEAD) \
	    harbor.adevshankar.id/petrosync/api:$(shell git rev-parse --short HEAD)
	docker push harbor.adevshankar.id/petrosync/api:$(shell git rev-parse --short HEAD)
