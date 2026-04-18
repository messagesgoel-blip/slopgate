package rules

import (
	"strings"
	"testing"
)

func TestSLP027_FiresOnAsyncThrow(t *testing.T) {
	d := parseDiff(t, `diff --git a/lib.js b/lib.js
--- a/lib.js
+++ b/lib.js
@@ -5,3 +5,4 @@
+async function emit() { if (!valid) throw new Error("bad"); return save(); }
`)
	got := SLP027{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP027_FiresOnPromiseReturnThrow(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -10,3 +10,4 @@
+function emit(): Promise<void> { if (!ok) throw new Error("fail"); return Promise.resolve(); }
`)
	got := SLP027{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP027_IgnoresSyncFunction(t *testing.T) {
	d := parseDiff(t, `diff --git a/validate.js b/validate.js
--- a/validate.js
+++ b/validate.js
@@ -3,3 +3,4 @@
+function validate() { if (!input) throw new Error("missing"); return true; }
`)
	got := SLP027{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 for sync function, got %d", len(got))
	}
}

func TestSLP027_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/lib.test.js b/lib.test.js
--- a/lib.test.js
+++ b/lib.test.js
@@ -1,2 +1,3 @@
+async function bad() { throw new Error("test"); }
`)
	got := SLP027{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 in test file, got %d", len(got))
	}
}

func TestSLP027_Description(t *testing.T) {
	r := SLP027{}
	if r.ID() != "SLP027" {
		t.Errorf("ID = %q", r.ID())
	}
	if !strings.Contains(strings.ToLower(r.Description()), "async") {
		t.Errorf("description should mention async: %q", r.Description())
	}
}