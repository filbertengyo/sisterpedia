//go:build js && wasm

// Command wasm is the browser-side crypto bridge: Lattigo (CKKS) compiled to
// WebAssembly (GOOS=js GOARCH=wasm). It runs inside a Web Worker and exposes the
// functions listed in docs/CRYPTO_BRIDGE.md on the global "sisterpedia" object.
//
// Initial stub: only registers "version" and "ping". The real bridge is
// implemented in issue A3.1+ on top of pkg/ckks and pkg/codec.
package main

import "syscall/js"

func main() {
	api := js.Global().Get("Object").New()
	api.Set("version", "stub")
	api.Set("ping", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		return "pong"
	}))
	js.Global().Set("sisterpedia", api)

	// Keep the Go runtime alive so exported functions stay callable.
	select {}
}
