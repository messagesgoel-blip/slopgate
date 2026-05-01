package rules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

const slp007GitShowTimeout = 2 * time.Second

// SLP007 flags imports that are added in a diff but never referenced in any
// other added line of the same file. This catches the classic AI "just in
// case" import slop where an agent adds an import but never uses the package.
//
// Supported languages:
//   - Go: import "pkg" / import alias "pkg" / import ( ... ) groups
//   - JS/TS: import { X } from 'y' / import X from 'y'
//   - Python: import X / from Y import X
//   - Java: import com.foo.Bar;
//   - Rust: use crate::foo::Bar; / use std::foo::Bar;
//
// Exempt:
//   - Go blank imports: import _ "pkg" (side-effect imports)
//   - Go dot imports: import . "pkg" (too ambiguous)
//   - Java wildcard imports: import com.foo.*; (too ambiguous)
//   - Rust glob imports: use foo::*; (too ambiguous)
//   - Pre-existing imports (only newly added import lines are checked)
type SLP007 struct{}

func (SLP007) ID() string                { return "SLP007" }
func (SLP007) DefaultSeverity() Severity { return SeverityWarn }
func (SLP007) Description() string {
	return "import added in diff but not used in any added line"
}

// importInfo holds a parsed import with the identifier to search for.
type importInfo struct {
	content   string // full added line content (for snippet)
	lineNo    int    // new file line number
	ident     string // identifier to search for in other added lines
	pkgPath   string // full import path (for message)
	isGrouped bool   // true if inside import ( ... )
}

// --- Go import patterns ---

// slp007GoSingleImport matches a single Go import outside a group.
// Captures: 1=alias (optional: word alias with optional space, or . or _ with
// optional space), 2=pkg path
var slp007GoSingleImport = regexp.MustCompile(`^\s*import\s+((?:\w+(?:\s+)?)|[._](?:\s+)?)?"([^"]+)"`)

// slp007GoGroupedImport matches a Go import line inside import ( ... ).
// Captures: 1=alias (optional), 2=pkg path
var slp007GoGroupedImport = regexp.MustCompile(`^\s*((?:\w+(?:\s+)?)|[._](?:\s+)?)?"([^"]+)"`)

// slp007GoGroupOpen matches the opening of a Go grouped import block.
var slp007GoGroupOpen = regexp.MustCompile(`^\s*import\s*\(\s*$`)

// slp007GoGroupClose matches the closing of a Go grouped import block.
var slp007GoGroupClose = regexp.MustCompile(`^\s*\)\s*$`)

// --- JS/TS import patterns ---

// slp007JSNamedImport matches `import { X, Y, Z } from 'pkg'` or
// `import { X as Y } from 'pkg'`.
var slp007JSNamedImport = regexp.MustCompile(`^\s*import\s*\{([^}]+)\}\s*from\s*['"][^'"]+['"]`)

// slp007JSDefaultImport matches `import X from 'pkg'` where X is not
// a brace or asterisk.
var slp007JSDefaultImport = regexp.MustCompile(`^\s*import\s+([A-Za-z_$][\w$]*)\s+from\s*['"][^'"]+['"]`)

// slp007JSStarImport matches `import * as X from 'pkg'`.
var slp007JSStarImport = regexp.MustCompile(`^\s*import\s*\*\s*as\s+([A-Za-z_$][\w$]*)\s+from\s*['"][^'"]+['"]`)

// slp007JSNamedItem matches a single item inside braces: X or X as Y.
var slp007JSNamedItem = regexp.MustCompile(`(\w+)\s*(?:as\s+(\w+))?`)

// --- Python import patterns ---

// slp007PyPlainImport matches `import X` or `import X, Y` or `import X as Y`.
var slp007PyPlainImport = regexp.MustCompile(`^\s*import\s+(.+)`)

