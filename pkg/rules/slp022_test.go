package rules

import (
	"strings"
	"testing"
)

func TestSLP022_ErrorWrappingWithoutW(t *testing.T) {
	tests := []struct {
		name    string
		diff    string
		want    int
		wantMsg string
	}{
		{
			name: "percent-v with err arg",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 func foo() error {
-	return nil
+	err := doThing()
+	return fmt.Errorf("doing thing: %v", err)
 }`,
			want:    1,
			wantMsg: "%v",
		},
		{
			name: "percent-s with err arg",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo() error {
-	return nil
+	return fmt.Errorf("failed: %s", myErr)
 }`,
			want:    1,
			wantMsg: "%s",
		},
		{
			name: "percent-w is correct no finding",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo() error {
-	return nil
+	return fmt.Errorf("doing thing: %w", err)
 }`,
			want: 0,
		},
		{
			name: "errors.Wrap not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo() error {
-	return nil
+	return errors.Wrap(err, "doing thing")
 }`,
			want: 0,
		},
		{
			name: "no err arg not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo() error {
-	return nil
+	return fmt.Errorf("invalid count: %v", count)
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
+	_ = fmt.Errorf("failed: %v", err)
 }`,
			want: 0,
		},
		{
			name: "percent-v with returnErr arg",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo() error {
-	return nil
+	return fmt.Errorf("bad: %v", returnErr)
 }`,
			want:    1,
			wantMsg: "%v",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP022{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d", len(got), tt.want)
			}
			if tt.want > 0 && tt.wantMsg != "" {
				if !strings.Contains(got[0].Message, tt.wantMsg) {
					t.Errorf("message = %q, want to contain %q", got[0].Message, tt.wantMsg)
				}
			}
		})
	}
}

func TestSLP022_IDAndDescription(t *testing.T) {
	var r SLP022
	if r.ID() != "SLP022" {
		t.Errorf("ID() = %q, want SLP022", r.ID())
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("DefaultSeverity() = %v, want warn", r.DefaultSeverity())
	}
	if r.Description() == "" {
		t.Error("Description() is empty")
	}
}
