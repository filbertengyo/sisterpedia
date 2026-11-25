# Hasil pengujian U01-U12 dan rangkaian transaksi

Dibuat otomatis oleh `go test -tags e2e ./tests/e2e/` pada 2026-11-25T09:09:06Z.

- Profil: p1, p2; pustaka github.com/tuneinsight/lattigo/v6 v6.2.0; Go go1.27.4, windows/amd64
- Server: biner `cmd/server` sebagai proses terpisah, SQLite sementara; klien: `pkg/ckks` (kode yang sama dengan WASM di browser) di proses uji.
- **Toleransi galat uang (ditetapkan sebelum uji): e_abs <= 0.01**; metadata: |nilai - bulat| <= 0.25. Galat relatif e_rel = e_abs/|y| dan ditulis `n/a` bila y = 0.
- Acuan dihitung dengan `math/big` (rasional eksak); nilai aktual adalah hasil dekripsi **sebelum** pembulatan.
- Ringkasan: **24/24 skenario lulus**.

## Ringkasan

| ID | Profil | Kategori | Diharapkan | Aktual | Lulus |
|---|---|---|---|---|---|
| U01 | p1 | Aktivasi dan saldo awal | 201 Created, CheckBalance didekripsi mendekati 0 (e_abs <= 0.01, e_rel n/a) | saldo mentah -6.538904584e-10 (e_abs 6.539e-10, e_rel n/a) | ya |
| U01 | p2 | Aktivasi dan saldo awal | 201 Created, CheckBalance didekripsi mendekati 0 (e_abs <= 0.01, e_rel n/a) | saldo mentah -1.045642559e-09 (e_abs 1.046e-09, e_rel n/a) | ya |
| U02 | p1 | Persistensi kunci dan sesi | Setelah server dan klien dijalankan ulang serta kunci lama diimpor: saldo 53000 dan 1 riwayat tetap terdekripsi; tidak ada pembangkitan kunci baru | saldo 53000.000003, riwayat 1 entri (true), kunci sama: true | ya |
| U02 | p2 | Persistensi kunci dan sesi | Setelah server dan klien dijalankan ulang serta kunci lama diimpor: saldo 53000 dan 1 riwayat tetap terdekripsi; tidak ada pembangkitan kunci baru | saldo 52999.999895, riwayat 1 entri (true), kunci sama: true | ya |
| U03 | p1 | Isolasi antarpengguna | Saldo A dan B terpisah dan benar dengan kuncinya masing-masing; sk B pada ciphertext A tidak menghasilkan 12345.67 | A=12345.670000, B=-4500.000003; sk B pada ciphertext A -> 1.63105e+31 (salah, tanpa pesan galat) | ya |
| U03 | p2 | Isolasi antarpengguna | Saldo A dan B terpisah dan benar dengan kuncinya masing-masing; sk B pada ciphertext A tidak menghasilkan 12345.67 | A=12345.670000, B=-4499.999999; sk B pada ciphertext A -> 1.4906e+43 (salah, tanpa pesan galat) | ya |
| U04 | p1 | Topup berulang | Saldo hasil dekripsi pada setiap langkah = jumlah acuan (e_abs <= 0.01) | saldo akhir 148181.25000000 (acuan 148181.25), max e_abs 1.630e-09 | ya |
| U04 | p2 | Topup berulang | Saldo hasil dekripsi pada setiap langkah = jumlah acuan (e_abs <= 0.01) | saldo akhir 148181.25000000 (acuan 148181.25), max e_abs 3.507e-09 | ya |
| U05 | p1 | Transaksi dan riwayat | Tagihan t=(p*q*(1-d))-k dan saldo sesuai acuan di setiap langkah; GetTransact memuat semua transaksi dengan metadata (produk, qty, diskon) dan total yang benar | 4 transaksi tercatat, metadata cocok: true, max e_abs 4.529e-05 | ya |
| U05 | p2 | Transaksi dan riwayat | Tagihan t=(p*q*(1-d))-k dan saldo sesuai acuan di setiap langkah; GetTransact memuat semua transaksi dengan metadata (produk, qty, diskon) dan total yang benar | 4 transaksi tercatat, metadata cocok: true, max e_abs 7.863e-06 | ya |
| U06 | p1 | Saldo tidak mencukupi | Server tetap mengurangi saldo (tanpa pemeriksaan homomorfik); klien mendeteksi saldo negatif setelah dekripsi | transaksi diterima, saldo -7500.000010, negatif terdeteksi: true | ya |
| U06 | p2 | Saldo tidak mencukupi | Server tetap mengurangi saldo (tanpa pemeriksaan homomorfik); klien mendeteksi saldo negatif setelah dekripsi | transaksi diterima, saldo -7499.999998, negatif terdeteksi: true | ya |
| U07 | p1 | Batas kedalaman perkalian | Perkalian ke-1..2 berhasil dalam toleransi; perkalian ke-3 melewati kapasitas: pustaka menolak rescale atau hasil di luar toleransi | 2 perkalian berhasil dalam toleransi; perkalian ke-3: ditolak pustaka=true, hasil di luar toleransi=true | ya |
| U07 | p2 | Batas kedalaman perkalian | Perkalian ke-1..3 berhasil dalam toleransi; perkalian ke-4 melewati kapasitas: pustaka menolak rescale atau hasil di luar toleransi | 3 perkalian berhasil dalam toleransi; perkalian ke-4: ditolak pustaka=true, hasil di luar toleransi=true | ya |
| U08 | p1 | Muatan dan ciphertext tidak valid | Kasus a-c: HTTP 400 dengan kode sesuai kontrak dan saldo/riwayat tidak berubah; kasus d: ciphertext termodifikasi tetap diterima (tidak terdeteksi) dan saldo menjadi salah | a-c: 11/11 kasus sesuai, data tidak berubah: true; d: diterima=true, saldo salah (e_abs 1.119e+31) | ya |
| U08 | p2 | Muatan dan ciphertext tidak valid | Kasus a-c: HTTP 400 dengan kode sesuai kontrak dan saldo/riwayat tidak berubah; kasus d: ciphertext termodifikasi tetap diterima (tidak terdeteksi) dan saldo menjadi salah | a-c: 11/11 kasus sesuai, data tidak berubah: true; d: diterima=true, saldo salah (e_abs 3.043e+43) | ya |
| U09 | p1 | Kunci evaluasi berbeda | Pustaka tidak dapat mendeteksi rlk yang salah; hasil dekripsi dengan sk A tidak sesuai acuan (dicatat nilainya) | terisolasi: 3.73751e+07 (e_abs 3.735e+07); via server: aktivasi diterima=true, total 3.19513e+07 (e_abs 3.193e+07) | ya |
| U09 | p2 | Kunci evaluasi berbeda | Pustaka tidak dapat mendeteksi rlk yang salah; hasil dekripsi dengan sk A tidak sesuai acuan (dicatat nilainya) | terisolasi: -1.19771e+19 (e_abs 1.198e+19); via server: aktivasi diterima=true, total -5.68134e+18 (e_abs 5.681e+18) | ya |
| U10 | p1 | Akun belum aktif | CheckBalance, Topup, Transact, GetTransact -> HTTP 404 account_not_found (pilihan kontrak: 404) | 4/4 panggilan mengembalikan 404 account_not_found | ya |
| U10 | p2 | Akun belum aktif | CheckBalance, Topup, Transact, GetTransact -> HTTP 404 account_not_found (pilihan kontrak: 404) | 4/4 panggilan mengembalikan 404 account_not_found | ya |
| U11 | p1 | Batas nilai (overflow) | Penjaga klien menolak > batas; di bawah batas hasil benar; di atas batas (vektor penuh 600000, skalar 1e10) hasil salah TANPA pesan galat | penjaga menolak: true; vektor penuh 600000 -> -448576; skalar 1e10 -> 4.76811e+08 | ya |
| U11 | p2 | Batas nilai (overflow) | Penjaga klien menolak > batas; di bawah batas hasil benar; di atas batas (vektor penuh 600000, skalar 1e10) hasil salah TANPA pesan galat | penjaga menolak: true; vektor penuh 600000 -> -448576; skalar 1e10 -> 1.23364e+08 | ya |
| U12 | p1 | Permintaan serentak | Semua permintaan berhasil; saldo akhir = 20000 + 10000 - 15000 = 15000; riwayat berisi 3 transaksi (tidak ada update yang hilang) | gagal 0, saldo 15000.000003, riwayat 3 entri | ya |
| U12 | p2 | Permintaan serentak | Semua permintaan berhasil; saldo akhir = 20000 + 10000 - 15000 = 15000; riwayat berisi 3 transaksi (tidak ada update yang hilang) | gagal 0, saldo 15000.000050, riwayat 3 entri | ya |

