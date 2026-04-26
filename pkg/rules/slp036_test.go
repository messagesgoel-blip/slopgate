package rules

import (
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

func TestSLP036(t *testing.T) {
	tests := []struct {
		name         string
		input        *diff.Diff
		wantFindings int
	}{
		{
			name: "suspicious required list in OpenAPI",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "docs/api.yaml",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "  required: [repo, path, content, saved_at]"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "legitimate required list without suspicious words",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "docs/api.yaml",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "  required: [repo, path, content]"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "short required list with suspicious word — not flagged",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "docs/api.yaml",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 5, Content: "  required: [repo, size]"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "flow-style required with suspicious word and large list",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "docs/api.yaml",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 10, Content: "  required:"},
									{Kind: diff.LineAdd, NewLineNo: 11, Content: "    - repo"},
									{Kind: diff.LineAdd, NewLineNo: 12, Content: "    - path"},
									{Kind: diff.LineAdd, NewLineNo: 13, Content: "    - content"},
									{Kind: diff.LineAdd, NewLineNo: 14, Content: "    - size"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "flow-style required with small list — not flagged",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "docs/api.yaml",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 10, Content: "  required:"},
									{Kind: diff.LineAdd, NewLineNo: 11, Content: "    - repo"},
									{Kind: diff.LineAdd, NewLineNo: 12, Content: "    - size"},
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
			r := SLP036{}
			out := r.Check(tt.input)
			if len(out) != tt.wantFindings {
				t.Errorf("got %d findings, want %d", len(out), tt.wantFindings)
			}
		})
	}
}
