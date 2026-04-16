package report

import (
	"encoding/json"
	"io"

	"github.com/messagesgoel-blip/slopgate/pkg/rules"
)

// JSONFinding is the per-finding shape in JSON output.
type JSONFinding struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	Snippet  string `json:"snippet,omitempty"`
}

// JSONSummary holds aggregate counts.
type JSONSummary struct {
	Total int `json:"total"`
	Block int `json:"block"`
	Warn  int `json:"warn"`
	Info  int `json:"info"`
}

// JSONReport is the top-level JSON output.
type JSONReport struct {
	Findings []JSONFinding `json:"findings"`
	Summary  JSONSummary   `json:"summary"`
}

// WriteJSON writes machine-readable JSON to w.
func WriteJSON(w io.Writer, findings []rules.Finding) {
	r := JSONReport{
		Findings: make([]JSONFinding, 0, len(findings)),
	}
	for _, f := range findings {
		jf := JSONFinding{
			RuleID:   f.RuleID,
			Severity: f.Severity.String(),
			File:     f.File,
			Line:     f.Line,
			Message:  f.Message,
			Snippet:  f.Snippet,
		}
		r.Findings = append(r.Findings, jf)
		switch f.Severity {
		case rules.SeverityBlock:
			r.Summary.Block++
		case rules.SeverityWarn:
			r.Summary.Warn++
		default:
			r.Summary.Info++
		}
	}
	r.Summary.Total = len(findings)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(r)
}
