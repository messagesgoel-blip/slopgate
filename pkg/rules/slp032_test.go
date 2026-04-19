package rules

import (
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

func TestSLP032(t *testing.T) {
	tests := []struct {
		name        string
		input       *diff.Diff
		wantFindings int
	}{
		{
			name: "TSX component without React import",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "export function MyComponent() {"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "  return <div>Hello</div>;"},
									{Kind: diff.LineAdd, NewLineNo: 3, Content: "}"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "TSX with React import",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "import React from 'react';"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "export function MyComponent() {"},
									{Kind: diff.LineAdd, NewLineNo: 3, Content: "  return <div>Hello</div>;"},
									{Kind: diff.LineAdd, NewLineNo: 4, Content: "}"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "JSX button without accessibility",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "import React from 'react';"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "export function MyComponent() {"},
									{Kind: diff.LineAdd, NewLineNo: 3, Content: "  return <button>Click me</button>;"},
									{Kind: diff.LineAdd, NewLineNo: 4, Content: "}"},
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
			rule := SLP032{}
			findings := rule.Check(tt.input)
			if len(findings) != tt.wantFindings {
				t.Errorf("SLP032 got %d findings, want %d", len(findings), tt.wantFindings)
				for _, f := range findings {
					t.Logf("Finding: %s:%d - %s", f.File, f.Line, f.Message)
				}
			}
		})
	}
}