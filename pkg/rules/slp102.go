package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP102 flags async functions that contain no await expression. These are
// likely stubs where an AI agent declared the function async but never
// added the async work.
type SLP102 struct{}

func (SLP102) ID() string                { return "SLP102" }
func (SLP102) DefaultSeverity() Severity { return SeverityWarn }
func (SLP102) Description() string {
	return "async function has no await — likely an incomplete stub"
}

var slp102AsyncFunc = regexp.MustCompile(`(?i)(?:async\s+(?:function\s+)?|async\s*\(\s*\)\s*=>|async\s+\w+\s*=>|async\s*\w+\s*\([^)]*\)\s*\{)`)

var slp102AwaitRe = regexp.MustCompile(`\bawait\b`)

func (r SLP102) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isJSOrTSFile(f.Path) {
			continue
		}
		if strings.Contains(strings.ToLower(f.Path), ".test.") ||
			strings.Contains(strings.ToLower(f.Path), ".spec.") {
			continue
		}

		for _, h := range f.Hunks {
			inAsync := false
			asyncLine := 0
			asyncSnippet := ""
			braceDepth := 0
			hasAwait := false

			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)

				if !inAsync && slp102AsyncFunc.MatchString(content) {
					// brace-less arrow expression: handle single-line
					if !strings.Contains(content, "{") {
						if !slp102AwaitRe.MatchString(content) {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     ln.NewLineNo,
								Message:  "async function contains no await — remove async or add the async work",
								Snippet:  content,
							})
						}
						continue
					}
					inAsync = true
					asyncLine = ln.NewLineNo
					asyncSnippet = content
					braceDepth = 0
					hasAwait = false
				}

				if inAsync {
					braceDepth += strings.Count(content, "{")
					braceDepth -= strings.Count(content, "}")

					if slp102AwaitRe.MatchString(content) {
						hasAwait = true
					}

					if braceDepth <= 0 && strings.Contains(content, "}") {
						if !hasAwait {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     asyncLine,
								Message:  "async function contains no await — remove async or add the async work",
								Snippet:  asyncSnippet,
							})
						}
						inAsync = false
					}
				}
			}
		}
	}
	return out
}