// slp007PyFromImport matches `from X import Y` (or `from X import Y as Z`).
// Group 1: module path, Group 2: the items after import.
var slp007PyFromImport = regexp.MustCompile(`^\s*from\s+([\w.]+)\s+import\s+(.+)`)

// --- Java import patterns ---

// slp007JavaImport matches `import com.foo.Bar;`. Captures the fully-qualified name.
var slp007JavaImport = regexp.MustCompile(`^\s*import\s+([\w.]+)\s*;`)

// --- Rust use patterns ---

// slp007RustUse matches `use foo::bar::Baz;` or `use foo::bar::Baz as Qux;`.
// Captures the last segment (or the alias if present).
//
// Limitation: grouped imports (use std::{collections::HashMap, io::Read};)
// and self imports (use crate::module::self;) are not matched. Only simple
// single-path use statements are detected. Grouped imports produce false
// negatives — they are silently skipped rather than falsely flagged.
var slp007RustUse = regexp.MustCompile(`^\s*use\s+([\w:]+)::(\w+)\s*(?:as\s+(\w+))?\s*;`)

// goImportIdent returns the identifier to search for given a Go import path
// and optional alias. If the import is blank or dot, it returns ("", false).
func goImportIdent(pkgPath, prefix string) (string, bool) {
	prefix = strings.TrimSpace(prefix)
	switch prefix {
	case "_":
		// Blank import — side-effect import, not a finding.
		return "", false
	case ".":
		// Dot import — too ambiguous, skip.
		return "", false
	}
	if prefix != "" {
		// Explicit alias: search for "alias."
		return prefix, true
	}
	// No alias: use last segment of the import path.
	// e.g. "fmt" -> "fmt", "encoding/json" -> "json"
	parts := strings.Split(pkgPath, "/")
	last := parts[len(parts)-1]
	return last, true
}

// parseGoImports extracts newly added Go import identifiers from the added
// lines of a file. It handles both single imports and grouped import ( ... )
// blocks.
func parseGoImports(added []diff.Line) []importInfo {
	var result []importInfo

	inGroup := false
	for _, ln := range added {
		content := ln.Content

		if inGroup {
			if slp007GoGroupClose.MatchString(content) {
				inGroup = false
				continue
			}
			m := slp007GoGroupedImport.FindStringSubmatch(content)
			if m == nil {
				continue
			}
			prefix := m[1]
			pkgPath := m[2]
			ident, ok := goImportIdent(pkgPath, prefix)
			if !ok {
				continue
			}
			result = append(result, importInfo{
				content:   content,
				lineNo:    ln.NewLineNo,
				ident:     ident,
				pkgPath:   pkgPath,
				isGrouped: true,
			})
			continue
		}

		// Check for group opening: import (
		if slp007GoGroupOpen.MatchString(content) {
			inGroup = true
			continue
		}

		// Check for single import.
		m := slp007GoSingleImport.FindStringSubmatch(content)
		if m == nil {
			continue
		}
		prefix := m[1]
		pkgPath := m[2]
		ident, ok := goImportIdent(pkgPath, prefix)
		if !ok {
			continue
		}
		result = append(result, importInfo{
			content: content,
			lineNo:  ln.NewLineNo,
			ident:   ident,
			pkgPath: pkgPath,
		})
	}

	return result
}

// parseJSImports extracts newly added JS/TS import identifiers from the added
// lines of a file.
func parseJSImports(added []diff.Line) []importInfo {
	var result []importInfo

	for _, ln := range added {
		content := ln.Content

		// import * as X from 'y' — ident is X
		if m := slp007JSStarImport.FindStringSubmatch(content); m != nil {
			result = append(result, importInfo{
				content: content,
				lineNo:  ln.NewLineNo,
				ident:   m[1],
				pkgPath: "star import",
			})
			continue
		}

		// import { X, Y, Z } from 'y'
		if m := slp007JSNamedImport.FindStringSubmatch(content); m != nil {
			braces := m[1]
			for _, item := range parseJSNamedItems(braces) {
				result = append(result, importInfo{
					content: content,
					lineNo:  ln.NewLineNo,
					ident:   item,
					pkgPath: "named import",
				})
			}
			continue
		}

		// import X from 'y' — default import
		if m := slp007JSDefaultImport.FindStringSubmatch(content); m != nil {
			result = append(result, importInfo{
				content: content,
				lineNo:  ln.NewLineNo,
				ident:   m[1],
				pkgPath: "default import",
			})
			continue
		}
	}

	return result
}

