package rules

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP062 flags functions longer than 50 added lines or Go files with
// more than 500 added lines.
//
// Scope: Go files only.
type SLP062 struct{}

func (SLP062) ID() string                { return "SLP062" }
func (SLP062) DefaultSeverity() Severity { return SeverityWarn }
func (SLP062) Description() string {
	return "function or file complexity exceeds thresholds (>50 / >500 lines)"
}

// slp062FuncSignature matches a Go function signature line.
var slp062FuncSignature = regexp.MustCompile(`^func\s*(?:\([^)]+\)\s*)?\w+\s*(?:\[(?:[^\[\]]|\[[^\[\]]*\])*\])?\s*\(`)

func (r SLP062) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		added := f.AddedLines()
		// File-level check.
		if len(added) > 500 {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     0,
				Message:  "file adds " + strconv.Itoa(len(added)) + " lines — consider refactoring into smaller files",
				Snippet:  "",
			})
		}

		// Function-level check using added lines only.
		for i := 0; i < len(added); i++ {
			ln := added[i]
			trimmed := strings.TrimSpace(ln.Content)
			if !slp062FuncSignature.MatchString(trimmed) {
				continue
			}
			// Extract function name for the message.
			funcName := extractFuncName(trimmed)
			startLine := ln.NewLineNo
			// Brace depth counting on THIS line only (strip string literals and comments first).
			clean := stripGoLiteralsAndComments(ln.Content)
			depth := strings.Count(clean, "{") - strings.Count(clean, "}")
			if depth <= 0 {
				// Opening brace is on a subsequent added line.
				j := i + 1
				for j < len(added) && depth <= 0 {
					cleanNext := stripGoLiteralsAndComments(added[j].Content)
					depth += strings.Count(cleanNext, "{") - strings.Count(cleanNext, "}")
					j++
				}
				if depth <= 0 {
					continue
				}
				bodyLines, nextIdx := countBodyLines(added, j, depth)
				if bodyLines > 50 {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     startLine,
						Message:  "function " + funcName + " is " + strconv.Itoa(bodyLines) + " lines — consider breaking it up",
						Snippet:  trimmed,
					})
				}
				if nextIdx > i {
					i = nextIdx
				}
				continue
			}
			// Opening brace is on the signature line.
			bodyLines, nextIdx := countBodyLines(added, i+1, depth)
			if bodyLines > 50 {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     startLine,
					Message:  "function " + funcName + " is " + strconv.Itoa(bodyLines) + " lines — consider breaking it up",
					Snippet:  trimmed,
				})
			}
			if nextIdx > i {
				i = nextIdx
			}
		}
	}
	return out
}

// countBodyLines counts lines from index `start` in `added` until `depth`
// reaches 0, using cleaned (literal/comment-free) content for brace counting.
// It returns the number of body lines and the last consumed index.
func countBodyLines(added []diff.Line, start, depth int) (int, int) {
	bodyLines := 1 // signature counts as part of the function
	j := start
	for j < len(added) && depth > 0 {
		bodyLines++
		clean := stripGoLiteralsAndComments(added[j].Content)
		depth += strings.Count(clean, "{") - strings.Count(clean, "}")
		j++
	}
	return bodyLines, j - 1
}

// stripGoLiteralsAndComments removes Go string literals, comments, and rune
// literals from a line so brace counting doesn't miscount braces inside them.
func stripGoLiteralsAndComments(s string) string {
	var b strings.Builder
	inString := false
	inRawString := false
	inBlockComment := false
	inRune := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inBlockComment {
			if c == '*' && i+1 < len(s) && s[i+1] == '/' {
				inBlockComment = false
				i++ // skip past the '/'
			}
			continue
		}
		if inRune {
			if c == '\\' && i+1 < len(s) {
				i++ // skip escaped char
				continue
			}
			if c == '\'' {
				inRune = false
			}
			continue
		}
		if inRawString {
			if c == '`' {
				inRawString = false
			}
			continue
		}
		if inString {
			if c == '\\' && i+1 < len(s) {
				i++ // skip escaped char
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '/' {
			break // line comment: skip rest of line
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			inBlockComment = true
			i++ // skip past the '*'
			continue
		}
		if c == '"' {
			inString = true
			continue
		}
		if c == '`' {
			inRawString = true
			continue
		}
		if c == '\'' {
			inRune = true
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func extractFuncName(line string) string {
	// Heuristic: after `func`, optionally `(recv) `, then the name before `(`.
	s := strings.TrimPrefix(line, "func")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") {
		idx := strings.Index(s, ")")
		if idx >= 0 {
			s = strings.TrimSpace(s[idx+1:])
		}
	}
	idx := strings.Index(s, "(")
	if idx > 0 {
		return strings.TrimSpace(s[:idx])
	}
	return "<unknown>"
}
