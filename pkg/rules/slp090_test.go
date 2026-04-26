package rules

import (
	"testing"
)

func TestSLP090(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int
	}{
		{
			name: "API route without error handling",
			diff: `diff --git a/src/routes/users.js b/src/routes/users.js
index 123..456 100644
--- a/src/routes/users.js
+++ b/src/routes/users.js
@@ -1,5 +1,8 @@
-router.post('/users', async (req, res) => {
-  const user = await User.create(req.body)
-  res.status(201).json(user)
+router.post('/users', async (req, res) => {
+  const user = await User.create(req.body)
+  res.status(201).json(user)
 })
`,
			expected: 1,
		},
		{
			name: "API route with try-catch is ok",
			diff: `diff --git a/src/routes/users.js b/src/routes/users.js
index 123..456 100644
--- a/src/routes/users.js
+++ b/src/routes/users.js
@@ -1,5 +1,8 @@
-router.post('/users', async (req, res) => {
-  try {
-    const user = await User.create(req.body)
-    res.status(201).json(user)
-  } catch (err) {
-    res.status(500).json({ error: err.message })
-  }
+router.post('/users', async (req, res) => {
+  try {
+    const user = await User.create(req.body)
+    res.status(201).json(user)
+  } catch (err) {
+    res.status(500).json({ error: err.message })
+  }
 })
`,
			expected: 0,
		},
		{
			name: "Express route with error middleware",
			diff: `diff --git a/src/routes/users.js b/src/routes/users.js
index 123..456 100644
--- a/src/routes/users.js
+++ b/src/routes/users.js
@@ -1,5 +1,8 @@
-router.post('/users', (req, res, next) => {
-  User.create(req.body)
-    .then(user => res.status(201).json(user))
-    .catch(next)
+router.post('/users', (req, res, next) => {
+  User.create(req.body)
+    .then(user => res.status(201).json(user))
+    .catch(next)
 })
`,
			expected: 0,
		},
		{
			name: "non-API file should be skipped",
			diff: `diff --git a/src/utils.js b/src/utils.js
index 123..456 100644
--- a/src/utils.js
+++ b/src/utils.js
@@ -1,3 +1,5 @@
-const process = () => { return true }
+const process = () => { return true }
 export { process }
`,
			expected: 0,
		},
		{
			name: "Go handler without error handling",
			diff: `diff --git a/internal/handlers/user.go b/internal/handlers/user.go
index 123..456 100644
--- a/internal/handlers/user.go
+++ b/internal/handlers/user.go
@@ -1,5 +1,8 @@
-func CreateUser(w http.ResponseWriter, r *http.Request) {
-  var user User
-  json.NewDecoder(r.Body).Decode(&user)
+func CreateUser(w http.ResponseWriter, r *http.Request) {
+  var user User
+  json.NewDecoder(r.Body).Decode(&user)
   db.Create(&user)
   w.WriteHeader(http.StatusCreated)
   json.NewEncoder(w).Encode(user)
 }
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP090{}
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