## Rangkaian transaksi (>= 5 per profil)

Setiap perubahan saldo dibandingkan dengan acuan (`chains.csv`); total tagihan di `chains_totals.csv`.

| Profil | Rangkaian | Operasi | Topup | Tanpa diskon | Dengan diskon | max e_abs | max e_rel | Lulus |
|---|---|---|---|---|---|---|---|---|
| p1 | C1-campuran | 6 | 2 | 1 | 3 | 3.715e-05 | 1.256e-09 | ya |
| p1 | C2-desimal | 6 | 3 | 1 | 2 | 4.567e-05 | 6.436e-09 | ya |
| p1 | C3-nilai-besar | 5 | 2 | 1 | 2 | 9.831e-05 | 1.314e-08 | ya |
| p1 | C4-panjang | 20 | 10 | 0 | 10 | 1.706e-05 | 1.958e-09 | ya |
| p1 | C5-saldo-negatif | 5 | 2 | 1 | 2 | 2.942e-06 | 8.554e-10 | ya |
| p2 | C1-campuran | 6 | 2 | 1 | 3 | 9.904e-05 | 4.564e-09 | ya |
| p2 | C2-desimal | 6 | 3 | 1 | 2 | 9.850e-05 | 1.388e-08 | ya |
| p2 | C3-nilai-besar | 5 | 2 | 1 | 2 | 1.908e-04 | 1.540e-08 | ya |
| p2 | C4-panjang | 20 | 10 | 0 | 10 | 3.374e-05 | 6.782e-09 | ya |
| p2 | C5-saldo-negatif | 5 | 2 | 1 | 2 | 1.397e-05 | 1.012e-08 | ya |

## Detail skenario

### U01 (p1) — Aktivasi dan saldo awal

- **Masukan:** user_id=u01-p1-546200, saldo awal Enc(0)
- **Diharapkan:** 201 Created, CheckBalance didekripsi mendekati 0 (e_abs <= 0.01, e_rel n/a)
- **Aktual:** saldo mentah -6.538904584e-10 (e_abs 6.539e-10, e_rel n/a)
- **Lulus:** true
- **Langkah:**
  1. Klien membangkitkan sk, pk, rlk (p1) dan mengenkripsi 0; POST /accounts tanpa secret key -> 201
  2. GET /accounts/u01-p1-546200/balance -> ciphertext; dekripsi di klien

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo dari respons Activate | 0.00 | -0.00000000 | 6.539e-10 | n/a | true |
| saldo dari CheckBalance | 0.00 | -0.00000000 | 6.539e-10 | n/a | true |

### U01 (p2) — Aktivasi dan saldo awal

- **Masukan:** user_id=u01-p2-546200, saldo awal Enc(0)
- **Diharapkan:** 201 Created, CheckBalance didekripsi mendekati 0 (e_abs <= 0.01, e_rel n/a)
- **Aktual:** saldo mentah -1.045642559e-09 (e_abs 1.046e-09, e_rel n/a)
- **Lulus:** true
- **Langkah:**
  1. Klien membangkitkan sk, pk, rlk (p2) dan mengenkripsi 0; POST /accounts tanpa secret key -> 201
  2. GET /accounts/u01-p2-546200/balance -> ciphertext; dekripsi di klien

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo dari respons Activate | 0.00 | -0.00000000 | 1.046e-09 | n/a | true |
| saldo dari CheckBalance | 0.00 | -0.00000000 | 1.046e-09 | n/a | true |

