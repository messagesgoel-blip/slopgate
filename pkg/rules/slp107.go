package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP107 flags cleanup/destroy/close operations that appear only inside
// an error block (catch/except/if err) but are missing from the success
// path. Resources must be cleaned up on ALL code paths.
type SLP107 struct{}

func (SLP107) ID() string                { return "SLP107" }
func (SLP107) DefaultSeverity() Severity { return SeverityBlock }
func (SLP107) Description() string {
	return "cleanup/destroy only in error path — ensure cleanup runs on success too"
}

var slp107Cleanup = regexp.MustCompile(`(?i)(?:Close|Destroy|Cleanup|Release|Remove|Delete|Cancel|rollback)\s*\(`)

func (r SLP107) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			inErrorBlock := false
			errorBraceDepth := 0
			var cleanupLines []diff.Line

			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				cLower := strings.ToLower(content)

				if !inErrorBlock {
					if strings.Contains(cLower, "if err") || strings.Contains(cLower, "catch") || strings.Contains(cLower, "except") {
						inErrorBlock = true
						errorBraceDepth = 0
						errorBraceDepth += strings.Count(content, "{")
						errorBraceDepth -= strings.Count(content, "}")
					}
					continue
				}

				errorBraceDepth += strings.Count(content, "{")
				errorBraceDepth -= strings.Count(content, "}")

				if slp107Cleanup.MatchString(content) {
					cleanupLines = append(cleanupLines, ln)
				}

				if errorBraceDepth <= 0 && strings.Contains(content, "}") {
					if len(cleanupLines) > 0 {
						for _, cl := range cleanupLines {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     cl.NewLineNo,
								Message:  "cleanup/destroy only found in error block — ensure resource is also released on success path",
								Snippet:  strings.TrimSpace(cl.Content),
							})
						}
					}
					inErrorBlock = false
					cleanupLines = nil
				}
			}
		}
	}
	return out
}
