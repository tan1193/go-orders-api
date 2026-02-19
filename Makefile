.PHONY: run test test-race docker-up

run:
	go run ./cmd/server

test:
	go test ./...

test-race:
	go test -race ./...

docker-up:
	docker compose up --build

docker-postgres:
	docker compose up -d postgres