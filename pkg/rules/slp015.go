package rules

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP015 flags linter-suppression comments added in the current diff.
// These are comments that suppress linting or type-checking warnings,
// which AI agents frequently add to silence legitimate errors instead
// of fixing the underlying issue.
//
// This is distinct from SLP013 (commented-out code) — SLP015 specifically
// targets directives that tell tools to ignore problems.
//
// Detected patterns:
//   - Go:     //nolint, //nolint:..., //lint:ignore
//   - JS/TS:  // eslint-disable, // @ts-ignore, // @ts-nocheck, /* eslint-disable */
//   - Python: # noqa, # type: ignore, # pylint: disable
//   - Java:   @SuppressWarnings(...), // NOPMD
//   - Rust:   #[allow(...)], #[allow(dead_code)], etc.
type SLP015 struct{}

func (SLP015) ID() string                { return "SLP015" }
func (SLP015) DefaultSeverity() Severity { return SeverityWarn }
func (SLP015) Description() string {
	return "linter-suppression comment added instead of fixing the underlying issue"
}

// slp015Patterns matches linter-suppression directives in comments.
// Each pattern includes the language context it applies to.
var slp015Patterns = []struct {
	re      *regexp.Regexp
	lang    string // "go", "js", "py", "java", "rust", or "" for all
	example string
}{
	// Go: //nolint, //nolint:..., //lint:ignore
	{regexp.MustCompile(`//\s*nolint\b`), "go", "//nolint"},
	{regexp.MustCompile(`//\s*lint:ignore\b`), "go", "//lint:ignore"},

	// JS/TS: // eslint-disable, // @ts-ignore, // @ts-nocheck, /* eslint-disable */
	{regexp.MustCompile(`//\s*@ts-ignore\b`), "js", "// @ts-ignore"},
	{regexp.MustCompile(`//\s*@ts-nocheck\b`), "js", "// @ts-nocheck"},
	{regexp.MustCompile(`//\s*eslint-disable(?:-next-line|-line)?\b`), "js", "// eslint-disable"},
	{regexp.MustCompile(`/\*\s*eslint-disable\b`), "js", "/* eslint-disable */"},

	// Python: # noqa, # type: ignore, # pylint: disable=
	{regexp.MustCompile(`#\s*noqa\b`), "py", "# noqa"},
	{regexp.MustCompile(`#\s*type:\s*ignore\b`), "py", "# type: ignore"},
	{regexp.MustCompile(`#\s*pylint:\s*disable=`), "py", "# pylint: disable="},

	// Java: @SuppressWarnings, // NOPMD
	{regexp.MustCompile(`@SuppressWarnings\s*\(`), "java", "@SuppressWarnings"},
	{regexp.MustCompile(`//\s*NOPMD\b`), "java", "// NOPMD"},

	// Rust: #[allow(...)], #[allow(dead_code)], etc.
	{regexp.MustCompile(`#!\[\s*allow\s*\(`), "rust", "#![allow(...)]"},
	{regexp.MustCompile(`#\[\s*allow\s*\(`), "rust", "#[allow(...)]"},
}

// slp015FileLang determines the language for a file path, returning
// the short name used in slp015Patterns.
func slp015FileLang(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return "js"
	case ".py", ".pyi", ".pyw":
		return "py"
	case ".java", ".kt":
		return "java"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}

// stripStringLiterals removes the content of string literals (double-quoted,
// single-quoted, backtick-quoted) from a line but preserves comments. This is
// used for SLP015 because we want to detect linter-suppression directives that
// appear in comments, which would be stripped by stripCommentAndStrings.
//
// Limitation: this function processes each line independently, so multi-line
// string literals (e.g. Python triple-quoted strings) that span multiple lines
// are not tracked across calls. A string literal opened on a previous line will
// not suppress false positives on subsequent lines. This is acceptable for
// SLP015's diff-based analysis since multi-line strings in diffs are rare.
//
// Raw-string variants are also handled:
//   - Python r"..." / r'...' / R"..." / R'...' — no backslash escaping
//   - Rust r"..." and r#"..."# (arbitrary # count) — no backslash escaping;
//     Rust raw strings terminate only when the closing quote is followed by the
//     same number of '#' chars as the opening delimiter
func stripStringLiterals(s string) string {
	var b strings.Builder
	var quote byte
	rawMode := false // true → backslashes are not escape sequences
	rustHashes := 0  // number of '#' chars in a Rust r#"..."# delimiter
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if !rawMode && quote != '`' && c == '\\' && i+1 < len(s) {
				b.WriteString("  ") // blank escaped char
				i++                 // skip next
				continue
			}
			if c == quote {
				if rawMode && rustHashes > 0 {
					// Rust raw string: the closing quote must be followed by exactly
					// rustHashes '#' chars to end the literal.
					trailing := 0
					for i+1+trailing < len(s) && s[i+1+trailing] == '#' {
						trailing++
					}
					if trailing < rustHashes {
						b.WriteByte(' ') // embedded quote — not the real closing delimiter
						continue
					}
					// Consume the trailing hashes.
					for k := 0; k < rustHashes; k++ {
						b.WriteByte('#')
					}
					i += rustHashes
				}
				quote = 0
				rawMode = false
				rustHashes = 0
				b.WriteByte(c) // closing quote stays
			} else {
				b.WriteByte(' ') // blank string content
			}
			continue
		}
		// Detect a raw-string prefix: r/R optionally followed by '#' chars and a quote.
		// Covers Python r"..."  and Rust r"..." / r#"..."# / r##"..."## etc.
		if (c == 'r' || c == 'R') && i+1 < len(s) {
			hashes := 0
			k := i + 1
			for k < len(s) && s[k] == '#' {
				hashes++
				k++
			}
			if k < len(s) && (s[k] == '"' || s[k] == '\'') {
				b.WriteByte(c) // emit 'r'/'R'
				for h := 0; h < hashes; h++ {
					b.WriteByte('#')
				}
				b.WriteByte(s[k]) // opening quote stays
				quote = s[k]
				rawMode = true
				rustHashes = hashes
				i = k
				continue
			}
		}
		switch c {
		case '"', '\'', '`':
			quote = c
			b.WriteByte(c) // opening quote stays
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func (r SLP015) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		lang := slp015FileLang(f.Path)
		for _, ln := range f.AddedLines() {
			content := ln.Content
			// We want to match patterns in comments but NOT inside string
			// literals. stripStringLiterals blanks string contents but
			// preserves comments, so the regex can match directives in
			// comments without false positives from string literals.
			clean := stripStringLiterals(content)
			for _, p := range slp015Patterns {
				// Skip patterns that don't match this language.
				if p.lang != "" && lang != p.lang {
					continue
				}
				if p.re.MatchString(clean) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  fmt.Sprintf("linter-suppression comment added — fix the underlying issue instead of silencing it (%s)", p.example),
						Snippet:  strings.TrimSpace(content),
					})
					break // one finding per line
				}
			}
		}
	}
	return out
}
