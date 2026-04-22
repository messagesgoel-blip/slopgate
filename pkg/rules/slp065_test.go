package rules

import (
	"strings"
	"testing"
)

func TestSLP065_FiresOnBlankAssignment(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,4 @@
 package foo
+func Bar() {
+	_ = doSomething()
+}
`)
	got := SLP065{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "error return ignored") {
		t.Errorf("message should mention ignored: %q", got[0].Message)
	}
}

func TestSLP065_FiresOnBlankTupleAssignment(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,4 @@
 package foo
+func Bar() {
+	_, _ = doSomething()
+}
`)
	got := SLP065{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP065_FiresOnErrNotChecked(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,5 @@
 package foo
+func Bar() {
+	err := doSomething()
+	_ = err
+}
`)
	got := SLP065{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP065_NoFireWhenErrChecked(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,6 @@
 package foo
+func Bar() {
+	err := doSomething()
+	if err != nil {
+		return
+	}
+}
`)
	got := SLP065{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP065_NoFireForNonGo(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.js b/foo.js
--- a/foo.js
+++ b/foo.js
@@ -1,1 +1,3 @@
 function bar() {
+  _ = doSomething();
 }
`)
	got := SLP065{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP065_Description(t *testing.T) {
	r := SLP065{}
	if r.ID() != "SLP065" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
