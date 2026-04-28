package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP122 flags async polling/retry patterns without nearby cancellation or
// in-flight guard logic. This catches common UI/task-loop race patterns.
type SLP122 struct{}

func (SLP122) ID() string                { return "SLP122" }
func (SLP122) DefaultSeverity() Severity { return SeverityWarn }
func (SLP122) Description() string {
	return "async polling/retry logic added without cancellation or in-flight guard"
}

var slp122AsyncTriggerRe = regexp.MustCompile(`(?i)(setInterval\s*\(|setTimeout\s*\(|await\s+new\s+Promise\s*\(|\bpoll\w*\s*\(|\bretry\w*\s*\()`)
var slp122AsyncWorkRe = regexp.MustCompile(`(?i)(fetch\s*\(|axios\.\w+\s*\(|api\.\w+\s*\(|enqueue|queue|scan|sync|await\s+)`)
var slp122GuardRe = regexp.MustCompile(`(?i)(clearInterval|clearTimeout|AbortController|controller\.abort|signal|isMounted|cancelled|canceled|inFlight|mutex|lock|return\s*\(\s*\)\s*=>)`)

func (r SLP122) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
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
				if !slp122AsyncTriggerRe.MatchString(content) {
					continue
				}
				if !slp122AsyncWorkRe.MatchString(content) {
					continue
				}

				hasGuard := false
				start := i - 8
				if start < 0 {
					start = 0
				}
				end := i + 8
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
					if slp122GuardRe.MatchString(windowLine) {
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
					Message:  "async poll/retry path has no nearby cancel/in-flight guard — add abort or lock to avoid overlap races",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
