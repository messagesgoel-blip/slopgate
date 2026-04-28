package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP124 flags external API/client calls that consume request/input payloads
// without nearby validation checks.
type SLP124 struct{}

func (SLP124) ID() string                { return "SLP124" }
func (SLP124) DefaultSeverity() Severity { return SeverityWarn }
func (SLP124) Description() string {
	return "external call uses request/input payload without nearby validation guard"
}

var slp124ExternalCallRe = regexp.MustCompile(`(?i)(fetch\s*\(|axios\.\w+\s*\(|http\.(?:Post|Do|NewRequest)\s*\(|client\.Do\s*\(|litellm|chat\.completions\.create|openai\.\w+\s*\()`)
var slp124InputPayloadRe = regexp.MustCompile(`(?i)(req\.(?:body|query|params)|\binput\b|\bpayload\b|\bprompt\b|\bmessages?\b|\bbody\b|\bquery\b|\bparams\b)`)
var slp124ValidationRe = regexp.MustCompile(`(?i)(validate|schema\.parse|zod|joi|if\s*\(\s*!|if\s*\(\s*len\(|if\s*\(\s*[^)]*==\s*["']\s*["']|trim\(\)|required|ensure|guard)`)

func (r SLP124) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) {
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
				if !slp124ExternalCallRe.MatchString(content) || !slp124InputPayloadRe.MatchString(content) {
					continue
				}
				if slp124ValidationRe.MatchString(content) {
					continue
				}

				hasValidation := false
				start := i - 6
				if start < 0 {
					start = 0
				}
				for j := start; j <= i; j++ {
					if h.Lines[j].Kind == diff.LineDelete {
						continue
					}
					windowLine := strings.TrimSpace(stripCommentAndStrings(h.Lines[j].Content))
					if windowLine == "" {
						continue
					}
					if slp124ValidationRe.MatchString(windowLine) {
						hasValidation = true
						break
					}
				}
				if hasValidation {
					continue
				}

				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "request/input payload passed to external call without nearby validation — validate before outbound request",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
