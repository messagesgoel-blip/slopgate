package rules

import (
	"strings"
	"testing"
)

func TestSLP053_FiresOnConfigWithoutComment(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.yaml b/config.yaml
--- a/config.yaml
+++ b/config.yaml
@@ -1,2 +1,3 @@
 app:
+  timeout: 3000
   name: demo
`)
	got := SLP053{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "timeout") {
		t.Errorf("expected message about timeout, got %q", got[0].Message)
	}
}

func TestSLP053_IgnoresWithComment(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.yaml b/config.yaml
--- a/config.yaml
+++ b/config.yaml
@@ -1,2 +1,4 @@
 app:
+  # 3 second timeout chosen empirically
+  timeout: 3000
   name: demo
`)
	got := SLP053{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings with comment, got %d: %+v", len(got), got)
	}
}

func TestSLP053_FiresInGoFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main

+const timeout = 3000
`)
	got := SLP053{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding in .go file, got %d: %+v", len(got), got)
	}
}

func TestSLP053_FiresWhenCommentSeparatedByContextLine(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.yaml b/config.yaml
--- a/config.yaml
+++ b/config.yaml
@@ -1,3 +1,5 @@
 app:
+  # 3 second timeout chosen empirically
   name: demo
+  timeout: 3000
`)
	got := SLP053{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (comment should not carry across context line), got %d: %+v", len(got), got)
	}
}

func TestSLP053_Description(t *testing.T) {
	r := SLP053{}
	if r.ID() != "SLP053" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("DefaultSeverity = %v, want SeverityInfo", r.DefaultSeverity())
	}
}
