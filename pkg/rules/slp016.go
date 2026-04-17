package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP016 flags variable shadowing — when an inner scope declares a
// variable with the same name as one already seen in an outer scope.
// AI agents frequently shadow outer variables unintentionally,
// causing subtle bugs.
//
// Single order-sensitive pass over hunk lines: context lines seed
// outerNames first; then each added line is checked against
// outerNames before its own names are added.
//
// Exempt: single-letter loop iterators (i, j, k, _); Go's err at
// info level only; test files; doc files.
type SLP016 struct{}

func (SLP016) ID() string                { return "SLP016" }
func (SLP016) DefaultSeverity() Severity { return SeverityWarn }
func (SLP016) Description() string {
	return "variable shadows an outer-scope declaration with the same name"
}

var slp016DeclPattern = regexp.MustCompile(`(?:var|let|const|int|long|float|double|string|bool|auto)\s+(\w+)\b`)
var slp016AssignDecl = regexp.MustCompile(`(\w+)\s*:=`)
var slp016ForVar = regexp.MustCompile(`for\s*\(\s*\w+\s+(\w+)\b`)

var slp016ExemptNames = map[string]bool{
	"i": true, "j": true, "k": true, "_": true,
}

var slp016GoErrPattern = regexp.MustCompile(`\berr\s*:?=`)

func slp016ExtractDecls(content string) []string {
	var names []string
	for _, m := range slp016DeclPattern.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			names = append(names, m[1])
		}
	}
	for _, m := range slp016AssignDecl.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			names = append(names, m[1])
		}
	}
	for _, m := range slp016ForVar.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			names = append(names, m[1])
		}
	}
	return names
}

func (r SLP016) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		isTest := isGoTestFile(f.Path) || isJavaTestFile(f.Path) ||
			isPythonTestFile(f.Path) || isJSTestFile(f.Path) || isRustTestFile(f.Path)
		if isTest {
			continue
		}
		for _, h := range f.Hunks {
			outerNames := map[string]int{} // name → indent level (only already-seen)

			for _, ln := range h.Lines {
				if ln.Kind == diff.LineContext {
					for _, name := range slp016ExtractDecls(stripCommentAndStrings(ln.Content)) {
						outerNames[name] = leadingSpaces(ln.Content)
					}
					continue
				}
				if ln.Kind != diff.LineAdd {
					continue
				}
				clean := stripCommentAndStrings(ln.Content)
				indent := leadingSpaces(ln.Content)
				seen := map[string]bool{}
				for _, name := range slp016ExtractDecls(clean) {
					if seen[name] {
						continue
					}
					seen[name] = true

					// Check against already-seen outer names.
					if outerIndent, exists := outerNames[name]; exists {
						if indent > outerIndent && !slp016ExemptNames[name] {
							sev := SeverityWarn
							msg := fmt.Sprintf("variable %q shadows outer-scope declaration", name)
							isErr := slp016GoErrPattern.MatchString(clean) && name == "err"
							if isErr {
								sev = SeverityInfo
								msg = fmt.Sprintf("variable %q shadows outer err — consider renaming for clarity", name)
							}
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: sev,
								File:     f.Path,
								Line:     ln.NewLineNo,
								Message:  msg,
								Snippet:  strings.TrimSpace(ln.Content),
							})
						}
					}

					// Now add this name to outer scope for subsequent lines.
					if _, exists := outerNames[name]; !exists || outerNames[name] > indent {
						outerNames[name] = indent
					}
				}
			}
		}
	}
	return out
}
