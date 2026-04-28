package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

type SLP114 struct{}

func (SLP114) ID() string                { return "SLP114" }
func (SLP114) DefaultSeverity() Severity { return SeverityWarn }
func (SLP114) Description() string {
	return "error-returning function called as statement — check the error return"
}

// slp114ErrGuardRe matches common Go error-check if forms.
// Known limitations: does not match named error identifiers other than "err"
// (e.g., dbErr) and does not cover positive "err == nil" branches.
var slp114ErrGuardRe = regexp.MustCompile(`^if\s+err\s*:=|^if\s+[^;]+;\s*err\s*!=`)

func (r SLP114) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || !isGoFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				if content == "" {
					continue
				}

				stripped := stripCommentAndStrings(content)

				if strings.HasPrefix(stripped, "return ") {
					continue
				}

				if strings.HasPrefix(stripped, "_ = ") {
					continue
				}

				var body string
				if strings.HasPrefix(stripped, "if ") {
					braceIdx := indexOutsideQuotes(stripped, "{")
					if braceIdx >= 0 && braceIdx < len(stripped) {
						initSegment := stripped[3:braceIdx]
						semiIdx := indexOutsideQuotes(initSegment, ";")
						if semiIdx >= 0 {
							initPart := strings.TrimSpace(initSegment[:semiIdx])
							if !slp114ErrGuardRe.MatchString("if " + initPart) {
								out = append(out, r.slp114ScanCalls(initPart, f.Path, ln.NewLineNo, content)...)
							}
						}
						body = stripped[braceIdx+1:]
						closeIdx := strings.LastIndex(body, "}")
						if closeIdx > 0 {
							body = body[:closeIdx]
						}
					} else if slp114ErrGuardRe.MatchString(stripped) {
						continue
					} else {
						body = stripped
					}
				} else {
					body = stripped
				}

				out = append(out, r.slp114ScanCalls(body, f.Path, ln.NewLineNo, content)...)
			}
		}
	}
	return out
}

func (r SLP114) slp114ScanCalls(body string, filePath string, lineNo int, raw string) []Finding {
	var out []Finding
	search := body
	for {
		idx := indexOutsideQuotes(search, "(")
		if idx <= 0 {
			break
		}
		funcCall := search[:idx]
		fullCallee := strings.TrimSpace(funcCall)
		lastIdent := fullCallee
		if dot := strings.LastIndexByte(funcCall, '.'); dot >= 0 {
			lastIdent = strings.TrimSpace(funcCall[dot+1:])
		}

		afterParen := search[idx:]
		if isErrorReturningFunc(fullCallee) || isErrorReturningFunc(lastIdent) {
			if strings.Contains(afterParen, ")") {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     filePath,
					Line:     lineNo,
					Message:  "error-returning function '" + fullCallee + "' called as statement — check for missing error check",
					Snippet:  raw,
				})
			}
		}

		closeIdx := indexOutsideQuotes(afterParen, ")")
		if closeIdx < 0 {
			break
		}
		search = afterParen[closeIdx+1:]
		if search == "" {
			break
		}
	}
	return out
}

var slp114ErrorReturnNames = map[string]bool{
	"Insert": true, "Update": true, "Delete": true, "Query": true,
	"Exec": true, "Write": true, "Read": true, "Close": true,
	"Send": true, "Recv": true, "Flush": true, "Sync": true,
	"Create": true, "Save": true, "Remove": true, "Parse": true,
	"Decode": true, "Encode": true, "Marshal": true, "Unmarshal": true,
	"Commit": true, "Rollback": true, "Run": true, "Do": true,
	"Apply": true, "Call": true, "ExecContext": true,
}

var slp114ErrorReturnPrefixes = []string{
	"err", "Err", "Error", "errors.",
}

func isErrorReturningFunc(name string) bool {
	if slp114ErrorReturnNames[name] {
		return true
	}
	for _, prefix := range slp114ErrorReturnPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
