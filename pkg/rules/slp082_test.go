package rules

import (
	"strings"
	"testing"
)

func TestSLP082(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int
	}{
		{
			name: "map returning JSX without key - inline",
			diff: strings.TrimSpace(`diff --git a/src/components/List.tsx b/src/components/List.tsx
index 123..456 100644
--- a/src/components/List.tsx
+++ b/src/components/List.tsx
@@ -1,3 +1,3 @@
-const items = [1, 2, 3]; items.map(i => <li>{i}</li>)
+const items = [1, 2, 3]; items.map(i => <li className="item">{i}</li>)
`),
			expected: 1,
		},
		{
			name: "map with key is ok - inline",
			diff: strings.TrimSpace(`diff --git a/src/components/List.tsx b/src/components/List.tsx
index 123..456 100644
--- a/src/components/List.tsx
+++ b/src/components/List.tsx
@@ -1,3 +1,3 @@
-const items = [1, 2, 3]; items.map(i => <li key={i}>{i}</li>)
+const items = [1, 2, 3]; items.map(i => <li key={i} className="item">{i}</li>)
`),
			expected: 0,
		},
		{
			name: "forEach with JSX without key - inline",
			diff: strings.TrimSpace(`diff --git a/src/components/List.tsx b/src/components/List.tsx
index 123..456 100644
--- a/src/components/List.tsx
+++ b/src/components/List.tsx
@@ -1,3 +1,3 @@
-const items = [1, 2, 3]; items.forEach(i => <li>{i}</li>)
+const items = [1, 2, 3]; items.forEach(i => <li className="item">{i}</li>)
`),
			expected: 1,
		},
		{
			name: "ts file should be skipped",
			diff: strings.TrimSpace(`diff --git a/src/utils.ts b/src/utils.ts
index 123..456 100644
--- a/src/utils.ts
+++ b/src/utils.ts
@@ -1,3 +1,5 @@
-const items = [1, 2, 3]
+const items = [1, 2, 3]
 export { items }
`),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP082{}
			findings := r.Check(d)

			if len(findings) != tt.expected {
				t.Errorf("expected %d findings, got %d", tt.expected, len(findings))
				for _, f := range findings {
					t.Logf("  - %s:%d: %s", f.File, f.Line, f.Message)
				}
			}
		})
	}
}
