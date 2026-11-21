// Command server is the online-shop REST API. It only handles public artifacts
// and ciphertexts; it must never hold or use a secret key.
// Initial stub: serves /healthz only. Real handlers are implemented in B1-B3.
package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := getenv("SERVER_ADDR", ":8000")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"server"}`))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second, // request bodies carry large keys/ciphertexts
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("server listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
