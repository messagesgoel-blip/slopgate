package rules

import "testing"

func TestSLP116_FiresOnNestedQuantifier(t *testing.T) {
	d := parseDiff(t, `diff --git a/validate.go b/validate.go
--- a/validate.go
+++ b/validate.go
@@ -1,1 +1,3 @@
 package main
+
+var re = regexp.MustCompile("(.)*+")
`)
	got := SLP116{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for nested quantifier regex")
	}
}

func TestSLP116_NoFireOnSafeRegex(t *testing.T) {
	d := parseDiff(t, `diff --git a/validate.go b/validate.go
--- a/validate.go
+++ b/validate.go
@@ -1,1 +1,3 @@
 package main
+
+var re = regexp.MustCompile("^[a-z]+$")
`)
	got := SLP116{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for safe regex, got %d", len(got))
	}
}

func TestSLP116_Description(t *testing.T) {
	r := SLP116{}
	if r.ID() != "SLP116" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
