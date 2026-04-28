package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP125 flags role/share/access mutations that lack nearby audit logging.
type SLP125 struct{}

func (SLP125) ID() string                { return "SLP125" }
func (SLP125) DefaultSeverity() Severity { return SeverityWarn }
func (SLP125) Description() string {
	return "share/role/access mutation without nearby audit logging call"
}

var slp125MutationTargetRe = regexp.MustCompile(`(?i)\b(grant|revoke|share|permission|role|member(?:ship)?|access)\b`)
var slp125WriteVerbRe = regexp.MustCompile(`(?i)\b(insert|update|delete|patch|post|put|set|create)\b`)
var slp125AuditRe = regexp.MustCompile(`(?i)(audit|activity|logEvent|appendActivity|recordAudit|trackEvent|emitAudit|writeAudit)`)

func slp125LikelyBackendFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "api") || strings.Contains(lower, "route") ||
		strings.Contains(lower, "handler") || strings.Contains(lower, "controller") ||
		strings.Contains(lower, "service") {
		return true
	}
	return isGoFile(path) || isJSOrTSFile(path) || isPythonFile(path)
}

func (r SLP125) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !slp125LikelyBackendFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(stripCommentAndStrings(ln.Content))
				if content == "" {
					continue
				}
				if !slp125MutationTargetRe.MatchString(content) || !slp125WriteVerbRe.MatchString(content) {
					continue
				}

				hasAudit := false
				start := i - 10
				if start < 0 {
					start = 0
				}
				end := i + 10
				if end >= len(h.Lines) {
					end = len(h.Lines) - 1
				}
				for j := start; j <= end; j++ {
					if h.Lines[j].Kind == diff.LineDelete {
						continue
					}
					windowLine := strings.TrimSpace(stripCommentAndStrings(h.Lines[j].Content))
					if windowLine == "" {
						continue
					}
					if slp125AuditRe.MatchString(windowLine) {
						hasAudit = true
						break
					}
				}
				if hasAudit {
					continue
				}

				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "role/share/access mutation without nearby audit/activity log write — add audit trail entry",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
