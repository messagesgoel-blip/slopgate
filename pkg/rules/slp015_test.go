package rules

import (
	"strings"
	"testing"
)

func TestSLP015_GoNolint(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+//nolint:revive
+func Foo() {}
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for //nolint, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "nolint") {
		t.Errorf("message should mention nolint: %q", got[0].Message)
	}
}

func TestSLP015_GoLintIgnore(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+//lint:ignore U1000
+func Bar() {}
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for //lint:ignore, got %d: %+v", len(got), got)
	}
}

func TestSLP015_TsIgnore(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.ts b/a.ts
--- a/a.ts
+++ b/a.ts
@@ -1,1 +1,3 @@
 export {}
+// @ts-ignore
+const x = a.b
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for @ts-ignore, got %d: %+v", len(got), got)
	}
}

func TestSLP015_TsNoCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.ts b/a.ts
--- a/a.ts
+++ b/a.ts
@@ -1,1 +1,3 @@
 export {}
+// @ts-nocheck
+const x = 1
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for @ts-nocheck, got %d: %+v", len(got), got)
	}
}

func TestSLP015_EslintDisable(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.js b/a.js
--- a/a.js
+++ b/a.js
@@ -1,1 +1,3 @@
 const a = 1;
+// eslint-disable-next-line no-console
+console.log(a);
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for eslint-disable, got %d: %+v", len(got), got)
	}
}

func TestSLP015_PythonNoqa(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.py b/a.py
--- a/a.py
+++ b/a.py
@@ -1,1 +1,3 @@
 import os
+x = os.environ["KEY"]  # noqa: E501
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for # noqa, got %d: %+v", len(got), got)
	}
}

func TestSLP015_PythonTypeIgnore(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.py b/a.py
--- a/a.py
+++ b/a.py
@@ -1,1 +1,3 @@
 import os
+x = os.environ["KEY"]  # type: ignore
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for type: ignore, got %d: %+v", len(got), got)
	}
}

func TestSLP015_PythonPylintDisable(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.py b/a.py
--- a/a.py
+++ b/a.py
@@ -1,1 +1,3 @@
 import os
+# pylint: disable=bare-except
+try:
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for pylint: disable, got %d: %+v", len(got), got)
	}
}

func TestSLP015_JavaSuppressWarnings(t *testing.T) {
	d := parseDiff(t, `diff --git a/Svc.java b/Svc.java
--- a/Svc.java
+++ b/Svc.java
@@ -1,1 +1,3 @@
 package svc;
+@SuppressWarnings("unchecked")
+public List getList() { return null; }
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for @SuppressWarnings, got %d: %+v", len(got), got)
	}
}

func TestSLP015_JavaNopmd(t *testing.T) {
	d := parseDiff(t, `diff --git a/Svc.java b/Svc.java
--- a/Svc.java
+++ b/Svc.java
@@ -1,1 +1,3 @@
 package svc;
+// NOPMD
+int x = 1;
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for NOPMD, got %d: %+v", len(got), got)
	}
}

func TestSLP015_RustAllow(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.rs b/main.rs
--- a/main.rs
+++ b/main.rs
@@ -1,1 +1,3 @@
 fn main() {}
+#[allow(dead_code)]
+fn unused() {}
`)
	got := SLP015{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for #[allow], got %d: %+v", len(got), got)
	}
}

func TestSLP015_IgnoresDocFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,1 +1,3 @@
 # Project
+<!-- eslint-disable no-console -->
+Content here
`)
	got := SLP015{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in markdown file, got %d: %+v", len(got), got)
	}
}

func TestSLP015_IgnoresNolintInString(t *testing.T) {
	// //nolint inside a string literal should NOT fire.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+msg := "add //nolint to suppress"
+func Foo() {}
`)
	got := SLP015{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for nolint in string, got %d: %+v", len(got), got)
	}
}

func TestSLP015_Description(t *testing.T) {
	r := SLP015{}
	if r.ID() != "SLP015" {
		t.Errorf("ID = %q, want SLP015", r.ID())
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
	if !strings.Contains(strings.ToLower(r.Description()), "linter") {
		t.Errorf("description should mention linter: %q", r.Description())
	}
}
