package rules

import "testing"

func TestSLP119_FiresOnTrimSuffixUse(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var result = strings.TrimSuffix(name, ".txt")
`)
	got := SLP119{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for TrimSuffix without empty check")
	}
}

func TestSLP119_FiresOnTrimPrefixUse(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var result = strings.TrimPrefix(path, "/api/")
`)
	got := SLP119{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for TrimPrefix without empty check")
	}
}

func TestSLP119_NoFireOnNonTrimCode(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var result = name + ".bak"
`)
	got := SLP119{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for non-trim code, got %d", len(got))
	}
}

func TestSLP119_NoFireOnHasSuffixCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,4 @@
 package main
+
+if strings.HasSuffix(name, ".txt") {
+    var result = strings.TrimSuffix(name, ".txt")
+}
`)
	got := SLP119{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when HasSuffix guard present, got %d", len(got))
	}
}

func TestSLP119_NoFireOnHasSuffixOnAdjacentLine(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+if strings.HasSuffix(name, ".txt") {}
+var result = strings.TrimSuffix(name, ".txt")
`)
	got := SLP119{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when HasSuffix on adjacent line, got %d", len(got))
	}
}

func TestSLP119_NoFireOnEmptyStringCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var result = strings.TrimSuffix(name, ".txt"); if result == "" {}
`)
	got := SLP119{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when empty string check present, got %d", len(got))
	}
}

func TestSLP119_NoFireOnTrimLeft(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var result = strings.TrimLeft(name, " ")
`)
	got := SLP119{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for TrimLeft (not a suffix/prefix op), got %d", len(got))
	}
}

func TestSLP119_NoFireOnStartsWith(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,1 +1,3 @@
 const x = 1
+
+if (str.startsWith("/api/")) {}
`)
	got := SLP119{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for startsWith (a safety check itself), got %d", len(got))
	}
}

func TestSLP119_Description(t *testing.T) {
	r := SLP119{}
	if r.ID() != "SLP119" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
