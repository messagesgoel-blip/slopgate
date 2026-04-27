package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

type SLP114 struct{}

func (SLP114) ID() string                { return "SLP114" }
func (SLP114) DefaultSeverity() Severity { return SeverityWarn }
func (SLP114) Description() string {
	return "error-returning function called as statement — check the error return"
}

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
				if strings.HasPrefix(stripped, "if ") {
					continue
				}

				if strings.HasPrefix(stripped, "_ = ") {
					continue
				}

				if idx := indexOutsideQuotes(stripped, "("); idx > 0 {
					funcCall := stripped[:idx]
					lastIdent := funcCall
					if dot := strings.LastIndexByte(funcCall, '.'); dot >= 0 {
						lastIdent = funcCall[dot+1:]
					}
					lastIdent = strings.TrimSpace(lastIdent)
					if lastIdent != "" {
						if strings.HasSuffix(stripped, ")") || strings.HasSuffix(stripped, "){") {
							hasErrorReturn := false
							if isErrorReturningFunc(lastIdent, stripped) {
								hasErrorReturn = true
							}

							if hasErrorReturn {
								out = append(out, Finding{
									RuleID:   r.ID(),
									Severity: r.DefaultSeverity(),
									File:     f.Path,
									Line:     ln.NewLineNo,
									Message:  "error-returning function '" + lastIdent + "' called as statement — check for missing error check",
									Snippet:  content,
								})
							}
						}
					}
				}
			}
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

func isErrorReturningFunc(name string, callSnippet string) bool {
	if slp114ErrorReturnNames[name] {
		return true
	}
	if strings.HasPrefix(name, "New") || strings.HasPrefix(name, "new") {
		return true
	}
	for _, prefix := range slp114ErrorReturnPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
