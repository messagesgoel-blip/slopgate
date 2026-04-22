package rules

import (
	"strings"
	"testing"
)

func TestSLP051_FiresOnUndefinedCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,4 @@
 package a

+func Run() {
+	undefinedHelper()
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "undefinedHelper") {
		t.Errorf("expected message to mention undefinedHelper, got %q", got[0].Message)
	}
}

func TestSLP051_IgnoresDefinedCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,8 @@
 package a

+func helper() {}
+
+func Run() {
+	helper()
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresBuiltins(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func Run() {
+	make([]int, 10)
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for builtins, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresMethodCalls(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func Run() {
+	srv.DoThing()
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for method calls, got %d: %+v", len(got), got)
	}
}

func TestSLP051_Description(t *testing.T) {
	r := SLP051{}
	if r.ID() != "SLP051" {
		t.Errorf("ID = %q", r.ID())
	}
}
