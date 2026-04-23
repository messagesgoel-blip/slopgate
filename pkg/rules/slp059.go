package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP059 flags unsanitized exec.Command usage in Go files.
type SLP059 struct{}

func (SLP059) ID() string                { return "SLP059" }
func (SLP059) DefaultSeverity() Severity { return SeverityBlock }
func (SLP059) Description() string {
	return "unsanitized os/exec command with user input"
}

var (
	goIdentPattern          = regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)
	execCommandRe           = regexp.MustCompile(`\bexec\.Command\s*\(`)
	slp059ConstStringDeclRe = regexp.MustCompile("^\\s*const\\s+([A-Za-z_][A-Za-z0-9_]*)(?:\\s+[A-Za-z_][A-Za-z0-9_]*)?\\s*=\\s*(?:\"(?:\\\\.|[^\"\\\\])*\"|`[^`]*`)\\s*$")
	slp059VarStringDeclRe   = regexp.MustCompile("^\\s*var\\s+([A-Za-z_][A-Za-z0-9_]*)(?:\\s+[A-Za-z_][A-Za-z0-9_]*)?\\s*=\\s*(?:\"(?:\\\\.|[^\"\\\\])*\"|`[^`]*`)\\s*$")
	slp059ShortStringDeclRe = regexp.MustCompile("^\\s*([A-Za-z_][A-Za-z0-9_]*)\\s*:=\\s*(?:\"(?:\\\\.|[^\"\\\\])*\"|`[^`]*`)\\s*$")
	slp059FunctionStartLine = regexp.MustCompile(`^\s*func\b`)
)

func slp059FileLines(f diff.File) []diff.Line {
	var out []diff.Line
	for _, h := range f.Hunks {
		for _, ln := range h.Lines {
			if ln.NewLineNo > 0 {
				out = append(out, ln)
			}
		}
	}
	return out
}

func slp059LineIndex(lines []diff.Line, lineNo int) int {
	for i, ln := range lines {
		if ln.NewLineNo == lineNo {
			return i
		}
	}
	return -1
}

func slp059LiteralStringName(line string) (string, bool) {
	for _, re := range []*regexp.Regexp{
		slp059ConstStringDeclRe,
		slp059VarStringDeclRe,
		slp059ShortStringDeclRe,
	} {
		if m := re.FindStringSubmatch(line); m != nil {
			return m[1], true
		}
	}
	return "", false
}

func slp059TopLevelStringConsts(lines []diff.Line, upto int) map[string]struct{} {
	safe := make(map[string]struct{})
	depth := 0
	for i := 0; i < upto; i++ {
		clean := stripCommentAndStrings(lines[i].Content)
		if depth == 0 {
			if name, ok := slp059LiteralStringName(clean); ok && strings.HasPrefix(strings.TrimSpace(clean), "const ") {
				safe[name] = struct{}{}
			}
		}
		depth += strings.Count(clean, "{") - strings.Count(clean, "}")
		if depth < 0 {
			depth = 0
		}
	}
	return safe
}

func slp059LocalSafeStrings(lines []diff.Line, upto int) map[string]struct{} {
	safe := make(map[string]struct{})
	reverseDepth := 0
	for i := upto - 1; i >= 0; i-- {
		clean := stripCommentAndStrings(lines[i].Content)
		reverseDepth += strings.Count(clean, "}")
		if reverseDepth == 0 {
			if name, ok := slp059LiteralStringName(clean); ok {
				safe[name] = struct{}{}
			}
		}
		reverseDepth -= strings.Count(clean, "{")
		if reverseDepth < 0 {
			reverseDepth = 0
		}
		if reverseDepth == 0 && slp059FunctionStartLine.MatchString(strings.TrimSpace(clean)) {
			break
		}
	}
	return safe
}

func slp059SafeStrings(f diff.File, lineNo int) map[string]struct{} {
	lines := slp059FileLines(f)
	idx := slp059LineIndex(lines, lineNo)
	if idx < 0 {
		return nil
	}
	safe := slp059TopLevelStringConsts(lines, idx)
	for name := range slp059LocalSafeStrings(lines, idx) {
		safe[name] = struct{}{}
	}
	return safe
}

func slp059HasUnsafeIdentifier(args string, safe map[string]struct{}) bool {
	for _, ident := range goIdentPattern.FindAllString(args, -1) {
		if safe != nil {
			if _, ok := safe[ident]; ok {
				continue
			}
		}
		return true
	}
	return false
}

func slp059CollectExecCall(added []diff.Line, start int) string {
	var parts []string
	depth := 0
	foundCall := false
	for i := start; i < len(added); i++ {
		if i > start && added[i].NewLineNo != added[i-1].NewLineNo+1 {
			break
		}
		clean := stripCommentAndStrings(added[i].Content)
		if !foundCall {
			m := execCommandRe.FindStringIndex(clean)
			if m == nil {
				break
			}
			clean = clean[m[0]:]
			foundCall = true
		}
		parts = append(parts, clean)
		depth += strings.Count(clean, "(") - strings.Count(clean, ")")
		if foundCall && depth <= 0 {
			break
		}
	}
	return strings.Join(parts, "\n")
}

func (r SLP059) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		added := f.AddedLines()
		for i, ln := range added {
			cleanLine := stripCommentAndStrings(ln.Content)
			if execCommandRe.FindStringIndex(cleanLine) == nil {
				continue
			}
			call := slp059CollectExecCall(added, i)
			if call == "" {
				continue
			}
			callMatch := execCommandRe.FindStringIndex(call)
			if callMatch == nil {
				continue
			}
			args := call[callMatch[1]:]
			unquoted := args
			// Any interpolation or concatenation is an immediate red flag.
			if strings.Contains(unquoted, "$") || strings.Contains(unquoted, "+") || strings.Contains(unquoted, "fmt.Sprintf") {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "exec.Command argument may contain user input — sanitize before executing",
					Snippet:  strings.TrimSpace(ln.Content),
				})
				continue
			}
			if slp059HasUnsafeIdentifier(unquoted, slp059SafeStrings(f, ln.NewLineNo)) {
				// Note: this is still a best-effort heuristic. We skip local
				// literal strings and compile-time consts that are visible in the
				// current diff hunk, but identifiers resolved outside that scope
				// are conservatively treated as potentially unsafe.
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "exec.Command argument may contain user input — sanitize before executing",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
