package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP141 detects missing guards in async logic triggered by React effects.
// High-signal pattern: useEffect calling an async function without checking
// a loading/mounted state or using an AbortController.
type SLP141 struct{}

func (SLP141) ID() string                { return "SLP141" }
func (SLP141) DefaultSeverity() Severity { return SeverityWarn }
func (SLP141) Description() string {
	return "async useEffect without in-flight request guard or AbortController"
}

var (
	slp141UseEffect   = regexp.MustCompile(`\buseEffect\s*\(`)
	slp141AsyncCall   = regexp.MustCompile(`\b(?:await\s+)?(?:fetch|load|refresh|get|post|put|delete|update|sync|fetch\w+|load\w+)\s*\(`)
	slp141AsyncFunc   = regexp.MustCompile(`\basync\s+(?:function|\w+\s*=)`)
	slp141GuardCheck  = regexp.MustCompile(`\b(?:if\s*\(|&&\s*)(?:loading|isLoading|isFetching|active|mounted|ref\s*\.\s*current)\b`)
	slp141AbortSignal = regexp.MustCompile(`\b(?:signal|AbortController|abort)\b`)
)

func (r SLP141) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isJSOrTSFile(f.Path) {
			continue
		}

		added := f.AddedLines()
		for i, ln := range added {
			clean := stripCommentAndStrings(ln.Content)
			if !slp141UseEffect.MatchString(clean) {
				continue
			}

			// Collect the useEffect block
			block := collectHunkBlock(added, i, 15, true)
			if block == "" {
				continue
			}

			// If it contains an async call or async function definition
			if slp141AsyncCall.MatchString(block) || slp141AsyncFunc.MatchString(block) {
				// But lacks a guard check or AbortController
				if !slp141GuardCheck.MatchString(block) && !slp141AbortSignal.MatchString(block) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "missing in-flight request guard or AbortController in useEffect — may cause race conditions",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}

// collectHunkBlock concatenates up to 'limit' lines from 'added' starting at 'start',
// stopping if there's a gap in line numbers. If trackBraces is true, it also stops
// when the brace depth returns to zero (assuming it started at or above zero).
func collectHunkBlock(added []diff.Line, start int, limit int, trackBraces bool) string {
	var lines []string
	depth := 0
	started := false

	for i := start; i < len(added) && i < start+limit; i++ {
		if i > start && added[i].NewLineNo != added[i-1].NewLineNo+1 {
			break
		}
		content := added[i].Content
		stripped := stripCommentAndStrings(content)
		lines = append(lines, stripped)

		if trackBraces {
			// Track brace depth.
			open := strings.Count(stripped, "{")
			closeBraces := strings.Count(stripped, "}")

			if open > 0 {
				started = true
			}
			depth += open - closeBraces

			if started && depth <= 0 {
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}
