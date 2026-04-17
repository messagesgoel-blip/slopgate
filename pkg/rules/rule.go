// Package rules defines the slopgate rule interface, the Finding type,
// the severity levels, and a minimal Registry that fires each registered
// rule against a parsed diff.
package rules

import (
	"fmt"

	"github.com/messagesgoel-blip/slopgate/pkg/config"
	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// Severity controls whether a finding blocks a commit or is advisory.
type Severity int

const (
	// SeverityInfo is advisory only — visible in PR mode, ignored in pre-commit.
	SeverityInfo Severity = iota
	// SeverityWarn is printed but does not fail the run.
	SeverityWarn
	// SeverityBlock fails the run (non-zero exit code).
	SeverityBlock
)

// String returns a short name for the severity.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarn:
		return "warn"
	case SeverityBlock:
		return "block"
	default:
		return "unknown"
	}
}

// Finding is a single slop detection reported by a rule.
type Finding struct {
	RuleID   string
	Severity Severity
	File     string
	Line     int    // 1-indexed line in the new file; 0 if not applicable
	Message  string // one-line explanation
	Snippet  string // the offending source line, unmodified
}

// Rule is the interface every detection must implement.
type Rule interface {
	// ID returns the stable rule identifier, e.g. "SLP012".
	ID() string
	// Description returns a human-readable one-liner.
	Description() string
	// DefaultSeverity is the severity used when config does not override it.
	DefaultSeverity() Severity
	// Check runs the rule against the parsed diff and returns any findings.
	Check(d *diff.Diff) []Finding
}

// Registry is an ordered collection of rules. Order matters: the reporter
// walks findings in registration order so output is deterministic.
type Registry struct {
	rules []Rule
	seen  map[string]bool
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{seen: map[string]bool{}}
}

// Register adds a rule to the registry. Panics on duplicate ID to catch
// configuration mistakes at startup rather than silently masking rules.
func (r *Registry) Register(rule Rule) {
	if r.seen[rule.ID()] {
		panic(fmt.Sprintf("slopgate: duplicate rule ID %q", rule.ID()))
	}
	r.seen[rule.ID()] = true
	r.rules = append(r.rules, rule)
}

// All returns the registered rules in registration order.
func (r *Registry) All() []Rule {
	out := make([]Rule, len(r.rules))
	copy(out, r.rules)
	return out
}

// Run applies every registered rule to the diff and returns the concatenated
// findings. If cfg is non-nil, per-rule severity overrides, ignores, and
// path ignores are applied. A nil cfg means all rules run with their defaults.
func (r *Registry) Run(d *diff.Diff, cfg *config.Config) []Finding {
	var out []Finding
	for _, rule := range r.rules {
		// Check if rule is globally ignored via config.
		if cfg != nil {
			if rc, ok := cfg.Rules[rule.ID()]; ok && rc.Ignore {
				continue
			}
		}
		// Apply per-rule path ignores: filter files this rule shouldn't see.
		ruleDiff := d
		if cfg != nil {
			if rc, ok := cfg.Rules[rule.ID()]; ok && len(rc.IgnorePaths) > 0 {
				ruleDiff = diff.FilterIgnored(d, rc.IgnorePaths)
			}
		}
		def := rule.DefaultSeverity()
		for _, f := range rule.Check(ruleDiff) {
			if f.RuleID == "" {
				f.RuleID = rule.ID()
			}
			// Apply config severity override.
			if cfg != nil {
				if rc, ok := cfg.Rules[rule.ID()]; ok && rc.Severity != "" {
					switch rc.Severity {
					case "block":
						f.Severity = SeverityBlock
					case "warn":
						f.Severity = SeverityWarn
					case "info":
						f.Severity = SeverityInfo
					case "off":
						continue // skip this finding
					default:
						// Unknown severity string — fall through to default.
					}
				} else if f.Severity == SeverityInfo && def != SeverityInfo {
					f.Severity = def
				}
			} else if f.Severity == SeverityInfo && def != SeverityInfo {
				f.Severity = def
			}
			out = append(out, f)
		}
	}
	return out
}
