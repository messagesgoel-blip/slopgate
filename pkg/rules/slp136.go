package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP136 flags catch blocks that wrap a caught error in AppError without
// preserving the original cause. This is especially important when the code
// also logs or captures the original error for Sentry-style diagnostics.
type SLP136 struct{}

func (SLP136) ID() string                { return "SLP136" }
func (SLP136) DefaultSeverity() Severity { return SeverityWarn }
func (SLP136) Description() string {
	return "caught error wrapped in AppError without preserving the original cause"
}

var (
	slp136CatchHeader  = regexp.MustCompile(`\bcatch\s*\(\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*\)`)
	slp136NewAppError  = regexp.MustCompile(`\bnew\s+AppError\s*\(`)
	slp136CauseField   = regexp.MustCompile(`\bcause\s*:`)
	slp136ImmediateUse = regexp.MustCompile(`\b(?:error|next)\s*\(|\bthrow\b|\breturn\b`)
)

type slp136PendingFinding struct {
	line    int
	snippet string
	depth   int
}

func slp136MentionsCaughtError(line, errName string) bool {
	return wordInLine(line, errName)
}

func slp136PreservesCause(line, errName string) bool {
	if errName == "" {
		return false
	}
	quotedErrName := regexp.QuoteMeta(errName)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\bcause\s*:\s*` + quotedErrName + `\b`),
		regexp.MustCompile(`\b[A-Za-z_$][A-Za-z0-9_$]*\s*\.\s*cause\s*=\s*` + quotedErrName + `\b`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

func slp136ExpressionDepthDelta(line string) int {
	return strings.Count(line, "(") + strings.Count(line, "{") - strings.Count(line, ")") - strings.Count(line, "}")
}

func (r SLP136) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) || !isJSOrTSFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			inCatch := false
			catchDepth := 0
			errName := ""
			observedErrUse := false
			causePreserved := false
			var pending *slp136PendingFinding

			reset := func() {
				inCatch = false
				catchDepth = 0
				errName = ""
				observedErrUse = false
				causePreserved = false
				pending = nil
			}

			for _, ln := range h.Lines {
				if ln.Kind == diff.LineDelete {
					continue
				}
				content := ln.Content
				trimmed := strings.TrimSpace(content)

				if !inCatch {
					if m := slp136CatchHeader.FindStringSubmatch(trimmed); m != nil {
						inCatch = true
						errName = m[1]
						catchSub := content
						if idx := strings.Index(strings.ToLower(content), "catch"); idx >= 0 {
							catchSub = content[idx:]
						}
						catchDepth = strings.Count(catchSub, "{") - strings.Count(catchSub, "}")
						if catchDepth <= 0 {
							catchDepth = 1
						}
						observedErrUse = false
						causePreserved = false
						if slp136MentionsCaughtError(trimmed, errName) {
							observedErrUse = true
						}
					}
					continue
				}

				if errName != "" && slp136MentionsCaughtError(trimmed, errName) {
					observedErrUse = true
				}
				if errName != "" && slp136PreservesCause(trimmed, errName) {
					causePreserved = true
				}

				if pending != nil {
					pending.depth += slp136ExpressionDepthDelta(content)
					if pending.depth <= 0 {
						if !causePreserved {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     pending.line,
								Message:  "caught error is wrapped in AppError without preserving cause — attach the original error for diagnostics",
								Snippet:  pending.snippet,
							})
						}
						pending = nil
					}
				}

				if pending == nil && ln.Kind == diff.LineAdd && errName != "" && observedErrUse && !causePreserved &&
					slp136NewAppError.MatchString(trimmed) && slp136ImmediateUse.MatchString(trimmed) {
					depth := slp136ExpressionDepthDelta(content)
					if depth > 0 {
						pending = &slp136PendingFinding{
							line:    ln.NewLineNo,
							snippet: strings.TrimSpace(ln.Content),
							depth:   depth,
						}
					} else {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "caught error is wrapped in AppError without preserving cause — attach the original error for diagnostics",
							Snippet:  strings.TrimSpace(ln.Content),
						})
					}
				}

				catchDepth += strings.Count(content, "{")
				catchDepth -= strings.Count(content, "}")
				if catchDepth <= 0 {
					if pending != nil && !causePreserved {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     pending.line,
							Message:  "caught error is wrapped in AppError without preserving cause — attach the original error for diagnostics",
							Snippet:  pending.snippet,
						})
					}
					reset()
				}
			}
		}
	}
	return out
}