### U02 (p1) — Persistensi kunci dan sesi

- **Masukan:** user_id=u02-p1-546200; topup 75000; beli P001 x2 COMBO; ekspor kunci ke params.json/secret.key/public.key/relin.key
- **Diharapkan:** Setelah server dan klien dijalankan ulang serta kunci lama diimpor: saldo 53000 dan 1 riwayat tetap terdekripsi; tidak ada pembangkitan kunci baru
- **Aktual:** saldo 53000.000003, riwayat 1 entri (true), kunci sama: true
- **Lulus:** true
- **Langkah:**
  1. Aktivasi, topup 75000, transaksi P001 x2 COMBO; ekspor kunci ke berkas: params.json, public.key, relin.key, secret.key
  2. Server dihentikan (PID 5792) dan dijalankan ulang (PID 19700) dengan basis data SQLite yang sama
  3. Klien dijalankan ulang: kunci dimuat dari berkas (validasi pasangan kunci), tanpa keygen; public key identik: true
  4. CheckBalance dan GetTransact setelah restart; riwayat: 1 entri

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo setelah restart | 53000.00 | 53000.00000333 | 3.331e-06 | 6.286e-11 | true |
| total riwayat setelah restart | 22000.00 | 21999.99999667 | 3.330e-06 | 1.514e-10 | true |

### U02 (p2) — Persistensi kunci dan sesi

- **Masukan:** user_id=u02-p2-546200; topup 75000; beli P001 x2 COMBO; ekspor kunci ke params.json/secret.key/public.key/relin.key
- **Diharapkan:** Setelah server dan klien dijalankan ulang serta kunci lama diimpor: saldo 53000 dan 1 riwayat tetap terdekripsi; tidak ada pembangkitan kunci baru
- **Aktual:** saldo 52999.999895, riwayat 1 entri (true), kunci sama: true
- **Lulus:** true
- **Langkah:**
  1. Aktivasi, topup 75000, transaksi P001 x2 COMBO; ekspor kunci ke berkas: params.json, public.key, relin.key, secret.key
  2. Server dihentikan (PID 19700) dan dijalankan ulang (PID 20184) dengan basis data SQLite yang sama
  3. Klien dijalankan ulang: kunci dimuat dari berkas (validasi pasangan kunci), tanpa keygen; public key identik: true
  4. CheckBalance dan GetTransact setelah restart; riwayat: 1 entri

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo setelah restart | 53000.00 | 52999.99989523 | 1.048e-04 | 1.977e-09 | true |
| total riwayat setelah restart | 22000.00 | 22000.00010477 | 1.048e-04 | 4.762e-09 | true |

### U03 (p1) — Isolasi antarpengguna

- **Masukan:** akun A=u03a-p1-546200 topup 12345.67; akun B=u03b-p1-546200 topup 500 lalu beli P005 x1
- **Diharapkan:** Saldo A dan B terpisah dan benar dengan kuncinya masing-masing; sk B pada ciphertext A tidak menghasilkan 12345.67
- **Aktual:** A=12345.670000, B=-4500.000003; sk B pada ciphertext A -> 1.63105e+31 (salah, tanpa pesan galat)
- **Lulus:** true
- **Langkah:**
  1. Dua akun dengan pasangan kunci berbeda; transaksi masing-masing
  2. Ciphertext saldo A didekripsi dengan sk B: hasil 1.63105e+31

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo A dengan sk A | 12345.67 | 12345.67000000 | 2.782e-10 | 2.254e-14 | true |
| saldo B dengan sk B | -4500.00 | -4500.00000297 | 2.975e-06 | 6.610e-10 | true |

> Dekripsi dengan kunci yang salah tidak menimbulkan galat dari pustaka; hasilnya angka acak (e_abs 1.631e+31 terhadap 12345.67).

### U03 (p2) — Isolasi antarpengguna

- **Masukan:** akun A=u03a-p2-546200 topup 12345.67; akun B=u03b-p2-546200 topup 500 lalu beli P005 x1
- **Diharapkan:** Saldo A dan B terpisah dan benar dengan kuncinya masing-masing; sk B pada ciphertext A tidak menghasilkan 12345.67
- **Aktual:** A=12345.670000, B=-4499.999999; sk B pada ciphertext A -> 1.4906e+43 (salah, tanpa pesan galat)
- **Lulus:** true
- **Langkah:**
  1. Dua akun dengan pasangan kunci berbeda; transaksi masing-masing
  2. Ciphertext saldo A didekripsi dengan sk B: hasil 1.4906e+43

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo A dengan sk A | 12345.67 | 12345.67000000 | 3.636e-09 | 2.945e-13 | true |
| saldo B dengan sk B | -4500.00 | -4499.99999873 | 1.268e-06 | 2.817e-10 | true |

> Dekripsi dengan kunci yang salah tidak menimbulkan galat dari pustaka; hasilnya angka acak (e_abs 1.491e+43 terhadap 12345.67).

### U04 (p1) — Topup berulang

- **Masukan:** topup berturut-turut: 100000, 2500.75, 0.5, 33333.33, 1, 12345.67
- **Diharapkan:** Saldo hasil dekripsi pada setiap langkah = jumlah acuan (e_abs <= 0.01)
- **Aktual:** saldo akhir 148181.25000000 (acuan 148181.25), max e_abs 1.630e-09
- **Lulus:** true
- **Langkah:**
  1. 6 kali POST /accounts/u04-p1-546200/topup dengan Enc(nominal); dekripsi saldo setelah setiap langkah

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo setelah topup #1 (+100000) | 100000.00 | 100000.00000000 | 6.694e-10 | 6.694e-15 | true |
| saldo setelah topup #2 (+2500.75) | 102500.75 | 102500.75000000 | 7.567e-10 | 7.382e-15 | true |
| saldo setelah topup #3 (+0.5) | 102501.25 | 102501.25000000 | 3.783e-10 | 3.691e-15 | true |
| saldo setelah topup #4 (+33333.33) | 135834.58 | 135834.58000000 | 5.402e-10 | 3.977e-15 | true |
| saldo setelah topup #5 (+1) | 135835.58 | 135835.58000000 | 3.946e-10 | 2.905e-15 | true |
| saldo setelah topup #6 (+12345.67) | 148181.25 | 148181.25000000 | 1.630e-09 | 1.100e-14 | true |

