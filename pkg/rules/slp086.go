package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP086 flags potential missing authorization checks on sensitive endpoints.
// This can lead to privilege escalation and unauthorized access.
type SLP086 struct{}

func (SLP086) ID() string                { return "SLP086" }
func (SLP086) DefaultSeverity() Severity { return SeverityWarn }
func (SLP086) Description() string {
	return "missing authorization check on sensitive endpoint - verify user permissions before processing"
}

var (
	// Sensitive action keywords
	slp086SensitiveActions = []string{
		"delete", "remove", "destroy", "erase",
		"update", "modify", "change", "set",
		"grant", "assign", "add",
		"password", "secret", "token", "key",
		"salary", "finance", "bill",
		"transfer", "withdraw", "deposit",
	}
	// Auth patterns that must appear on same side of route body
	slp086AuthPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(checkPermission|hasPermission|authorize|hasRole|isAuthorized|requireAuth)\s*\(`),
		regexp.MustCompile(`(?i)\.isAdmin|\.isAuth|\.isAuthorized|session\.user|session\.id`),
		regexp.MustCompile(`(?i)ctx\.Value\s*\([^)]*(?:user|auth|session)`),
		regexp.MustCompile(`(?i)if\s*\(\s*\w+\s*\.\s*(isAdmin|isAuth|isAuthorized)`),
		regexp.MustCompile(`(?i)if\s*\(\s*!?\s*\w+\.(isAdmin|isAuth|isAuthorized)`), // Auth check with optional negation
		regexp.MustCompile(`(?i)res\.(status|send|json)\s*\([^)]*403|res\.(status|send|json)\s*\([^)]*Forbidden`),
	}
)

func (r SLP086) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		// Only check API/backend files
		fileLower := strings.ToLower(f.Path)
		// Check directory names with substring match (api, route, controller, handler)
		hasDirName := strings.Contains(fileLower, "api") ||
			strings.Contains(fileLower, "route") ||
			strings.Contains(fileLower, "controller") ||
			strings.Contains(fileLower, "handler")
		// Check extension strictly using filepath.Ext
		ext := strings.ToLower(filepath.Ext(f.Path))
		hasExtension := ext == ".js" || ext == ".ts" || ext == ".go" || ext == ".py"

		if !hasDirName && !hasExtension {
			continue
		}

		for _, h := range f.Hunks {
			// Detect route boundaries and track which lines belong to each route
			type routeInfo struct {
				startIdx             int
				endIdx               int
				content              string
				sensitiveActionFound bool
			}
			var routes []routeInfo

			// First pass: identify all route definitions
			routePattern := regexp.MustCompile(`(?i)(router|app|express)\.(post|put|patch|delete)\s*\(`)
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				if routePattern.MatchString(content) {
					routes = append(routes, routeInfo{
						startIdx:           i,
						endIdx:             -1, // Will be filled by closing brace
						content:            content,
						sensitiveActionFound: false,
					})

					// Check if this route handles sensitive actions
					contentLower := strings.ToLower(content)
					for _, action := range slp086SensitiveActions {
						pattern := `(?i)\b` + regexp.QuoteMeta(action) + `\b`
						if regexp.MustCompile(pattern).MatchString(contentLower) {
							routes[len(routes)-1].sensitiveActionFound = true
							break
						}
					}
				}
			}

			// Second pass: determine route boundaries using brace depth
			for i, route := range routes {
				depth := 0
				routeStarted := false
				for j := route.startIdx; j < len(h.Lines); j++ {
					ln := h.Lines[j]
					// Count braces in this line
					for _, ch := range ln.Content {
						if ch == '(' || ch == '{' {
							depth++
							routeStarted = true
						} else if ch == ')' || ch == '}' {
							depth--
						}
					}
					// Route ends when we've seen opening and depth returns to 0
					if routeStarted && depth == 0 {
						routes[i].endIdx = j
						break
					}
				}
				if routes[i].endIdx == -1 {
					routes[i].endIdx = len(h.Lines) - 1
				}
			}

			// Third pass: for each sensitive route without auth, report finding
			for _, route := range routes {
				if !route.sensitiveActionFound {
					continue
				}

				// Check for auth patterns only within this route's lines
				hasAuthInRoute := false
				for j := route.startIdx; j <= route.endIdx; j++ {
					ln := h.Lines[j]
					if ln.Kind == diff.LineAdd {
						for _, pattern := range slp086AuthPatterns {
							if pattern.MatchString(ln.Content) {
								hasAuthInRoute = true
								break
							}
						}
					}
					if hasAuthInRoute {
						break
					}
				}

				if !hasAuthInRoute {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     h.Lines[route.startIdx].NewLineNo,
						Message:  "sensitive action route may be missing authorization check - verify user permissions before processing",
						Snippet:  route.content,
					})
				}
			}
		}
	}
	return out
}
