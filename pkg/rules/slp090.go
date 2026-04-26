package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP090 flags API endpoints that don't handle error responses properly.
// This can lead to unhandled exceptions and poor user experience.
type SLP090 struct{}

func (SLP090) ID() string                { return "SLP090" }
func (SLP090) DefaultSeverity() Severity { return SeverityWarn }
func (SLP090) Description() string {
	return "API endpoint missing error response handling - add error handling for 4xx/5xx cases"
}

var (
	// Route definition patterns
	slp090RoutePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(app|router|express)\.(get|post|put|patch|delete|all)\s*\(`),
		regexp.MustCompile(`(?i)\.route\s*\([^)]*\)\s*(?:\.get|\.post|\.put|\.patch|\.delete)`),

		// Go HTTP handlers
		regexp.MustCompile(`(?i)func\s+\w+\s*\(.*http\.ResponseWriter\s*,\s*.*\*http\.Request`),

		// Python Flask/FastAPI routes
		regexp.MustCompile(`(?i)(@app\.(route|get|post|put|patch|delete)|@router\.(route|get|post|put|patch|delete))`),
		regexp.MustCompile(`(?i)(@get|@post|@put|@patch|@delete)\s*\(`),
	}

	// Error handling patterns (what we're looking for)
	slp090ErrorPatterns = []*regexp.Regexp{
		// Try-catch
		regexp.MustCompile(`(?i)try\s*\{`),
		regexp.MustCompile(`(?i)catch\s*\(`),
		regexp.MustCompile(`(?i)except\s+`),

		// Error handling
		regexp.MustCompile(`(?i)if\s*\(.*err\s*[:=]`),
		regexp.MustCompile(`(?i)err\s*!=\s*nil`),
		regexp.MustCompile(`(?i)if\s*\(\s*!ok`),
		regexp.MustCompile(`(?i)handleError|throw\s+new|return\s+error|respondWithError`),
		regexp.MustCompile(`(?i)next\s*\(\s*err|\.catch\s*\(|\.then\s*\([^)]*\)\s*=>\s*\{?\s*if\s*\(.*err|catch\s*\(\s*err`),

		// Response status patterns - only match error responses
		regexp.MustCompile(`(?i)status\s*\(\s*(4|5)\d{2}\s*\)|res\.(status|send|json)\s*\([^)]*(?:error|fail|4xx|5xx)`),
		// Match 4xx/5xx status codes or error/fail in the call
		regexp.MustCompile(`(?i)res\.(status|send|json)\s*\(\s*(4|5)\d{2}\s*\)|res\.(status|send|json)\s*\([^)]*(?:error|fail)`),
	}
)

// Check scans API routes for missing error handling.
func (r SLP090) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		fileLower := strings.ToLower(f.Path)

		// Only check API/backend files
		if !strings.Contains(fileLower, "route") &&
			!strings.Contains(fileLower, "controller") &&
			!strings.Contains(fileLower, "handler") &&
			!strings.Contains(fileLower, "api") {
			continue
		}

		// Track routes and their error handling
		for _, h := range f.Hunks {
			inRoute := false
			routeStartLine := -1
			hasErrorHandling := false

			for j, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}

				content := strings.TrimSpace(ln.Content)

				// Check if this is a route definition
				isRoute := false
				for _, pattern := range slp090RoutePatterns {
					if pattern.MatchString(content) {
						isRoute = true
						break
					}
				}

				if isRoute {
					inRoute = true
					routeStartLine = j
					hasErrorHandling = false
					continue
				}

				// Check for error handling within a route
				if inRoute {
					for _, pattern := range slp090ErrorPatterns {
						if pattern.MatchString(content) {
							hasErrorHandling = true
							break
						}
					}

					// Check if we've exited the route (closing brace, next route, or end of function)
					if strings.Contains(content, "}") && j > routeStartLine {
						if !hasErrorHandling && routeStartLine >= 0 && !isDocFile(f.Path) {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     h.Lines[routeStartLine].NewLineNo,
								Message:  "API endpoint may be missing error response handling - add try-catch, error middleware, or return error response",
								Snippet:  strings.TrimSpace(h.Lines[routeStartLine].Content),
							})
						}
						inRoute = false
					}
				}
			}

			// Handle case where file ends with a route without error handling
			if inRoute && !hasErrorHandling && routeStartLine >= 0 && !isDocFile(f.Path) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     h.Lines[routeStartLine].NewLineNo,
					Message:  "API endpoint may be missing error response handling - add try-catch, error middleware, or return error response",
					Snippet:  strings.TrimSpace(h.Lines[routeStartLine].Content),
				})
			}
		}
	}
	return out
}
