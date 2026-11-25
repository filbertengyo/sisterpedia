// Package config reads the server configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is the server configuration.
type Config struct {
	Addr            string   // SERVER_ADDR
	DBPath          string   // DB_PATH
	CatalogPath     string   // CATALOG_PATH
	ProfilesFile    string   // CKKS_PROFILES_FILE (empty = embedded config/profiles.yaml)
	AllowedProfiles []string // ALLOWED_PROFILES
	AllowedOrigins  []string // ALLOWED_ORIGIN (comma separated)
	MaxBodyBytes    int64    // MAX_BODY_BYTES
}

// FromEnv loads the configuration, applying the documented defaults.
func FromEnv() (Config, error) {
	c := Config{
		Addr:            get("SERVER_ADDR", ":8000"),
		DBPath:          get("DB_PATH", "server_data/shop.db"),
		CatalogPath:     get("CATALOG_PATH", "data/catalog.json"),
		ProfilesFile:    os.Getenv("CKKS_PROFILES_FILE"),
		AllowedProfiles: list(get("ALLOWED_PROFILES", "p1,p2")),
		AllowedOrigins:  list(get("ALLOWED_ORIGIN", "http://localhost:3000")),
		MaxBodyBytes:    16 << 20,
	}
	if v := os.Getenv("MAX_BODY_BYTES"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			return c, fmt.Errorf("MAX_BODY_BYTES: invalid value %q", v)
		}
		c.MaxBodyBytes = n
	}
	if len(c.AllowedProfiles) == 0 {
		return c, fmt.Errorf("ALLOWED_PROFILES is empty")
	}
	return c, nil
}

func get(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func list(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