### U04 (p2) — Topup berulang

- **Masukan:** topup berturut-turut: 100000, 2500.75, 0.5, 33333.33, 1, 12345.67
- **Diharapkan:** Saldo hasil dekripsi pada setiap langkah = jumlah acuan (e_abs <= 0.01)
- **Aktual:** saldo akhir 148181.25000000 (acuan 148181.25), max e_abs 3.507e-09
- **Lulus:** true
- **Langkah:**
  1. 6 kali POST /accounts/u04-p2-546200/topup dengan Enc(nominal); dekripsi saldo setelah setiap langkah

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo setelah topup #1 (+100000) | 100000.00 | 100000.00000000 | 3.507e-09 | 3.507e-14 | true |
| saldo setelah topup #2 (+2500.75) | 102500.75 | 102500.75000000 | 3.492e-09 | 3.407e-14 | true |
| saldo setelah topup #3 (+0.5) | 102501.25 | 102501.25000000 | 3.201e-09 | 3.123e-14 | true |
| saldo setelah topup #4 (+33333.33) | 135834.58 | 135834.58000000 | 2.257e-09 | 1.662e-14 | true |
| saldo setelah topup #5 (+1) | 135835.58 | 135835.58000000 | 1.821e-09 | 1.340e-14 | true |
| saldo setelah topup #6 (+12345.67) | 148181.25 | 148181.25000000 | 1.397e-09 | 9.428e-15 | true |

### U05 (p1) — Transaksi dan riwayat

- **Masukan:** topup 100000; P002 x1 NONE (tanpa diskon); P003 x2 DISC10 (diskon persentase); P004 x1 FLAT500 (potongan tetap); P001 x2 COMBO (kombinasi (acuan spesifikasi: 22000))
- **Diharapkan:** Tagihan t=(p*q*(1-d))-k dan saldo sesuai acuan di setiap langkah; GetTransact memuat semua transaksi dengan metadata (produk, qty, diskon) dan total yang benar
- **Aktual:** 4 transaksi tercatat, metadata cocok: true, max e_abs 4.529e-05
- **Lulus:** true
- **Langkah:**
  1. Topup lalu 4 transaksi (POST /accounts/u05-p1-546200/transactions; product_id dan discount_code terbuka, kuantitas & metadata terenkripsi)
  2. GET /accounts/u05-p1-546200/transactions: 4 entri, metadata dibulatkan ke bilangan bulat terdekat (toleransi 0.25)

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| tagihan P002 x1 NONE | 15000.00 | 14999.99998980 | 1.020e-05 | 6.802e-10 | true |
| saldo setelah transaksi #1 | 85000.00 | 85000.00001020 | 1.020e-05 | 1.200e-10 | true |
| tagihan P003 x2 DISC10 | 31500.00 | 31500.00004529 | 4.529e-05 | 1.438e-09 | true |
| saldo setelah transaksi #2 | 53500.00 | 53499.99996491 | 3.509e-05 | 6.559e-10 | true |
| tagihan P004 x1 FLAT500 | 8500.00 | 8499.99998611 | 1.389e-05 | 1.634e-09 | true |
| saldo setelah transaksi #3 | 45000.00 | 44999.99997880 | 2.120e-05 | 4.711e-10 | true |
| tagihan P001 x2 COMBO | 22000.00 | 22000.00001858 | 1.858e-05 | 8.443e-10 | true |
| saldo setelah transaksi #4 | 23000.00 | 22999.99996022 | 3.978e-05 | 1.729e-09 | true |
| riwayat #1 total (P002 x1 NONE, metadata mentah 2.000000/1.000000/-0.000000) | 15000.00 | 14999.99998980 | 1.020e-05 | 6.802e-10 | true |
| riwayat #2 total (P003 x2 DISC10, metadata mentah 3.000000/2.000000/1.000000) | 31500.00 | 31500.00004529 | 4.529e-05 | 1.438e-09 | true |
| riwayat #3 total (P004 x1 FLAT500, metadata mentah 4.000000/1.000000/2.000000) | 8500.00 | 8499.99998611 | 1.389e-05 | 1.634e-09 | true |
| riwayat #4 total (P001 x2 COMBO, metadata mentah 1.000000/2.000000/3.000000) | 22000.00 | 22000.00001858 | 1.858e-05 | 8.443e-10 | true |

### U05 (p2) — Transaksi dan riwayat

