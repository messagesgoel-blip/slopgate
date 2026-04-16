// Package config parses .slopgate.toml configuration files and discovers
// them from the repo root upward.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the top-level .slopgate.toml structure.
type Config struct {
	Rules map[string]RuleConfig `toml:"rules"`
}

// RuleConfig configures a single rule's behaviour.
type RuleConfig struct {
	Severity    string   `toml:"severity"`     // "block", "warn", "info", "off"
	Ignore      bool     `toml:"ignore"`       // true to skip the rule entirely
	IgnorePaths []string `toml:"ignore_paths"` // glob patterns, per-rule file ignores
}

// Load parses a TOML config file at path. Returns a nil Config without
// error if the file exists but is empty.
func Load(path string) (*Config, error) {
	cfg := &Config{}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Discover walks up from dir to find .slopgate.toml.
// Returns ("", nil) if not found.
func Discover(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	cur := abs
	for {
		p := filepath.Join(cur, ".slopgate.toml")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break // filesystem root
		}
		cur = parent
	}
	return "", nil
}
