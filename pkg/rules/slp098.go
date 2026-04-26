package rules

import (
	"regexp"
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
	hasNewRoute := false
	hasTestChange := false
	routeFiles := make(map[string]bool)

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if isTestFile(f.Path) || strings.Contains(f.Path, "_test.") || strings.Contains(f.Path, ".test.") || strings.Contains(f.Path, ".spec.") {
			hasTestChange = true
			continue
		}
		if isDocFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			for _, pat := range slp098RoutePatterns {
				if pat.MatchString(content) {
					hasNewRoute = true
					routeFiles[f.Path] = true
					break
				}
			}
			if hasNewRoute && len(routeFiles) > 0 {
				break
			}
		}
	}

	if hasNewRoute && !hasTestChange {
		for _, f := range d.Files {
			if !routeFiles[f.Path] {
				continue
			}
			for _, ln := range f.AddedLines() {
				content := strings.TrimSpace(ln.Content)
				for _, pat := range slp098RoutePatterns {
					if pat.MatchString(content) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "new route added without corresponding test changes in this diff",
							Snippet:  content,
						})
						break
					}
				}
			}
		}
	}

	return out
}