- **Masukan:** topup 100000; P002 x1 NONE (tanpa diskon); P003 x2 DISC10 (diskon persentase); P004 x1 FLAT500 (potongan tetap); P001 x2 COMBO (kombinasi (acuan spesifikasi: 22000))
- **Diharapkan:** Tagihan t=(p*q*(1-d))-k dan saldo sesuai acuan di setiap langkah; GetTransact memuat semua transaksi dengan metadata (produk, qty, diskon) dan total yang benar
- **Aktual:** 4 transaksi tercatat, metadata cocok: true, max e_abs 7.863e-06
- **Lulus:** true
- **Langkah:**
  1. Topup lalu 4 transaksi (POST /accounts/u05-p2-546200/transactions; product_id dan discount_code terbuka, kuantitas & metadata terenkripsi)
  2. GET /accounts/u05-p2-546200/transactions: 4 entri, metadata dibulatkan ke bilangan bulat terdekat (toleransi 0.25)

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| tagihan P002 x1 NONE | 15000.00 | 15000.00000097 | 9.688e-07 | 6.459e-11 | true |
| saldo setelah transaksi #1 | 85000.00 | 84999.99999903 | 9.696e-07 | 1.141e-11 | true |
| tagihan P003 x2 DISC10 | 31500.00 | 31499.99999646 | 3.539e-06 | 1.124e-10 | true |
| saldo setelah transaksi #2 | 53500.00 | 53500.00000257 | 2.570e-06 | 4.803e-11 | true |
| tagihan P004 x1 FLAT500 | 8500.00 | 8500.00000786 | 7.863e-06 | 9.251e-10 | true |
| saldo setelah transaksi #3 | 45000.00 | 44999.99999471 | 5.294e-06 | 1.176e-10 | true |
| tagihan P001 x2 COMBO | 22000.00 | 21999.99999608 | 3.923e-06 | 1.783e-10 | true |
| saldo setelah transaksi #4 | 23000.00 | 22999.99999863 | 1.370e-06 | 5.958e-11 | true |
| riwayat #1 total (P002 x1 NONE, metadata mentah 2.000000/1.000000/0.000000) | 15000.00 | 15000.00000097 | 9.688e-07 | 6.459e-11 | true |
| riwayat #2 total (P003 x2 DISC10, metadata mentah 3.000000/2.000000/1.000000) | 31500.00 | 31499.99999646 | 3.539e-06 | 1.124e-10 | true |
| riwayat #3 total (P004 x1 FLAT500, metadata mentah 4.000000/1.000000/2.000000) | 8500.00 | 8500.00000786 | 7.863e-06 | 9.251e-10 | true |
| riwayat #4 total (P001 x2 COMBO, metadata mentah 1.000000/2.000000/3.000000) | 22000.00 | 21999.99999608 | 3.923e-06 | 1.783e-10 | true |

### U06 (p1) — Saldo tidak mencukupi

- **Masukan:** topup 10000; beli P003 x1 NONE (tagihan di atas saldo)
- **Diharapkan:** Server tetap mengurangi saldo (tanpa pemeriksaan homomorfik); klien mendeteksi saldo negatif setelah dekripsi
- **Aktual:** transaksi diterima, saldo -7500.000010, negatif terdeteksi: true
- **Lulus:** true
- **Langkah:**
  1. POST transaksi diterima (201) meskipun tagihan 17500 > saldo 10000
  2. Klien mendekripsi saldo: -7500.000010 -> negatif terdeteksi: true (GUI menampilkan 'Saldo tidak mencukupi')

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| tagihan | 17500.00 | 17500.00001007 | 1.007e-05 | 5.755e-10 | true |
| saldo akhir | -7500.00 | -7500.00001007 | 1.007e-05 | 1.343e-09 | true |

### U06 (p2) — Saldo tidak mencukupi

- **Masukan:** topup 10000; beli P003 x1 NONE (tagihan di atas saldo)
- **Diharapkan:** Server tetap mengurangi saldo (tanpa pemeriksaan homomorfik); klien mendeteksi saldo negatif setelah dekripsi
- **Aktual:** transaksi diterima, saldo -7499.999998, negatif terdeteksi: true
- **Lulus:** true
- **Langkah:**
  1. POST transaksi diterima (201) meskipun tagihan 17500 > saldo 10000
  2. Klien mendekripsi saldo: -7499.999998 -> negatif terdeteksi: true (GUI menampilkan 'Saldo tidak mencukupi')

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| tagihan | 17500.00 | 17499.99999767 | 2.326e-06 | 1.329e-10 | true |
| saldo akhir | -7500.00 | -7499.99999768 | 2.324e-06 | 3.099e-10 | true |

### U07 (p1) — Batas kedalaman perkalian

- **Masukan:** x = 1.5 dikalikan berulang dengan Enc(1.5) (ciphertext x ciphertext + relin + rescale) sebanyak 3 kali; level maks 2
- **Diharapkan:** Perkalian ke-1..2 berhasil dalam toleransi; perkalian ke-3 melewati kapasitas: pustaka menolak rescale atau hasil di luar toleransi
- **Aktual:** 2 perkalian berhasil dalam toleransi; perkalian ke-3: ditolak pustaka=true, hasil di luar toleransi=true
- **Lulus:** true
- **Langkah:**
  1. Enc(1.5): level 2, skala 2^40.0000
  2. perkalian #1 mul -> level 2, skala 2^80.0000
  3. perkalian #1 relin -> level 2, skala 2^80.0000
  4. perkalian #1 rescale -> level 1, skala 2^40.0000
  5. perkalian #2 mul -> level 1, skala 2^80.0000
  6. perkalian #2 relin -> level 1, skala 2^80.0000
  7. perkalian #2 rescale -> level 0, skala 2^40.0000
  8. perkalian #3 mul -> level 0, skala 2^100.0000
  9. perkalian #3 relin -> level 0, skala 2^100.0000
  10. perkalian #3 DITOLAK pustaka: homomorphic evaluation failed: rescale: cannot Rescale: input Ciphertext level is too low
  11. tanpa rescale, skala 2^100.0 melebihi q0 (60 bit): dekripsi -1.31311e-11, acuan 5.062500

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| perkalian #1 (level 1) | 2.25 | 2.25000000 | 6.234e-11 | 2.771e-11 | true |
| perkalian #2 (level 0) | 3.38 | 3.37500000 | 1.143e-09 | 3.386e-10 | true |
| perkalian #3 tanpa rescale (level 0, skala 2^100.0) | 5.06 | -0.00000000 | 5.063e+00 | 1.000e+00 | false |

> Tanpa bootstrapping, setiap perkalian+rescale menghabiskan satu level; profil p1 mendukung tepat 2 perkalian beruntun (tagihan butuh 2).

### U07 (p2) — Batas kedalaman perkalian

