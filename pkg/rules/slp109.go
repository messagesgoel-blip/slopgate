package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP109 flags two or more functions added in the same file with highly
// similar bodies (>60% identical). This is a common AI slop pattern:
// copy-pasting entire functions with minor changes instead of extracting
// shared logic.
type SLP109 struct{}

func (SLP109) ID() string                { return "SLP109" }
func (SLP109) DefaultSeverity() Severity { return SeverityWarn }
func (SLP109) Description() string {
	return "duplicate function body — extract shared logic instead of copy-pasting"
}

func slp109Normalize(line string) string {
	s := strings.TrimSpace(line)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32)
		}
	}
	return b.String()
}

func slp109BodySimilarity(a, b []string) float64 {
	setA := make(map[string]bool)
	for _, la := range a {
		na := slp109Normalize(la)
		if len(na) >= 3 {
			setA[na] = true
		}
	}
	setB := make(map[string]bool)
	for _, lb := range b {
		nb := slp109Normalize(lb)
		if len(nb) >= 3 {
			setB[nb] = true
		}
	}
	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}
	intersection := 0
	for k := range setA {
		if setB[k] {
			intersection++
		}
	}
	maxLen := len(setA)
	if len(setB) > maxLen {
		maxLen = len(setB)
	}
	return float64(intersection) / float64(maxLen)
}

func slp109HasFuncKeyword(content string) bool {
	cLower := strings.ToLower(content)
	return strings.Contains(cLower, "func ") ||
		strings.Contains(cLower, "function ") ||
		strings.Contains(cLower, "public ") ||
		strings.Contains(cLower, "private ") ||
		strings.Contains(cLower, "static ")
}

func slp109NextNonEmptyLine(lines []diff.Line, start int) string {
	for i := start; i < len(lines); i++ {
		content := strings.TrimSpace(lines[i].Content)
		if content != "" {
			return content
		}
	}
	return ""
}

func (r SLP109) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isJavaFile(f.Path) {
			continue
		}

		type funcBody struct {
			sigLine int
			sig     string
			body    []string
		}
		var funcs []funcBody

		for _, h := range f.Hunks {
			inFunc := false
			braceDepth := 0
			var cur funcBody

			for idx, ln := range h.Lines {
				content := strings.TrimSpace(ln.Content)

				if !inFunc && ln.Kind != diff.LineAdd {
					continue
				}

				if !inFunc && strings.Contains(content, "(") && strings.Contains(content, ")") {
					if slp109HasFuncKeyword(content) {
						inFunc = true
						braceDepth = 0
						cur = funcBody{sigLine: ln.NewLineNo, sig: content}
						if !strings.Contains(content, "{") {
							next := slp109NextNonEmptyLine(h.Lines, idx+1)
							if !strings.Contains(next, "{") {
								inFunc = false
								continue
							}
						}
					}
				}

				if inFunc {
					braceDepth += strings.Count(content, "{")
					braceDepth -= strings.Count(content, "}")
					if ln.Kind == diff.LineAdd && ln.NewLineNo != cur.sigLine && content != "{" && content != "}" {
						cur.body = append(cur.body, content)
					}
					if braceDepth <= 0 && strings.Contains(content, "}") {
						if len(cur.body) > 0 {
							funcs = append(funcs, cur)
						}
						inFunc = false
					}
				}
			}
		}

		seen := make(map[int]bool)
		for i := 0; i < len(funcs); i++ {
			for j := i + 1; j < len(funcs); j++ {
				if seen[funcs[j].sigLine] {
					continue
				}
				sim := slp109BodySimilarity(funcs[i].body, funcs[j].body)
				if sim > 0.6 {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     funcs[j].sigLine,
						Message:  "function body is highly similar to another added function — extract shared logic",
						Snippet:  funcs[j].sig,
					})
					seen[funcs[j].sigLine] = true
				}
			}
		}
	}
	return out
}
