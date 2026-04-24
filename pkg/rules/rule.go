// Package rules defines the slopgate rule interface, the Finding type,
// the severity levels, and a minimal Registry that fires each registered
// rule against a parsed diff. This file also defines the SemanticRule
// interface for AST-aware rules.
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
	File    string
	Line    int    // 1-indexed line in the new file; 0 if not applicable
	Message string // one-line explanation
	Snippet string // the offending source line, unmodified
}

// Rule is the interface every detection must implement.
// This is the original rule interface for regex-on-diff rules.
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

// SemanticRule is the interface for AST-aware rules that can
// query cross-function type information and perform semantic analysis.
// These rules receive the full AST for new Go files, enabling deeper
// analysis than what's possible with regex alone.
type SemanticRule interface {
	// ID returns the stable rule identifier, e.g. "SLP071".
	ID() string
	// Description returns a human-readable one-liner.
	Description() string
	// DefaultSeverity is the severity used when config does not override it.
	DefaultSeverity() Severity
	// Check runs the rule against the AST analysis and returns any findings.
	Check(a *diff.AnalysisResult) []Finding
}

// Registry is an ordered collection of rules. Order matters: the reporter
// walks findings in registration order so output is deterministic.
type Registry struct {
	rules        []Rule
	semantic     []SemanticRule
	seen         map[string]bool
	semanticSeen map[string]bool
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		seen:         map[string]bool{},
		semanticSeen: map[string]bool{},
	}
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

// RegisterSemantic adds a semantic (AST-aware) rule to the registry.
// Panics on duplicate ID.
func (r *Registry) RegisterSemantic(rule SemanticRule) {
	if r.semanticSeen[rule.ID()] {
		panic(fmt.Sprintf("slopgate: duplicate semantic rule ID %q", rule.ID()))
	}
	r.semanticSeen[rule.ID()] = true
	r.semantic = append(r.semantic, rule)
}

// All returns the registered rules in registration order.
func (r *Registry) All() []Rule {
	out := make([]Rule, len(r.rules))
	copy(out, r.rules)
	return out
}

// AllSemantic returns the registered semantic rules in registration order.
func (r *Registry) AllSemantic() []SemanticRule {
	out := make([]SemanticRule, len(r.semantic))
	copy(out, r.semantic)
	return out
}

// HasSemanticRules returns true if any semantic rules are registered.
func (r *Registry) HasSemanticRules() bool {
	return len(r.semantic) > 0
}

// Run applies every registered rule to the diff and returns the concatenated
// findings. If cfg is non-nil, per-rule severity overrides, ignores, and
// path ignores are applied. A nil cfg means all rules run with their defaults.
//
// Run also processes semantic rules if AST analysis is available and
// semantic rules are registered.
func (r *Registry) Run(d *diff.Diff, cfg *config.Config) []Finding {
	var out []Finding

	// Run regex rules.
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

	// Run semantic rules if AST is available.
	// Only run semantic rules on Go files.
	if r.HasSemanticRules() && diff.HasGoFiles(d, true) {
		astResult := diff.LoadASTAnalysis(d)
		if len(astResult.Files) > 0 {
			for _, rule := range r.semantic {
				// Check if rule is globally ignored via config.
				if cfg != nil {
					if rc, ok := cfg.Rules[rule.ID()]; ok && rc.Ignore {
						continue
					}
				}
				def := rule.DefaultSeverity()
				for _, f := range rule.Check(astResult) {
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
								continue
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
		}
	}

	return out
}