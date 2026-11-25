# Ringkasan benchmark (Tabel 2.7)

Dibuat otomatis oleh `go run ./cmd/bench -report` pada 2026-11-25T06:12:27Z dari data mentah di `bench/results/raw/`.

**Prosedur.** Untuk setiap profil: 5 pengulangan pemanasan (tidak dicatat) lalu 30 pengukuran dengan masukan dan rangkaian operasi yang sama. Waktu diukur dengan jam monotonik (`time.Now`/`time.Since` di Go, `performance.now()` di Node). Median = nilai tengah (rata-rata dua nilai tengah bila n genap). **P95 nearest-rank** = elemen ke-⌈0,95·n⌉ dari data terurut dengan posisi mulai 1 (n = 30 → elemen ke-29).

Lingkungan: Intel(R) Core(TM) i5-10400F CPU @ 2.90GHz, 12 core logis, RAM 15.9 GiB, Microsoft Windows 11 Pro 10.0.26200 amd64, go1.27.4, github.com/tuneinsight/lattigo/v6 v6.2.0 (detail: `bench/ENVIRONMENT.md`).

## Waktu

### Pembangkitan kunci (klien, native Go)

Dipisahkan dari operasi layanan. `params` = pembentukan parameter (`NewParametersFromLiteral`, termasuk tabel NTT); `total` = jumlah keempat tahap pada run yang sama.

| Operasi | p1 median (ms) | p1 P95 (ms) | n | p2 median (ms) | p2 P95 (ms) | n |
|---|---|---|---|---|---|---|
| params | 3.532 | 4.356 | 30 | 4.636 | 5.693 | 30 |
| sk | 0.6777 | 0.9323 | 30 | 1.679 | 1.786 | 30 |
| pk | 1.006 | 1.419 | 30 | 2.740 | 2.891 | 30 |
| rlk | 3.526 | 4.118 | 30 | 11.938 | 12.670 | 30 |
| total | 8.929 | 9.812 | 30 | 21.089 | 22.287 | 30 |

### Kriptografi klien (native Go, tanpa jaringan)

`encode` dan `encrypt` diukur terpisah pada run yang sama; `encode+encrypt` = keduanya. Dekripsi `balance`/`total` pada level L-2 (hasil transaksi), `metadata` pada level L.

| Operasi | p1 median (ms) | p1 P95 (ms) | n | p2 median (ms) | p2 P95 (ms) | n |
|---|---|---|---|---|---|---|
| encode amount | 0.6473 | 1.148 | 30 | 1.648 | 2.378 | 30 |
| encrypt amount | 2.945 | 4.119 | 30 | 7.657 | 8.386 | 30 |
| encode+encrypt amount | 3.615 | 5.350 | 30 | 9.361 | 10.169 | 30 |
| encode quantity | 0.6525 | 1.162 | 30 | 1.685 | 2.215 | 30 |
| encrypt quantity | 2.871 | 4.048 | 30 | 7.703 | 8.458 | 30 |
| encode+encrypt quantity | 3.537 | 5.134 | 30 | 9.617 | 10.408 | 30 |
| encode metadata | 0.6594 | 1.234 | 30 | 1.674 | 2.131 | 30 |
| encrypt metadata | 2.894 | 4.802 | 30 | 7.671 | 8.425 | 30 |
| encode+encrypt metadata | 3.561 | 5.952 | 30 | 9.440 | 10.128 | 30 |
| decrypt balance | 0.0415 | 0.0582 | 30 | 0.1500 | 0.2873 | 30 |
| decode balance | 0.2501 | 0.4186 | 30 | 8.745 | 9.919 | 30 |
| decrypt+decode balance | 0.2911 | 0.4782 | 30 | 8.910 | 10.076 | 30 |
| decrypt total | 0.0371 | 0.0564 | 30 | 0.1485 | 0.1664 | 30 |
| decode total | 0.2182 | 0.3714 | 30 | 8.776 | 9.709 | 30 |
| decrypt+decode total | 0.2552 | 0.4251 | 30 | 8.929 | 9.858 | 30 |
| decrypt metadata | 0.1091 | 0.2406 | 30 | 0.3081 | 0.4451 | 30 |
| decode metadata | 5.479 | 6.712 | 30 | 12.133 | 13.682 | 30 |
| decrypt+decode metadata | 5.630 | 6.850 | 30 | 12.461 | 13.967 | 30 |

### Evaluasi server (tanpa jaringan dan basis data)

