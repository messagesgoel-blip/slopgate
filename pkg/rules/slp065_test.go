package rules

import (
	"testing"
)

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

func TestSLP065_IgnoresExplicitSuppression(t *testing.T) {
	// Explicit `_ = doSomething()` is a deliberate acknowledged suppression — skip it.
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
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for explicit suppression, got %d: %+v", len(got), got)
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

func TestSLP065_NoFireForNamedErrOnLHS(t *testing.T) {
	// `_, err := doSomething()` — err is named on LHS, so it's handled.
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,4 @@
 package foo
+func Bar() {
+	_, err := doSomething()
+}
`)
	got := SLP065{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for named err on LHS, got %d: %+v", len(got), got)
	}
}

func TestSLP065_NoFireForInlineErrInit(t *testing.T) {
	// `if err := doSomething(); err != nil {` — inline if-init is handled.
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,3 @@
 package foo
+if err := doSomething(); err != nil {
+	return
+}
`)
	got := SLP065{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for inline if-init, got %d: %+v", len(got), got)
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
