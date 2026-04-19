package rules

import (
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

func TestSLP034(t *testing.T) {
	tests := []struct {
		name        string
		input       *diff.Diff
		wantFindings int
	}{
		{
			name: "Multiple sequential state updates",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "setCount(count + 1);"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "setName('new name');"},
									{Kind: diff.LineAdd, NewLineNo: 3, Content: "setActive(true);"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "Single state update",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "setCount(count + 1);"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "Complex state management pattern",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "useEffect(() => {"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "  const fetchData = async () => {"},
									{Kind: diff.LineAdd, NewLineNo: 3, Content: "    // async logic"},
									{Kind: diff.LineAdd, NewLineNo: 4, Content: "  };"},
									{Kind: diff.LineAdd, NewLineNo: 5, Content: "}, []);"},
								},
							},
						},
					},
				},
			},
			wantFindings: 0, // This is a proper async useEffect
		},
		{
			name: "Anti-pattern useEffect with async function",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "useEffect(async () => {"},
									{Kind: diff.LineAdd, NewLineNo: 2, Content: "  const response = await fetch('/api/data');"},
									{Kind: diff.LineAdd, NewLineNo: 3, Content: "}, []);"},
								},
							},
						},
					},
				},
			},
			wantFindings: 1,
		},
		{
			name: "Anti-pattern useEffect with async function in different hunk",
			input: &diff.Diff{
				Files: []diff.File{
					{
						Path: "Component.tsx",
						Hunks: []diff.Hunk{
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 1, Content: "const [data, setData] = useState(null);"},
								},
							},
							{
								Lines: []diff.Line{
									{Kind: diff.LineAdd, NewLineNo: 5, Content: "useEffect(async () => {"},
									{Kind: diff.LineAdd, NewLineNo: 6, Content: "  const response = await fetch('/api/data');"},
									{Kind: diff.LineAdd, NewLineNo: 7, Content: "}, []);"},
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
			rule := SLP034{}
			findings := rule.Check(tt.input)
			if len(findings) != tt.wantFindings {
				t.Errorf("SLP034 got %d findings, want %d", len(findings), tt.wantFindings)
				for _, f := range findings {
					t.Logf("Finding: %s:%d - %s", f.File, f.Line, f.Message)
				}
			}
		})
	}
}