Jalur kode yang sama dengan server (`internal/server/transact`). `Add`=topup, `Sub`=pengurangan potongan dan saldo, `Mult`=perkalian ciphertext×ciphertext (tensor, derajat 2), `Relin`/`Rescale` terpisah. `encrypt_*` = enkripsi konstanta katalog dengan pk pengguna; `align_*` = penyelarasan level (DropLevel). `topup.total`/`transact.total` = seluruh evaluasi.

| Operasi | p1 median (ms) | p1 P95 (ms) | n | p2 median (ms) | p2 P95 (ms) | n |
|---|---|---|---|---|---|---|
| topup.align | 0.0530 | 0.0606 | 30 | 0.1268 | 0.1426 | 30 |
| topup.add | 0.0247 | 0.0349 | 30 | 0.0915 | 0.2793 | 30 |
| topup.total | 0.1433 | 0.2043 | 30 | 0.2855 | 0.4892 | 30 |
| encrypt_price | 3.639 | 4.850 | 30 | 9.243 | 10.569 | 30 |
| encrypt_discount | 3.583 | 4.511 | 30 | 9.340 | 11.118 | 30 |
| encrypt_fixed | 3.495 | 4.995 | 30 | 9.269 | 10.405 | 30 |
| mul_1 | 0.2689 | 0.3841 | 30 | 0.7733 | 0.9810 | 30 |
| relin_1 | 3.168 | 4.131 | 30 | 9.962 | 10.733 | 30 |
| rescale_1 | 0.6274 | 0.7288 | 30 | 1.759 | 2.194 | 30 |
| align_discount | 0.0518 | 0.1618 | 30 | 0.1326 | 0.3380 | 30 |
| mul_2 | 0.1528 | 0.4431 | 30 | 0.4928 | 0.7758 | 30 |
| relin_2 | 1.941 | 3.294 | 30 | 6.755 | 7.216 | 30 |
| rescale_2 | 0.4243 | 0.7663 | 30 | 1.337 | 1.646 | 30 |
| align_sub_fixed | 0.0518 | 0.0659 | 30 | 0.1311 | 0.1584 | 30 |
| sub_fixed | 0.0233 | 0.0356 | 30 | 0.0898 | 0.1204 | 30 |
| align_sub_balance | 0.0005 | 0.0007 | 30 | 0.0005 | 0.0007 | 30 |
| sub_balance | 0.0262 | 0.0346 | 30 | 0.0966 | 0.1152 | 30 |
| transact.total | 18.865 | 20.464 | 30 | 50.532 | 53.245 | 30 |

### End-to-end (enkripsi klien → HTTP → evaluasi + SQLite → dekripsi)

Klien Go native terhadap biner server nyata (proses terpisah, SQLite) melalui HTTP lokal.

| Operasi | p1 median (ms) | p1 P95 (ms) | n | p2 median (ms) | p2 P95 (ms) | n |
|---|---|---|---|---|---|---|
| topup | 25.719 | 64.784 | 30 | 88.832 | 209.7 | 30 |
| topup.client_encrypt | 3.961 | 4.765 | 30 | 10.619 | 11.583 | 30 |
| topup.client_decrypt | 6.192 | 6.834 | 30 | 13.998 | 15.047 | 30 |
| transact | 77.980 | 162.6 | 30 | 163.1 | 208.1 | 30 |
| transact.client_encrypt | 8.454 | 10.120 | 30 | 21.226 | 22.621 | 30 |
| transact.client_decrypt | 1.021 | 1.587 | 30 | 19.590 | 21.762 | 30 |

### Pembangkitan kunci (WebAssembly, Node)

Fungsi jembatan `genKeys` menggabungkan params+sk+pk+rlk dan serialisasi Base64 dalam satu panggilan; diukur gabungan.

| Operasi | p1 median (ms) | p1 P95 (ms) | n | p2 median (ms) | p2 P95 (ms) | n |
|---|---|---|---|---|---|---|
| genKeys | 35.696 | 40.901 | 30 | 104.0 | 111.3 | 30 |

### Kriptografi klien (WebAssembly, Node)

Fungsi jembatan menggabungkan decode Base64 kunci (di-cache), encode, enkripsi, dan serialisasi Base64 dalam satu panggilan.

| Operasi | p1 median (ms) | p1 P95 (ms) | n | p2 median (ms) | p2 P95 (ms) | n |
|---|---|---|---|---|---|---|
| encryptScalar amount | 17.684 | 20.571 | 30 | 44.127 | 48.208 | 30 |
| encryptScalar quantity | 17.570 | 20.758 | 30 | 43.890 | 46.802 | 30 |
| encryptVector metadata | 16.925 | 18.142 | 30 | 41.733 | 44.378 | 30 |
| decryptScalar balance | 2.895 | 3.138 | 30 | 40.569 | 41.745 | 30 |
| decryptScalar total | 2.741 | 3.256 | 30 | 52.714 | 66.621 | 30 |
| decryptVector metadata | 36.100 | 40.277 | 30 | 73.842 | 93.547 | 30 |

