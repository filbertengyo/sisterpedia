// Penanda modul Go terpisah agar "go build/vet/test ./..." dari root repo
// tidak ikut memindai paket Go yang terselip di web/node_modules.
// Tidak ada kode Go di folder ini.
module sisterpedia.invalid/web-nonmodule

go 1.23
