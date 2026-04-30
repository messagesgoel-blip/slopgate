package rules

import (
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

func TestSLP035(t *testing.T) {
	tests := []struct {
		name         string
		input        *diff.Diff
		wantFindings int
	}{
		{
			name: "Console log statement",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "console.log('debugging');"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "Debugger statement",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "debugger;"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "TODO without ticket reference",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "// TODO: fix this later"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "TODO with ticket reference",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "// TODO: CR-123 fix this later"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Long line",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "const reallyLongVariableNameThatExceedsTheRecommendedLengthLimitForCodeQualityAndReadabilityAndWouldBeHardToReviewInADiffBecauseItKeepsGoing = getValue();"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "Normal code without issues",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "const value = getValue();"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "return <div>{value}</div>;"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Long docs line ignored",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "docs/plan.md",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "This is a long planning paragraph that intentionally exceeds one hundred characters because prose is wrapped by editors and should not be treated as code quality slop."},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Console log in different hunk",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "const value = getValue();"},
								},
							},
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 5, Content: "console.log('debugging');"},
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
			rule := SLP035{}
			findings := rule.Check(tt.input)
			if len(findings) != tt.wantFindings {
				t.Errorf("SLP035 got %d findings, want %d", len(findings), tt.wantFindings)
				for _, f := range findings {
					t.Logf("Finding: %s:%d - %s", f.File, f.Line, f.Message)
				}
			}
		})
	}
}
