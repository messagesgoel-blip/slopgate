package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP087 flags webhook handlers that don't have timeout configurations.
// This can cause hanging requests and resource exhaustion.
type SLP087 struct{}

func (SLP087) ID() string                { return "SLP087" }
func (SLP087) DefaultSeverity() Severity { return SeverityWarn }
func (SLP087) Description() string {
	return "webhook handler may be missing timeout configuration - set appropriate timeout to prevent hanging requests"
}

var (
	// Webhook-related file patterns
	slp087WebhookFilePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)webhook`),

		// Payment gateway webhooks
		regexp.MustCompile(`(?i)stripe|stripe\.`),
		regexp.MustCompile(`(?i)hubspot`),

		// GitHub/GitLab webhooks
		regexp.MustCompile(`(?i)github`),
		regexp.MustCompile(`(?i)gitlab`),
		regexp.MustCompile(`(?i)slack`),

		// Event patterns in file path
		regexp.MustCompile(`(?i)handler.*event|event.*handler`),
	}

	// Timeout-related patterns (what we're looking for)
	slp087TimeoutPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)timeout(?:Ms|ms|millis|OUT)?\s*[:=]?\s*\d+`),
		regexp.MustCompile(`(?i)signal\.abort|AbortController`),
		regexp.MustCompile(`(?i)context\.With(?:Timeout|Deadline|Cancel)`),
		regexp.MustCompile(`(?i)http\.Client\s*\{[^}]*Timeout`),
		regexp.MustCompile(`(?i)timeout\s*=\s*\d+`),
		regexp.MustCompile(`(?i)setTimeout|setImmediate`),
		regexp.MustCompile(`(?i)abortSignal|signal\s*=`),
		regexp.MustCompile(`(?i)controller\s*=\s*new\s+(AbortController|TimeoutController)`),
	}
)

// Check scans webhook files for missing timeout configurations.
func (r SLP087) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		fileLower := strings.ToLower(f.Path)

		// Check if this is a webhook-related file
		isWebhookFile := false
		for _, pattern := range slp087WebhookFilePatterns {
			if pattern.MatchString(fileLower) {
				isWebhookFile = true
				break
			}
		}

		// Also check for files that look like webhook handlers
		if strings.Contains(fileLower, "webhook") ||
			strings.Contains(fileLower, "stripe") ||
			strings.Contains(fileLower, "github") ||
			strings.Contains(fileLower, "gitlab") ||
			strings.Contains(fileLower, "slack") ||
			strings.Contains(fileLower, "handler") ||
			strings.Contains(fileLower, "callback") ||
			strings.Contains(fileLower, "endpoint") ||
			strings.Contains(fileLower, "event") {
			isWebhookFile = true
		}

		if !isWebhookFile {
			continue
		}

		// Check if timeout is configured in the file
		hasTimeout := false
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				for _, pattern := range slp087TimeoutPatterns {
					if pattern.MatchString(ln.Content) {
						hasTimeout = true
						break
					}
				}
				if hasTimeout {
					break
				}
			}
		}

		// Check for webhook-related code patterns - look for webhook handling
		// Pattern 1: webhook handlers with event payloads
		// Pattern 2: function parameters that suggest webhook-like handling
		webhookCodePatterns := regexp.MustCompile(`(?i)webhook.*(?:payload|event|delivery|signature)|event.*(?:data|payload|body)|req\.(body|params|query)|res\.(status|json|end)\s*\(`)

		hasWebhookCode := false
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if webhookCodePatterns.MatchString(ln.Content) {
					hasWebhookCode = true
					break
				}
			}
		}

		if hasWebhookCode && !hasTimeout {
			for _, h := range f.Hunks {
				for _, ln := range h.Lines {
					if ln.Kind != diff.LineAdd {
						continue
					}
					if webhookCodePatterns.MatchString(ln.Content) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "webhook handler may be missing timeout configuration - add timeout, abort controller, or context.WithTimeout",
							Snippet:  strings.TrimSpace(ln.Content),
						})
						break
					}
				}
			}
		}
	}
	return out
}
