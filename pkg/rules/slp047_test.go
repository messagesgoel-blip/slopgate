package rules

import (
	"strings"
	"testing"
)

func TestSLP047_FiresOnRestatingComment(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+// counter++
+ counter++
`)
	got := SLP047{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "a.go" {
		t.Errorf("file: %q", got[0].File)
	}
	if !strings.Contains(got[0].Message, "restates") {
		t.Errorf("message should mention restates: %q", got[0].Message)
	}
}

func TestSLP047_IgnoresDifferentComment(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+// increment counter to avoid off-by-one
+ counter++
`)
	got := SLP047{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP047_FiresWithTrailingBrace(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+// if x > 0 {
+ if x > 0 {
`)
	got := SLP047{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP047_IgnoresNonCodeFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,1 +1,2 @@
 # Project
+// hello world
+hello world
`)
	got := SLP047{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in markdown, got %d: %+v", len(got), got)
	}
}

func TestSLP047_Description(t *testing.T) {
	r := SLP047{}
	if r.ID() != "SLP047" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
