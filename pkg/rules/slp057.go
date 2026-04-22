package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP057 flags dynamic code execution patterns in added lines.
type SLP057 struct{}

func (SLP057) ID() string                { return "SLP057" }
func (SLP057) DefaultSeverity() Severity { return SeverityBlock }
func (SLP057) Description() string {
	return "dynamic code execution detected"
}

var langAgnosticDynamic = []struct {
	re   *regexp.Regexp
	desc string
}{
	{regexp.MustCompile(`\beval\s*\(`), "eval("},
	{regexp.MustCompile(`\bnew\s+Function\s*\(`), "new Function("},
	{regexp.MustCompile(`\bFunction\s*\(`), "Function("},
}

var pythonDynamic = []struct {
	re   *regexp.Regexp
	desc string
}{
	{regexp.MustCompile(`\bexec\s*\(`), "exec("},
	{regexp.MustCompile(`\b__import__\b`), "__import__"},
	{regexp.MustCompile(`\bimportlib\.import_module\b`), "importlib.import_module"},
}

var goDynamic = []struct {
	re   *regexp.Regexp
	desc string
}{
	{regexp.MustCompile(`\breflect\.Value\.Call\b`), "reflect.Value.Call"},
	{regexp.MustCompile(`\bunsafe\b`), "unsafe"},
}

func (SLP057) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		for _, ln := range f.AddedLines() {
			matched := false
			desc := ""

			for _, p := range langAgnosticDynamic {
				if p.re.MatchString(ln.Content) {
					matched = true
					desc = p.desc
					break
				}
			}

			if !matched && isPythonFile(f.Path) {
				for _, p := range pythonDynamic {
					if p.re.MatchString(ln.Content) {
						matched = true
						desc = p.desc
						break
					}
				}
			}

			if !matched && isGoFile(f.Path) {
				for _, p := range goDynamic {
					if p.re.MatchString(ln.Content) {
						matched = true
						desc = p.desc
						break
					}
				}
			}

			if matched {
				out = append(out, Finding{
					RuleID:   "SLP057",
					Severity: SeverityBlock,
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "dynamic code execution detected — " + desc + " is dangerous with untrusted input",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
