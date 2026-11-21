# Target dasar; isi lengkap menyusul di issue OPS/C2/C3.
.PHONY: build test run-server run-client web-dev up down test-all bench

build:
	go build -o bin/ ./cmd/...

test:
	go test -race ./...

run-server:
	go run ./cmd/server

run-client:
	go run ./cmd/clientsvc

web-dev:
	cd web && npm run dev

up:
	docker compose up --build

down:
	docker compose down

test-all: test
	@echo "TODO: e2e U01-U10 (issue C2)"

bench:
	@echo "TODO: go run ./cmd/bench (issue C3)"
