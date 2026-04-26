package rules

import (
	"path"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP099 detects when a response struct/type field is added, renamed, or
// retyped in a non-test file without corresponding test file changes in
// the same diff. This is a common AI slop pattern: the agent changes a
// response shape but doesn't update the tests, causing test drift.
type SLP099 struct{}

func (SLP099) ID() string                { return "SLP099" }
func (SLP099) DefaultSeverity() Severity { return SeverityWarn }
func (SLP099) Description() string {
	return "response field changed without test update — tests may be stale"
}

var slp099GoStructField = regexp.MustCompile(`^\s*\w+\s+(?:\[\])?\*?\w+(?:\.\w+)?(?:\s+\x60[^\x60]*\x60)?$`)

var slp099TSInterfaceProp = regexp.MustCompile(`(?i)^(?:readonly\s+)?\w+(?:\?)?:\s*(?:string|number|boolean|Date|\[\]\w+|\w+\[\])[;,]?$`)

var slp099ResponseKeywords = map[string]struct{}{
	"response": {},
	"dto":      {},
	"output":   {},
	"result":   {},
	"payload":  {},
}

var slp099IgnoredTrailingTokens = map[string]struct{}{
	"model":  {},
	"schema": {},
	"type":   {},
}

var slp099CamelBoundaryLowerToUpper = regexp.MustCompile(`([a-z0-9])([A-Z])`)
var slp099CamelBoundaryAcronym = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
var slp099NonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

func hasResponseKeyword(name string) bool {
	tokens := slp099FilenameTokens(name)
	for i := len(tokens) - 1; i >= 0; i-- {
		if _, ok := slp099IgnoredTrailingTokens[tokens[i]]; ok {
			continue
		}
		_, ok := slp099ResponseKeywords[tokens[i]]
		return ok
	}
	return false
}

func slp099FilenameTokens(name string) []string {
	base := path.Base(name)
	stem := strings.TrimSuffix(base, path.Ext(base))
	stem = slp099CamelBoundaryAcronym.ReplaceAllString(stem, `$1 $2`)
	stem = slp099CamelBoundaryLowerToUpper.ReplaceAllString(stem, `$1 $2`)
	stem = slp099NonAlnum.ReplaceAllString(stem, " ")
	return strings.Fields(strings.ToLower(stem))
}

func (r SLP099) Check(d *diff.Diff) []Finding {
	var out []Finding
	changedFiles := make(map[string]bool)
	changedTestFiles := make(map[string]bool)

	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if isTestFile(f.Path) {
			changedTestFiles[f.Path] = true
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			if slp099GoStructField.MatchString(content) || slp099TSInterfaceProp.MatchString(content) {
				if hasResponseKeyword(f.Path) {
					changedFiles[f.Path] = true
				}
			}
		}
	}

	for _, f := range d.Files {
		if !changedFiles[f.Path] {
			continue
		}
		if testMatchesResponse(f.Path, changedTestFiles) {
			continue
		}
		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			if slp099GoStructField.MatchString(content) || slp099TSInterfaceProp.MatchString(content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "response field added/changed without test update — verify tests still match",
					Snippet:  content,
				})
			}
		}
	}
	return out
}

func testMatchesResponse(respPath string, testFiles map[string]bool) bool {
	if len(testFiles) == 0 {
		return false
	}
	respStem := slp099FileStem(respPath)
	respDir := path.Dir(respPath)

	for tf := range testFiles {
		if slp099FileStem(tf) != respStem {
			continue
		}
		if slp099RelatedDir(respDir, path.Dir(tf)) {
			return true
		}
	}
	return false
}

func slp099FileStem(filePath string) string {
	base := path.Base(filePath)
	stem := strings.TrimSuffix(base, path.Ext(base))
	switch {
	case strings.HasSuffix(stem, "_test"):
		return strings.TrimSuffix(stem, "_test")
	case strings.HasSuffix(stem, ".test"):
		return strings.TrimSuffix(stem, ".test")
	case strings.HasSuffix(stem, ".spec"):
		return strings.TrimSuffix(stem, ".spec")
	default:
		return stem
	}
}

func slp099RelatedDir(respDir, testDir string) bool {
	if respDir == testDir {
		return true
	}
	if path.Dir(testDir) == respDir || path.Dir(respDir) == testDir {
		return true
	}
	return path.Dir(respDir) == path.Dir(testDir)
}