- **Masukan:** x = 1.5 dikalikan berulang dengan Enc(1.5) (ciphertext x ciphertext + relin + rescale) sebanyak 4 kali; level maks 3
- **Diharapkan:** Perkalian ke-1..3 berhasil dalam toleransi; perkalian ke-4 melewati kapasitas: pustaka menolak rescale atau hasil di luar toleransi
- **Aktual:** 3 perkalian berhasil dalam toleransi; perkalian ke-4: ditolak pustaka=true, hasil di luar toleransi=true
- **Lulus:** true
- **Langkah:**
  1. Enc(1.5): level 3, skala 2^40.0000
  2. perkalian #1 mul -> level 3, skala 2^80.0000
  3. perkalian #1 relin -> level 3, skala 2^80.0000
  4. perkalian #1 rescale -> level 2, skala 2^40.0000
  5. perkalian #2 mul -> level 2, skala 2^80.0000
  6. perkalian #2 relin -> level 2, skala 2^80.0000
  7. perkalian #2 rescale -> level 1, skala 2^40.0000
  8. perkalian #3 mul -> level 1, skala 2^80.0000
  9. perkalian #3 relin -> level 1, skala 2^80.0000
  10. perkalian #3 rescale -> level 0, skala 2^40.0000
  11. perkalian #4 mul -> level 0, skala 2^100.0000
  12. perkalian #4 relin -> level 0, skala 2^100.0000
  13. perkalian #4 DITOLAK pustaka: homomorphic evaluation failed: rescale: cannot Rescale: input Ciphertext level is too low
  14. tanpa rescale, skala 2^100.0 melebihi q0 (60 bit): dekripsi -4.57811e-11, acuan 7.593750

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| perkalian #1 (level 2) | 2.25 | 2.25000000 | 1.341e-09 | 5.958e-10 | true |
| perkalian #2 (level 1) | 3.38 | 3.37500000 | 2.106e-09 | 6.241e-10 | true |
| perkalian #3 (level 0) | 5.06 | 5.06250001 | 8.058e-09 | 1.592e-09 | true |
| perkalian #4 tanpa rescale (level 0, skala 2^100.0) | 7.59 | -0.00000000 | 7.594e+00 | 1.000e+00 | false |

> Tanpa bootstrapping, setiap perkalian+rescale menghabiskan satu level; profil p2 mendukung tepat 3 perkalian beruntun (tagihan butuh 2).

### U08 (p1) — Muatan dan ciphertext tidak valid

- **Masukan:** JSON rusak, tipe salah, field tak dikenal, Base64 tidak valid, biner tak dapat dimuat, ciphertext profil lain, skala salah, secret key sebagai ciphertext; lalu ciphertext yang diubah 1 byte
- **Diharapkan:** Kasus a-c: HTTP 400 dengan kode sesuai kontrak dan saldo/riwayat tidak berubah; kasus d: ciphertext termodifikasi tetap diterima (tidak terdeteksi) dan saldo menjadi salah
- **Aktual:** a-c: 11/11 kasus sesuai, data tidak berubah: true; d: diterima=true, saldo salah (e_abs 1.119e+31)
- **Lulus:** true
- **Langkah:**
  1. a1 JSON rusak -> HTTP 400 invalid_json (diharapkan 400 invalid_json) OK
  2. a2 tipe salah (angka) -> HTTP 400 invalid_json (diharapkan 400 invalid_json) OK
  3. a3 field tak dikenal (amount) -> HTTP 400 invalid_json (diharapkan 400 invalid_json) OK
  4. a4 field wajib hilang -> HTTP 400 invalid_json (diharapkan 400 invalid_json) OK
  5. b1 Base64 tidak valid -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  6. b2 Base64 kosong -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  7. c1 biner acak -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  8. c2 ciphertext terpotong -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  9. c3 ciphertext profil p2 -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  10. c4 skala 2^30 (konteks salah) -> HTTP 400 context_mismatch (diharapkan 400 context_mismatch) OK
  11. c5 secret key sebagai ciphertext -> HTTP 400 secret_key_rejected (diharapkan 400 secret_key_rejected) OK
  12. Ciphertext saldo byte-per-byte sama: true; jumlah riwayat 1 -> 1
  13. d: akun terpisah u08d-p1-546200, Enc(500) diubah 1 byte (koefisien) -> diterima server: true; saldo didekripsi -1.11923e+31 (acuan 1500)

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo setelah semua penolakan | -4000.00 | -3999.99999985 | 1.505e-07 | 3.763e-11 | true |

> Validasi format hanya memeriksa struktur, level, dan skala; perubahan isi ciphertext yang tetap valid tidak terdeteksi karena CKKS tidak menyediakan autentikasi/integritas (diperlukan MAC/tanda tangan di luar CKKS).

### U08 (p2) — Muatan dan ciphertext tidak valid

- **Masukan:** JSON rusak, tipe salah, field tak dikenal, Base64 tidak valid, biner tak dapat dimuat, ciphertext profil lain, skala salah, secret key sebagai ciphertext; lalu ciphertext yang diubah 1 byte
- **Diharapkan:** Kasus a-c: HTTP 400 dengan kode sesuai kontrak dan saldo/riwayat tidak berubah; kasus d: ciphertext termodifikasi tetap diterima (tidak terdeteksi) dan saldo menjadi salah
- **Aktual:** a-c: 11/11 kasus sesuai, data tidak berubah: true; d: diterima=true, saldo salah (e_abs 3.043e+43)
- **Lulus:** true
- **Langkah:**
  1. a1 JSON rusak -> HTTP 400 invalid_json (diharapkan 400 invalid_json) OK
  2. a2 tipe salah (angka) -> HTTP 400 invalid_json (diharapkan 400 invalid_json) OK
  3. a3 field tak dikenal (amount) -> HTTP 400 invalid_json (diharapkan 400 invalid_json) OK
  4. a4 field wajib hilang -> HTTP 400 invalid_json (diharapkan 400 invalid_json) OK
  5. b1 Base64 tidak valid -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  6. b2 Base64 kosong -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  7. c1 biner acak -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  8. c2 ciphertext terpotong -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  9. c3 ciphertext profil p1 -> HTTP 400 invalid_payload (diharapkan 400 invalid_payload) OK
  10. c4 skala 2^30 (konteks salah) -> HTTP 400 context_mismatch (diharapkan 400 context_mismatch) OK
  11. c5 secret key sebagai ciphertext -> HTTP 400 secret_key_rejected (diharapkan 400 secret_key_rejected) OK
  12. Ciphertext saldo byte-per-byte sama: true; jumlah riwayat 1 -> 1
  13. d: akun terpisah u08d-p2-546200, Enc(500) diubah 1 byte (koefisien) -> diterima server: true; saldo didekripsi -3.04324e+43 (acuan 1500)

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo setelah semua penolakan | -4000.00 | -3999.99999710 | 2.899e-06 | 7.247e-10 | true |

