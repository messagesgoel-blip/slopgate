package rules

import "testing"

func TestSLP108_FiresOnOpenWithoutDefer(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,1 +1,3 @@
+  db, err := sql.Open("postgres", connStr)
`)
	got := SLP108{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for open without defer")
	}
}

func TestSLP108_FiresOnFetchWithoutTimeout(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  const res = fetch("https://api.example.com/data")
`)
	got := SLP108{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for fetch without timeout")
	}
}

func TestSLP108_NoFireOnOpenWithDefer(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,1 +1,5 @@
+  db, err := sql.Open("postgres", connStr)
+  defer db.Close()
`)
	got := SLP108{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for open with defer, got %d", len(got))
	}
}

func TestSLP108_Description(t *testing.T) {
	r := SLP108{}
	if r.ID() != "SLP108" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
