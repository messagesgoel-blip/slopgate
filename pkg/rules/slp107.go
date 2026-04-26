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

var slp107Cleanup = regexp.MustCompile(`(?i)\b(?:Close|Destroy|Cleanup|Release|Remove|Delete|Cancel|Free)\b\s*(?:\(|$)`)
var slp107IdentifierPattern = regexp.MustCompile(`(?i)\b([A-Za-z_][A-Za-z0-9_]*)\s*\.\s*(?:close|destroy|cleanup|release|remove|delete|cancel|free)\b\s*(?:\(|$)`)
var slp107ErrorBlockStart = regexp.MustCompile(`(?i)(?:\bif\s+err\b|\bcatch\b|\bexcept\b)`)

func (r SLP107) Check(d *diff.Diff) []Finding {
	var out []Finding
	for i := range d.Files {
		f := &d.Files[i]
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) {
			continue
		}

		isPython := isPythonFile(f.Path)

		for j := range f.Hunks {
			h := &f.Hunks[j]
			inErrorBlock := false
			errorBraceDepth := 0
			errorIndentLevel := -1
			errorBlockStart := -1
			var cleanupLines []diff.Line

			for k := range h.Lines {
				ln := &h.Lines[k]
				if ln.Kind == diff.LineDelete {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				cLower := strings.ToLower(content)

				if !inErrorBlock {
					if strings.Contains(cLower, "if err") || slp107ErrorBlockStart.MatchString(content) {
						inErrorBlock = true
						errorBlockStart = k
						if isPython {
							errorIndentLevel = len(ln.Content) - len(strings.TrimLeft(ln.Content, " \t"))
						} else {
							errorBraceDepth = 0
							errorBraceDepth += strings.Count(content, "{")
							errorBraceDepth -= strings.Count(content, "}")
						}

						// Handle cases where the cleanup might be on the same line as the if err (e.g. Go one-liners)
						// ONLY if it's an added line
						if ln.Kind == diff.LineAdd && slp107Cleanup.MatchString(content) {
							cleanupLines = append(cleanupLines, *ln)
						}

						// Check if the block closed immediately (one-liner)
						if (!isPython && errorBraceDepth <= 0 && strings.Contains(content, "}")) || (isPython && slp107NextNonDeletedIndent(h.Lines, k+1) <= errorIndentLevel) {
							if len(cleanupLines) > 0 {
								r.emitIfNoSuccessCleanup(&out, f.Path, cleanupLines, h, errorBlockStart, k+1)
							}
							inErrorBlock = false
							errorBlockStart = -1
							cleanupLines = nil
						}
					}
					continue
				}

				if isPython {
					// In Python, a line with less or equal indentation than the except/try line ends the block
					// (Ignoring blank lines)
					if content != "" && getIndent(ln.Content) <= errorIndentLevel {
						if len(cleanupLines) > 0 {
							r.emitIfNoSuccessCleanup(&out, f.Path, cleanupLines, h, errorBlockStart, k)
						}
						inErrorBlock = false
						errorBlockStart = -1
						cleanupLines = nil
						// Re-check if this line starts a new error block
						if strings.Contains(cLower, "if err") || slp107ErrorBlockStart.MatchString(content) {
							inErrorBlock = true
							errorBlockStart = k
							errorIndentLevel = len(ln.Content) - len(strings.TrimLeft(ln.Content, " \t"))
							if ln.Kind == diff.LineAdd && slp107Cleanup.MatchString(content) {
								cleanupLines = append(cleanupLines, *ln)
							}
						}
						continue
					}
				} else {
					errorBraceDepth += strings.Count(content, "{")
					errorBraceDepth -= strings.Count(content, "}")
				}

				if ln.Kind == diff.LineAdd && slp107Cleanup.MatchString(content) {
					cleanupLines = append(cleanupLines, *ln)
				}

				if !isPython && errorBraceDepth <= 0 && strings.Contains(content, "}") {
					if len(cleanupLines) > 0 {
						r.emitIfNoSuccessCleanup(&out, f.Path, cleanupLines, h, errorBlockStart, k+1)
					}
					inErrorBlock = false
					errorBlockStart = -1
					cleanupLines = nil
				}
			}
			// End of hunk also closes any open error block
			if inErrorBlock && len(cleanupLines) > 0 {
				r.emitIfNoSuccessCleanup(&out, f.Path, cleanupLines, h, errorBlockStart, len(h.Lines))
			}
		}
	}
	return out
}

func (r SLP107) emitIfNoSuccessCleanup(out *[]Finding, filePath string, cleanupLines []diff.Line, hunk *diff.Hunk, blockStart, startIndex int) {
	for _, cl := range cleanupLines {
		identifier := extractIdentifier(cl.Content)
		foundSuccess := false
		for j := blockStart - 1; j >= 0; j-- {
			ln := hunk.Lines[j]
			if ln.Kind == diff.LineDelete {
				continue
			}
			if slp107LineMatchesCleanup(ln.Content, identifier) {
				foundSuccess = true
				break
			}
		}
		if foundSuccess {
			continue
		}
		for j := startIndex; j < len(hunk.Lines); j++ {
			ln := hunk.Lines[j]
			if ln.Kind == diff.LineDelete {
				continue
			}
			if slp107LineMatchesCleanup(ln.Content, identifier) {
				foundSuccess = true
				break
			}
		}

		if !foundSuccess {
			*out = append(*out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     filePath,
				Line:     cl.NewLineNo,
				Message:  "cleanup/destroy only found in error block — ensure resource is also released on success path",
				Snippet:  strings.TrimSpace(cl.Content),
			})
		}
	}
}

func slp107LineMatchesCleanup(content string, identifier string) bool {
	lower := strings.ToLower(content)
	if !slp107Cleanup.MatchString(content) && !strings.Contains(lower, "defer ") {
		return false
	}
	if identifier == "" {
		// For bare cleanup calls (no receiver), only accept other bare cleanup calls,
		// not deferred method calls with receivers.
		return slp107Cleanup.MatchString(content) && extractIdentifier(content) == ""
	}
	return extractIdentifier(content) == identifier
}

func slp107NextNonDeletedIndent(lines []diff.Line, start int) int {
	for i := start; i < len(lines); i++ {
		if lines[i].Kind == diff.LineDelete {
			continue
		}
		return getIndent(lines[i].Content)
	}
	return -1
}

func getIndent(s string) int {
	trimmed := strings.TrimLeft(s, " \t")
	if trimmed == "" {
		return 1000 // Blank lines don't end blocks
	}
	return len(s) - len(trimmed)
}

func extractIdentifier(content string) string {
	content = strings.TrimSpace(content)
	match := slp107IdentifierPattern.FindStringSubmatch(content)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}
