# Sisterpedia - FHE (CKKS) Online Shop

Final project for IF4020 Cryptography (2026/2027): a simple online shop where balances, top-ups, quantities and bills stay **encrypted** end to end. The server computes the bill and updates the balance directly on ciphertexts using the CKKS homomorphic encryption scheme; decryption only ever happens in the user's browser.

Five services: **Activate**, **CheckBalance**, **Topup**, **Transact**, **GetTransact**.

> Status: repository skeleton and contracts are in place; implementation is tracked in the GitHub issues (epic #16).

## Architecture

```
browser (Next.js GUI + Web Worker running Lattigo as WebAssembly, keys in IndexedDB)
   |  ciphertext only (CORS)
   v
server (Go + SQLite :8000)  - public keys + ciphertexts, never a secret key
```

- **web** (`web/`) - Next.js GUI, also serves the WebAssembly crypto bundle (`crypto.wasm`).
- **browser crypto** (`cmd/wasm` + `web/lib/crypto`) - key generation, encode/encrypt, decrypt/decode inside a Web Worker. The secret key lives only in the browser (IndexedDB) and is never sent anywhere. No local service to install. See [`docs/CRYPTO_BRIDGE.md`](docs/CRYPTO_BRIDGE.md).
- **server** (`cmd/server`) - REST API and persistent storage. Holds public keys and ciphertexts; never a secret key.
- The same `pkg/ckks` / `pkg/codec` code is compiled natively for the server and to WebAssembly for the browser, so serialized keys and ciphertexts are compatible.

FHE library: [Lattigo](https://github.com/tuneinsight/lattigo) v6 (Go, CKKS, built-in NTT). Parameters: `config/profiles.yaml`, explained in `docs/PARAMS.md`.

## Quick start

```bash
cp .env.example .env        # optional, defaults work
docker compose up --build   # GUI http://localhost:3000, server :8000
```

Local development (Linux/WSL for Go):
```bash
make test     # go test (with -race when gcc is available)
make lint     # golangci-lint
make wasm     # build web/public/crypto.wasm (GOOS=js GOARCH=wasm)
make run-server
make web-dev  # builds the wasm bundle, then next dev
```

## Repository layout

| Path | Purpose |
|---|---|
| `cmd/server`, `cmd/bench` | binaries |
| `cmd/wasm` | browser crypto bridge (GOOS=js GOARCH=wasm) |
| `internal/server` | server application code |
| `pkg/ckks`, `pkg/codec` | CKKS wrapper and Base64 serialization (shared by server and WASM) |
| `web/` | Next.js GUI, crypto worker, keystore |
| `config/` | CKKS parameter profiles |
| `data/` | product catalog and discount rules |
| `deploy/` | Dockerfiles |
| `tests/`, `bench/` | end-to-end tests, benchmarks, results |
| `docs/` | API contracts, parameters, conventions |
| `report/` | report figures |

## Documentation
- [`docs/API_CONTRACT.md`](docs/API_CONTRACT.md) - server REST API
- [`docs/CRYPTO_BRIDGE.md`](docs/CRYPTO_BRIDGE.md) - browser crypto bridge (WASM), shop module and keystore
- [`docs/PARAMS.md`](docs/PARAMS.md) - CKKS parameter profiles
- [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) - workflow and conventions

## Security rules (non-negotiable)
- No secret keys or `.env` in the repository (only the TEST ONLY fixtures in `tests/fixtures/keys/`). The secret key never leaves the browser.
- Server code never references `SecretKey` or `Decryptor`; there is no decryption endpoint.
- Balance, amounts, quantities and totals appear only as ciphertext in requests, responses, database and logs.
- CKKS parameters live in `config/`, not in code.

The full README (tech stack versions, GUI usage, how to reproduce tests and benchmarks) is completed in the final phase.
