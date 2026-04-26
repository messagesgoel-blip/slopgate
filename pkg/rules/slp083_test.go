package rules

import (
	"strings"
	"testing"
)

func TestSLP083(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int
	}{
		{
			name: "useCallback without deps",
			diff: strings.TrimSpace(`diff --git a/src/components/MyComponent.tsx b/src/components/MyComponent.tsx
index 123..456 100644
--- a/src/components/MyComponent.tsx
+++ b/src/components/MyComponent.tsx
@@ -1,5 +1,8 @@
 import { useCallback } from 'react'
 const MyComponent = () => {
-  const handleClick = useCallback(() => { console.log('clicked') })
+  const handleClick = useCallback(() => { console.log('clicked') })
   return <button onClick={handleClick}>Click</button>
 }
`),
			expected: 1,
		},
		{
			name: "useCallback with empty deps is ok",
			diff: strings.TrimSpace(`diff --git a/src/components/MyComponent.tsx b/src/components/MyComponent.tsx
index 123..456 100644
--- a/src/components/MyComponent.tsx
+++ b/src/components/MyComponent.tsx
@@ -1,5 +1,8 @@
 import { useCallback } from 'react'
 const MyComponent = () => {
-  const handleClick = useCallback(() => { console.log('clicked') }, [])
+  const handleClick = useCallback(() => { console.log('clicked') }, [])
   return <button onClick={handleClick}>Click</button>
 }
`),
			expected: 0,
		},
		{
			name: "useMemo without deps",
			diff: strings.TrimSpace(`diff --git a/src/components/MyComponent.tsx b/src/components/MyComponent.tsx
index 123..456 100644
--- a/src/components/MyComponent.tsx
+++ b/src/components/MyComponent.tsx
@@ -1,5 +1,8 @@
 import { useMemo } from 'react'
 const MyComponent = () => {
-  const value = useMemo(() => computeExpensiveValue())
+  const value = useMemo(() => computeExpensiveValue())
   return <div>{value}</div>
 }
`),
			expected: 1,
		},
		{
			name: "useMemo with empty deps is ok",
			diff: strings.TrimSpace(`diff --git a/src/components/MyComponent.tsx b/src/components/MyComponent.tsx
index 123..456 100644
--- a/src/components/MyComponent.tsx
+++ b/src/components/MyComponent.tsx
@@ -1,5 +1,8 @@
 import { useMemo } from 'react'
 const MyComponent = () => {
-  const value = useMemo(() => computeExpensiveValue(), [])
+  const value = useMemo(() => computeExpensiveValue(), [])
   return <div>{value}</div>
 }
`),
			expected: 0,
		},
		{
			name: "regular function should be skipped",
			diff: strings.TrimSpace(`diff --git a/src/utils.ts b/src/utils.ts
index 123..456 100644
--- a/src/utils.ts
+++ b/src/utils.ts
@@ -1,3 +1,5 @@
 const add = (a: number, b: number) => a + b
 export { add }
`),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP083{}
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