// parseJSNamedItems extracts individual identifiers from the braces content
// of a named import. Handles "X" and "X as Y" — returns the local name (Y
// for aliases, X otherwise).
func parseJSNamedItems(braces string) []string {
	var items []string
	for _, item := range strings.Split(braces, ",") {
		item = strings.TrimSpace(item)
		item = strings.TrimPrefix(item, "type ")
		if item == "" {
			continue
		}
		m := slp007JSNamedItem.FindStringSubmatch(item)
		if m == nil {
			continue
		}
		name := m[1]
		alias := m[2]
		if alias != "" {
			items = append(items, alias)
			continue
		}
		items = append(items, name)
	}
	return items
}

// --- Python import parsing ---

// parsePythonImports extracts newly added Python import identifiers from
// the added lines of a file.
func parsePythonImports(added []diff.Line) []importInfo {
	var result []importInfo
	for _, ln := range added {
		content := ln.Content

		// from X import Y [as Z], A [as B]
		if m := slp007PyFromImport.FindStringSubmatch(content); m != nil {
			modulePath := m[1]
			items := m[2]
			for _, item := range strings.Split(items, ",") {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				// Handle "X as Y" — the local name is Y.
				parts := strings.SplitN(item, " as ", 2)
				localName := strings.TrimSpace(parts[0])
				if len(parts) == 2 {
					localName = strings.TrimSpace(parts[1])
				}
				result = append(result, importInfo{
					content: content,
					lineNo:  ln.NewLineNo,
					ident:   localName,
					pkgPath: modulePath,
				})
			}
			continue
		}

		// import X, Y
		if m := slp007PyPlainImport.FindStringSubmatch(content); m != nil {
			names := strings.Split(m[1], ",")
			for _, name := range names {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				// Handle "import X as Y".
				parts := strings.SplitN(name, " as ", 2)
				localName := strings.TrimSpace(parts[0])
				if len(parts) == 2 {
					localName = strings.TrimSpace(parts[1])
				}
				// For bare "import os.path", the local name is the first segment.
				if idx := strings.Index(localName, "."); idx >= 0 {
					localName = localName[:idx]
				}
				result = append(result, importInfo{
					content: content,
					lineNo:  ln.NewLineNo,
					ident:   localName,
					pkgPath: name,
				})
			}
			continue
		}
	}
	return result
}

// --- Java import parsing ---

// parseJavaImports extracts newly added Java import identifiers from the
// added lines of a file.
func parseJavaImports(added []diff.Line) []importInfo {
	var result []importInfo
	for _, ln := range added {
		content := ln.Content
		m := slp007JavaImport.FindStringSubmatch(content)
		if m == nil {
			continue
		}
		fqn := m[1]
		// Skip wildcard imports (import com.foo.*;) — too ambiguous.
		if strings.HasSuffix(fqn, ".*") {
			continue
		}
		// The identifier is the last segment: com.foo.Bar -> Bar.
		lastDot := strings.LastIndex(fqn, ".")
		if lastDot < 0 || lastDot == len(fqn)-1 {
			continue
		}
		ident := fqn[lastDot+1:]
		result = append(result, importInfo{
			content: content,
			lineNo:  ln.NewLineNo,
			ident:   ident,
			pkgPath: fqn,
		})
	}
	return result
}

// --- Rust use parsing ---

