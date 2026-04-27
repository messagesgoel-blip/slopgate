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

var slp102AsyncFunc = regexp.MustCompile(`(?i)(?:async\s+(?:function\s+)?|async\s*\(\s*\)\s*=>|async\s+\w+\s*=>|async\s+\w+\s*\([^)]*\)\s*\{)`)

var slp102AwaitRe = regexp.MustCompile(`\bawait\b`)

func slp102HasOpeningBraceLookahead(lines []diff.Line, start int) bool {
	seenNonEmpty := 0
	for i := start; i < len(lines) && seenNonEmpty < 3; i++ {
		content := strings.TrimSpace(lines[i].Content)
		if content == "" {
			continue
		}
		seenNonEmpty++
		stripped := stripCommentAndStrings(content)
		// Stop if a new async function declaration is found — its { belongs to a different function.
		if slp102AsyncFunc.MatchString(stripped) {
			return false
		}
		if strings.Contains(stripped, "{") {
			return true
		}
	}
	return false
}

func (r SLP102) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isJSOrTSFile(f.Path) {
			continue
		}
		if isTestFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			inAsync := false
			asyncLine := 0
			asyncSnippet := ""
			braceDepth := 0
			hasAwait := false

			for idx, ln := range h.Lines {
				content := strings.TrimSpace(ln.Content)

				if !inAsync && ln.Kind == diff.LineAdd && slp102AsyncFunc.MatchString(content) {
					// brace-less arrow expression: handle single-line
					if !strings.Contains(content, "{") {
						if slp102HasOpeningBraceLookahead(h.Lines, idx+1) {
							inAsync = true
							asyncLine = ln.NewLineNo
							asyncSnippet = content
							braceDepth = 0
							hasAwait = false
						} else {
							// Only emit if it's a complete arrow expression on one line
							// e.g. const x = async () => 1
							// Avoid false positives on multiline like:
							// const x = async () =>
							//    1
							arrowIdx := strings.Index(content, "=>")
							if arrowIdx != -1 {
								rhs := strings.TrimSpace(content[arrowIdx+2:])
								if rhs != "" && !slp102AwaitRe.MatchString(content) {
									out = append(out, Finding{
										RuleID:   r.ID(),
										Severity: r.DefaultSeverity(),
										File:     f.Path,
										Line:     ln.NewLineNo,
										Message:  "async function contains no await — remove async or add the async work",
										Snippet:  content,
									})
								}
							}
							// If it's a multiline async arrow (no { but trailing =>), we might miss it
							// because we don't have a good way to track the end of a brace-less arrow func.
							// But the instructions specifically mentioned not emitting if it's not complete.
							continue
						}
					} else {
						inAsync = true
						asyncLine = ln.NewLineNo
						asyncSnippet = content
						braceDepth = 0
						hasAwait = false
					}
				}

				if !inAsync {
					continue
				}

				cleanContent := stripCommentAndStrings(content)
				if !inAsync || ln.Kind != diff.LineDelete {
					braceDepth += strings.Count(cleanContent, "{")
					braceDepth -= strings.Count(cleanContent, "}")
				}

				if ln.Kind != diff.LineDelete && slp102AwaitRe.MatchString(content) {
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
	return out
}
