package rules

import (
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

func TestSLP031(t *testing.T) {
	tests := []struct {
		name        string
		input       *diff.Diff
		wantFindings int
	}{
		{
			name: "Direct code intake without license validation",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "ATTRIBUTION.md",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "This directory is a direct-code intake from `codero-sparkle-start`"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "Upstream repo: https://github.com/example/repo"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "Intake with license validation mentioned",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "ATTRIBUTION.md",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "This directory is a direct-code intake from `codero-sparkle-start`"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "License validation completed by legal team"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Non-documentation file",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "main.go",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "fmt.Println(\"hello\")"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Split hunks in same file",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "ATTRIBUTION.md",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "Some unrelated content"},
								},
							},
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 5, Content: "This directory is a direct-code intake from `codero-sparkle-start`"},
									{Kind: diff.LineAdd, NewLineNo: 6, Content: "Upstream repo: https://github.com/example/repo"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := SLP031{}
			findings := rule.Check(tt.input)
			if len(findings) != tt.wantFindings {
				t.Errorf("SLP031 got %d findings, want %d", len(findings), tt.wantFindings)
			}
		})
	}
}