// parseRustImports extracts newly added Rust use identifiers from the
// added lines of a file.
func parseRustImports(added []diff.Line) []importInfo {
	var result []importInfo
	for _, ln := range added {
		content := ln.Content
		m := slp007RustUse.FindStringSubmatch(content)
		if m == nil {
			continue
		}
		// m[1]: path (e.g. "std::collections"), m[2]: name (e.g. "HashMap"),
		// m[3]: optional alias (e.g. "MyMap")
		ident := m[2]
		if m[3] != "" {
			ident = m[3]
		}
		// Skip glob imports: use foo::*; won't match the regex, but be safe.
		if ident == "*" {
			continue
		}
		pkgPath := m[1] + "::" + m[2]
		result = append(result, importInfo{
			content: content,
			lineNo:  ln.NewLineNo,
			ident:   ident,
			pkgPath: pkgPath,
		})
	}
	return result
}

// identUsedInAddedLines reports whether any added line (other than import
// lines themselves) contains a reference to the given identifier.
// For Go: looks for "ident." (package qualifier).
// For JS/TS: looks for the bare identifier as a word boundary.
// skipLineN is the line number of the import line to skip (so we don't
// falsely detect the identifier in its own import declaration).
func identUsedInAddedLines(ident string, added []diff.Line, goMode bool, skipLineN int) bool {
	var searchPat string
	if goMode {
		searchPat = ident + "."
	} else {
		searchPat = ident
	}

	for _, ln := range added {
		// Skip the import line itself.
		if ln.NewLineNo == skipLineN {
			continue
		}
		content := ln.Content

		if goMode {
			// For Go, search for ident. as a package qualifier.
			if strings.Contains(content, searchPat) {
				return true
			}
		} else {
			// For JS/TS, use word-boundary check.
			if wordInLine(content, ident) {
				return true
			}
		}
	}
	return false
}

func slp007ResolveFile(repoRoot, relPath string) (string, bool) {
	if repoRoot == "" {
		return "", false
	}
	cleanSlash, ok := slp007CleanRelativePath(relPath)
	if !ok {
		return "", false
	}

	rootAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", false
	}
	rootEval, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", false
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(cleanSlash)))
	if err != nil {
		return "", false
	}
	targetEval, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(rootEval, targetEval)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return targetEval, true
}

func slp007CleanRelativePath(relPath string) (string, bool) {
	if filepath.IsAbs(filepath.FromSlash(relPath)) {
		return "", false
	}
	cleanSlash := path.Clean(strings.ReplaceAll(relPath, "\\", "/"))
	if cleanSlash == "." || cleanSlash == ".." || strings.HasPrefix(cleanSlash, "../") {
		return "", false
	}
	return cleanSlash, true
}

func slp007FileContent(d *diff.Diff, relPath string) (string, bool) {
	if d == nil || d.RepoRoot == "" {
		return "", false
	}
	if d.SnapshotWorktree {
		resolved, ok := slp007ResolveFile(d.RepoRoot, relPath)
		if !ok {
			return "", false
		}
		content, err := os.ReadFile(resolved) // #nosec G304 -- path is constrained to the repo root above.
		if err != nil {
			return "", false
		}
		return string(content), true
	}
	if d.SnapshotRef == "" {
		return "", false
	}
	cleanSlash, ok := slp007CleanRelativePath(relPath)
	if !ok {
		return "", false
	}
	switch d.SnapshotRef {
	case ":":
		ctx, cancel := context.WithTimeout(context.Background(), slp007GitShowTimeout)
		defer context.CancelFunc(cancel)()
		cmd := exec.CommandContext(ctx, "git", "show")
		cmd.Dir = d.RepoRoot
		cmd.Args = append(cmd.Args, ":"+cleanSlash)
		out, err := cmd.Output()
		if err != nil {
			return "", false
		}
		return string(out), true
	case "HEAD":
		ctx, cancel := context.WithTimeout(context.Background(), slp007GitShowTimeout)
		defer context.CancelFunc(cancel)()
		cmd := exec.CommandContext(ctx, "git", "show")
		cmd.Dir = d.RepoRoot
		cmd.Args = append(cmd.Args, "HEAD:"+cleanSlash)
		out, err := cmd.Output()
		if err != nil {
			return "", false
		}
		return string(out), true
	default:
		return "", false
	}
}

