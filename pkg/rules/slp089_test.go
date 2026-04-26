package rules

import (
	"strings"
	"testing"
)

func TestSLP089(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int
	}{
		{
			name: "export without docstring",
			diff: strings.TrimSpace(`diff --git a/src/utils.js b/src/utils.js
index 123..456 100644
--- a/src/utils.js
+++ b/src/utils.js
@@ -1,5 +1,8 @@
 const add = (a, b) => a + b
 const multiply = (a, b) => a * b
-export { add, multiply }
+export { add, multiply }
`),
			expected: 2,
		},
		{
			name: "export with JSDoc is ok",
			diff: strings.TrimSpace(`diff --git a/src/utils.js b/src/utils.js
index 123..456 100644
--- a/src/utils.js
+++ b/src/utils.js
@@ -1,5 +1,8 @@
 /**
  * Adds two numbers
  * @param {number} a
  * @param {number} b
  * @returns {number}
  */
 const add = (a, b) => a + b
 /**
  * Multiplies two numbers
  * @param {number} a
  * @param {number} b
  * @returns {number}
  */
 const multiply = (a, b) => a * b
-export { add, multiply }
+export { add, multiply }
`),
			expected: 0,
		},
		{
			name: "export with comment is ok",
			diff: strings.TrimSpace(`diff --git a/src/utils.js b/src/utils.js
index 123..456 100644
--- a/src/utils.js
+++ b/src/utils.js
@@ -1,5 +1,8 @@
+// Adds two numbers
 const add = (a, b) => a + b
+// Multiplies two numbers
 const multiply = (a, b) => a * b
-export { add, multiply }
+export { add, multiply }
`),
			expected: 0,
		},
		{
			name: "JavaScript file should be checked",
			diff: strings.TrimSpace(`diff --git a/src/helper.js b/src/helper.js
index 123..456 100644
--- a/src/helper.js
+++ b/src/helper.js
@@ -1,3 +1,5 @@
 const formatData = (data) => JSON.stringify(data)
-export { formatData }
+export { formatData }
`),
			expected: 1,
		},
		{
			name: "TypeScript export without docstring",
			diff: strings.TrimSpace(`diff --git a/src/types.ts b/src/types.ts
index 123..456 100644
--- a/src/types.ts
+++ b/src/types.ts
@@ -1,5 +1,8 @@
-export interface User {
-  id: number
-  name: string
+export interface User {
+  id: number
+  name: string
 }
`),
			expected: 1,
		},
		{
			name: "Go export without docstring",
			diff: strings.TrimSpace(`diff --git a/internal/db/user.go b/internal/db/user.go
index 123..456 100644
--- a/internal/db/user.go
+++ b/internal/db/user.go
@@ -1,5 +1,8 @@
-func GetUser(id int) User {
-  // Get user from database
+func GetUser(id int) User {
+  // Get user from database
   return User{ID: id}
 }
`),
			expected: 1,
		},
		{
			name: "Go export with docstring is ok",
			diff: strings.TrimSpace(`diff --git a/internal/db/user.go b/internal/db/user.go
index 123..456 100644
--- a/internal/db/user.go
+++ b/internal/db/user.go
@@ -1,5 +1,8 @@
+// GetUser retrieves a user by ID
+// Returns User with matching ID
 func GetUser(id int) User {
-  // Get user from database
+  // Get user from database
   return User{ID: id}
 }
`),
			expected: 0,
		},
		{
			name: "Python export without docstring",
			diff: strings.TrimSpace(`diff --git a/src/utils.py b/src/utils.py
index 123..456 100644
--- a/src/utils.py
+++ b/src/utils.py
@@ -1,5 +1,8 @@
-def add(a, b):
-    return a + b
-def multiply(a, b):
-    return a * b
+def add(a, b):
+    return a + b
+def multiply(a, b):
+    return a * b
`),
			expected: 2,
		},
		{
			name: "test file should be skipped",
			diff: strings.TrimSpace(`diff --git a/src/utils_test.js b/src/utils_test.js
index 123..456 100644
--- a/src/utils_test.js
+++ b/src/utils_test.js
@@ -1,3 +1,5 @@
-const testAdd = () => { assert(add(1,2) === 3) }
+const testAdd = () => { assert(add(1,2) === 3) }
`),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP089{}
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
