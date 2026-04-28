package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP121 flags sensitive access/share/role mutations that appear to be missing
// explicit tenant/membership/authorization guard checks in nearby code.
type SLP121 struct{}

func (SLP121) ID() string                { return "SLP121" }
func (SLP121) DefaultSeverity() Severity { return SeverityWarn }
func (SLP121) Description() string {
	return "sensitive access mutation may be missing tenant/membership authorization guard"
}

var slp121SensitiveMutationRe = regexp.MustCompile(`(?i)(grant|revoke|share|permission|role|member|tenant|access)`)
var slp121MutationVerbRe = regexp.MustCompile(`(?i)\b(insert|update|delete|upsert|patch|post|put|remove|set)\b`)
var slp121RouteMutationRe = regexp.MustCompile(`(?i)\b(?:router|app)\.(?:post|put|patch|delete)\s*\(`)
var slp121GuardRe = regexp.MustCompile(`(?i)(require(?:Auth|Role|Permission|Tenant|[A-Za-z]*Access)|check(?:Permission|Role|Tenant|Membership)|has(?:Permission|Role|Tenant|Access)|is(?:Admin|Owner|Member)|authorize|authz|tenant.*member|member.*tenant)`)

func slp121LikelyAccessFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "route") || strings.Contains(lower, "handler") ||
		strings.Contains(lower, "controller") || strings.Contains(lower, "api") ||
		strings.Contains(lower, "service") {
		return true
	}
	return isGoFile(path) || isJSOrTSFile(path) || isPythonFile(path)
}

func (r SLP121) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !slp121LikelyAccessFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(slp115StripCommentsPreservingStrings(ln.Content))
				if content == "" {
					continue
				}
				if !slp121SensitiveMutationRe.MatchString(content) {
					continue
				}
				if !slp121MutationVerbRe.MatchString(content) && !slp121RouteMutationRe.MatchString(content) {
					continue
				}

				hasGuard := false
				start := i - 8
				if start < 0 {
					start = 0
				}
				end := i + 12
				if end >= len(h.Lines) {
					end = len(h.Lines) - 1
				}
				for j := start; j <= end; j++ {
					if h.Lines[j].Kind == diff.LineDelete {
						continue
					}
					windowLine := strings.TrimSpace(slp115StripCommentsPreservingStrings(h.Lines[j].Content))
					if windowLine == "" {
						continue
					}
					if slp121GuardRe.MatchString(windowLine) {
						hasGuard = true
						break
					}
				}
				if hasGuard {
					continue
				}

				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "access/share/role mutation added without nearby tenant or membership guard — verify authorization before write",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
