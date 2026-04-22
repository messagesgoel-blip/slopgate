package rules

import (
	"strings"
	"testing"
)

func TestSLP057_FiresOnEval(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,1 +1,2 @@
 const x = 1
+
+eval(userInput)
`)
	got := SLP057{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "eval(") {
		t.Errorf("message should mention eval(: %q", got[0].Message)
	}
}

func TestSLP057_FiresOnNewFunction(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,1 +1,2 @@
 const x = 1
+
+const f = new Function("x", "return x")
`)
	got := SLP057{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP057_FiresOnPythonExec(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,2 +1,3 @@
 def main():
+
+    exec(cmd)
`)
	got := SLP057{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "exec(") {
		t.Errorf("message: %q", got[0].Message)
	}
}

func TestSLP057_FiresOnPythonImport(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,2 +1,3 @@
 def main():
+
+    mod = __import__(name)
`)
	got := SLP057{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP057_FiresOnGoReflectCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+reflect.Value.Call(vals)
`)
	got := SLP057{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP057_FiresOnUnsafe(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+import "unsafe"
`)
	got := SLP057{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP057_IgnoresExecInGo(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+exec.Command("echo", "hi")
`)
	got := SLP057{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for exec in Go, got %d: %+v", len(got), got)
	}
}

func TestSLP057_Description(t *testing.T) {
	r := SLP057{}
	if r.ID() != "SLP057" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
