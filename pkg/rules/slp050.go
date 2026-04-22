package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP050 flags Go functions that accept pointer, slice, map, interface, or
// string parameters without performing any nil/empty validation.
//
// Rationale: Missing validation leads to runtime panics. If a function
// receives a `*T` or `[]T` and never checks it before use, it will crash on
// nil input. AI-generated code often omits these guards.
type SLP050 struct{}

func (SLP050) ID() string                { return "SLP050" }
func (SLP050) DefaultSeverity() Severity { return SeverityWarn }
func (SLP050) Description() string {
	return "function accepts pointer or reference param without nil/empty validation"
}

var (
	// funcDeclRe captures the function name and parenthesised parameter list.
	slp050FuncDeclRe = regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?([A-Za-z_]\w*)\s*\((.*)\)`)
	// validationRe matches common nil/empty checks.
	slp050ValidationRe = regexp.MustCompile(`==\s*nil|==\s*""|==\s*0|len\s*\(\w+\)\s*==\s*0`)
)

func (r SLP050) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		added := f.AddedLines()
		for i := 0; i < len(added); i++ {
			ln := added[i]
			m := slp050FuncDeclRe.FindStringSubmatch(ln.Content)
			if m == nil {
				continue
			}
			funcName := m[1]
			paramsLine := m[2]

			// Extract named parameters.
			var paramNames []string
			for _, raw := range strings.Split(paramsLine, ",") {
				raw = strings.TrimSpace(raw)
				parts := strings.Fields(raw)
				if len(parts) < 2 {
					continue
				}
				name := parts[len(parts)-2]
				typeStr := parts[len(parts)-1]
				if needsValidation(typeStr) {
					paramNames = append(paramNames, name)
				}
			}
			if len(paramNames) == 0 {
				continue
			}

			// Scan subsequent added lines for validation.
			validated := make(map[string]bool)
			for j := i + 1; j < len(added); j++ {
				next := added[j]
				c := next.Content
				// Stop scanning at the next top-level declaration.
				trimmed := strings.TrimSpace(c)
				if strings.HasPrefix(trimmed, "func ") ||
					strings.HasPrefix(trimmed, "type ") {
					break
				}
				if slp050ValidationRe.MatchString(c) {
					for _, p := range paramNames {
						if strings.Contains(c, p) {
							validated[p] = true
						}
					}
				}
			}

			for _, p := range paramNames {
				if !validated[p] {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "function " + funcName + " accepts " + p + " without validation — add nil/empty check",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}

func needsValidation(typeStr string) bool {
	return strings.HasPrefix(typeStr, "*") ||
		strings.HasPrefix(typeStr, "[]") ||
		strings.HasPrefix(typeStr, "map[") ||
		typeStr == "interface{}" ||
		typeStr == "any" ||
		typeStr == "string"
}
