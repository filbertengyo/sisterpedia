# Target dasar; isi lengkap menyusul di issue OPS/C2/C3.
.PHONY: build test lint run-server run-client web-dev up down test-all bench

# The race detector needs cgo (a C compiler). Enable it only when gcc is present;
# CI always runs with -race.
RACE := $(shell command -v gcc >/dev/null 2>&1 && echo -race)

build:
	go build -o bin/ ./cmd/...

test:
	go test $(RACE) -cover ./...

lint:
	golangci-lint run ./...

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
