package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP059 flags unsanitized exec.Command usage in Go files.
type SLP059 struct{}

func (SLP059) ID() string                { return "SLP059" }
func (SLP059) DefaultSeverity() Severity { return SeverityBlock }
func (SLP059) Description() string {
	return "unsanitized os/exec command with user input"
}

var goIdentPattern = regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)
var execCommandRe = regexp.MustCompile(`\bexec\.Command\s*\(`)

func slp059CollectExecCall(added []diff.Line, start int) string {
	var parts []string
	depth := 0
	foundCall := false
	for i := start; i < len(added); i++ {
		if i > start && added[i].NewLineNo != added[i-1].NewLineNo+1 {
			break
		}
		clean := stripCommentAndStrings(added[i].Content)
		if !foundCall {
			m := execCommandRe.FindStringIndex(clean)
			if m == nil {
				break
			}
			clean = clean[m[0]:]
			foundCall = true
		}
		parts = append(parts, clean)
		depth += strings.Count(clean, "(") - strings.Count(clean, ")")
		if foundCall && depth <= 0 {
			break
		}
	}
	return strings.Join(parts, "\n")
}

func (r SLP059) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		added := f.AddedLines()
		for i, ln := range added {
			cleanLine := stripCommentAndStrings(ln.Content)
			if execCommandRe.FindStringIndex(cleanLine) == nil {
				continue
			}
			call := slp059CollectExecCall(added, i)
			if call == "" {
				continue
			}
			callMatch := execCommandRe.FindStringIndex(call)
			if callMatch == nil {
				continue
			}
			args := call[callMatch[1]:]
			unquoted := args
			// Any interpolation or concatenation is an immediate red flag.
			if strings.Contains(unquoted, "$") || strings.Contains(unquoted, "+") || strings.Contains(unquoted, "fmt.Sprintf") {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "exec.Command argument may contain user input — sanitize before executing",
					Snippet:  strings.TrimSpace(ln.Content),
				})
				continue
			}
			if goIdentPattern.MatchString(unquoted) {
				// Note: we cannot statically resolve whether a variable is a safe
				// compile-time constant. A local const string is safe, but a
				// variable assigned elsewhere may contain user input. We flag all
				// non-literal variables as potentially unsafe.
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "exec.Command argument may contain user input — sanitize before executing",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
