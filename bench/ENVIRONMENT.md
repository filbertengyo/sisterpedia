# Lingkungan benchmark

Ditulis otomatis oleh `go run ./cmd/bench`. Angka di `bench/results/` hanya berlaku untuk lingkungan ini.

| Komponen | Nilai |
|---|---|
| Waktu pengukuran (UTC) | 2026-11-25T06:11:41Z |
| CPU | Intel(R) Core(TM) i5-10400F CPU @ 2.90GHz |
| Core logis | 12 |
| RAM | 15.9 GiB (17071464448 byte) |
| Sistem operasi | Microsoft Windows 11 Pro 10.0.26200 (windows/amd64) |
| Bahasa / runtime | Go go1.27.4, GOMAXPROCS=12 |
| Pustaka CKKS | github.com/tuneinsight/lattigo/v6 v6.2.0 (NTT bawaan) |
| Commit | 2335e4b (perubahan belum di-commit: true) |
| WASM node | v24.11.1 |
| WASM v8 | 13.6.233.10-node.28 |
| WASM platform | win32/x64 |
| WASM go_wasm | sisterpedia-bridge/1 github.com/tuneinsight/lattigo/v6 v6.2.0 sisterpedia-ckks/1 (lattigo v6 binary) |

Perintah ulang: `make bench` (Linux/macOS/WSL) atau `scripts/run_bench.ps1` (Windows).
