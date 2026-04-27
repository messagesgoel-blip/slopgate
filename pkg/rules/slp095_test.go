package rules

import "testing"

func TestSLP095_FiresOnCatchReturnNull(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.js b/handler.js
--- a/handler.js
+++ b/handler.js
@@ -1,1 +1,5 @@
+  try { doThing(); } catch (e) { return null; }
`)
	got := SLP095{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for catch return null")
	}
}

func TestSLP095_NoFireOnCatchWithThrow(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.js b/handler.js
--- a/handler.js
+++ b/handler.js
@@ -1,1 +1,5 @@
+  try { doThing(); } catch (e) { console.error(e); throw e; }
`)
	got := SLP095{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for catch with throw, got %d", len(got))
	}
}

func TestSLP095_FiresOnExceptReturnNone(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,1 +1,5 @@
+  try:
+      do_thing()
+  except Exception:
+      return None
`)
	got := SLP095{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for except return None")
	}
}

func TestSLP095_NoFireOnExceptWithRaiseAndOuterReturn(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,1 +1,6 @@
+  try:
+      do_thing()
+  except Exception:
+      raise
+  return None
`)
	got := SLP095{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for except with raise and outer return, got %d", len(got))
	}
}

func TestSLP095_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.test.js b/handler.test.js
--- a/handler.test.js
+++ b/handler.test.js
@@ -1,1 +1,5 @@
+  try { doThing(); } catch (e) { return null; }
`)
	got := SLP095{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for test file, got %d", len(got))
	}
}

func TestSLP095_Description(t *testing.T) {
	r := SLP095{}
	if r.ID() != "SLP095" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
