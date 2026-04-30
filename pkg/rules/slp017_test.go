package rules

import (
	"strings"
	"testing"
)

func TestSLP017_MagicNumber(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "magic number flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func calc(x int) int {
-	return 0
+	return x * 7
 }`,
			want: 1,
		},
		{
			name: "0 not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func calc(x int) int {
-	return x
+	return x + 0
 }`,
			want: 0,
		},
		{
			name: "1 not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func calc(x int) int {
-	return 0
+	return x + 1
 }`,
			want: 0,
		},
		{
			name: "2 not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func calc(x int) int {
-	return 0
+	return x / 2
 }`,
			want: 0,
		},
		{
			name: "const declaration not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func calc() {
-	var x = 0
+	const BatchSize = 100
 }`,
			want: 0,
		},
		{
			name: "hex literal not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func calc() int {
-	return 0
+	return 0xFF
 }`,
			want: 0,
		},
		{
			name: "array index not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func calc() int {
-	return 0
+	return arr[3]
 }`,
			want: 0,
		},
		{
			name: "ALL_CAPS assignment not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func calc() {
-	var x = 0
+	MAX_RETRIES = 42
 }`,
			want: 0,
		},
		{
			name: "test file not flagged",
			diff: `diff --git a/main_test.go b/main_test.go
--- a/main_test.go
+++ b/main_test.go
@@ -1,3 +1,4 @@
 func TestCalc(t *testing.T) {
-	_ = 0
+	if result := calc(); result != 42 { t.Fail() }
 }`,
			want: 0,
		},
		{
			name: "python magic number flagged",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,3 +1,4 @@
 def calc(x):
-    pass
+    return x * 7
 }`,
			want: 1,
		},
		{
			name: "python define not flagged",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,3 +1,4 @@
 def setup():
-    pass
+    #define THRESHOLD 100
 }`,
			want: 0,
		},
		{
			name: "float literal flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func calc() float64 {
-	return 0.0
+	return 3.14
 }`,
			want: 1,
		},
		{
			name: "HTTP status 400 not flagged",
			diff: `diff --git a/handler.js b/handler.js
--- a/handler.js
+++ b/handler.js
@@ -1,3 +1,4 @@
 async function handle() {
-	return res.json({})
+	if (!valid) return res.status(400).json({ error: 'bad' })
 }`,
			want: 0,
		},
		{
			name: "HTTP status 500 not flagged",
			diff: `diff --git a/handler.js b/handler.js
--- a/handler.js
+++ b/handler.js
@@ -1,3 +1,4 @@
 async function handle() {
-	return res.json({})
+	return res.status(500).json({ error: 'internal' })
 }`,
			want: 0,
		},
		{
			name: "SQL LIMIT 10 not flagged",
			diff: `diff --git a/query.sql b/query.sql
--- a/query.sql
+++ b/query.sql
@@ -1,2 +1,3 @@
 SELECT * FROM users
+LIMIT 10;
`,
			want: 0,
		},
		{
			name: "limit(50) not flagged",
			diff: `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -1,2 +1,3 @@
 const result = await query()
+  .limit(50);
`,
			want: 0,
		},
		{
			name: "pageSize 20 not flagged",
			diff: `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -1,2 +1,3 @@
 const pageSize = 20
+const result = await query.limit(pageSize)
`,
			want: 0,
		},
		{
			name: "non-standard number still flagged",
			diff: `diff --git a/tax.js b/tax.js
--- a/tax.js
+++ b/tax.js
@@ -1,2 +1,3 @@
+const taxRate = 86.9;  // not ALL_CAPS, should flag
return total * rate;
`,
			want: 1,
		},
		{
			name: "roadmap yaml not flagged",
			diff: `diff --git a/docs/roadmap.yml b/docs/roadmap.yml
--- a/docs/roadmap.yml
+++ b/docs/roadmap.yml
@@ -1,2 +1,5 @@
+milestone: 10
+target_days: 45
+phase: 3
`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP017{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestSLP017_IDAndDescription(t *testing.T) {
	var r SLP017
	if r.ID() != "SLP017" {
		t.Errorf("ID() = %q, want SLP017", r.ID())
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("DefaultSeverity() = %v, want info", r.DefaultSeverity())
	}
	if !strings.Contains(r.Description(), "constant") {
		t.Errorf("Description() should mention constant: %q", r.Description())
	}
}
