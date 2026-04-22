package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP051 flags bare function calls in added code that may be undefined.
// We look for identifier(arg...) patterns and skip builtins, keywords,
// and method calls (which come from imported packages).
type SLP051 struct{}

func (SLP051) ID() string                { return "SLP051" }
func (SLP051) DefaultSeverity() Severity { return SeverityBlock }
func (SLP051) Description() string {
	return "call to potentially undefined function — implement or import it"
}

// goKeywords are Go control-flow keywords that are not function calls.
var goKeywords = map[string]bool{
	"if": true, "for": true, "switch": true, "select": true,
	"return": true, "defer": true, "go": true, "panic": true,
	"recover": true, "print": true, "println": true,
	"new": true, "make": true, "len": true, "cap": true,
	"append": true, "copy": true, "delete": true, "close": true,
	"complex": true, "real": true, "imag": true,
}

// undefinedCallPattern matches a bare identifier followed immediately by '('.
var undefinedCallPattern = regexp.MustCompile(`\b([a-zA-Z_]\w*)\s*\(`)

func (r SLP051) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				// Skip comments.
				if strings.HasPrefix(content, "//") {
					continue
				}
				// Find all bare calls.
				for _, m := range undefinedCallPattern.FindAllStringSubmatch(content, -1) {
					name := m[1]
					if goKeywords[name] || isSafetyTestName(name) {
						continue
					}
					// Skip method calls (contain dot before identifier).
					idx := strings.Index(content, name+"(")
					if idx > 0 && content[idx-1] == '.' {
						continue
					}
					// Skip if there is a local func definition for this name somewhere in the file.
					if hasLocalFunc(f, name) {
						continue
					}
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "call to undefined function " + name + " — implement or import it",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}

// funcDefPattern matches a function/method definition.
var funcDefPattern = regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?([a-zA-Z_]\w*)\s*\(`)

func hasLocalFunc(f diff.File, name string) bool {
	for _, h := range f.Hunks {
		for _, ln := range h.Lines {
			if ln.Kind != diff.LineAdd {
				continue
			}
			if m := funcDefPattern.FindStringSubmatch(strings.TrimSpace(ln.Content)); m != nil {
				if m[1] == name {
					return true
				}
			}
		}
	}
	return false
}
