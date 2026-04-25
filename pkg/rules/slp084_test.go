package rules

import (
	"testing"
)

func TestSLP084(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int
	}{
		{
			name: "useEffect with addEventListener no cleanup",
			diff: `diff --git a/src/hooks/useWindowListener.ts b/src/hooks/useWindowListener.ts
index 123..456 100644
--- a/src/hooks/useWindowListener.ts
+++ b/src/hooks/useWindowListener.ts
@@ -1,5 +1,8 @@
-import { useEffect } from 'react'
-const useWindowListener = (event, handler) => {
-  useEffect(() => { window.addEventListener(event, handler) })
+import { useEffect } from 'react'
+const useWindowListener = (event, handler) => {
+  useEffect(() => { window.addEventListener(event, handler) })
 }
 export { useWindowListener }
`,
			expected: 1,
		},
		{
			name: "useEffect with cleanup is ok",
			diff: `diff --git a/src/hooks/useWindowListener.ts b/src/hooks/useWindowListener.ts
index 123..456 100644
--- a/src/hooks/useWindowListener.ts
+++ b/src/hooks/useWindowListener.ts
@@ -1,5 +1,8 @@
-import { useEffect } from 'react'
-const useWindowListener = (event, handler) => {
-  useEffect(() => {
-    window.addEventListener(event, handler)
-    return () => { window.removeEventListener(event, handler) }
-  })
+import { useEffect } from 'react'
+const useWindowListener = (event, handler) => {
+  useEffect(() => {
+    window.addEventListener(event, handler)
+    return () => { window.removeEventListener(event, handler) }
+  })
 }
 export { useWindowListener }
`,
			expected: 0,
		},
		{
			name: "useEffect with setTimeout no cleanup",
			diff: `diff --git a/src/components/MyComponent.tsx b/src/components/MyComponent.tsx
index 123..456 100644
--- a/src/components/MyComponent.tsx
+++ b/src/components/MyComponent.tsx
@@ -1,5 +1,8 @@
-const MyComponent = () => {
-  useEffect(() => { const id = setTimeout(() => {}, 1000) })
+const MyComponent = () => {
+  useEffect(() => { const id = setTimeout(() => {}, 1000) })
   return <div />
 }
`,
			expected: 1,
		},
		{
			name: "useEffect with async but cleanup present",
			diff: `diff --git a/src/components/MyComponent.tsx b/src/components/MyComponent.tsx
index 123..456 100644
--- a/src/components/MyComponent.tsx
+++ b/src/components/MyComponent.tsx
@@ -1,5 +1,8 @@
-const MyComponent = () => {
-  useEffect(() => {
-    const controller = new AbortController()
-    fetch('/api/data', { signal: controller.signal })
-    return () => controller.abort()
-  })
+const MyComponent = () => {
+  useEffect(() => {
+    const controller = new AbortController()
+    fetch('/api/data', { signal: controller.signal })
+    return () => controller.abort()
+  })
   return <div />
 }
`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP084{}
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
