// Command clientsvc is the client local service: key generation, encryption and
// decryption happen here. The secret key only lives in this process/volume.
// Initial stub: serves /healthz only. Real implementation is in A1-A3.
package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := getenv("LISTEN_ADDR", ":8001")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"client"}`))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("client service listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
