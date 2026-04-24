package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP067 flags resource acquisitions without deferred or explicit close.
type SLP067 struct{}

func (SLP067) ID() string                { return "SLP067" }
func (SLP067) DefaultSeverity() Severity { return SeverityWarn }
func (SLP067) Description() string {
	return "resource acquired without deferred close"
}

var resourcePatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bhttp\.(?:Get|Post|Do)\s*\(`),
	regexp.MustCompile(`\bdb\.Query(?:Context)?\s*\(`),
	regexp.MustCompile(`\bos\.(?:Open|Create)\s*\(`),
	regexp.MustCompile(`\bsql\.Open\s*\(`),
}

func hasResourceAcquisition(line string) bool {
	clean := stripCommentAndStrings(line)
	for _, re := range resourcePatterns {
		if re.MatchString(clean) {
			return true
		}
	}
	return false
}

// resourceVar extracts a likely variable name from a resource acquisition line.
// For assignments like "resp, err := http.Get(...)" it returns "resp".
// Strips leading control keywords like "if " or "for " from the LHS.
func resourceVar(line string) string {
	line = strings.TrimSpace(line)
	// Strip leading control keywords.
	if strings.HasPrefix(line, "if ") {
		line = strings.TrimPrefix(line, "if ")
		line = strings.TrimSpace(line)
	}
	if strings.HasPrefix(line, "for ") {
		line = strings.TrimPrefix(line, "for ")
		line = strings.TrimSpace(line)
	}
	if idx := strings.Index(line, ":="); idx > 0 {
		lhs := strings.TrimSpace(line[:idx])
		parts := strings.Split(lhs, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if idx := strings.Index(line, "="); idx > 0 {
		lhs := strings.TrimSpace(line[:idx])
		parts := strings.Split(lhs, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return ""
}

func slp067BraceDelta(line string) int {
	clean := stripCommentAndStrings(line)
	return strings.Count(clean, "{") - strings.Count(clean, "}")
}

func slp067ScopeDepth(added []diff.Line, end int) int {
	depth := 0
	for i := 0; i <= end && i < len(added); i++ {
		depth += slp067BraceDelta(added[i].Content)
	}
	return depth
}

func slp067LineHasClose(line, varName string) bool {
	line = stripCommentAndStrings(line)
	trimmed := strings.TrimSpace(line)
	if varName == "" {
		return strings.Contains(trimmed, ".Close()") || strings.HasPrefix(trimmed, "defer ")
	}
	return strings.Contains(trimmed, varName+".Close()") ||
		strings.Contains(trimmed, varName+".Body.Close()") ||
		strings.HasPrefix(trimmed, "defer "+varName+".") ||
		strings.Contains(trimmed, "defer "+varName+".Body.")
}

func (r SLP067) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		added := f.AddedLines()
		for i, ln := range added {
			if !hasResourceAcquisition(ln.Content) {
				continue
			}
			varName := resourceVar(ln.Content)
			foundClose := false
			startDepth := slp067ScopeDepth(added, i)
			runningDepth := startDepth
			for j := i + 1; j < len(added); j++ {
				next := added[j].Content
				if slp067LineHasClose(next, varName) {
					foundClose = true
					break
				}
				// Check for anonymous defer closure: "defer func() { ... varName.Close() ... }()"
				if strings.Contains(next, "defer func(") || strings.Contains(next, "defer func ()") {
					// Scan the anon-defer block for a Close() call.
					blockDepth := runningDepth + slp067BraceDelta(next)
					for k := j + 1; k < len(added) && k < j+10; k++ {
						blockLine := added[k].Content
						if slp067LineHasClose(blockLine, varName) {
							foundClose = true
							break
						}
						if strings.Contains(blockLine, "}()") {
							break
						}
						blockDepth += slp067BraceDelta(blockLine)
						if blockDepth < startDepth {
							break
						}
					}
					if foundClose {
						break
					}
				}
				runningDepth += slp067BraceDelta(next)
				if runningDepth < startDepth {
					break
				}
			}
			if !foundClose {
				msg := "resource acquired without deferred Close() — add defer resource.Close() or similar"
				if varName != "" {
					msg = "resource acquired without deferred Close() — add defer " + varName + ".Close() or similar"
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  msg,
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
