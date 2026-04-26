package rules

import (
	"strings"
	"testing"
)

func TestSLP026_FiresOnHashNotNull(t *testing.T) {
	d := parseDiff(t, `diff --git a/query.sql b/query.sql
--- a/query.sql
+++ b/query.sql
@@ -1,2 +1,3 @@
+SELECT * FROM files WHERE hash IS NOT NULL
`)
	got := SLP026{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP026_IgnoresWithSentinel(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -10,3 +10,4 @@
+  "SELECT * FROM files WHERE hash IS NOT NULL AND hash != 'folder-marker'"
`)
	got := SLP026{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings with sentinel, got %d", len(got))
	}
}

func TestSLP026_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/query.test.js b/query.test.js
--- a/query.test.js
+++ b/query.test.js
@@ -1,2 +1,3 @@
+  "SELECT * WHERE hash IS NOT NULL"
`)
	got := SLP026{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in test file, got %d", len(got))
	}
}

func TestSLP026_Description(t *testing.T) {
	r := SLP026{}
	if r.ID() != "SLP026" {
		t.Errorf("ID = %q", r.ID())
	}
	if !strings.Contains(strings.ToLower(r.Description()), "sentinel") {
		t.Errorf("description should mention sentinel: %q", r.Description())
	}
}
