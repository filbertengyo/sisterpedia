# Konvensi Kerja (S0.7)

## Alur
1. Ambil sub-issue yang di-assign ke kamu; cek "Depends on" sudah selesai.
2. Buat branch dari `main`: `feat/<kode-issue>-<slug>`, mis. `feat/b3.1-evaluate`. Tipe lain: `fix/`, `test/`, `docs/`, `chore/`.
3. Commit kecil dan sering. Pesan commit gaya conventional:
   - `feat(crypto): tambah GenKeys dan ExportPublic`
   - `fix(server): tolak payload berisi secret key`
   - `test(transact): tambah E1-E5`
   - `docs: draf API_CONTRACT`, `chore: ...`
   Scope: `crypto`, `client`, `server`, `transact`, `web`, `tests`, `bench`, `ci`, `docs`.
4. Buka PR ke `main` dengan judul sama seperti commit utama dan `Closes #<nomor sub-issue>` di deskripsi. Centang DoD di sub-issue.
5. Minimal **1 review** dari anggota lain dan CI hijau sebelum merge (squash merge disarankan).

## Aturan keras (jangan dilanggar)
- Tidak ada secret key, `.env`, atau kunci privat nyata di repo. Satu-satunya kunci yang boleh masuk: fixture bertanda TEST ONLY di `tests/fixtures/keys/`.
- Kode server tidak boleh menyebut `SecretKey` atau `Decryptor`.
- Nilai sensitif hanya dalam bentuk ciphertext di request, respons, DB, dan log.
- Parameter CKKS hanya di `config/`, bukan di kode.

## Lokal
- Go: `go build ./...`, `go vet ./...`, `go test -race ./...` (di Windows jalankan lewat WSL).
- Web: `cd web && npm ci && npm run lint && npm run build`.
- Stack: `docker compose up --build`; GUI di `http://localhost:3000`, client service `:8001`, server `:8000`.
- Format: `gofmt` untuk Go; ESLint untuk web.
