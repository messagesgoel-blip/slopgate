package rules

import (
	"strings"
	"testing"
)

func TestSLP067_FiresOnHTTPGetWithoutClose(t *testing.T) {
	d := parseDiff(t, `diff --git a/client.go b/client.go
--- a/client.go
+++ b/client.go
@@ -1,1 +1,4 @@
 package client
+
+func Fetch() {
+	resp, _ := http.Get("http://example.com")
+}
`)
	got := SLP067{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "client.go" {
		t.Errorf("file: %q", got[0].File)
	}
	if !strings.Contains(got[0].Message, "without deferred Close()") {
		t.Errorf("message: %q", got[0].Message)
	}
}

func TestSLP067_NoFireWithDeferClose(t *testing.T) {
	d := parseDiff(t, `diff --git a/client.go b/client.go
--- a/client.go
+++ b/client.go
@@ -1,1 +1,5 @@
 package client
+
+func Fetch() {
+	resp, _ := http.Get("http://example.com")
+	defer resp.Body.Close()
+}
`)
	got := SLP067{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP067_FiresOnDBQueryWithoutClose(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,1 +1,4 @@
 package db
+
+func List() {
+	rows, _ := db.Query("SELECT * FROM items")
+}
`)
	got := SLP067{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP067_NoFireWithExplicitClose(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,1 +1,5 @@
 package db
+
+func List() {
+	rows, _ := db.Query("SELECT * FROM items")
+	rows.Close()
+}
`)
	got := SLP067{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP067_Meta(t *testing.T) {
	r := SLP067{}
	if r.ID() != "SLP067" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
