package rules

import (
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

func TestSLP037(t *testing.T) {
	tests := []struct {
		name         string
		input        *diff.Diff
		wantFindings int
		wantRuleID   string
		wantLine     int
	}{
		{
			name: "INSERT without transaction handling",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "db.go",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 10, Content: "\t_, err := db.Exec(`INSERT INTO events (repo, payload) VALUES (?, ?)`, repo, payload)"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
			wantRuleID:   "SLP037",
			wantLine:     10,
		},
		{
			name: "INSERT with transaction handling present",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "db.go",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 10, Content: "tx, err := db.BeginTx(ctx, nil)"},
									{Kind: diff.LineAdd, NewLineNo: 11, Content: "\t_, err := tx.Exec(`INSERT INTO events (repo, payload) VALUES (?, ?)`, repo, payload)"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
			wantRuleID:   "",
			wantLine:     0,
		},
		{
			name: "ctx assignment should not suppress finding",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "db.go",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 5, Content: "ctx := r.Context()"},
									{Kind: diff.LineAdd, NewLineNo: 6, Content: "\t_, err := db.Exec(`INSERT INTO events (repo, payload) VALUES (?, ?)`, repo, payload)"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
			wantRuleID:   "SLP037",
			wantLine:     6,
		},
		{
			name: "Query with SELECT should not be flagged",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "db.go",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 10, Content: "\trows, err := db.Query(`SELECT updated_at FROM events WHERE id = ?`, id)"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
			wantRuleID:   "",
			wantLine:     0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := SLP037{}
			out := r.Check(tt.input)
			if len(out) != tt.wantFindings {
				t.Errorf("got %d findings, want %d", len(out), tt.wantFindings)
			}
			if tt.wantFindings > 0 && len(out) > 0 {
				if out[0].RuleID != tt.wantRuleID {
					t.Errorf("got RuleID %q, want %q", out[0].RuleID, tt.wantRuleID)
				}
				if out[0].Line != tt.wantLine {
					t.Errorf("got Line %d, want %d", out[0].Line, tt.wantLine)
				}
			}
		})
	}
}