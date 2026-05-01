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
	if len(m) > 1 {
		if m[1] != "" {
			return m[1]
		}
		if len(m) > 2 {
			return m[2]
		}
	}
	return ""
}

func slp136AssignedWrapperVarPrefix(line string) string {
	m := slp136VarAssignPrefix.FindStringSubmatch(line)
	if len(m) > 1 {
		if m[1] != "" {
			return m[1]
		}
		if len(m) > 2 && m[2] != "" {
			return m[2]
		}
	}
	return ""
}

func slp136CatchBodyText(line string) string {
	idx := slp136CatchHeader.FindStringIndex(line)
	if len(idx) >= 2 {
		return line[idx[1]:]
	}
	return line
}

func slp136MarkPreservedWrapper(line, errName string, wrappers map[string]*slp136PendingFinding) {
	m := slp136CauseAssign.FindStringSubmatch(line)
	if len(m) >= 3 {
		if m[2] != errName {
			return
		}
		if wrapper := wrappers[m[1]]; wrapper != nil {
			wrapper.preserved = true
		}
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

func slp136AggregateSinkText(lines []diff.Line, start int, fallback string) string {
	if start < 0 || start >= len(lines) {
		return fallback
	}

	var b strings.Builder
	depth := 0
	sawOpen := false
	for i := start; i < len(lines); i++ {
		ln := lines[i]
		if ln.Kind == diff.LineDelete {
			continue
		}
		part := strings.TrimSpace(ln.Content)
		if part == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(part)

		if strings.Contains(part, "(") {
			sawOpen = true
		}
		depth += strings.Count(part, "(")
		depth -= strings.Count(part, ")")
		if sawOpen && depth <= 0 {
			break
		}
		if !sawOpen && (strings.Contains(part, ";") || strings.Contains(part, "}")) {
			break
		}
		if !sawOpen && (strings.Contains(part, "throw") || strings.Contains(part, "return")) {
			break
		}
	}
	if b.Len() == 0 {
		return fallback
	}
	return b.String()
}

func slp136SinkText(lines []diff.Line, start int, fallback string) string {
	if !slp136ImmediateUse.MatchString(fallback) {
		return fallback
	}
	return slp136AggregateSinkText(lines, start, fallback)
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

func slp136ObserveContextWrappers(
	lines []diff.Line,
	errName string,
	observedErrUse bool,
	causePatterns []*regexp.Regexp,
	wrappers map[string]*slp136PendingFinding,
	observedWrappers *[]*slp136PendingFinding,
) {
	if errName == "" {
		return
	}
	lastAssignedVar := ""
	for _, ln := range lines {
		if ln.Kind != diff.LineContext {
			continue
		}
		trimmed := strings.TrimSpace(ln.Content)
		if trimmed == "" {
			continue
		}
		if slp136NewAppError.MatchString(trimmed) {
			variable := slp136AssignedWrapperVar(trimmed)
			if variable == "" {
				variable = lastAssignedVar
			}
			if variable != "" && wrappers[variable] == nil {
				wrapper := &slp136PendingFinding{
					line:      ln.NewLineNo,
					snippet:   trimmed,
					preserved: slp136PreservesCause(trimmed, causePatterns),
					sawErrUse: observedErrUse,
					variable:  variable,
				}
				slp136FinalizePending(wrapper, wrappers, observedWrappers)
			}
			lastAssignedVar = ""
		} else {
			lastAssignedVar = slp136AssignedWrapperVarPrefix(trimmed)
		}
		slp136MarkPreservedWrapper(trimmed, errName, wrappers)
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
		if wrapper == nil || !wrapper.sawSink || wrapper.preserved {
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
			catchLines := []diff.Line{}

			for lineIndex, ln := range h.Lines {
				if ln.Kind == diff.LineDelete {
					continue
				}
				content := ln.Content
				trimmed := strings.TrimSpace(content)
				skipCatchDepthUpdate := false

				if !inCatch {
					if m := slp136CatchHeader.FindStringSubmatch(trimmed); len(m) >= 2 {
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
								catchLines = nil
							}
							continue
						}
						skipCatchDepthUpdate = true
					} else {
						continue
					}
				}

				catchLines = append(catchLines, diff.Line{
					Kind:      ln.Kind,
					Content:   content,
					NewLineNo: ln.NewLineNo,
					OldLineNo: ln.OldLineNo,
				})
				sinkText := slp136SinkText(h.Lines, lineIndex, trimmed)

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

				if pending == nil && ln.Kind == diff.LineAdd && errName != "" && !slp136NewAppError.MatchString(trimmed) && slp136InlineAppErrorSink(sinkText) {
					pending = &slp136PendingFinding{
						line:      ln.NewLineNo,
						snippet:   strings.TrimSpace(sinkText),
						preserved: slp136PreservesCause(sinkText, causePatterns),
						sawErrUse: observedErrUse,
						directUse: true,
					}
					slp136FinalizePending(pending, wrappers, &observedWrappers)
					pending = nil
				}

				if pending == nil && ln.Kind == diff.LineAdd && errName != "" && slp136NewAppError.MatchString(trimmed) {
					variable := slp136AssignedWrapperVar(trimmed)
					if variable == "" {
						variable = lastAssignedVar
					}
					directUse := slp136InlineAppErrorSink(sinkText) && variable == ""
					if !directUse && variable == "" {
						goto catchDepthUpdate
					}
					depth := slp136ExpressionDepthDelta(content)
					pending = &slp136PendingFinding{
						line:      ln.NewLineNo,
						snippet:   strings.TrimSpace(ln.Content),
						depth:     depth,
						preserved: slp136PreservesCause(sinkText, causePatterns),
						sawErrUse: observedErrUse,
						variable:  variable,
						directUse: directUse,
					}
					if pending.depth <= 0 {
						slp136FinalizePending(pending, wrappers, &observedWrappers)
						pending = nil
					}
				}

				if ln.Kind == diff.LineAdd && slp136ImmediateUse.MatchString(sinkText) {
					slp136ObserveContextWrappers(catchLines, errName, observedErrUse, causePatterns, wrappers, &observedWrappers)
				}
				if errName != "" {
					slp136MarkPreservedWrapper(trimmed, errName, wrappers)
				}
				slp136MaybeSinkWrappers(sinkText, wrappers)
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
					catchLines = nil
				}
			}
		}
	}
	return out
}
