package rules

import (
	"strings"
	"testing"
)

func TestSLP059_FiresOnVariableArg(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+cmd := exec.Command("sh", "-c", userInput)
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "exec.Command argument may contain user input") {
		t.Errorf("message: %q", got[0].Message)
	}
}

func TestSLP059_FiresOnFmtSprintfArg(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+cmd := exec.Command(fmt.Sprintf("echo %s", name))
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP059_FiresOnConcatArg(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+cmd := exec.Command("echo " + msg)
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP059_IgnoresHardcodedArgs(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+cmd := exec.Command("echo", "hello")
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for hardcoded args, got %d: %+v", len(got), got)
	}
}

func TestSLP059_IgnoresNonGoFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/script.py b/script.py
--- a/script.py
+++ b/script.py
@@ -1,2 +1,3 @@
 def run():
+
+    os.execvp("echo", ["hello"])
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in non-Go file, got %d: %+v", len(got), got)
	}
}

func TestSLP059_Description(t *testing.T) {
	r := SLP059{}
	if r.ID() != "SLP059" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
