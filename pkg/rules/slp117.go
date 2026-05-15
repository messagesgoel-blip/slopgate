package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP117 checks for unanchored regex patterns that could match unintended substrings.
type SLP117 struct{}

func (SLP117) ID() string                { return "SLP117" }
func (SLP117) DefaultSeverity() Severity { return SeverityInfo }
func (SLP117) Description() string {
	return "unanchored regex — add ^, $, or \\b anchor to prevent unintended substring matches"
}

func slp117HasAnchor(s string) bool {
	return strings.Contains(s, `^`) || strings.Contains(s, `$`) ||
		strings.Contains(s, `\b`) || strings.Contains(s, `\A`) ||
		strings.Contains(s, `\z`) || strings.Contains(s, `\Z`)
}

var (
	slp117JSRegExpCtor   = regexp.MustCompile(`(?:^|[^A-Za-z0-9_$])(?:new\s+)?RegExp\s*\(`)
	slp117JSRegexLiteral = regexp.MustCompile(`(?:=|return|\(|:|,)\s*/(?:\\.|[^/\\\n])+/[a-zA-Z]*`)
)

func slp117LooksLikeRegex(cleaned string) bool {
	if strings.Contains(cleaned, "regexp.") ||
		slp117JSRegExpCtor.MatchString(cleaned) {
		return true
	}
	return slp117JSRegexLiteral.MatchString(cleaned)
}

func (r SLP117) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				raw := strings.TrimSpace(ln.Content)
				cleaned := stripCommentAndStrings(ln.Content)
				cleaned = strings.TrimSpace(cleaned)

				if cleaned == "" || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/*") || strings.HasPrefix(raw, "#") {
					continue
				}

				indicatorSource := cleaned
				if indicatorSource == "" {
					indicatorSource = raw
				}

				if !slp117LooksLikeRegex(indicatorSource) {
					continue
				}

				if slp117HasAnchor(cleaned) || slp117HasAnchor(raw) {
					continue
				}

				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "unanchored regex pattern — add ^, $, or \\b anchors to prevent unintended substring matches",
					Snippet:  ln.Content,
				})
			}
		}
	}
	return out
}
