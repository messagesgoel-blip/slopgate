package rules

import "testing"

func assertSLP108Fires(t *testing.T, got []Finding) {
	t.Helper()
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

func TestSLP108_FiresOnOpenWithoutDefer(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,1 +1,1 @@
+  db, err := sql.Open("postgres", connStr)
`)
	got := SLP108{}.Check(d)
	assertSLP108Fires(t, got)
	if got[0].Message != "open/connect without defer close — add resource lifecycle management" {
		t.Errorf("Message = %q", got[0].Message)
	}
}

func TestSLP108_FiresOnFetchWithoutTimeout(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,1 @@
+  const res = fetch("https://api.example.com/data")
`)
	got := SLP108{}.Check(d)
	assertSLP108Fires(t, got)
	if got[0].Message != "fetch without timeout — add a timeout or AbortController to prevent hanging" {
		t.Errorf("Message = %q", got[0].Message)
	}
}

func TestSLP108_NoFireOnOpenWithDefer(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,1 +1,2 @@
+  db, err := sql.Open("postgres", connStr)
+  defer db.Close()
`)
	got := SLP108{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for open with defer, got %d", len(got))
	}
}

func TestSLP108_NoFireOnFetchWithTimeout(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,1 @@
+  const res = fetch("url", { signal: AbortSignal.timeout(5000) });
`)
	got := SLP108{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for fetch with timeout, got %d", len(got))
	}
}

func TestSLP108_FiresOnOpenWithTimeoutNoDefer(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,1 +1,2 @@
+  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
+  db, err := sql.Open("postgres", connStr)
`)
	assertSLP108Fires(t, SLP108{}.Check(d))
}

func TestSLP108_FiresOnOpenWithDeferCancelOnly(t *testing.T) {
	d := parseDiff(t, `diff --git a/db.go b/db.go
--- a/db.go
+++ b/db.go
@@ -1,1 +1,3 @@
+  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
+  db, err := sql.Open("postgres", connStr)
+  defer cancel()
`)
	assertSLP108Fires(t, SLP108{}.Check(d))
}

func TestSLP108_IgnoresGenericNewClient(t *testing.T) {
	d := parseDiff(t, `diff --git a/client.go b/client.go
--- a/client.go
+++ b/client.go
@@ -1,1 +1,1 @@
+  client := NewClient(cfg)
`)
	got := SLP108{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for generic NewClient, got %d", len(got))
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
