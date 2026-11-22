# Browser Crypto Bridge (v1)

Status: **proposal, replaces the former `CLIENT_API.md` (local Go service)**. To be frozen in the architecture-change discussion (epic #16).

All cryptography on the user's side runs **in the browser**: Lattigo (CKKS) compiled to WebAssembly (`cmd/wasm`, `GOOS=js GOARCH=wasm`) and executed inside a **Web Worker**. There is no local service to install. The same `pkg/ckks` and `pkg/codec` packages are used natively by the server, so serialized keys and ciphertexts are byte-compatible.

```
GUI (React)  ->  crypto worker (TS) -> WASM bridge (Go, Lattigo)
                      |
                      +-> IndexedDB keystore (secret key lives here, never leaves the browser)
                      +-> fetch -> server REST API (ciphertext only)
```

## Layers
| Layer | Location | Responsibility |
|---|---|---|
| WASM bridge | `cmd/wasm` (Go) | pure crypto functions, no I/O, no storage, no network |
| Worker + shop module | `web/lib/crypto/*.ts` | loads WASM, keystore, orchestrates the five services, talks to the server |
| GUI | `web/app`, `web/components` | calls the shop module only; never touches keys or raw WASM |

## 1. WASM bridge (global `sisterpedia`, called from the worker)
All keys/ciphertexts are **standard Base64 strings** of the library's binary serialization. Every function is synchronous and returns `{ "ok": true, "value": ... }` or `{ "ok": false, "error": { "code": "...", "message": "..." } }`. A Go panic is recovered and returned as `internal_error`.

| Function | Arguments | `value` |
|---|---|---|
| `profiles()` | - | `[{name, logN, maxLevel, slots}]` |
| `genKeys(profile)` | `"p1"\|"p2"` | `{params, secret_key, public_key, relin_key}` |
| `validateKeys(params, secret_key, public_key)` | b64 x3 | `true` (pk matches sk, params loadable) |
| `encryptScalar(params, public_key, x)` | b64, b64, number | `ciphertext` (b64) |
| `encryptVector(params, public_key, values)` | b64, b64, number[] | `ciphertext` (b64) |
| `decryptScalar(params, secret_key, ciphertext)` | b64 x3 | `number` (slot 0, raw) |
| `decryptVector(params, secret_key, ciphertext, n)` | b64 x3, int | `number[]` (first `n` slots) |
| `ciphertextInfo(params, ciphertext)` | b64 x2 | `{level, log2Scale, bytes}` |
| `version()` | - | build/library version string |

Error codes: `invalid_payload` (bad Base64 or unloadable binary), `params_mismatch`, `keys_mismatch`, `value_out_of_range`, `unknown_profile`, `internal_error`.

Rules: no secret material in error messages; no logging of arguments; limit accepted Base64 size per call.

## 2. Shop module (`web/lib/crypto/shop.ts`, runs in the worker)
The GUI uses this async API through `postMessage` (a thin proxy in the main thread). Result shapes keep the fields the GUI needs: the decrypted value **and** a ciphertext preview to display side by side.

```ts
interface CipherView { preview: string; bytes: number }          // first 60 b64 chars + "...", binary size

interface BalanceView {
  user_id: string; balance: number; balance_raw: number; negative: boolean;
  enc_balance: CipherView;
}
interface TransactResult extends BalanceView {
  transaction_id: string; total: number; total_raw: number;
  enc_total: CipherView; warnings: ("insufficient_balance")[];
}
interface HistoryItem {
  transaction_id: string; created_at: string;
  product_id: string; quantity: number; discount_code: string;
  total: number; total_raw: number;
  enc_total: CipherView; enc_metadata: CipherView;
  warning: "metadata_uncertain" | null;
}

init(): Promise<void>
getConfig(): Promise<{profiles: string[]; default_profile: string; max_value: number; tolerance: number}>
listUsers(): Promise<{user_id: string; profile: string}[]>
getCatalog(): Promise<Catalog>                                    // proxies server GET /catalog
activate(userId: string, profile: string): Promise<{status: "activated"; enc_balance: CipherView;
                                                    artifact_bytes: {params: number; public_key: number; relin_key: number}}>
getBalance(userId: string): Promise<BalanceView>
topup(userId: string, amount: number): Promise<BalanceView>
transact(userId: string, productId: string, quantity: number, discountCode?: string): Promise<TransactResult>
getHistory(userId: string): Promise<HistoryItem[]>
exportKeys(userId: string): Promise<Blob>                         // zip, CONTAINS THE SECRET KEY
importKeys(userId: string, file: File): Promise<{status: "imported"; profile: string}>
```
Errors are thrown as `ShopError{ code, message }`. Codes: `invalid_input`, `value_out_of_range`, `keys_not_found`, `keys_corrupt`, `keys_mismatch`, `user_exists_locally`, plus codes forwarded from the server (`account_not_found`, `account_exists`, `product_not_found`, `discount_not_found`, `evaluation_failed`, `invalid_payload`, `context_mismatch`), and `server_unreachable`.

Behavior:
- `activate`: generate keys -> validate -> store in IndexedDB -> encrypt `0` -> `POST /accounts`. If the server rejects, delete the new local keys.
- `topup` / `transact`: validate input (`amount > 0`, integer `quantity >= 1`, `|value| <= max_value`) -> encrypt -> server -> decrypt the returned ciphertexts.
- `transact` sends `product_id`/`discount_code` in the clear and `enc_quantity` + `enc_metadata = Enc([product_num, quantity, discount_num])` as ciphertext (`API_CONTRACT.md`).
- `getHistory`: decrypt metadata, round each slot to the nearest integer, map back through the catalog; `warning = "metadata_uncertain"` if a slot is more than 0.25 from an integer.
- Rounded display: `balance`/`total` rounded to 2 decimals; `*_raw` is the raw decrypted CKKS value.
- `max_value` default 500000 (silent overflow beyond that, see `PARAMS.md`).

## 3. Keystore (IndexedDB, `web/lib/crypto/keystore.ts`)
- DB `sisterpedia-keys`, object store `users`, key `user_id` (`^[a-z0-9_-]{3,32}$`), record `{user_id, profile, params, secret_key, public_key, relin_key, created_at}`.
- **Only the worker reads/writes this store.** The main thread never receives `secret_key` except as the blob returned by `exportKeys`.
- Export/import bundle = zip with `params.json`, `secret.key`, `public.key`, `relin.key` (same names as `docs`/README, Table 2.3 of the spec). Import validates names, size (max 16 MiB), then `validateKeys` before saving. Zip-slip is prevented by accepting only these four names.
- Browser data loss (clearing site data) loses the secret key: the GUI must warn and offer export.

## 4. Security notes specific to the browser
- XSS is the main threat to the secret key: strict **Content-Security-Policy** (no inline scripts, no third-party scripts), `Cross-Origin-Opener-Policy`/`X-Content-Type-Options`, dependencies pinned.
- The secret key is never sent over the network; the server must still reject any payload that contains one.
- WASM binary is served same-origin (`/crypto.wasm`, `/wasm_exec.js`), versioned with the build; `wasm_exec.js` must come from the same Go version that built the WASM.
- Heavy operations (keygen, relin key, p2) run in the worker so the UI does not freeze.

## 5. Measured feasibility (spike, Lattigo v6.2.0, median of 5, WASM run under Node)
| | p1 native | p1 WASM | p2 native | p2 WASM |
|---|---|---|---|---|
| keygen (sk+pk) | 3.3 ms | 5.4 ms | 7.2 ms | 13 ms |
| relin key | 5.6 ms | 12 ms | 21 ms | 40 ms |
| encrypt | 3.9 ms | 9.7 ms | 9.0 ms | 30 ms |
| 2x mul+relin+rescale | 6.6 ms | 17 ms | 20 ms | 54 ms |
| decrypt | 0.4 ms | 0.95 ms | 7.7 ms | 32 ms |
Artifact sizes (binary): pk 512 / 1280 KiB, relin key 1536 / 5120 KiB, ciphertext 384 / 1024 KiB (p1 / p2). WASM binary 12 MB raw, about 2.9 MB gzip. Max error vs reference (12500 x 2 x 0.9 = 22500): 8.5e-5 (p1), 1.4e-4 (p2). Not yet measured in a real browser.
