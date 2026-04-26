package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP094 flags shell commands that suppress failures with || true or || :
// This is a common anti-pattern where AI agents silence errors instead of
// handling them, leading to builds that appear green but are actually broken.
type SLP094 struct{}

func (SLP094) ID() string                { return "SLP094" }
func (SLP094) DefaultSeverity() Severity { return SeverityBlock }
func (SLP094) Description() string {
	return "shell command suppresses failure with || true or || : — handle the error instead"
}

var slp094SilentFail = regexp.MustCompile(`\|\|\s*(?:true|:)\s*(?:;|\s*$|&&|\)|\s)`)
var slp094YAMLRunLine = regexp.MustCompile(`^\s*(?:-\s*)?run\s*:\s*(.*)$`)

func (r SLP094) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isShellLikeFile(f.Path) {
			continue
		}
		for _, candidate := range slp094CommandCandidates(f) {
			if slp094IsCommentOnlyLine(candidate.command) {
				continue
			}
			if slp094SilentFail.MatchString(candidate.command) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     candidate.line.NewLineNo,
					Message:  "|| true or || : suppresses command failure — handle the error or explicitly comment why it's safe",
					Snippet:  candidate.line.Content,
				})
			}
		}
	}
	return out
}

type slp094CommandCandidate struct {
	line    diff.Line
	command string
}

func slp094CommandCandidates(f diff.File) []slp094CommandCandidate {
	if !slp094IsYAMLFile(f.Path) {
		out := make([]slp094CommandCandidate, 0, len(f.AddedLines()))
		for _, ln := range f.AddedLines() {
			out = append(out, slp094CommandCandidate{line: ln, command: ln.Content})
		}
		return out
	}

	var out []slp094CommandCandidate
	for _, h := range f.Hunks {
		inRunBlock := false
		runIndent := -1
		for _, ln := range h.Lines {
			if ln.Kind == diff.LineDelete {
				continue
			}

			content := ln.Content
			trim := strings.TrimSpace(content)
			indent := slp094Indent(content)

			if inRunBlock {
				if trim != "" && indent <= runIndent {
					inRunBlock = false
				} else {
					if ln.Kind == diff.LineAdd {
						out = append(out, slp094CommandCandidate{line: ln, command: trim})
					}
					continue
				}
			}

			match := slp094YAMLRunLine.FindStringSubmatch(content)
			if len(match) == 0 {
				continue
			}

			value := strings.TrimSpace(match[1])
			if value == "" {
				continue
			}
			if strings.HasPrefix(value, "|") || strings.HasPrefix(value, ">") {
				inRunBlock = true
				if strings.HasPrefix(trim, "-") {
					runIndent = indent + 2
				} else {
					runIndent = indent
				}
				continue
			}
			if ln.Kind == diff.LineAdd {
				out = append(out, slp094CommandCandidate{line: ln, command: value})
			}
		}
	}
	return out
}

func slp094IsCommentOnlyLine(content string) bool {
	trim := strings.TrimSpace(content)
	return trim == "" ||
		strings.HasPrefix(trim, "#") ||
		strings.HasPrefix(trim, "//") ||
		strings.HasPrefix(trim, "/*") ||
		strings.HasPrefix(trim, "*/")
}

func slp094IsYAMLFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml")
}

func slp094Indent(content string) int {
	return len(content) - len(strings.TrimLeft(content, " \t"))
}

func isShellLikeFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash") {
		return true
	}
	base := lower
	if i := strings.LastIndex(lower, "/"); i >= 0 {
		base = lower[i+1:]
	}
	if base == "makefile" || strings.HasSuffix(base, ".mk") {
		return true
	}
	if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") {
		// CI token as standalone segment
		isCI := base == "ci.yml" || base == "ci.yaml" ||
			strings.HasPrefix(base, "ci.") || strings.HasPrefix(base, "ci-") ||
			strings.HasSuffix(base, "-ci.yml") || strings.HasSuffix(base, "-ci.yaml") ||
			strings.HasSuffix(base, ".ci.yml") || strings.HasSuffix(base, ".ci.yaml")

		inGitHubWorkflows := strings.HasPrefix(lower, ".github/workflows/") || strings.Contains(lower, "/.github/workflows/")
		isWorkflow := strings.HasPrefix(base, "workflow.") ||
			strings.HasPrefix(base, "workflow-")

		return inGitHubWorkflows || isCI || isWorkflow
	}
	return false
}
