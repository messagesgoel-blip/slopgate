package rules

import (
	"strings"
	"testing"
)

func TestSLP060_NoInterfaceNoFinding(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+type Server struct{}
`)
	got := SLP060{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP060_OneInterfaceZeroStructs(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+type Reader interface {
`)
	got := SLP060{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "Reader") {
		t.Errorf("message should contain interface name: %q", got[0].Message)
	}
}

func TestSLP060_OneInterfaceOneStruct(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,5 @@
 package main
+
+type Reader interface {}
+
+type fileReader struct{}
`)
	got := SLP060{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP060_OneInterfaceTwoStructs(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,6 @@
 package main
+
+type Reader interface {}
+
+type fileReader struct{}
+type netReader struct{}
`)
	got := SLP060{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP060_TwoInterfacesOneStruct(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,6 @@
 package main
+
+type Reader interface {}
+type Writer interface {}
+
+type impl struct{}
`)
	got := SLP060{}.Check(d)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP060_IgnoresNonGoFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,2 +1,3 @@
 def main():
+
+class MyInterface:
`)
	got := SLP060{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in non-Go file, got %d: %+v", len(got), got)
	}
}

func TestSLP060_Description(t *testing.T) {
	r := SLP060{}
	if r.ID() != "SLP060" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
