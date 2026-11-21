// Command clientsvc adalah client local service: tempat keygen, enkripsi, dan
// dekripsi berlangsung. Secret key hanya hidup di proses/volume ini.
// Stub awal: baru melayani /healthz. Implementasi sebenarnya ada di issue A1-A3.
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := getenv("LISTEN_ADDR", ":8001")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"client"}`))
	})

	log.Printf("client service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
