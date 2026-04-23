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

			paramChecks := make(map[string]*slp050ParamChecks, len(paramNames))
			for _, p := range paramNames {
				paramChecks[p] = newSLP050ParamChecks(p)
			}

			// Scan subsequent added lines for validation until the current function ends.
			validated := make(map[string]bool)
			depth := slp050BraceDelta(ln.Content)
			bodyStarted := strings.Contains(stripCommentAndStrings(ln.Content), "{")
			for j := i + 1; j < len(added) && (!bodyStarted || depth > 0); j++ {
				next := added[j]
				clean := stripCommentAndStrings(next.Content)
				if clean != "" {
					for _, p := range paramNames {
						if paramChecks[p].matches(clean) {
							validated[p] = true
						}
					}
				}
				depth += slp050BraceDelta(next.Content)
				if !bodyStarted && strings.Contains(clean, "{") {
					bodyStarted = true
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

type slp050ParamChecks struct {
	nilCheck   *regexp.Regexp
	emptyCheck *regexp.Regexp
	lenCheck   *regexp.Regexp
}

func newSLP050ParamChecks(name string) *slp050ParamChecks {
	ident := regexp.QuoteMeta(name)
	return &slp050ParamChecks{
		nilCheck: regexp.MustCompile(`\b` + ident + `\b\s*(?:==|!=)\s*nil|\bnil\s*(?:==|!=)\s*\b` + ident + `\b`),
		emptyCheck: regexp.MustCompile(
			`\b` + ident + `\b\s*(?:==|!=)\s*""|""\s*(?:==|!=)\s*\b` + ident + `\b`,
		),
		lenCheck: regexp.MustCompile(
			`len\s*\(\s*` + ident + `\s*\)\s*(?:==|!=|>=|<=)\s*0|len\s*\(\s*` + ident + `\s*\)\s*>\s*0|len\s*\(\s*` + ident + `\s*\)\s*<\s*1`,
		),
	}
}

func (c *slp050ParamChecks) matches(line string) bool {
	return c.nilCheck.MatchString(line) || c.emptyCheck.MatchString(line) || c.lenCheck.MatchString(line)
}

func slp050BraceDelta(line string) int {
	clean := stripCommentAndStrings(line)
	return strings.Count(clean, "{") - strings.Count(clean, "}")
}

// paramRegex returns a regex matching the parameter name as a bare identifier.
func paramRegex(name string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)(^|[^\w])` + regexp.QuoteMeta(name) + `($|[^\w])`)
}

func needsValidation(typeStr string) bool {
	return strings.HasPrefix(typeStr, "*") ||
		strings.HasPrefix(typeStr, "[]") ||
		strings.HasPrefix(typeStr, "map[") ||
		typeStr == "interface{}" ||
		typeStr == "any" ||
		typeStr == "string"
}
