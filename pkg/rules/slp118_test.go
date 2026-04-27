package rules

import "testing"

func TestSLP118_FiresOnDirectIndexAccess(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var first = items[0]
`)
	got := SLP118{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for index access without guard")
	}
}

func TestSLP118_FiresOnDirectIndexAccessBeyondOne(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var third = items[2]
`)
	got := SLP118{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for index [2] access without guard")
	}
}

func TestSLP118_NoFireOnSlicing(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var subset = items[1:3]
`)
	got := SLP118{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for slicing, got %d", len(got))
	}
}

func TestSLP118_NoFireOnGuardedAccess(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,4 @@
 package main
+
+if len(items) > 0 {
+    var first = items[0]
+}
`)
	got := SLP118{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for guarded access, got %d", len(got))
	}
}

func TestSLP118_NoFireOnJSGuardedAccess(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.ts b/app.ts
--- a/app.ts
+++ b/app.ts
@@ -1,1 +1,4 @@
 const x = 1
+
+if (items.length > 0) {
+    const first = items[0]
+}
`)
	got := SLP118{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for JS guarded access, got %d", len(got))
	}
}

func TestSLP118_NoFireOnPyGuardedAccess(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.py b/app.py
--- a/app.py
+++ b/app.py
@@ -1,1 +1,4 @@
 x = 1
+
+if len(items) > 0:
+    first = items[0]
`)
	got := SLP118{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for Python guarded access, got %d", len(got))
	}
}

func TestSLP118_PrevContentPreservedAcrossContextLines(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,5 @@
 package main
+
+if len(items) > 0 {
+    // safe access
+    var first = items[0]
+}
`)
	got := SLP118{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when guard is present, got %d", len(got))
	}
}

func TestSLP118_Description(t *testing.T) {
	r := SLP118{}
	if r.ID() != "SLP118" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}