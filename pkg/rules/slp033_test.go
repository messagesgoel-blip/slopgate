package rules

import (
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

func TestSLP033(t *testing.T) {
	tests := []struct {
		name         string
		input        *diff.Diff
		wantFindings int
	}{
		{
			name: "useState used without import",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "const [count, setCount] = useState(0);"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "useState used with import",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "import { useState } from 'react';"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "const [count, setCount] = useState(0);"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Type used in type annotation without import",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "function MyComponent(props: ComponentProps) {"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "Type used with import",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "import { ComponentProps } from 'react';"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "function MyComponent(props: ComponentProps) {"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Non-JS/TS file",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "main.go",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "useState := 0"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Namespace import satisfies React availability",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "import * as React from 'react';"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "const [count, setCount] = React.useState(0);"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Import in different hunk",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "import { useState } from 'react';"},
								},
							},
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 5, Content: "const [count, setCount] = useState(0);"},
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
			rule := SLP033{}
			findings := rule.Check(tt.input)
			if len(findings) != tt.wantFindings {
				t.Errorf("SLP033 got %d findings, want %d", len(findings), tt.wantFindings)
				for _, f := range findings {
					t.Logf("Finding: %s:%d - %s", f.File, f.Line, f.Message)
				}
			}
		})
	}
}
