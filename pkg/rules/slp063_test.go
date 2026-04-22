package rules

import (
	"strings"
	"testing"
)

func TestSLP063_FiresOnStructWith16Fields(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,19 @@
 package foo
+type BigStruct struct {
`)
	for i := 0; i < 16; i++ {
		sb.WriteString("+\tField" + string(rune('A'+i)) + " string\n")
	}
	sb.WriteString("+}\n")
	d := parseDiff(t, sb.String())
	got := SLP063{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "BigStruct") {
		t.Errorf("message should mention BigStruct: %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "16") {
		t.Errorf("message should mention 16 fields: %q", got[0].Message)
	}
}

func TestSLP063_NoFireFor15Fields(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,18 @@
 package foo
+type MediumStruct struct {
`)
	for i := 0; i < 15; i++ {
		sb.WriteString("+\tField" + string(rune('A'+i)) + " string\n")
	}
	sb.WriteString("+}\n")
	d := parseDiff(t, sb.String())
	got := SLP063{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP063_NoFireForNonGo(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.java b/foo.java
--- a/foo.java
+++ b/foo.java
@@ -1,1 +1,5 @@
 class Foo {
+    private String a;
+    private String b;
+    private String c;
 }
`)
	got := SLP063{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP063_Description(t *testing.T) {
	r := SLP063{}
	if r.ID() != "SLP063" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
