package rules

import (
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
		regexp.MustCompile(`(?i)if\s*\(\s*!`), // Check for "if (!" pattern
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
		if !strings.Contains(fileLower, "api") &&
			!strings.Contains(fileLower, "route") &&
			!strings.Contains(fileLower, "controller") &&
			!strings.Contains(fileLower, "handler") &&
			!strings.Contains(fileLower, ".js") &&
			!strings.Contains(fileLower, ".ts") &&
			!strings.Contains(fileLower, ".go") &&
			!strings.Contains(fileLower, ".py") {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}

				content := strings.TrimSpace(ln.Content)

				// Check if this is a sensitive route definition
				routePattern := regexp.MustCompile(`(?i)(router|app|express)\.(post|put|patch|delete)\s*\(`)
				if routePattern.MatchString(content) {
					// Check if route handles sensitive actions
					sensitiveActionFound := false
					for _, action := range slp086SensitiveActions {
						if strings.Contains(strings.ToLower(content), action) {
							sensitiveActionFound = true
							break
						}
					}

					if sensitiveActionFound {
						// Check if auth pattern exists in the *entire* route body (hunk)
						hasAuthInRoute := false

						// Look for auth patterns in this hunk
						for _, checkLn := range h.Lines {
							if checkLn.Kind == diff.LineAdd {
								for _, pattern := range slp086AuthPatterns {
									if pattern.MatchString(checkLn.Content) {
										hasAuthInRoute = true
										break
									}
								}
							}
						}

						if !hasAuthInRoute {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     ln.NewLineNo,
								Message:  "sensitive action route may be missing authorization check - verify user permissions before processing",
								Snippet:  content,
							})
						}
					}
				}
			}
		}
	}
	return out
}
