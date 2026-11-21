# Client Local Service API (v1)

Status: **proposal, to be frozen together with `API_CONTRACT.md` (issue #1)**. The web GUI (Next.js) talks **only** to this service, never directly to the server.

The client service (`cmd/clientsvc`, default `http://localhost:8001`) performs key generation, encoding, encryption, decryption and decoding. The secret key lives only in this service's data directory (`CLIENT_DATA_DIR`, Docker volume `client_data`) and is never sent to the server.

Plaintext amounts/quantities sent *to this local service* are expected and allowed (it is the user's own device). The service encrypts them before calling the server. Responses contain the decrypted value **and** a ciphertext preview, so the GUI can show both side by side.

## Conventions
- JSON, `Content-Type: application/json`. CORS allows only `ALLOWED_ORIGIN`.
- `user_id`: `^[a-z0-9_-]{3,32}$`.
- `*_preview`: first 60 characters of the Base64 ciphertext followed by `...`. `*_bytes`: size of the binary ciphertext.
- Money fields: `balance` / `total` are rounded to 2 decimals for display; `balance_raw` / `total_raw` are the raw decrypted CKKS values (for error analysis). `negative` is `true` when the decrypted balance is below `-tolerance`.
- Maximum absolute value accepted for amounts: from `GET /local/config` (`max_value`, default 500000). Larger values are rejected with `400 value_out_of_range`, because CKKS can overflow silently at the lowest level (see `PARAMS.md`).

## Error format
```json
{"error": "<code>", "detail": "<human readable text>"}
```
| HTTP | `error` | Meaning |
|---|---|---|
| 400 | `invalid_json`, `invalid_payload`, `value_out_of_range` | bad input |
| 400 | `keys_not_found` | no local key files for this user (activate or import first) |
| 400 | `keys_corrupt`, `keys_mismatch` | key files unreadable or pk/sk do not match |
| 404 | `account_not_found`, `product_not_found`, `discount_not_found` | forwarded from the server |
| 409 | `account_exists`, `user_exists_locally` | duplicate activation / local keys already exist |
| 502 | `server_unreachable`, `server_error` | server down or returned 5xx |

## Endpoints

### `GET /local/config`
```json
{"profiles": ["p1", "p2"], "default_profile": "p1", "max_value": 500000, "tolerance": 0.01}
```

### `GET /local/users`
Users that have key files on this device.
```json
{"users": [{"user_id": "vanessa", "profile": "p1"}]}
```

### `GET /local/catalog`
Proxy of the server's `GET /catalog`.
```json
{"products": [{"product_id": "P001", "name": "Chocolate Cake", "base_price": 12500}],
 "discounts": [{"code": "COMBO", "percent": 0.1, "fixed": 500}]}
```

### `POST /local/users` (Activate)
Generates keys, stores them locally, encrypts `0`, and calls the server `POST /accounts`. If the server rejects the activation, the newly created local keys are removed.
Request: `{"user_id": "vanessa", "profile": "p1"}`
Response `201`:
```json
{"status": "activated", "user_id": "vanessa", "profile": "p1",
 "enc_balance_preview": "AQAAAAAAAAB...", "enc_balance_bytes": 262424,
 "artifact_bytes": {"params": 120, "public_key": 262400, "relin_key": 1572864}}
```

### `GET /local/users/{user_id}/balance` (CheckBalance)
```json
{"user_id": "vanessa", "balance": 0.0, "balance_raw": 3.1e-07, "negative": false,
 "enc_balance_preview": "AQAAAAAAAAB...", "enc_balance_bytes": 262424}
```

### `POST /local/users/{user_id}/topup` (Topup)
Request: `{"amount": 100000}` (integers and decimals allowed, `> 0`).
Response `200`: same shape as CheckBalance.

### `POST /local/users/{user_id}/transactions` (Transact)
Request: `{"product_id": "P001", "quantity": 2, "discount_code": "COMBO"}` (`discount_code` optional, `quantity` integer `>= 1`).
Response `201`:
```json
{"transaction_id": "01JABC...",
 "total": 22000.0, "total_raw": 21999.99999,
 "balance": 78000.0, "balance_raw": 77999.99998, "negative": false,
 "enc_total_preview": "AQAAAAAAAAB...", "enc_balance_preview": "AQAAAAAAAAB...",
 "warnings": []}
```
`warnings` may contain `"insufficient_balance"` when the new balance is negative. The server never checks sufficiency; this is a client-side display rule.

### `GET /local/users/{user_id}/transactions` (GetTransact)
```json
{"user_id": "vanessa",
 "transactions": [{
   "transaction_id": "01JABC...", "created_at": "2026-11-25T10:00:00Z",
   "product_id": "P001", "quantity": 2, "discount_code": "COMBO",
   "total": 22000.0, "total_raw": 21999.99999,
   "enc_total_preview": "AQAAAAAAAAB...", "enc_metadata_preview": "AQAAAAAAAAB...",
   "warning": null}]}
```
`product_id`, `quantity`, `discount_code` are recovered by decrypting `enc_metadata` and rounding each slot to the nearest integer, then mapping back through the catalog. `warning` is `"metadata_uncertain"` when a slot is more than 0.25 away from an integer.

### `GET /local/users/{user_id}/keys/export`
Returns `application/zip` with `params.json`, `secret.key`, `public.key`, `relin.key`. **Contains the secret key**; the GUI must show a strong warning.

### `POST /local/users/{user_id}/keys/import`
`multipart/form-data` field `file` (zip, small size limit, e.g. 1 MiB). Only the four file names above are accepted; the bundle is validated (loadable, pk matches sk) before being stored. Response `200`: `{"status": "imported", "user_id": "vanessa", "profile": "p1"}`.

### `GET /healthz`
`{"status": "ok", "service": "client"}`

## Notes for implementers
- Client -> server calls follow `API_CONTRACT.md`. The client sends `product_id` and `discount_code` in the clear and everything numeric as ciphertext.
- Never log request/response bodies. Log method, path, status, duration, byte sizes.
- Test mode (`AUDIT=1`) may append reference values and raw decrypted results to `client_data/<user_id>/audit.jsonl`. This file is never sent to the server.
