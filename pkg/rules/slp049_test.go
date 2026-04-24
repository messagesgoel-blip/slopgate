package rules

import (
	"testing"
)

func TestSLP049_FiresOnVacuousAssertion(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,3 @@
 package foo
+
+assert.Equal(t, input, input)
`)
	got := SLP049{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "foo_test.go" {
		t.Errorf("file: %q", got[0].File)
	}
}

func TestSLP049_FiresOnTError(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,3 @@
 package foo
+
+if input == input { t.Error("same") }
`)
	got := SLP049{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP049_IgnoresNonTestFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,2 @@
 package foo
+assert.Equal(t, input, input)
`)
	got := SLP049{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in non-test file, got %d: %+v", len(got), got)
	}
}

func TestSLP049_IgnoresDifferentAssertion(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,2 @@
 package foo
+assert.Equal(t, 42, result)
`)
	got := SLP049{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP049_Description(t *testing.T) {
	r := SLP049{}
	if r.ID() != "SLP049" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
