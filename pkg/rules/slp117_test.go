package rules

import "testing"

func TestSLP117_FiresOnUnanchoredRegex(t *testing.T) {
	d := parseDiff(t, `diff --git a/validate.go b/validate.go
--- a/validate.go
+++ b/validate.go
@@ -1,1 +1,3 @@
 package main
+
+var re = regexp.MustCompile("\\d+")
`)
	got := SLP117{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for unanchored regex")
	}
}

func TestSLP117_NoFireOnAnchoredRegex(t *testing.T) {
	d := parseDiff(t, `diff --git a/validate.go b/validate.go
--- a/validate.go
+++ b/validate.go
@@ -1,1 +1,3 @@
 package main
+
+var re = regexp.MustCompile("^\\d+$")
`)
	got := SLP117{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for anchored regex, got %d", len(got))
	}
}

func TestSLP117_Description(t *testing.T) {
	r := SLP117{}
	if r.ID() != "SLP117" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