### WASM vs native (median)

| Operasi | p1 native (ms) | p1 WASM (ms) | rasio | p2 native (ms) | p2 WASM (ms) | rasio |
|---|---|---|---|---|---|---|
| pembangkitan kunci (params+sk+pk+rlk) | 8.929 | 35.696 | 4.00x | 21.089 | 104.0 | 4.93x |
| encode+enkripsi nominal | 3.615 | 17.684 | 4.89x | 9.361 | 44.127 | 4.71x |
| encode+enkripsi metadata (3 slot) | 3.561 | 16.925 | 4.75x | 9.440 | 41.733 | 4.42x |
| dekripsi+decode saldo | 0.2911 | 2.895 | 9.95x | 8.910 | 40.569 | 4.55x |
| dekripsi+decode metadata | 5.630 | 36.100 | 6.41x | 12.461 | 73.842 | 5.93x |

## Ukuran artefak (data hasil serialisasi)

Ukuran biner = keluaran `MarshalBinary`/`MarshalJSON` Lattigo; Base64 = panjang string di JSON. 1 KiB = 1024 byte.

| Artefak | p1 biner (byte) | p1 biner (KiB) | p1 Base64 (byte) | p1 Base64 (KiB) | p2 biner (byte) | p2 biner (KiB) | p2 Base64 (byte) | p2 Base64 (KiB) |
|---|---|---|---|---|---|---|---|---|
| crypto.wasm (raw / gzip) | 13038055 | 12732.5 | 3206676 | 3131.5 | 13038055 | 12732.5 | 3206676 | 3131.5 |
| params | 255 | 0.2 | 340 | 0.3 | 269 | 0.3 | 360 | 0.4 |
| secret_key | 262192 | 256.0 | 349592 | 341.4 | 655416 | 640.1 | 873888 | 853.4 |
| public_key | 524392 | 512.1 | 699192 | 682.8 | 1310840 | 1280.1 | 1747788 | 1706.8 |
| relin_key | 1573216 | 1536.3 | 2097624 | 2048.5 | 5243408 | 5120.5 | 6991212 | 6827.4 |
| ct_balance_fresh | 393566 | 384.3 | 524756 | 512.5 | 1048942 | 1024.4 | 1398592 | 1365.8 |
| ct_topup_amount | 393566 | 384.3 | 524756 | 512.5 | 1048942 | 1024.4 | 1398592 | 1365.8 |
| ct_quantity | 393566 | 384.3 | 524756 | 512.5 | 1048942 | 1024.4 | 1398592 | 1365.8 |
| ct_metadata | 393566 | 384.3 | 524756 | 512.5 | 1048942 | 1024.4 | 1398592 | 1365.8 |
| ct_total | 131390 | 128.3 | 175188 | 171.1 | 524622 | 512.3 | 699496 | 683.1 |
| ct_balance_after_transact | 131390 | 128.3 | 175188 | 171.1 | 524622 | 512.3 | 699496 | 683.1 |
| json_activate_request | 3321996 | 3244.1 | 3321996 | 3244.1 | 10138036 | 9900.4 | 10138036 | 9900.4 |
| json_topup_request | 524773 | 512.5 | 524773 | 512.5 | 1398609 | 1365.8 | 1398609 | 1365.8 |
| json_transact_request | 1049593 | 1025.0 | 1049593 | 1025.0 | 2797265 | 2731.7 | 2797265 | 2731.7 |
| json_transact_response | 350455 | 342.2 | 350455 | 342.2 | 1399071 | 1366.3 | 1399071 | 1366.3 |

## Perbandingan otomatis p2 terhadap p1 (rasio median)

| Besaran | Rasio |
|---|---|
| pembangkitan kunci total | 2.36x |
| relinearization key | 3.39x |
| encode+enkripsi | 2.59x |
| dekripsi saldo | 30.61x |
| Mult | 2.88x |
| Relin | 3.14x |
| Rescale | 2.80x |
| evaluasi Transact | 2.68x |
| end-to-end Transact | 2.09x |
| ukuran public key | 2.50x |
| ukuran relinearization key | 3.33x |
| ukuran ciphertext saldo (level L) | 2.67x |

Lingkungan WASM: `{"go_wasm":"sisterpedia-bridge/1 github.com/tuneinsight/lattigo/v6 v6.2.0 sisterpedia-ckks/1 (lattigo v6 binary)","node":"v24.11.1","platform":"win32/x64","v8":"13.6.233.10-node.28"}`
