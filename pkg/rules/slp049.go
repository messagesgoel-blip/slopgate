package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP049 flags vacuous test assertions that compare an input parameter to
// the output without testing any real transformation logic.
//
// Rationale: Such tests provide no value and give false confidence.
// Asserting that `input == result` when result is just the input parameter
// means the implementation was never actually exercised.
type SLP049 struct{}

func (SLP049) ID() string                { return "SLP049" }
func (SLP049) DefaultSeverity() Severity { return SeverityWarn }
func (SLP049) Description() string {
	return "vacuous test asserts input equals output — test actual transformation logic"
}

// assertKeywordRe detects common testify / t assertion helpers.
var assertKeywordRe = regexp.MustCompile(`\b(assert|require|t\.Error|t\.Fatal)\b`)

func (r SLP049) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		// Only inspect test files.
		name := strings.ToLower(f.Path)
		if !strings.Contains(name, "_test.go") && !strings.Contains(name, ".test.") &&
			!strings.Contains(name, ".spec.") && !strings.HasSuffix(name, "_test.rs") {
			continue
		}
		for _, ln := range f.AddedLines() {
			if !assertKeywordRe.MatchString(ln.Content) {
				continue
			}
			clean := strings.TrimSpace(ln.Content)
			// Check for same identifier on both sides of == or =.
			if hasSameIdentifierBothSides(clean) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  r.Description(),
					Snippet:  clean,
				})
			}
		}
	}
	return out
}

// hasSameIdentifierBothSides checks if the same identifier appears on both
// sides of == or = within the string, or as duplicate arguments in an
// assert/require call.
func hasSameIdentifierBothSides(s string) bool {
	// Replace common separators with spaces to tokenize.
	s = strings.ReplaceAll(s, ",", " ")
	s = strings.ReplaceAll(s, "(", " ")
	s = strings.ReplaceAll(s, ")", " ")
	// Find all tokens.
	tokens := strings.Fields(s)
	for i := 0; i < len(tokens)-2; i++ {
		if tokens[i+1] == "==" || tokens[i+1] == "=" {
			if tokens[i] == tokens[i+2] {
				return true
			}
		}
	}
	// Check for duplicate tokens among the remaining tokens (catches assert.Equal(t, input, input))
	seen := make(map[string]bool)
	for _, tok := range tokens {
		// Only consider identifier-like tokens.
		if matched, _ := regexp.MatchString(`^[A-Za-z_][A-Za-z0-9_]*$`, tok); !matched {
			continue
		}
		if seen[tok] {
			return true
		}
		seen[tok] = true
	}
	return false
}
