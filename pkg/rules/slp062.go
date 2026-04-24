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
			bodyLines, nextIdx, closed := slp062CountFunctionLines(added, i)
			if closed && bodyLines > 50 {
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

// slp062CountFunctionLines counts added lines belonging to the function
// starting at `start`. It stops when the function scope closes or when added
// lines become non-contiguous, and reports whether the function was fully
// closed within the scanned region.
func slp062CountFunctionLines(added []diff.Line, start int) (int, int, bool) {
	bodyLines := 0
	depth := 0
	started := false
	last := start
	for j := start; j < len(added); j++ {
		if j > start && added[j].NewLineNo != added[j-1].NewLineNo+1 {
			return bodyLines, last, false
		}
		bodyLines++
		clean := stripGoLiteralsAndComments(added[j].Content)
		if strings.Contains(clean, "{") {
			started = true
		}
		depth += strings.Count(clean, "{") - strings.Count(clean, "}")
		last = j
		if started && depth <= 0 {
			return bodyLines, last, true
		}
	}
	return bodyLines, last, false
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
