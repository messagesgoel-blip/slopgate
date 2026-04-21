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
