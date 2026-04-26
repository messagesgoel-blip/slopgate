package rules

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP098 flags new API routes or handlers added without any corresponding
// test changes in the same diff. This is a common AI slop pattern: adding
// route handlers without tests.
type SLP098 struct{}

func (SLP098) ID() string                { return "SLP098" }
func (SLP098) DefaultSeverity() Severity { return SeverityWarn }
func (SLP098) Description() string {
	return "new route/handler added without test — add a test for the new endpoint"
}

var slp098RoutePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:app|router|r)\.(?:get|post|put|delete|patch|head|options)\s*\(`),
	regexp.MustCompile(`(?i)@(?:Get|Post|Put|Delete|Patch|RequestMapping)\s*\(`),
	regexp.MustCompile(`(?i)\.(?:HandleFunc|Handle)\s*\(`),
	regexp.MustCompile(`(?i)(?:mux|router)\.(?:HandleFunc|Handle|NewRoute)\s*\(`),
	regexp.MustCompile(`(?i)(?:group|route)\s*\(\s*["'\x60]/`),
	regexp.MustCompile(`(?i)@(?:route|app\.route|blueprint\.route)\s*\(`),
	regexp.MustCompile(`(?i)\.(?:AddRoute|MapPath|HandlePath)\s*\(`),
}

func (r SLP098) Check(d *diff.Diff) []Finding {
	var out []Finding
	routeFiles := make(map[string]bool)
	testFiles := make(map[string]bool)

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if isTestFile(f.Path) {
			if len(f.AddedLines()) > 0 {
				testFiles[f.Path] = true
			}
			continue
		}
		if isDocFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			for _, pat := range slp098RoutePatterns {
				if pat.MatchString(content) {
					routeFiles[f.Path] = true
					break
				}
			}
			if routeFiles[f.Path] {
				break
			}
		}
	}

	routePaths := make([]string, 0, len(routeFiles))
	for rf := range routeFiles {
		routePaths = append(routePaths, rf)
	}
	sort.Strings(routePaths)

	for _, rf := range routePaths {
		base := strings.TrimSuffix(rf, path.Ext(rf))

		foundTest := false
		for tf := range testFiles {
			if slp098TestMatches(tf, base) {
				foundTest = true
				break
			}
		}

		if !foundTest {
			// Emit findings for this file
			for _, f := range d.Files {
				if f.Path != rf {
					continue
				}
				for _, ln := range f.AddedLines() {
					for _, pat := range slp098RoutePatterns {
						if pat.MatchString(ln.Content) {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     ln.NewLineNo,
								Message:  "new route added without corresponding test changes for this module in this diff",
								Snippet:  ln.Content,
							})
							break
						}
					}
				}
			}
		}
	}

	return out
}

// slp098TestMatches returns true if testPath is considered related to sourceBase
// (sourceBase is the route file path without extension).
func slp098TestMatches(testPath, sourceBase string) bool {
	testBase := slp098TestTarget(testPath)
	if testBase == sourceBase {
		return true
	}

	if testBase == "" {
		return false
	}

	// Also check if the test path lives under a parallel test/tests directory.
	// Strip common test root prefixes and compare the remainder against the
	// module-relative source path (exact match only — no loose suffix/base checks).
	for _, root := range []string{"tests/", "test/", "src/tests/", "src/test/"} {
		if strings.HasPrefix(testBase, root) {
			remainder := testBase[len(root):]
			// Exact match
			if sourceBase == remainder {
				return true
			}
			// Exact module-relative match: strip a known source root from sourceBase
			for _, srcRoot := range []string{"src", "lib", "app"} {
				prefix := srcRoot + "/"
				if strings.HasPrefix(sourceBase, prefix) {
					rel := strings.TrimPrefix(sourceBase, prefix)
					if rel == remainder {
						return true
					}
				}
			}
		}
	}

	// Handle source files under src/, lib/, app/ — check parallel test(s)/ directory.
	srcBase := path.Base(sourceBase)
	testBaseName := path.Base(testBase)
	if srcBase == testBaseName {
		// Same filename stem — check if directories are parallel (e.g., src/foo vs tests/foo)
		srcDir := path.Dir(sourceBase)
		testDir := path.Dir(testBase)
		for _, root := range []string{"tests", "test", "src/tests", "src/test"} {
			if testDir == root || strings.HasPrefix(testDir, root+"/") {
				remainder := strings.TrimPrefix(testDir, root)
				remainder = strings.TrimPrefix(remainder, "/")
				if remainder == "" || strings.HasSuffix(srcDir, remainder) {
					return true
				}
			}
		}
		for _, srcRoot := range []string{"src", "lib", "app"} {
			srcReplaced := strings.TrimPrefix(srcDir, srcRoot+"/")
			if srcReplaced == srcDir {
				continue
			}
			for _, testRoot := range []string{"tests", "test"} {
				if testDir == testRoot+"/"+srcReplaced {
					return true
				}
			}
			// Handle src/<module>/test[s] pattern (e.g., src/routes/test vs src/routes)
			for _, testSuffix := range []string{"/test", "/tests"} {
				if testDir == srcRoot+"/"+srcReplaced+testSuffix {
					return true
				}
				// Also handle when testDir has the srcRoot prefix and a test suffix
				if strings.TrimSuffix(testDir, testSuffix) == srcRoot+"/"+srcReplaced {
					return true
				}
			}
		}
	}
	return false
}

func slp098TestTarget(testPath string) string {
	ext := path.Ext(testPath)
	stem := strings.TrimSuffix(testPath, ext)
	switch {
	case strings.HasSuffix(stem, "_test"):
		return strings.TrimSuffix(stem, "_test")
	case strings.HasSuffix(stem, ".test"):
		return strings.TrimSuffix(stem, ".test")
	case strings.HasSuffix(stem, ".spec"):
		return strings.TrimSuffix(stem, ".spec")
	default:
		// Strip "test"/"tests" only when they form the standalone basename
		// or are preceded by a delimiter, to avoid truncating words like "latest".
		base := path.Base(stem)
		lowerBase := strings.ToLower(base)
		dir := path.Dir(stem)
		for _, sfx := range []string{"tests", "test"} {
			if lowerBase == sfx {
				return dir
			}
			for _, delim := range []string{".", "-"} {
				if strings.HasSuffix(lowerBase, delim+sfx) {
					newBase := base[:len(base)-len(delim+sfx)]
					return path.Join(dir, newBase)
				}
			}
		}
		return stem
	}
}
