# Sisterpedia - FHE (CKKS) Online Shop

Final project for IF4020 Cryptography (2026/2027): a simple online shop where balances, top-ups, quantities and bills stay **encrypted** end to end. The server computes the bill and updates the balance directly on ciphertexts using the CKKS homomorphic encryption scheme; decryption only ever happens on the user's device.

Five services: **Activate**, **CheckBalance**, **Topup**, **Transact**, **GetTransact**.

> Status: repository skeleton and contracts are in place; implementation is tracked in the GitHub issues (epic #16).

## Architecture

```
browser -> web (Next.js :3000) -> client service (Go :8001) -> server (Go + SQLite :8000)
                                  keygen / encrypt / decrypt     ciphertext + public artifacts only
```

- **web** - Next.js GUI. UI only, no cryptography, talks only to the client service.
- **client service** (`cmd/clientsvc`) - key generation, encode/encrypt, decrypt/decode, key files. The secret key never leaves this service.
- **server** (`cmd/server`) - REST API and persistent storage. Holds public keys and ciphertexts; never a secret key.
- Docker Compose isolates the networks: `web` cannot reach `server`; volume `client_data` is mounted only into `client`, `server_data` only into `server`.

FHE library: [Lattigo](https://github.com/tuneinsight/lattigo) v6 (Go, CKKS, built-in NTT). Parameters: `config/profiles.yaml`, explained in `docs/PARAMS.md`.

## Quick start

```bash
cp .env.example .env        # optional, defaults work
docker compose up --build   # GUI http://localhost:3000, client :8001, server :8000
```

Local development (Linux/WSL for Go):
```bash
make test     # go test (with -race when gcc is available)
make lint     # golangci-lint
make build
cd web && npm ci && npm run dev
```

## Repository layout

| Path | Purpose |
|---|---|
| `cmd/server`, `cmd/clientsvc`, `cmd/bench` | binaries |
| `internal/server`, `internal/client` | application code |
| `pkg/ckks`, `pkg/codec` | CKKS wrapper and Base64 serialization |
| `web/` | Next.js GUI |
| `config/` | CKKS parameter profiles |
| `data/` | product catalog and discount rules |
| `deploy/` | Dockerfiles |
| `tests/`, `bench/` | end-to-end tests, benchmarks, results |
| `docs/` | API contracts, parameters, conventions |
| `report/` | report figures |

## Documentation
- [`docs/API_CONTRACT.md`](docs/API_CONTRACT.md) - server REST API
- [`docs/CLIENT_API.md`](docs/CLIENT_API.md) - client local service API used by the GUI
- [`docs/PARAMS.md`](docs/PARAMS.md) - CKKS parameter profiles
- [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) - workflow and conventions

## Security rules (non-negotiable)
- No secret keys or `.env` in the repository (only the TEST ONLY fixtures in `tests/fixtures/keys/`).
- Server code never references `SecretKey` or `Decryptor`; there is no decryption endpoint.
- Balance, amounts, quantities and totals appear only as ciphertext in requests, responses, database and logs.
- CKKS parameters live in `config/`, not in code.

The full README (tech stack versions, GUI usage, how to reproduce tests and benchmarks) is completed in the final phase.