> Validasi format hanya memeriksa struktur, level, dan skala; perubahan isi ciphertext yang tetap valid tidak terdeteksi karena CKKS tidak menyediakan autentikasi/integritas (diperlukan MAC/tanda tangan di luar CKKS).

### U09 (p1) — Kunci evaluasi berbeda

- **Masukan:** Evaluasi tagihan P001 x2 COMBO (acuan 22000) dengan pk akun A tetapi rlk dari pasangan kunci B
- **Diharapkan:** Pustaka tidak dapat mendeteksi rlk yang salah; hasil dekripsi dengan sk A tidak sesuai acuan (dicatat nilainya)
- **Aktual:** terisolasi: 3.73751e+07 (e_abs 3.735e+07); via server: aktivasi diterima=true, total 3.19513e+07 (e_abs 3.193e+07)
- **Lulus:** true
- **Langkah:**
  1. Evaluasi terisolasi dengan rlk B: galat pustaka=false; hasil dekripsi sk A = 3.73751e+07 (e_abs 3.735e+07)
  2. Aktivasi server dengan pk A + rlk B: diterima=true (format valid tidak membuktikan pasangan kunci)
  3. Topup tetap benar (tanpa relinearization); transaksi P001 x2 COMBO -> total didekripsi 3.19513e+07

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| kontrol: rlk A | 22000.00 | 22000.00000128 | 1.285e-06 | 5.840e-11 | true |

> Lattigo tidak memverifikasi bahwa rlk berasal dari sk yang sama; hanya pemeriksaan di klien (ValidateKeys sebelum aktivasi/impor) yang dapat mencegahnya.

### U09 (p2) — Kunci evaluasi berbeda

- **Masukan:** Evaluasi tagihan P001 x2 COMBO (acuan 22000) dengan pk akun A tetapi rlk dari pasangan kunci B
- **Diharapkan:** Pustaka tidak dapat mendeteksi rlk yang salah; hasil dekripsi dengan sk A tidak sesuai acuan (dicatat nilainya)
- **Aktual:** terisolasi: -1.19771e+19 (e_abs 1.198e+19); via server: aktivasi diterima=true, total -5.68134e+18 (e_abs 5.681e+18)
- **Lulus:** true
- **Langkah:**
  1. Evaluasi terisolasi dengan rlk B: galat pustaka=false; hasil dekripsi sk A = -1.19771e+19 (e_abs 1.198e+19)
  2. Aktivasi server dengan pk A + rlk B: diterima=true (format valid tidak membuktikan pasangan kunci)
  3. Topup tetap benar (tanpa relinearization); transaksi P001 x2 COMBO -> total didekripsi -5.68134e+18

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| kontrol: rlk A | 22000.00 | 22000.00007789 | 7.789e-05 | 3.541e-09 | true |

> Lattigo tidak memverifikasi bahwa rlk berasal dari sk yang sama; hanya pemeriksaan di klien (ValidateKeys sebelum aktivasi/impor) yang dapat mencegahnya.

### U10 (p1) — Akun belum aktif

- **Masukan:** user_id=u10-p1-546200 (tidak pernah diaktifkan)
- **Diharapkan:** CheckBalance, Topup, Transact, GetTransact -> HTTP 404 account_not_found (pilihan kontrak: 404)
- **Aktual:** 4/4 panggilan mengembalikan 404 account_not_found
- **Lulus:** true
- **Langkah:**
  1. CheckBalance -> HTTP 404 account_not_found: account "u10-p1-546200" is not activated
  2. Topup -> HTTP 404 account_not_found: account "u10-p1-546200" is not activated
  3. Transact -> HTTP 404 account_not_found: account "u10-p1-546200" is not activated
  4. GetTransact -> HTTP 404 account_not_found: account "u10-p1-546200" is not activated

### U10 (p2) — Akun belum aktif

- **Masukan:** user_id=u10-p2-546200 (tidak pernah diaktifkan)
- **Diharapkan:** CheckBalance, Topup, Transact, GetTransact -> HTTP 404 account_not_found (pilihan kontrak: 404)
- **Aktual:** 4/4 panggilan mengembalikan 404 account_not_found
- **Lulus:** true
- **Langkah:**
  1. CheckBalance -> HTTP 404 account_not_found: account "u10-p2-546200" is not activated
  2. Topup -> HTTP 404 account_not_found: account "u10-p2-546200" is not activated
  3. Transact -> HTTP 404 account_not_found: account "u10-p2-546200" is not activated
  4. GetTransact -> HTTP 404 account_not_found: account "u10-p2-546200" is not activated

### U11 (p1) — Batas nilai (overflow)

