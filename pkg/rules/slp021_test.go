package rules

import (
	"strings"
	"testing"
)

func TestSLP021_InconsistentNaming(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "mixed camel and snake flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 func foo() {
-	return
+	userName := "alice"
+	user_email := "bob@test.com"
 }`,
			want: 1,
		},
		{
			name: "all camelCase not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 func foo() {
-	return
+	userName := "alice"
+	lastName := "smith"
 }`,
			want: 0,
		},
		{
			name: "all snake_case not flagged",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,3 +1,5 @@
 def foo():
-    pass
+    user_name = "alice"
+    user_email = "bob@test.com"
 }`,
			want: 0,
		},
		{
			name: "SCREAMING_SNAKE not counted",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 func foo() {
-	return
+	userName := "alice"
+	MAX_RETRIES := 3
 }`,
			want: 0,
		},
		{
			name: "test file not flagged",
			diff: `diff --git a/main_test.go b/main_test.go
--- a/main_test.go
+++ b/main_test.go
@@ -1,3 +1,5 @@
 func TestFoo(t *testing.T) {
-	return
+	userName := "alice"
+	user_email := "bob@test.com"
 }`,
			want: 0,
		},
		{
			name: "one of each still flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 func foo() {
-	return
+	userName := "alice"
+	user_email := "bob@test.com"
 }`,
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP021{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestSLP021_IDAndDescription(t *testing.T) {
	var r SLP021
	if r.ID() != "SLP021" {
		t.Errorf("ID() = %q, want SLP021", r.ID())
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("DefaultSeverity() = %v, want info", r.DefaultSeverity())
	}
	if !strings.Contains(r.Description(), "camelCase") || !strings.Contains(r.Description(), "snake_case") {
		t.Errorf("Description() should mention camelCase and snake_case: %q", r.Description())
	}
}
