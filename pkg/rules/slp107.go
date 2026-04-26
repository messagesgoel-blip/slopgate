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

var slp107Cleanup = regexp.MustCompile(`(?i)(?:Close|Destroy|Cleanup|Release|Remove|Delete|Cancel)\s*\(`)

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
			var cleanupLines []diff.Line

			for k := range h.Lines {
				ln := &h.Lines[k]
				content := strings.TrimSpace(ln.Content)
				cLower := strings.ToLower(content)

				if !inErrorBlock {
					if strings.Contains(cLower, "if err") || strings.Contains(cLower, "catch") || strings.Contains(cLower, "except") {
						inErrorBlock = true
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
						if (!isPython && errorBraceDepth <= 0 && strings.Contains(content, "}")) || (isPython && k+1 < len(h.Lines) && getIndent(h.Lines[k+1].Content) <= errorIndentLevel) {
							if len(cleanupLines) > 0 {
								r.emitIfNoSuccessCleanup(&out, f.Path, cleanupLines, h, k+1)
							}
							inErrorBlock = false
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
							r.emitIfNoSuccessCleanup(&out, f.Path, cleanupLines, h, k)
						}
						inErrorBlock = false
						cleanupLines = nil
						// Re-check if this line starts a new error block
						if strings.Contains(cLower, "if err") || strings.Contains(cLower, "catch") || strings.Contains(cLower, "except") {
							inErrorBlock = true
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
						r.emitIfNoSuccessCleanup(&out, f.Path, cleanupLines, h, k+1)
					}
					inErrorBlock = false
					cleanupLines = nil
				}
			}
			// End of hunk also closes any open error block
			if inErrorBlock && len(cleanupLines) > 0 {
				r.emitIfNoSuccessCleanup(&out, f.Path, cleanupLines, h, len(h.Lines))
			}
		}
	}
	return out
}

func (r SLP107) emitIfNoSuccessCleanup(out *[]Finding, filePath string, cleanupLines []diff.Line, hunk *diff.Hunk, startIndex int) {
	for _, cl := range cleanupLines {
		identifier := extractIdentifier(cl.Content)
		foundSuccess := false
		for j := startIndex; j < len(hunk.Lines); j++ {
			ln := hunk.Lines[j]
			if ln.Kind == diff.LineDelete {
				continue
			}
			// Success path cleanup could be an existing line or a newly added line
			content := ln.Content
			if slp107Cleanup.MatchString(content) || strings.Contains(strings.ToLower(content), "defer ") {
				if identifier == "" || strings.Contains(content, identifier) {
					foundSuccess = true
					break
				}
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

func getIndent(s string) int {
	trimmed := strings.TrimLeft(s, " \t")
	if trimmed == "" {
		return 1000 // Blank lines don't end blocks
	}
	return len(s) - len(trimmed)
}

func extractIdentifier(content string) string {
	content = strings.TrimSpace(content)
	idx := strings.Index(content, ".Close")
	if idx == -1 {
		idx = strings.Index(content, ".Destroy")
	}
	if idx == -1 {
		idx = strings.Index(content, ".Cleanup")
	}
	if idx > 0 {
		start := idx - 1
		for start >= 0 && isAlphaNumeric(content[start]) {
			start--
		}
		return content[start+1 : idx]
	}
	return ""
}

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
