package rules

import (
	"strings"
	"testing"
)

func TestSLP023_BareTypeAssertion(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "bare assertion flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo(v interface{}) {
-	_ = nil
+	s := v.(string)
 }`,
			want: 1,
		},
		{
			name: "comma-ok not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo(v interface{}) {
-	_ = nil
+	s, ok := v.(string)
 }`,
			want: 0,
		},
		{
			name: "type switch not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo(v interface{}) {
-	_ = nil
+	switch v := v.(type) {
 }`,
			want: 0,
		},
		{
			name: "test file not flagged",
			diff: `diff --git a/main_test.go b/main_test.go
--- a/main_test.go
+++ b/main_test.go
@@ -1,3 +1,4 @@
 func TestFoo(t *testing.T) {
-	_ = nil
+	s := v.(string)
 }`,
			want: 0,
		},
		{
			name: "pointer type assertion flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo(v interface{}) {
-	_ = nil
+	p := v.(*Foo)
 }`,
			want: 1,
		},
		{
			name: "method call not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo(v interface{}) {
-	_ = nil
+	fmt.Println("hello")
 }`,
			want: 0,
		},
		{
			name: "comma-ok with equals not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo(v interface{}) {
-	_ = nil
+	s, ok = v.(string)
 }`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP023{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestSLP023_IDAndDescription(t *testing.T) {
	var r SLP023
	if r.ID() != "SLP023" {
		t.Errorf("ID() = %q, want SLP023", r.ID())
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("DefaultSeverity() = %v, want warn", r.DefaultSeverity())
	}
	if r.Description() == "" {
		t.Error("Description() is empty")
	}
	if !strings.Contains(r.Description(), "comma-ok") {
		t.Errorf("Description() should mention comma-ok: %q", r.Description())
	}
}
