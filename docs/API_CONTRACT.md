# Kontrak REST API Server (v1)

Status: **usulan untuk dibekukan di issue #1 (S0.4)**. Perubahan setelah dibekukan hanya lewat diskusi di issue tersebut.

Server hanya menerima/menyimpan artefak publik dan ciphertext. Semua nilai sensitif (saldo, nominal topup, kuantitas, total tagihan, metadata) berupa **ciphertext biner pustaka yang di-encode Base64 standar (RFC 4648, dengan padding)** di dalam JSON. Tidak ada endpoint yang mengembalikan plaintext hasil dekripsi.

## Ketentuan umum
- Base URL dev: `http://localhost:8000`. Content-Type request/respons: `application/json; charset=utf-8`.
- `user_id`: string `^[a-z0-9_-]{3,32}$`.
- Field tak dikenal ditolak (`400 invalid_json`). Body maksimum `MAX_BODY_BYTES` (default 8 MiB; kunci relinearization besar).
- Base64 string kosong atau tidak valid -> `400 invalid_payload`.
- Penolakan **tidak** mengubah saldo maupun riwayat. Pembaruan saldo + penambahan riwayat disimpan atomik.
- Waktu memakai RFC 3339 UTC.

## Endpoint

### 1. Activate - `POST /accounts`
Request:
```json
{
  "user_id": "vanessa",
  "params": "<b64 parameter CKKS terserialisasi>",
  "public_key": "<b64>",
  "relin_key": "<b64>",
  "enc_balance": "<b64 ciphertext Enc(0)>"
}
```
Respons `201 Created`:
```json
{"status": "activated", "user_id": "vanessa", "enc_balance": "<b64>"}
```
Catatan: payload **tidak boleh** memuat secret key. Server menolak bila salah satu blob bukan tipe yang diharapkan (`400 secret_key_rejected` bila terdeteksi sebagai sk). Parameter harus sama persis dengan salah satu profil di `ALLOWED_PROFILES` (`400 context_mismatch`). `user_id` sudah ada -> `409 account_exists`.

### 2. CheckBalance - `GET /accounts/{user_id}/balance`
Respons `200 OK`:
```json
{"user_id": "vanessa", "enc_balance": "<b64>"}
```
Akun belum aktif -> `404 account_not_found`.

### 3. Topup - `POST /accounts/{user_id}/topup`
Request: `{"enc_amount": "<b64>"}`
Respons `200 OK`: `{"user_id": "vanessa", "enc_balance": "<b64 saldo baru>"}`.
Server menghitung `c_b' = c_b (+) c_amount` setelah menyelaraskan level dan skala. Akun belum aktif -> `404`.

### 4. Transact - `POST /accounts/{user_id}/transactions`
Request:
```json
{
  "product_id": "P001",
  "discount_code": "COMBO",
  "enc_quantity": "<b64 Enc(q)>",
  "enc_metadata": "<b64 Enc([product_num, qty, discount_num])>"
}
```
`discount_code` boleh `null`/dihilangkan (dianggap `NONE`). Respons `201 Created`:
```json
{"transaction_id": "01JABC...", "enc_total": "<b64>", "enc_balance": "<b64>"}
```
Server menghitung `t = (p * q * (1 - d)) - k` dengan dua perkalian ciphertext x ciphertext (masing-masing diikuti relinearization dan rescaling), lalu `c_b' = c_b (-) c_t`. **Tidak ada pemeriksaan kecukupan saldo**; saldo boleh negatif dan klien yang menampilkannya. `product_id` atau `discount_code` tak dikenal -> `404 product_not_found` / `404 discount_not_found`. Evaluasi gagal (level habis, kunci tidak cocok) -> `400 evaluation_failed`.

### 5. GetTransact - `GET /accounts/{user_id}/transactions`
Respons `200 OK`:
```json
{
  "user_id": "vanessa",
  "transactions": [
    {"transaction_id": "01JABC...", "enc_metadata": "<b64>", "enc_total": "<b64>", "created_at": "2026-11-25T10:00:00Z"}
  ]
}
```
Daftar kosong dikembalikan sebagai `[]`. Urutan: terlama ke terbaru.

### Pendukung - `GET /catalog`
Data non-sensitif untuk klien (ID produk, nama, harga dasar, aturan diskon):
```json
{
  "products": [{"product_id": "P001", "numeric_id": 1, "name": "Kue Coklat", "base_price": 12500}],
  "discounts": [{"code": "COMBO", "numeric_id": 3, "percent": 0.1, "fixed": 500}]
}
```
`GET /healthz` -> `200 {"status":"ok","service":"server"}`.

## Kode status dan galat
| Status | `error` | Kondisi |
|---|---|---|
| 200 / 201 | - | berhasil |
| 400 | `invalid_json` | JSON/tipe/field salah |
| 400 | `invalid_payload` | Base64 tidak valid, biner tidak dapat dimuat |
| 400 | `context_mismatch` | parameter/kunci tidak cocok dengan profil atau akun |
| 400 | `secret_key_rejected` | payload memuat secret key |
| 400 | `evaluation_failed` | pustaka menolak operasi |
| 404 | `account_not_found` | akun belum aktif (dipilih 404, bukan 400) |
| 404 | `product_not_found`, `discount_not_found` | ID tidak ada di katalog |
| 409 | `account_exists` | aktivasi ganda |
Bentuk galat: `{"error": "<kode>", "detail": "<teks tanpa ciphertext>"}`.

## Skema pemetaan ID (S0.6)
- `product_id` string (mis. `P001`) dan `discount_code` string tetap **terbuka** (server membutuhkannya untuk mencari harga dan aturan diskon). Ini dibahas di laporan sebagai metadata yang masih terlihat.
- Untuk `enc_metadata`, klien memetakan ke bilangan bulat lewat `numeric_id` di katalog: `NONE`=0, `DISC10`=1, `FLAT500`=2, `COMBO`=3; produk `P001`=1, `P002`=2, dst. Vektor: `[product_num, quantity, discount_num]` di slot 0..2.
- Pemulihan: `round(nilai)`; bila `|nilai - round(nilai)| > 0.25` klien menampilkan peringatan "metadata tidak pasti".

## Toleransi dan batas nilai (S0.6)
- Galat uji uang: `e_abs <= 1e-2` sebagai awal; ketatkan menjadi `1e-3` setelah A1 mengukur galat nyata. Dasar acuan memakai `math/big`. `e_rel` tidak terdefinisi bila acuan = 0.
- Batas nilai: pada level terendah, `|nilai| * scale < q0/2`. Dengan scale 2^40 dan q0 60 bit, |nilai| maksimum sekitar 5,2 x 10^5. Nilai lebih besar bisa overflow **tanpa error** (hasil salah). Klien menolak topup/transaksi di atas batas (default 500000).

## Pertanyaan terbuka untuk QnA
1. Tabel 2.4 menulis masukan `Transact` = "ID pengguna dan ciphertext nominal pengisian saldo", sedangkan Algoritma 1 membutuhkan `c_q`, `c_m`, harga, diskon. Kontrak ini mengikuti Algoritma 1.
2. Apakah serialisasi biner bawaan pustaka untuk `params`, `public_key`, `relin_key` terpisah sudah memenuhi Tabel 2.3?
3. Apakah `409` untuk aktivasi ganda dapat diterima?
