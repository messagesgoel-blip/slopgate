package rules

import (
	"strings"
	"testing"
)

func TestSLP030_FiresOnOnlyWithoutFilter(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/files.js b/api/files.js
--- a/api/files.js
+++ b/api/files.js
@@ -10,3 +10,4 @@
+  const file = await File.query().only();
`)
	got := SLP030{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP030_FiresOnFirstWithoutFilter(t *testing.T) {
	d := parseDiff(t, `diff --git a/models/user.js b/models/user.js
--- a/models/user.js
+++ b/models/user.js
@@ -5,2 +5,3 @@
+  const user = await User.find().first();
`)
	got := SLP030{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP030_FiresOnFindOnePython(t *testing.T) {
	d := parseDiff(t, `diff --git a/models.py b/models.py
--- a/models.py
+++ b/models.py
@@ -5,2 +5,3 @@
+  record = Record.objects.find_one()
`)
	got := SLP030{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for Python, got %d", len(got))
	}
}

func TestSLP030_IgnoresWithSentinelFilter(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/files.js b/api/files.js
--- a/api/files.js
+++ b/api/files.js
@@ -10,3 +10,4 @@
+  const file = await File.query().where('hash', '!=', 'folder-marker').first();
`)
	got := SLP030{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings with sentinel filter, got %d", len(got))
	}
}

func TestSLP030_IgnoresWithExclude(t *testing.T) {
	d := parseDiff(t, `diff --git a/models.py b/models.py
--- a/models.py
+++ b/models.py
@@ -5,2 +5,3 @@
+  record = Record.objects.exclude(hash='folder-marker').first()
`)
	got := SLP030{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings with exclude, got %d", len(got))
	}
}

func TestSLP030_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/models.test.js b/models.test.js
--- a/models.test.js
+++ b/models.test.js
@@ -1,2 +1,3 @@
+  const file = await File.query().only();
`)
	got := SLP030{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in test file, got %d", len(got))
	}
}

func TestSLP030_IgnoresNonQueryFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.yml b/config.yml
--- a/config.yml
+++ b/config.yml
@@ -1,2 +1,3 @@
+first: true
`)
	got := SLP030{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 for non-query file, got %d", len(got))
	}
}

func TestSLP030_Description(t *testing.T) {
	r := SLP030{}
	if r.ID() != "SLP030" {
		t.Errorf("ID = %q", r.ID())
	}
	if !strings.Contains(strings.ToLower(r.Description()), "sentinel") {
		t.Errorf("description should mention sentinel: %q", r.Description())
	}
}
