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
	slp136ImmediateUse = regexp.MustCompile(`\b(?:error|next)\s*\(|\bthrow\b|\breturn\b`)
	slp136VarAssign    = regexp.MustCompile(`\b(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*new\s+AppError\s*\(|\b([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*new\s+AppError\s*\(`)
	slp136CauseAssign  = regexp.MustCompile(`\b([A-Za-z_$][A-Za-z0-9_$]*)\s*\.\s*cause\s*=\s*([A-Za-z_$][A-Za-z0-9_$]*)\b`)
)

type slp136PendingFinding struct {
	line      int
	snippet   string
	depth     int
	preserved bool
	variable  string
	directUse bool
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

func slp136AssignedWrapperVar(line string) string {
	m := slp136VarAssign.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	if m[1] != "" {
		return m[1]
	}
	return m[2]
}

func slp136MarkPreservedWrapper(line, errName string, wrappers map[string]*slp136PendingFinding) {
	m := slp136CauseAssign.FindStringSubmatch(line)
	if m == nil || m[2] != errName {
		return
	}
	if wrapper := wrappers[m[1]]; wrapper != nil {
		wrapper.preserved = true
	}
}

func slp136MaybeSinkWrappers(line string, wrappers map[string]*slp136PendingFinding, out *[]Finding, rule SLP136, filePath string) {
	if !slp136ImmediateUse.MatchString(line) {
		return
	}
	for variable, wrapper := range wrappers {
		if !wordInLine(line, variable) {
			continue
		}
		if !wrapper.preserved {
			*out = append(*out, Finding{
				RuleID:   rule.ID(),
				Severity: rule.DefaultSeverity(),
				File:     filePath,
				Line:     wrapper.line,
				Message:  "caught error is wrapped in AppError without preserving cause — attach the original error for diagnostics",
				Snippet:  wrapper.snippet,
			})
		}
		delete(wrappers, variable)
	}
}

func slp136FinalizePending(out *[]Finding, rule SLP136, filePath string, wrapper *slp136PendingFinding, wrappers map[string]*slp136PendingFinding) {
	if wrapper == nil {
		return
	}
	if wrapper.directUse {
		if !wrapper.preserved {
			*out = append(*out, Finding{
				RuleID:   rule.ID(),
				Severity: rule.DefaultSeverity(),
				File:     filePath,
				Line:     wrapper.line,
				Message:  "caught error is wrapped in AppError without preserving cause — attach the original error for diagnostics",
				Snippet:  wrapper.snippet,
			})
		}
		return
	}
	if wrapper.variable != "" {
		wrappers[wrapper.variable] = wrapper
	}
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
			var pending *slp136PendingFinding
			wrappers := map[string]*slp136PendingFinding{}

			reset := func() {
				inCatch = false
				catchDepth = 0
				errName = ""
				observedErrUse = false
				pending = nil
				clear(wrappers)
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
						if slp136MentionsCaughtError(trimmed, errName) {
							observedErrUse = true
						}
					}
					continue
				}

				if errName != "" && slp136MentionsCaughtError(trimmed, errName) {
					observedErrUse = true
				}

				if pending != nil {
					if errName != "" && slp136PreservesCause(trimmed, errName) {
						pending.preserved = true
					}
					pending.depth += slp136ExpressionDepthDelta(content)
					if pending.depth <= 0 {
						slp136FinalizePending(&out, r, f.Path, pending, wrappers)
						pending = nil
					}
				}

				if pending == nil && ln.Kind == diff.LineAdd && errName != "" && observedErrUse &&
					slp136NewAppError.MatchString(trimmed) {
					variable := slp136AssignedWrapperVar(trimmed)
					directUse := slp136ImmediateUse.MatchString(trimmed) && variable == ""
					if !directUse && variable == "" {
						goto catchDepthUpdate
					}
					depth := slp136ExpressionDepthDelta(content)
					pending = &slp136PendingFinding{
						line:      ln.NewLineNo,
						snippet:   strings.TrimSpace(ln.Content),
						depth:     depth,
						preserved: slp136PreservesCause(trimmed, errName),
						variable:  variable,
						directUse: directUse,
					}
					if pending.depth <= 0 {
						slp136FinalizePending(&out, r, f.Path, pending, wrappers)
						pending = nil
					}
				}
				if errName != "" {
					slp136MarkPreservedWrapper(trimmed, errName, wrappers)
				}
				slp136MaybeSinkWrappers(trimmed, wrappers, &out, r, f.Path)

			catchDepthUpdate:
				catchDepth += strings.Count(content, "{")
				catchDepth -= strings.Count(content, "}")
				if catchDepth <= 0 {
					if pending != nil {
						slp136FinalizePending(&out, r, f.Path, pending, wrappers)
					}
					reset()
				}
			}
		}
	}
	return out
}
