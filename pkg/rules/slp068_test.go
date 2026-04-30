package rules

import (
	"testing"
)

func TestSLP068_FiresOnDuplicateBlock(t *testing.T) {
	d := parseDiff(t, `diff --git a/utils.go b/utils.go
--- a/utils.go
+++ b/utils.go
@@ -1,1 +1,11 @@
 package utils
+a := 1
+b := 2
+c := 3
+d := 4
+e := 5
+a := 1
+b := 2
+c := 3
+d := 4
+e := 5
`)
	got := SLP068{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Line != 7 {
		t.Errorf("line: %d, want 7", got[0].Line)
	}
}

func TestSLP068_NoFireOnShortDuplicate(t *testing.T) {
	d := parseDiff(t, `diff --git a/utils.go b/utils.go
--- a/utils.go
+++ b/utils.go
@@ -1,1 +1,7 @@
 package utils
+a := 1
+b := 2
+c := 3
+a := 1
+b := 2
+c := 3
`)
	got := SLP068{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP068_IgnoresDuplicateDocsBlock(t *testing.T) {
	d := parseDiff(t, `diff --git a/docs/plan.md b/docs/plan.md
--- a/docs/plan.md
+++ b/docs/plan.md
@@ -1,1 +1,12 @@
 # Plan
+This paragraph is intentionally repeated in a long design note.
+It describes a migration sequence and acceptance criteria.
+It is prose, not duplicated production logic.
+It may exceed several lines in a Markdown document.
+It should not be surfaced as a code clone.
+This paragraph is intentionally repeated in a long design note.
+It describes a migration sequence and acceptance criteria.
+It is prose, not duplicated production logic.
+It may exceed several lines in a Markdown document.
+It should not be surfaced as a code clone.
`)
	got := SLP068{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for duplicated docs prose, got %d: %+v", len(got), got)
	}
}

func TestSLP068_Meta(t *testing.T) {
	r := SLP068{}
	if r.ID() != "SLP068" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