- **Masukan:** batas aman |x| <= q0/(2*skala) = 524287 (semua pola slot); penjaga klien 500000; nilai diturunkan ke level 0 lalu didekripsi
- **Diharapkan:** Penjaga klien menolak > batas; di bawah batas hasil benar; di atas batas (vektor penuh 600000, skalar 1e10) hasil salah TANPA pesan galat
- **Aktual:** penjaga menolak: true; vektor penuh 600000 -> -448576; skalar 1e10 -> 4.76811e+08
- **Lulus:** true
- **Langkah:**
  1. Penjaga klien EncryptScalar(600000): value out of range: slot 0 = 600000 (limit 524287)
  2. skalar 400000 (slot 0), level 0 -> slot 0 didekripsi 400000 (benar)
  3. vektor penuh 400000 (4096 slot), level 0 -> slot 0 didekripsi 400000 (benar)
  4. skalar 600000 (slot 0), level 0 -> slot 0 didekripsi 600000 (benar)
  5. vektor penuh 600000 (4096 slot), level 0 -> slot 0 didekripsi -448576 (SALAH, tanpa galat)
  6. skalar 1e10 (slot 0), level 0 -> slot 0 didekripsi 4.76811e+08 (SALAH, tanpa galat)

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| skalar 400000 (slot 0) pada level 0 | 400000.00 | 400000.00000000 | 0.000e+00 | 0.000e+00 | true |
| vektor penuh 400000 (4096 slot) pada level 0 | 400000.00 | 400000.00000000 | 5.821e-11 | 1.455e-16 | true |
| skalar 600000 (slot 0) pada level 0 | 600000.00 | 600000.00000000 | 6.985e-10 | 1.164e-15 | true |
| vektor penuh 600000 (4096 slot) pada level 0 | 600000.00 | -448575.99999998 | 1.049e+06 | 1.748e+00 | false |
| skalar 1e10 (slot 0) pada level 0 | 10000000000.00 | 476810794.04725504 | 9.523e+09 | 9.523e-01 | false |

> Overflow terjadi pada koefisien polinomial: |koefisien| >= q0/2 membungkus modulo q0 tanpa pesan galat. Untuk vektor penuh koefisien ke-0 = x*skala, sehingga batas 524287 berlaku umum; nilai di satu slot tersebar ke banyak koefisien (~x*skala*2/N) sehingga baru overflow jauh di atas batas. Klien memakai batas konservatif 500000 untuk saldo, nominal, dan subtotal.

### U11 (p2) — Batas nilai (overflow)

- **Masukan:** batas aman |x| <= q0/(2*skala) = 524287 (semua pola slot); penjaga klien 500000; nilai diturunkan ke level 0 lalu didekripsi
- **Diharapkan:** Penjaga klien menolak > batas; di bawah batas hasil benar; di atas batas (vektor penuh 600000, skalar 1e10) hasil salah TANPA pesan galat
- **Aktual:** penjaga menolak: true; vektor penuh 600000 -> -448576; skalar 1e10 -> 1.23364e+08
- **Lulus:** true
- **Langkah:**
  1. Penjaga klien EncryptScalar(600000): value out of range: slot 0 = 600000 (limit 524287)
  2. skalar 400000 (slot 0), level 0 -> slot 0 didekripsi 400000 (benar)
  3. vektor penuh 400000 (8192 slot), level 0 -> slot 0 didekripsi 400000 (benar)
  4. skalar 600000 (slot 0), level 0 -> slot 0 didekripsi 600000 (benar)
  5. vektor penuh 600000 (8192 slot), level 0 -> slot 0 didekripsi -448576 (SALAH, tanpa galat)
  6. skalar 1e10 (slot 0), level 0 -> slot 0 didekripsi 1.23364e+08 (SALAH, tanpa galat)

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| skalar 400000 (slot 0) pada level 0 | 400000.00 | 400000.00000000 | 3.201e-09 | 8.004e-15 | true |
| vektor penuh 400000 (8192 slot) pada level 0 | 400000.00 | 400000.00000000 | 5.821e-10 | 1.455e-15 | true |
| skalar 600000 (slot 0) pada level 0 | 600000.00 | 600000.00000000 | 1.048e-09 | 1.746e-15 | true |
| vektor penuh 600000 (8192 slot) pada level 0 | 600000.00 | -448575.99999991 | 1.049e+06 | 1.748e+00 | false |
| skalar 1e10 (slot 0) pada level 0 | 10000000000.00 | 123364421.84601450 | 9.877e+09 | 9.877e-01 | false |

> Overflow terjadi pada koefisien polinomial: |koefisien| >= q0/2 membungkus modulo q0 tanpa pesan galat. Untuk vektor penuh koefisien ke-0 = x*skala, sehingga batas 524287 berlaku umum; nilai di satu slot tersebar ke banyak koefisien (~x*skala*2/N) sehingga baru overflow jauh di atas batas. Klien memakai batas konservatif 500000 untuk saldo, nominal, dan subtotal.

### U12 (p1) — Permintaan serentak

- **Masukan:** saldo awal 20000; 10 topup 1000 dan 3 transaksi P005 x1 NONE dikirim bersamaan
- **Diharapkan:** Semua permintaan berhasil; saldo akhir = 20000 + 10000 - 15000 = 15000; riwayat berisi 3 transaksi (tidak ada update yang hilang)
- **Aktual:** gagal 0, saldo 15000.000003, riwayat 3 entri
- **Lulus:** true
- **Langkah:**
  1. 13 permintaan paralel selesai, 0 gagal; server menyerialkan operasi per akun (mutex per user_id) dan menyimpan saldo+riwayat dalam satu transaksi SQL

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo akhir | 15000.00 | 15000.00000317 | 3.170e-06 | 2.113e-10 | true |

### U12 (p2) — Permintaan serentak

- **Masukan:** saldo awal 20000; 10 topup 1000 dan 3 transaksi P005 x1 NONE dikirim bersamaan
- **Diharapkan:** Semua permintaan berhasil; saldo akhir = 20000 + 10000 - 15000 = 15000; riwayat berisi 3 transaksi (tidak ada update yang hilang)
- **Aktual:** gagal 0, saldo 15000.000050, riwayat 3 entri
- **Lulus:** true
- **Langkah:**
  1. 13 permintaan paralel selesai, 0 gagal; server menyerialkan operasi per akun (mutex per user_id) dan menyimpan saldo+riwayat dalam satu transaksi SQL

| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |
|---|---|---|---|---|---|
| saldo akhir | 15000.00 | 15000.00004974 | 4.974e-05 | 3.316e-09 | true |

## Contoh JSON

Contoh permintaan/respons (ciphertext dipotong, panjang aslinya dicatat) ada di `samples/`.
