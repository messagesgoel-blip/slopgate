// Package report formats slopgate findings for human and machine consumers.
package report

import (
	"fmt"
	"io"
	"sort"

	"github.com/messagesgoel-blip/slopgate/pkg/rules"
)

// ansi holds the minimal set of escapes used by the text reporter.
type ansi struct {
	bold, dim, red, yellow, cyan, reset string
}

var colored = ansi{
	bold: "\033[1m", dim: "\033[2m",
	red: "\033[31m", yellow: "\033[33m", cyan: "\033[36m",
	reset: "\033[0m",
}

var plain = ansi{}

// WriteText renders findings as a grouped, human-readable report.
// When there are no findings, it writes a single clean line.
func WriteText(w io.Writer, findings []rules.Finding, color bool) {
	a := plain
	if color {
		a = colored
	}
	if len(findings) == 0 {
		fmt.Fprintln(w, "slopgate: no findings")
		return
	}

	// Group by file path.
	byFile := map[string][]rules.Finding{}
	var paths []string
	for _, f := range findings {
		if _, ok := byFile[f.File]; !ok {
			paths = append(paths, f.File)
		}
		byFile[f.File] = append(byFile[f.File], f)
	}
	sort.Strings(paths)

	var blocks, warns, infos int
	for _, p := range paths {
		fmt.Fprintf(w, "\n%s%s%s\n", a.bold, p, a.reset)
		sort.Slice(byFile[p], func(i, j int) bool {
			return byFile[p][i].Line < byFile[p][j].Line
		})
		for _, f := range byFile[p] {
			switch f.Severity {
			case rules.SeverityBlock:
				blocks++
			case rules.SeverityWarn:
				warns++
			default:
				infos++
			}
			sevColor := severityColor(a, f.Severity)
			fmt.Fprintf(w, "  %s%s:%d%s %s[%s]%s %s[%s]%s %s\n",
				a.cyan, f.File, f.Line, a.reset,
				sevColor, f.Severity.String(), a.reset,
				a.dim, f.RuleID, a.reset,
				f.Message,
			)
			if f.Snippet != "" {
				fmt.Fprintf(w, "    %s%s%s\n", a.dim, f.Snippet, a.reset)
			}
		}
	}

	total := len(findings)
	fmt.Fprintf(w, "\nslopgate: %d finding%s (%d block, %d warn, %d info)\n",
		total, plural(total), blocks, warns, infos)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func severityColor(a ansi, s rules.Severity) string {
	switch s {
	case rules.SeverityBlock:
		return a.red
	case rules.SeverityWarn:
		return a.yellow
	default:
		return a.cyan
	}
}
