// Command server adalah REST API toko daring (hanya artefak publik + ciphertext).
// Stub awal: baru melayani /healthz. Handler sebenarnya dikerjakan di issue B1-B3.
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := getenv("SERVER_ADDR", ":8000")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"server"}`))
	})

	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
