package rules

import (
	"strings"
	"testing"
)

func TestSLP058_FiresOnSQLConcat(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,2 +1,3 @@
 package db
+
+query := "SELECT * FROM users WHERE id = " + userID
`)
	got := SLP058{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "SQL built with string concatenation") {
		t.Errorf("message: %q", got[0].Message)
	}
}

func TestSLP058_FiresOnFmtSprintf(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,2 +1,3 @@
 package db
+
+query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", id)
`)
	got := SLP058{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP058_FiresOnInterpolation(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.js b/db.js
--- a/db.js
+++ b/db.js
@@ -1,1 +1,2 @@
 const x = 1
+
+const q = "SELECT * FROM users WHERE id = ${userId}"
`)
	got := SLP058{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP058_IgnoresPythonDBPlaceholders(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.py b/db.py
--- a/db.py
+++ b/db.py
@@ -1,2 +1,3 @@
 def get_user(id):
+
     cursor.execute("SELECT * FROM users WHERE id = %s", id)
`)
	got := SLP058{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for Python DB placeholder, got %d: %+v", len(got), got)
	}
}

func TestSLP058_IgnoresPlainSQL(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,2 +1,3 @@
 package db
+
+query := "SELECT * FROM users"
`)
	got := SLP058{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for plain SQL, got %d: %+v", len(got), got)
	}
}

func TestSLP058_Description(t *testing.T) {
	r := SLP058{}
	if r.ID() != "SLP058" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
