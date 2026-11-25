//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// samples saves selected request/response pairs as example JSON for the
// report. Long Base64 strings (ciphertexts, keys) are truncated, and their
// original length is recorded, so the files show that sensitive values only
// travel as ciphertext.
var samples = &sampleStore{saved: map[string]any{}}

type sampleStore struct {
	mu    sync.Mutex
	next  string
	saved map[string]any
}

// expect names the next exchange to be captured.
func (s *sampleStore) expect(name string) {
	s.mu.Lock()
	s.next = name
	s.mu.Unlock()
}

func truncateJSON(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			out[k] = truncateJSON(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = truncateJSON(x[i])
		}
		return out
	case string:
		if len(x) > 120 {
			return fmt.Sprintf("%s... [Base64 dipotong: %d karakter, %d byte biner]", x[:80], len(x), len(x)/4*3-strings.Count(x[len(x)-2:], "="))
		}
		return x
	default:
		return v
	}
}

func decodeBody(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		if len(b) > 200 {
			b = b[:200]
		}
		return string(b)
	}
	return truncateJSON(v)
}

func (s *sampleStore) capture(method, path string, req []byte, status int, resp []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.next == "" {
		return
	}
	s.saved[s.next] = map[string]any{
		"request":  map[string]any{"method": method, "path": path, "body": decodeBody(req), "body_bytes": len(req)},
		"response": map[string]any{"status": status, "body": decodeBody(resp), "body_bytes": len(resp)},
	}
	s.next = ""
}

func (s *sampleStore) write() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(env.results, "samples")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	for name, v := range s.saved {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), b, 0o600); err != nil {
			return err
		}
	}
	return nil
}
