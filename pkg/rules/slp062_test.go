package rules

import (
	"strings"
	"testing"
)

func TestSLP062_FiresOnFunctionOver50Lines(t *testing.T) {
	// Build a diff with a 52-line function.
	var sb strings.Builder
	sb.WriteString(`diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,54 @@
 package foo
`)
	sb.WriteString("+func BigFunc() {\n")
	for i := 0; i < 50; i++ {
		sb.WriteString("+\t_ = 1\n")
	}
	sb.WriteString("+}\n")
	d := parseDiff(t, sb.String())
	got := SLP062{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "BigFunc") {
		t.Errorf("message should mention BigFunc: %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "52") {
		t.Errorf("message should mention line count 52: %q", got[0].Message)
	}
}

func TestSLP062_FiresOnFileOver500Lines(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,503 @@
 package foo
`)
	for i := 0; i < 501; i++ {
		sb.WriteString("+var x = 1\n")
	}
	d := parseDiff(t, sb.String())
	got := SLP062{}.Check(d)
	// Should get a file-level finding.
	var foundFile bool
	for _, f := range got {
		if f.Line == 0 && strings.Contains(f.Message, "file adds") {
			foundFile = true
			break
		}
	}
	if !foundFile {
		t.Fatalf("expected file-level finding, got %+v", got)
	}
}

func TestSLP062_NoFireForSmallFunction(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,5 @@
 package foo
+func Small() {
+	_ = 1
+	_ = 2
+}
`)
	got := SLP062{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP062_NoFireForSmallFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,5 @@
 package foo
+func Small() {
+	_ = 1
+}
`)
	got := SLP062{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP062_NoFireForNonGo(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`diff --git a/foo.js b/foo.js
--- a/foo.js
+++ b/foo.js
@@ -1,1 +1,55 @@
 function big() {
`)
	for i := 0; i < 52; i++ {
		sb.WriteString("+  var x = 1;\n")
	}
	sb.WriteString("+}\n")
	d := parseDiff(t, sb.String())
	got := SLP062{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP062_FiresOnMultilineSignatureOver50Lines(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,56 @@
 package foo
+func Big(
+	arg int,
+) int {
`)
	for i := 0; i < 49; i++ {
		sb.WriteString("+\t_ = arg\n")
	}
	sb.WriteString("+\treturn arg\n")
	sb.WriteString("+}\n")
	d := parseDiff(t, sb.String())
	got := SLP062{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for multiline signature, got %d: %+v", len(got), got)
	}
}

func TestSLP062_NoFireAcrossDisjointAddedRegions(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,4 @@
 package foo
+func Big() {
+	_ = 1
+	_ = 2
@@ -20,1 +23,52 @@
`)
	for i := 0; i < 51; i++ {
		sb.WriteString("+\t_ = 3\n")
	}
	sb.WriteString("+}\n")
	d := parseDiff(t, sb.String())
	got := SLP062{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings across disjoint added regions, got %d: %+v", len(got), got)
	}
}

func TestSLP062_Description(t *testing.T) {
	r := SLP062{}
	if r.ID() != "SLP062" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
