# Parameter CKKS (S0.5)

Status: **p1 and p2 verified in a spike** (Lattigo v6.2.0, native and WebAssembly): both parameter sets are accepted by the library and the reference computation 12500 x 2 x 0.9 = 22500 stays within 1.4e-4 (see the table at the end). A1.1 (#18) still formalizes the loader and re-checks them against the pinned version.

- Pustaka: **Lattigo** (Go) skema CKKS, NTT bawaan pustaka. Versi dipin di `go.mod` saat A1 dimulai (usulan: v6).
- Konfigurasi terpisah dari kode: `config/profiles.yaml` + env `CKKS_PROFILE=p1|p2`.
- Pemeriksaan keamanan bawaan pustaka tetap aktif; jangan menonaktifkannya.

## Profil
| Profil | LogN | Slot (N/2) | LogQ | LogP | LogDefaultScale | Level maks | Kegunaan |
|---|---|---|---|---|---|---|---|
| `p1` | 13 | 4096 | [60, 40, 40] | [61] | 40 | 2 | profil utama (tepat 2 perkalian) |
| `p2` | 14 | 8192 | [60, 40, 40, 40] | [61] | 40 | 3 | profil pembanding (1 level cadangan) |

Total bit modulus `p1` = 60+40+40+61 = 201 dan `p2` = 60+40+40+40+61 = 241; keduanya di bawah batas 128-bit yang umum (LogN=13: 218; LogN=14: 438) untuk parameter ini. **Perlu dikonfirmasi** oleh pemeriksaan pustaka.

## Alasan
- Minimal **dua perkalian beruntun tanpa bootstrapping**: `(p x q)` lalu `x (1-d)`. Setiap perkalian memakai satu level (relinearization + rescaling), sehingga dibutuhkan >= 3 prime Q (`p1`).
- `q0 = 60 bit` agar nilai pada level terendah muat: `|nilai| * 2^40 < q0/2` -> |nilai| maks sekitar 5,2 x 10^5. Dengan q0 = 55 bit batasnya hanya 16.384 (saldo 100000 akan overflow).
- `p2` dengan LogN lebih besar untuk perbandingan presisi, latensi, dan ukuran data di benchmark (C3).

## Rencana level
| Langkah | Level hasil |
|---|---|
| ciphertext segar (`c_q`, `c_p`, `c_d`, `c_k`, `c_b`) | L |
| `c_u = c_p (x) c_q` | L-1 |
| `c_v = c_u (x) c_d` | L-2 |
| `c_t = c_v (-) c_k`, `c_b' = c_b (-) c_t` | L-2 |
Setelah transaksi pertama saldo tetap di level L-2; topup dan pengurangan tidak menambah kedalaman perkalian. Perlu `L >= 2`.

## Measured (spike, median of 5; WASM executed under Node, not yet in a real browser)
| | p1 native | p1 WASM | p2 native | p2 WASM |
|---|---|---|---|---|
| keygen (sk+pk) | 3.3 ms | 5.4 ms | 7.2 ms | 13 ms |
| relin key | 5.6 ms | 12 ms | 21 ms | 40 ms |
| encrypt | 3.9 ms | 9.7 ms | 9.0 ms | 30 ms |
| 2x mul+relin+rescale | 6.6 ms | 17 ms | 20 ms | 54 ms |
| decrypt | 0.4 ms | 0.95 ms | 7.7 ms | 32 ms |

Artifact sizes (binary, p1 / p2): pk 512 / 1280 KiB, relin key 1536 / 5120 KiB, ciphertext 384 / 1024 KiB. Max abs error vs 22500: 8.5e-5 / 1.4e-4. WASM binary about 12 MB raw, 2.9 MB gzip.

## Yang harus dibuktikan di A1
1. Profil terbentuk tanpa error keamanan di pustaka.
2. Contoh acuan: harga 12500, q=2, d=0.1, k=500 -> tagihan 22000, saldo 100000 -> ~78000.
3. Galat 20 transaksi berulang (menjadi dasar toleransi).
4. Perilaku penyelarasan level/skala operand pada `Add`/`Sub` (eksplisit vs otomatis) dan batas nilai (overflow).
