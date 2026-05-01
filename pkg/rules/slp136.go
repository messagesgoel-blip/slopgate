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
	slp136CatchHeader        = regexp.MustCompile(`\bcatch\s*\(\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*\)`)
	slp136NewAppError        = regexp.MustCompile(`\bnew\s+AppError\s*\(`)
	slp136ImmediateUse       = regexp.MustCompile(`\b(?:error|next)\s*\(|\bthrow\b|\breturn\b`)
	slp136VarAssign          = regexp.MustCompile(`\b(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*new\s+AppError\s*\(|\b([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*new\s+AppError\s*\(`)
	slp136VarAssignPrefix    = regexp.MustCompile(`\b(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*$|\b([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*$`)
	slp136CauseAssign        = regexp.MustCompile(`\b([A-Za-z_$][A-Za-z0-9_$]*)\s*\.\s*cause\s*=\s*([A-Za-z_$][A-Za-z0-9_$]*)\b`)
	slp136InlineSinkPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\bthrow\s+new\s+AppError\s*\(`),
		regexp.MustCompile(`\breturn\s+new\s+AppError\s*\(`),
		regexp.MustCompile(`\bnext\s*\(\s*new\s+AppError\s*\(`),
		regexp.MustCompile(`\berror\s*\([^)]*new\s+AppError\s*\(`),
	}
)

type slp136PendingFinding struct {
	line      int
	snippet   string
	depth     int
	preserved bool
	sawErrUse bool
	sawSink   bool
	variable  string
	directUse bool
}

func slp136MentionsCaughtError(line, errName string) bool {
	return wordInLine(line, errName)
}

func buildSlp136Patterns(errName string) []*regexp.Regexp {
	if errName == "" {
		return nil
	}
	quotedErrName := regexp.QuoteMeta(errName)
	return []*regexp.Regexp{
		regexp.MustCompile(`\bcause\s*:\s*` + quotedErrName + `\b`),
		regexp.MustCompile(`\b[A-Za-z_$][A-Za-z0-9_$]*\s*\.\s*cause\s*=\s*` + quotedErrName + `\b`),
	}
}

func slp136PreservesCause(line string, patterns []*regexp.Regexp) bool {
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
	if len(m) <= 1 {
		return ""
	}
	if m[1] != "" {
		return m[1]
	}
	if len(m) > 2 {
		return m[2]
	}
	return ""
}

func slp136AssignedWrapperVarPrefix(line string) string {
	m := slp136VarAssignPrefix.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	if len(m) > 1 && m[1] != "" {
		return m[1]
	}
	if len(m) > 2 && m[2] != "" {
		return m[2]
	}
	return ""
}

func slp136CatchBodyText(line string) string {
	idx := slp136CatchHeader.FindStringIndex(line)
	if idx == nil {
		return line
	}
	return line[idx[1]:]
}

func slp136MarkPreservedWrapper(line, errName string, wrappers map[string]*slp136PendingFinding) {
	m := slp136CauseAssign.FindStringSubmatch(line)
	if m == nil || len(m) < 3 || m[2] != errName {
		return
	}
	if wrapper := wrappers[m[1]]; wrapper != nil {
		wrapper.preserved = true
	}
}

func slp136InlineAppErrorSink(line string) bool {
	if !slp136ImmediateUse.MatchString(line) || !slp136NewAppError.MatchString(line) {
		return false
	}
	for _, pattern := range slp136InlineSinkPatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

func slp136SinkUsesVariable(line, variable string) bool {
	if variable == "" || !slp136ImmediateUse.MatchString(line) {
		return false
	}
	quotedVar := regexp.QuoteMeta(variable)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\bthrow\s+` + quotedVar + `\s*(?:[;),}]|$)`),
		regexp.MustCompile(`\breturn\s+` + quotedVar + `\s*(?:[;),}]|$)`),
		regexp.MustCompile(`\bnext\s*\(\s*` + quotedVar + `\s*(?:[),}]|$)`),
		regexp.MustCompile(`\berror\s*\([^)]*\b` + quotedVar + `\s*(?:[,);}]|$)`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

func slp136MaybeSinkWrappers(line string, wrappers map[string]*slp136PendingFinding) {
	if !slp136ImmediateUse.MatchString(line) {
		return
	}
	for variable, wrapper := range wrappers {
		if !slp136SinkUsesVariable(line, variable) {
			continue
		}
		wrapper.sawSink = true
	}
}

func slp136AppendFinding(out *[]Finding, rule SLP136, filePath string, wrapper *slp136PendingFinding) {
	*out = append(*out, Finding{
		RuleID:   rule.ID(),
		Severity: rule.DefaultSeverity(),
		File:     filePath,
		Line:     wrapper.line,
		Message:  "caught error is wrapped in AppError without preserving cause — attach the original error for diagnostics",
		Snippet:  wrapper.snippet,
	})
}

func slp136FlushObservedWrappers(out *[]Finding, rule SLP136, filePath string, observedWrappers []*slp136PendingFinding) {
	for _, wrapper := range observedWrappers {
		if wrapper == nil || !wrapper.sawSink || !wrapper.sawErrUse || wrapper.preserved {
			continue
		}
		slp136AppendFinding(out, rule, filePath, wrapper)
	}
}

func slp136FinalizePending(wrapper *slp136PendingFinding, wrappers map[string]*slp136PendingFinding, observedWrappers *[]*slp136PendingFinding) {
	if wrapper == nil {
		return
	}
	if wrapper.directUse {
		wrapper.sawSink = true
	}
	if wrapper.variable != "" {
		wrappers[wrapper.variable] = wrapper
	}
	*observedWrappers = append(*observedWrappers, wrapper)
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
			var causePatterns []*regexp.Regexp
			observedErrUse := false
			var pending *slp136PendingFinding
			wrappers := map[string]*slp136PendingFinding{}
			observedWrappers := []*slp136PendingFinding{}
			lastAssignedVar := ""

			for _, ln := range h.Lines {
				if ln.Kind == diff.LineDelete {
					continue
				}
				content := ln.Content
				trimmed := strings.TrimSpace(content)
				skipCatchDepthUpdate := false

				if !inCatch {
					if m := slp136CatchHeader.FindStringSubmatch(trimmed); m != nil {
						inCatch = true
						errName = m[1]
						causePatterns = buildSlp136Patterns(errName)
						catchSub := content
						if idx := strings.Index(strings.ToLower(content), "catch"); idx >= 0 {
							catchSub = content[idx:]
						}
						catchDepth = strings.Count(catchSub, "{") - strings.Count(catchSub, "}")
						bodyText := slp136CatchBodyText(catchSub)
						content = bodyText
						trimmed = strings.TrimSpace(bodyText)
						observedErrUse = false
						if slp136MentionsCaughtError(trimmed, errName) {
							observedErrUse = true
						}
						if trimmed == "" {
							if catchDepth <= 0 {
								inCatch = false
								catchDepth = 0
								errName = ""
								causePatterns = nil
								observedErrUse = false
								pending = nil
								clear(wrappers)
								observedWrappers = nil
								lastAssignedVar = ""
							}
							continue
						}
						skipCatchDepthUpdate = true
					} else {
						continue
					}
				}

				mentionedErr := errName != "" && slp136MentionsCaughtError(trimmed, errName)
				if mentionedErr {
					observedErrUse = true
					if pending != nil {
						pending.sawErrUse = true
					}
					for _, wrapper := range observedWrappers {
						wrapper.sawErrUse = true
					}
				}

				if pending != nil {
					if slp136PreservesCause(trimmed, causePatterns) {
						pending.preserved = true
					}
					pending.depth += slp136ExpressionDepthDelta(content)
					if pending.depth <= 0 {
						slp136FinalizePending(pending, wrappers, &observedWrappers)
						pending = nil
					}
				}

				if pending == nil && ln.Kind == diff.LineAdd && errName != "" && slp136NewAppError.MatchString(trimmed) {
					variable := slp136AssignedWrapperVar(trimmed)
					if variable == "" {
						variable = lastAssignedVar
					}
					directUse := slp136InlineAppErrorSink(trimmed) && variable == ""
					if !directUse && variable == "" {
						goto catchDepthUpdate
					}
					depth := slp136ExpressionDepthDelta(content)
					pending = &slp136PendingFinding{
						line:      ln.NewLineNo,
						snippet:   strings.TrimSpace(ln.Content),
						depth:     depth,
						preserved: slp136PreservesCause(trimmed, causePatterns),
						sawErrUse: observedErrUse,
						variable:  variable,
						directUse: directUse,
					}
					if pending.depth <= 0 {
						slp136FinalizePending(pending, wrappers, &observedWrappers)
						pending = nil
					}
				}

				if errName != "" {
					slp136MarkPreservedWrapper(trimmed, errName, wrappers)
				}
				slp136MaybeSinkWrappers(trimmed, wrappers)
				if slp136NewAppError.MatchString(trimmed) {
					lastAssignedVar = ""
				} else {
					lastAssignedVar = slp136AssignedWrapperVarPrefix(trimmed)
				}

			catchDepthUpdate:
				if !skipCatchDepthUpdate {
					catchDepth += strings.Count(content, "{")
					catchDepth -= strings.Count(content, "}")
				}
				if catchDepth <= 0 {
					if pending != nil {
						slp136FinalizePending(pending, wrappers, &observedWrappers)
					}
					slp136FlushObservedWrappers(&out, r, f.Path, observedWrappers)
					inCatch = false
					catchDepth = 0
					errName = ""
					causePatterns = nil
					observedErrUse = false
					pending = nil
					clear(wrappers)
					observedWrappers = nil
					lastAssignedVar = ""
				}
			}
		}
	}
	return out
}
