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
}

var jsDynamic = []struct {
	re   *regexp.Regexp
	desc string
}{
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
	{regexp.MustCompile(`\bunsafe\.(Pointer|Sizeof|Alignof|Offsetof|Add|Slice)\b`), "unsafe.*"},
}

var slp057UnsafeImportBlockStartRe = regexp.MustCompile(`^\s*import\s*\(\s*$`)
var slp057UnsafeImportSpecRe = regexp.MustCompile(`^\s*(?:_\s*)?"unsafe"\s*$`)
var slp057SingleLineImportRe = regexp.MustCompile(`^\s*import\s+"unsafe"\s*$`)
var slp057ImportBlockEndRe = regexp.MustCompile(`^\s*\)\s*$`)

func (r SLP057) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		for _, h := range f.Hunks {
			inGoImportBlock := false
			for _, ln := range h.Lines {
				if ln.Kind == diff.LineDelete {
					continue
				}
				clean := strings.TrimSpace(stripCommentAndStrings(ln.Content))
				rawTrimmed := strings.TrimSpace(ln.Content)
				if isGoFile(f.Path) {
					// Check for single-line import "unsafe".
					if ln.Kind == diff.LineAdd && slp057SingleLineImportRe.MatchString(rawTrimmed) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "dynamic code execution detected — import \"unsafe\" is dangerous with untrusted input",
							Snippet:  strings.TrimSpace(ln.Content),
						})
						continue
					}
					if slp057UnsafeImportBlockStartRe.MatchString(clean) {
						inGoImportBlock = true
					} else if inGoImportBlock && slp057ImportBlockEndRe.MatchString(clean) {
						inGoImportBlock = false
					}
					if inGoImportBlock && ln.Kind == diff.LineAdd && slp057UnsafeImportSpecRe.MatchString(rawTrimmed) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "dynamic code execution detected — import \"unsafe\" is dangerous with untrusted input",
							Snippet:  strings.TrimSpace(ln.Content),
						})
						continue
					}
				}
				if ln.Kind != diff.LineAdd {
					continue
				}
				matched := false
				desc := ""

				// Use cleaned content for matching to avoid comments/strings.
				for _, p := range langAgnosticDynamic {
					if p.re.MatchString(clean) {
						matched = true
						desc = p.desc
						break
					}
				}

				if !matched && isJSOrTSFile(f.Path) {
					for _, p := range jsDynamic {
						if p.re.MatchString(clean) {
							matched = true
							desc = p.desc
							break
						}
					}
				}

				if !matched && isPythonFile(f.Path) {
					for _, p := range pythonDynamic {
						if p.re.MatchString(clean) {
							matched = true
							desc = p.desc
							break
						}
					}
				}

				if !matched && isGoFile(f.Path) {
					for _, p := range goDynamic {
						if p.re.MatchString(clean) {
							matched = true
							desc = p.desc
							break
						}
					}
				}

				if matched {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "dynamic code execution detected — " + desc + " is dangerous with untrusted input",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}
