package rules

import "testing"

func TestSLP106_FiresOnOpenWithoutClose(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,1 +1,3 @@
+  f, err := os.Open("config.json")
`)
	got := SLP106{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for open without close")
	}
}

func TestSLP106_NoFireOnOpenWithClose(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,1 +1,5 @@
+  f, err := os.Open("config.json")
+  defer f.Close()
`)
	got := SLP106{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for open with close, got %d", len(got))
	}
}

func TestSLP106_IgnoresDocFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,1 +1,3 @@
+  f, err := os.Open("config.json")
`)
	got := SLP106{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for doc file, got %d", len(got))
	}
}

func TestSLP106_Description(t *testing.T) {
	r := SLP106{}
	if r.ID() != "SLP106" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}

func TestSLP106_FiresOnSecondOpenWithoutClose(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,1 +1,6 @@
+  f1, err := os.Open("file1.txt")
+  f2, err := os.Open("file2.txt")
+  if err != nil {
+    return err
+  }
+  defer f1.Close()
`)
	got := SLP106{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for f2 without close, got %d", len(got))
	}
	if got[0].Line != 2 {
		t.Errorf("expected finding on line 2 (f2), got line %d", got[0].Line)
	}
}
