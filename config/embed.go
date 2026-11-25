// Package config embeds the default CKKS parameter profiles so that both the
// server and the WebAssembly bridge (which cannot read files) use the same
// configuration. The values live in profiles.yaml, separate from app logic;
// the server may override them at runtime with CKKS_PROFILES_FILE.
package config

import _ "embed"

// ProfilesYAML is the content of config/profiles.yaml at build time.
//
//go:embed profiles.yaml
var ProfilesYAML []byte
