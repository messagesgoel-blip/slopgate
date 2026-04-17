package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP014 flags debug prints (fmt.Println, console.log, print(, etc.)
// added in non-test, non-main, non-doc files.
//
// Rationale: print-to-stdout for debugging is the oldest AI-coding
// failure mode. The model adds a `fmt.Println("here")` to figure out
// why something broke, then commits it without cleanup. In tests and
// CLI entrypoints prints are legitimate; everywhere else they are slop.
type SLP014 struct{}

func (SLP014) ID() string                { return "SLP014" }
func (SLP014) DefaultSeverity() Severity { return SeverityBlock }
func (SLP014) Description() string {
	return "debug print statement added in a non-test, non-entrypoint file"
}

// debugPrintPatterns matches the most common per-language prints.
// Each pattern requires the call syntax to be present — "fmt.Println"
// inside a string or comment has its own filter below.
//
// We deliberately skip console.warn / console.error / console.info:
// those are almost always real error-logging calls, not leftover
// debugging output. console.log / console.debug / console.trace are
// the ones AI agents add mid-task and forget to remove.
var debugPrintPatterns = []*regexp.Regexp{
	// Go
	regexp.MustCompile(`\bfmt\.(Println|Printf|Print)\s*\(`),
	// TypeScript / JavaScript
	regexp.MustCompile(`\bconsole\.(log|debug|trace)\s*\(`),
	// Python
	regexp.MustCompile(`(^|\W)print\s*\(`),
	// Java
	regexp.MustCompile(`\bSystem\.(out|err)\.(println|printf|print)\s*\(`),
	// Rust
	regexp.MustCompile(`\b(println|eprintln|print|eprint)!\s*\(`),
	regexp.MustCompile(`\bdbg!\s*\(`),
}

// isSuppressedDebugFile reports whether the file path is a location
// where debug prints are legitimate: test files, the entire cmd/**
// tree (CLI entrypoints whose job is to print), docs, and scripts.
func isSuppressedDebugFile(path string) bool {
	base := filepath.Base(path)
	lower := strings.ToLower(path)

	// Go test files.
	if strings.HasSuffix(lower, "_test.go") {
		return true
	}
	// JS/TS test files by common conventions.
	if strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") {
		return true
	}
	// Python test files.
	if strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") {
		return true
	}
	// Java/Kotlin test files (JUnit convention: *Test.java, *Tests.java).
	if isJavaFile(path) && (strings.Contains(base, "Test") || strings.Contains(base, "test")) {
		return true
	}
	// Rust test files.
	if isRustTestFile(path) {
		return true
	}
	// Doc files.
	if isDocFile(path) {
		return true
	}
	// Go CLI packages: anything under cmd/** is a command entrypoint
	// whose role is to print output for the user. Suppressing by
	// directory (not filename) keeps a `pkg/cli/cmd_foo.go` honest
	// while letting a real CLI subcommand at `cmd/tool/cmd_foo.go`
	// print freely.
	if strings.HasPrefix(lower, "cmd/") || strings.Contains(lower, "/cmd/") {
		return true
	}
	// Top-level main.go in a single-package repo.
	if base == "main.go" && !strings.Contains(strings.TrimSuffix(path, base), "/") {
		return true
	}
	// Shell / scripts directories.
	if strings.HasPrefix(lower, "scripts/") || strings.HasPrefix(lower, "script/") {
		return true
	}
	return false
}

// stripCommentAndStrings removes comments (both line-end `//` and
// inline `/* ... */` block comments) and the contents of all three
// common string-literal kinds (double-quoted, single-quoted,
// backtick/raw) from a line so the debug-print patterns can't match
// inside them. It is intentionally simple — multi-line strings and
// unclosed block comments are out of scope for a single-line linter,
// and perfect escape handling is unnecessary.
func stripCommentAndStrings(s string) string {
	// Strip full-line comments first.
	trimmed := strings.TrimLeft(s, " \t")
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "* ") {
		return ""
	}

	var b strings.Builder
	var quote byte // 0 when not in a string; otherwise the opening quote char
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			// Raw strings (backtick) have no escape sequences.
			if quote != '`' && c == '\\' && i+1 < len(s) {
				i++ // skip the escaped char
				continue
			}
			if c == quote {
				quote = 0
				b.WriteByte(c)
			}
			continue
		}
		switch {
		case c == '"' || c == '\'' || c == '`':
			quote = c
			b.WriteByte(c)
		case c == '/' && i+1 < len(s) && s[i+1] == '/':
			// Line comment — discard rest of line.
			return b.String()
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			// Block comment — skip until closing */.
			i += 2
			for i < len(s)-1 {
				if s[i] == '*' && s[i+1] == '/' {
					i++ // advance past the '/'
					break
				}
				i++
			}
		case c == '#':
			return b.String()
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func (r SLP014) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isSuppressedDebugFile(f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			stripped := stripCommentAndStrings(ln.Content)
			if stripped == "" {
				continue
			}
			for _, p := range debugPrintPatterns {
				if p.MatchString(stripped) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "debug print committed — delete before merge or move to real logging",
						Snippet:  strings.TrimSpace(ln.Content),
					})
					break
				}
			}
		}
	}
	return out
}
