# Basic targets; the rest is filled in by the OPS/C2/C3 issues.
.PHONY: build test lint wasm run-server web-dev up down test-all bench

# The race detector needs cgo (a C compiler). Enable it only when gcc is present;
# CI always runs with -race.
RACE := $(shell command -v gcc >/dev/null 2>&1 && echo -race)

build:
	go build -o bin/ ./cmd/server

test:
	go test $(RACE) -cover ./...

lint:
	golangci-lint run ./...

# Browser crypto bridge: Lattigo -> WebAssembly, placed where Next.js serves it.
wasm:
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o web/public/crypto.wasm ./cmd/wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/public/wasm_exec.js

run-server:
	go run ./cmd/server

web-dev: wasm
	cd web && npm run dev

up:
	docker compose up --build

down:
	docker compose down

test-all: test
	@echo "TODO: e2e U01-U10 (issue C2)"

bench:
	@echo "TODO: go run ./cmd/bench (issue C3)"