func slp007FileLines(d *diff.Diff, relPath string) ([]string, bool) {
	content, ok := slp007FileContent(d, relPath)
	if !ok {
		return nil, false
	}
	return strings.Split(content, "\n"), true
}

func identUsedInFile(ident string, lines []string, goMode bool, skipLineN int) bool {
	if len(lines) == 0 {
		return false
	}

	var searchPat string
	if goMode {
		searchPat = ident + "."
	}

	for i, line := range lines {
		if i+1 == skipLineN {
			continue
		}
		if goMode {
			if strings.Contains(line, searchPat) {
				return true
			}
			continue
		}
		if slp007IsImportLikeLine(line) {
			continue
		}
		if wordInLine(stripCommentAndStrings(line), ident) {
			return true
		}
	}
	return false
}

func slp007IsImportLikeLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "import "):
		return true
	case strings.HasPrefix(trimmed, "from ") && strings.Contains(trimmed, " import "):
		return true
	case strings.HasPrefix(trimmed, "use "):
		return true
	case strings.HasPrefix(trimmed, "export ") && strings.Contains(trimmed, " from "):
		return true
	default:
		return false
	}
}

// wordInLine reports whether the given word appears as a whole word in the
// line content. This prevents false positives like "Stateful" matching
// "State".
func wordInLine(line, word string) bool {
	idx := 0
	for {
		pos := strings.Index(line[idx:], word)
		if pos < 0 {
			return false
		}
		pos += idx
		// Check char before the match.
		if pos > 0 {
			ch := line[pos-1]
			if isWordChar(ch) {
				idx = pos + 1
				continue
			}
		}
		// Check char after the match.
		end := pos + len(word)
		if end < len(line) {
			ch := line[end]
			if isWordChar(ch) {
				idx = pos + 1
				continue
			}
		}
		return true
	}
}

// isWordChar reports whether the byte is a word character (alphanumeric,
// underscore, or dollar sign).
func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '$'
}

func (r SLP007) Check(d *diff.Diff) []Finding {
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		added := f.AddedLines()
		if len(added) == 0 {
			continue
		}

		var imports []importInfo
		var goMode bool

		switch {
		case isGoFile(f.Path):
			imports = parseGoImports(added)
			goMode = true
		case isJSOrTSFile(f.Path):
			imports = parseJSImports(added)
			goMode = false
		case isPythonFile(f.Path):
			imports = parsePythonImports(added)
			goMode = false
		case isJavaFile(f.Path):
			imports = parseJavaImports(added)
			goMode = false
		case isRustFile(f.Path):
			imports = parseRustImports(added)
			goMode = false
		default:
			continue
		}

		var fileLines []string
		fileLinesLoaded := false
		haveFileLines := false

		for _, imp := range imports {
			if identUsedInAddedLines(imp.ident, added, goMode, imp.lineNo) {
				continue
			}
			if !fileLinesLoaded && d != nil {
				fileLines, haveFileLines = slp007FileLines(d, f.Path)
				fileLinesLoaded = true
			}
			if haveFileLines && identUsedInFile(imp.ident, fileLines, goMode, imp.lineNo) {
				continue
			}

			msg := fmt.Sprintf("import %q added but %s is never used in any added line", imp.pkgPath, imp.ident)
			if goMode && !imp.isGrouped {
				msg = fmt.Sprintf("import %q added but %s. is never used in any added line", imp.pkgPath, imp.ident)
			}

			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     imp.lineNo,
				Message:  msg,
				Snippet:  strings.TrimSpace(imp.content),
			})
		}
	}

	return out
}
