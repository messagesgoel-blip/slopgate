package rules

import (
	"testing"
)

func TestSLP081(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int
	}{
		{
			name: "tsx without React import has JSX",
			diff: `diff --git a/src/components/Button.tsx b/src/components/Button.tsx
index 123..456 100644
--- a/src/components/Button.tsx
+++ b/src/components/Button.tsx
@@ -1,5 +1,8 @@
-const Button = () => {
-  return <button>Click me</button>
+const Button = () => {
+  return <button className="btn">Click me</button>
 }
 export default Button
`,
			expected: 1,
		},
		{
			name: "tsx with React import is ok",
			diff: `diff --git a/src/components/Button.tsx b/src/components/Button.tsx
index 123..456 100644
--- a/src/components/Button.tsx
+++ b/src/components/Button.tsx
@@ -1,5 +1,8 @@
+import React from 'react'
 const Button = () => {
-  return <button>Click me</button>
+  return <button className="btn">Click me</button>
 }
 export default Button
`,
			expected: 0,
		},
		{
			name: "ts file without JSX is ok",
			diff: `diff --git a/src/utils.ts b/src/utils.ts
index 123..456 100644
--- a/src/utils.ts
+++ b/src/utils.ts
@@ -1,3 +1,5 @@
-const add = (a: number, b: number) => a + b
+const add = (a: number, b: number) => a + b
 export { add }
`,
			expected: 0,
		},
		{
			name: "jsx file without React import",
			diff: `diff --git a/src/App.jsx b/src/App.jsx
index 123..456 100644
--- a/src/App.jsx
+++ b/src/App.jsx
@@ -1,5 +1,6 @@
-const App = () => <div>Hello</div>
+const App = () => <div className="app">Hello</div>
 export default App
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP081{}
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

func TestSLP081_JSXPattern(t *testing.T) {
	r := SLP081{}
	_ = r

	// Test JSX pattern detection
	tests := []struct {
		line     string
		expected bool
	}{
		{"const Button = () => <button>Click</button>", true},
		{"export default function App() { return <div /> }", true},
		{"export const Component = ({ children }) => <div>{children}</div>", true},
		{"const x = 5", false},
		{"const Button = () => { const [count, setCount] = useState(0); return <button>{count}</button> }", true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			d := parseDiff(t, `diff --git a/test.tsx b/test.tsx
index 123..456 100644
--- a/test.tsx
+++ b/test.tsx
@@ -1,1 +1,1 @@
-`+tt.line+`
+`+tt.line+`
`)
			r := SLP081{}
			findings := r.Check(d)

			hasFinding := len(findings) > 0
			if hasFinding != tt.expected {
				t.Errorf("line %q: expected JSX finding=%v, got %v", tt.line, tt.expected, hasFinding)
			}
		})
	}
}
