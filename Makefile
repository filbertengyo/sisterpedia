# Build, test and benchmark targets. Windows equivalents: scripts/*.ps1.
.PHONY: build test lint wasm run-server web-dev up down test-e2e test-web test-ui test-all bench bench-report

GO ?= go
# The race detector needs cgo (a C compiler). Enable it only when gcc is present;
# CI always runs with -race.
RACE := $(shell command -v gcc >/dev/null 2>&1 && echo -race)

build:
	$(GO) build -o bin/ ./cmd/server ./cmd/bench

test:
	$(GO) test $(RACE) -cover ./...

lint:
	golangci-lint run ./...

# Browser crypto bridge: Lattigo -> WebAssembly, placed where Next.js serves it.
wasm:
	GOOS=js GOARCH=wasm $(GO) build -trimpath -ldflags="-s -w" -o web/public/crypto.wasm ./cmd/wasm
	cp "$$($(GO) env GOROOT)/lib/wasm/wasm_exec.js" web/public/wasm_exec.js

run-server:
	$(GO) run ./cmd/server

web-dev: wasm
	cd web && npm run dev

up:
	docker compose up --build

down:
	docker compose down

# U01-U12 + 5 transaction chains on both profiles against the real server
# process; writes tests/results/{RESULTS.md,results.json,chains*.csv,samples/}.
test-e2e:
	$(GO) test -tags e2e -count=1 -v ./tests/e2e/

# Vitest unit tests of the GUI helpers.
test-web:
	cd web && npm test

# Playwright GUI flows (real server + production web build); screenshots go
# to report/figures/gui and are copied to tests/results/screenshots.
test-ui: wasm
	cd web && GO_BIN="$$(command -v $(GO))" npx playwright test
	mkdir -p tests/results/screenshots && cp report/figures/gui/*.png tests/results/screenshots/

# Everything, from a clean state.
test-all: test test-e2e test-web test-ui
	@echo "All tests passed. Results: tests/results/"

# Table 2.7 metrics on both profiles (5 warm-up + 30 runs), native and WASM.
bench: wasm
	$(GO) run ./cmd/bench
	node bench/wasm/bench.mjs
	$(GO) run ./cmd/bench -report

bench-report:
	$(GO) run ./cmd/bench -report
