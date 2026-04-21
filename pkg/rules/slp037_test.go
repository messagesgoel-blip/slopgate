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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := SLP037{}
			out := r.Check(tt.input)
			if len(out) != tt.wantFindings {
				t.Errorf("got %d findings, want %d", len(out), tt.wantFindings)
			}
		})
	}
}
