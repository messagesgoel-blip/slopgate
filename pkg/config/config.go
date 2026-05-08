// Package config parses .slopgate.toml configuration files and discovers
// them from the repo root upward.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the top-level .slopgate.toml structure.
type Config struct {
	Rules       map[string]RuleConfig `toml:"rules"`
	MinSeverity string                `toml:"min_severity"` // global severity floor: "block", "warn", "info"
}

// RuleConfig configures a single rule's behaviour.
type RuleConfig struct {
	Severity    string   `toml:"severity"`     // "block", "warn", "info", "off"
	Ignore      bool     `toml:"ignore"`       // true to skip the rule entirely
	IgnorePaths []string `toml:"ignore_paths"` // glob patterns, per-rule file ignores
}

// Load parses a TOML config file at path. Returns a non-nil (empty)
// *Config when the file exists but is empty. Callers should check the
// returned error and treat a nil error with a non-nil Config as success.
func Load(path string) (*Config, error) {
	cfg := &Config{}
	meta, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, err
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return nil, fmt.Errorf("config: unknown keys: %v", undecoded)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks that the Config values are valid.
func (c *Config) Validate() error {
	validSeverity := map[string]bool{"block": true, "warn": true, "info": true, "off": true}
	for ruleID, rule := range c.Rules {
		if rule.Severity != "" && !validSeverity[rule.Severity] {
			return fmt.Errorf("config: rule %q: invalid severity %q (want block|warn|info|off)", ruleID, rule.Severity)
		}
	}
	if c.MinSeverity != "" && !validSeverity[c.MinSeverity] {
		return fmt.Errorf("config: min_severity %q invalid (want block|warn|info)", c.MinSeverity)
	}
	return nil
}

// Discover walks up from dir to find .slopgate.toml.
// Stops at the repository root (directory containing .git or go.mod).
// Returns ("", nil) if not found.
func Discover(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	cur := abs
	for {
		p := filepath.Join(cur, ".slopgate.toml")
		if _, err := os.Stat(p); err != nil {
			if os.IsNotExist(err) {
				// Not found here, continue upward.
			} else {
				// Real I/O error — surface it.
				return "", err
			}
		} else {
			return p, nil
		}

		// Stop at repo root sentinel.
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %s: %w", filepath.Join(cur, ".git"), err)
		}
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %s: %w", filepath.Join(cur, "go.mod"), err)
		}

		parent := filepath.Dir(cur)
		if parent == cur {
			break // filesystem root
		}
		cur = parent
	}
	return "", nil
}
