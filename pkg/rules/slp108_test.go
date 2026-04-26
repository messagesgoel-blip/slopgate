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
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
	if got[0].RuleID != "SLP108" {
		t.Errorf("RuleID = %q", got[0].RuleID)
	}
	if got[0].Severity != SeverityBlock {
		t.Errorf("Severity = %v", got[0].Severity)
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
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
	if got[0].RuleID != "SLP108" {
		t.Errorf("RuleID = %q", got[0].RuleID)
	}
	if got[0].Severity != SeverityBlock {
		t.Errorf("Severity = %v", got[0].Severity)
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

func TestSLP108_NoFireOnOpenWithTimeout(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,1 +1,5 @@
+  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
+  db, err := sql.Open("postgres", connStr)
`)
	got := SLP108{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for open with timeout, got %d", len(got))
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
