//go:build js && wasm

// Command wasm is the browser-side crypto bridge: Lattigo (CKKS) compiled to
// WebAssembly (GOOS=js GOARCH=wasm). It runs inside a Web Worker and exposes
// the functions of docs/CRYPTO_BRIDGE.md on the global "sisterpedia" object.
// Each function returns a JSON string {"ok":true,"value":...} or
// {"ok":false,"error":{"code","message"}}. The logic lives in pkg/bridge.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/filbertengyo/sisterpedia/pkg/bridge"
)

func encode(r bridge.Result) any {
	b, err := json.Marshal(r)
	if err != nil {
		b, _ = json.Marshal(bridge.Result{Error: &bridge.Error{Code: bridge.CodeInternal, Message: "cannot encode result"}})
	}
	return string(b)
}

func str(args []js.Value, i int) (string, error) {
	if i >= len(args) || args[i].Type() != js.TypeString {
		return "", bridge.InvalidArgs("argument %d must be a string", i)
	}
	return args[i].String(), nil
}

func optStr(args []js.Value, i int) string {
	if i >= len(args) || args[i].Type() != js.TypeString {
		return ""
	}
	return args[i].String()
}

func num(args []js.Value, i int) (float64, error) {
	if i >= len(args) || args[i].Type() != js.TypeNumber {
		return 0, bridge.InvalidArgs("argument %d must be a number", i)
	}
	return args[i].Float(), nil
}

func nums(args []js.Value, i int) ([]float64, error) {
	if i >= len(args) || !args[i].InstanceOf(js.Global().Get("Array")) {
		return nil, bridge.InvalidArgs("argument %d must be an array of numbers", i)
	}
	n := args[i].Length()
	out := make([]float64, n)
	for j := 0; j < n; j++ {
		v := args[i].Index(j)
		if v.Type() != js.TypeNumber {
			return nil, bridge.InvalidArgs("argument %d[%d] must be a number", i, j)
		}
		out[j] = v.Float()
	}
	return out, nil
}

func fn(f func(args []js.Value) (any, error)) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		return encode(bridge.Wrap(func() (any, error) { return f(args) }))
	})
}

func main() {
	b, err := bridge.New()
	api := js.Global().Get("Object").New()
	api.Set("version", bridge.Version())
	if err != nil {
		api.Set("initError", err.Error())
		js.Global().Set("sisterpedia", api)
		select {}
	}

	api.Set("profiles", fn(func([]js.Value) (any, error) { return b.Profiles() }))
	api.Set("paramsInfo", fn(func(a []js.Value) (any, error) {
		p, err := str(a, 0)
		if err != nil {
			return nil, err
		}
		return b.ParamsInfo(p)
	}))
	api.Set("genKeys", fn(func(a []js.Value) (any, error) {
		p, err := str(a, 0)
		if err != nil {
			return nil, err
		}
		return b.GenKeys(p)
	}))
	api.Set("validateKeys", fn(func(a []js.Value) (any, error) {
		p, err := str(a, 0)
		if err != nil {
			return nil, err
		}
		sk, err := str(a, 1)
		if err != nil {
			return nil, err
		}
		pk, err := str(a, 2)
		if err != nil {
			return nil, err
		}
		return b.ValidateKeys(p, sk, pk, optStr(a, 3))
	}))
	api.Set("encryptScalar", fn(func(a []js.Value) (any, error) {
		p, err := str(a, 0)
		if err != nil {
			return nil, err
		}
		pk, err := str(a, 1)
		if err != nil {
			return nil, err
		}
		x, err := num(a, 2)
		if err != nil {
			return nil, err
		}
		return b.EncryptScalar(p, pk, x)
	}))
	api.Set("encryptVector", fn(func(a []js.Value) (any, error) {
		p, err := str(a, 0)
		if err != nil {
			return nil, err
		}
		pk, err := str(a, 1)
		if err != nil {
			return nil, err
		}
		v, err := nums(a, 2)
		if err != nil {
			return nil, err
		}
		return b.EncryptVector(p, pk, v)
	}))
	api.Set("decryptScalar", fn(func(a []js.Value) (any, error) {
		p, err := str(a, 0)
		if err != nil {
			return nil, err
		}
		sk, err := str(a, 1)
		if err != nil {
			return nil, err
		}
		ct, err := str(a, 2)
		if err != nil {
			return nil, err
		}
		return b.DecryptScalar(p, sk, ct)
	}))
	api.Set("decryptVector", fn(func(a []js.Value) (any, error) {
		p, err := str(a, 0)
		if err != nil {
			return nil, err
		}
		sk, err := str(a, 1)
		if err != nil {
			return nil, err
		}
		ct, err := str(a, 2)
		if err != nil {
			return nil, err
		}
		n, err := num(a, 3)
		if err != nil {
			return nil, err
		}
		return b.DecryptVector(p, sk, ct, int(n))
	}))
	api.Set("ciphertextInfo", fn(func(a []js.Value) (any, error) {
		p, err := str(a, 0)
		if err != nil {
			return nil, err
		}
		ct, err := str(a, 1)
		if err != nil {
			return nil, err
		}
		return b.CiphertextInfo(p, ct)
	}))
	api.Set("ping", js.FuncOf(func(js.Value, []js.Value) any { return "pong" }))
	js.Global().Set("sisterpedia", api)

	// Keep the Go runtime alive so exported functions stay callable.
	select {}
}
