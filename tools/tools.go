//go:build tools

// Package tools pins the project's core dependencies in go.mod before the real
// code imports them, so parallel PRs do not all fight over go.mod/go.sum.
// Delete an import from here once real code uses it; delete this file when all
// of them are imported elsewhere. It is excluded from normal builds by the
// "tools" build tag.
package tools

import (
	_ "github.com/go-chi/chi/v5"
	_ "github.com/oklog/ulid/v2"
	_ "github.com/tuneinsight/lattigo/v6/core/rlwe"
	_ "github.com/tuneinsight/lattigo/v6/schemes/ckks"
	_ "gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)
