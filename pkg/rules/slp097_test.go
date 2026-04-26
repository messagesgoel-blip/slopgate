package rules

import "testing"

func TestSLP097_FiresOnDataDestructureWithoutOkCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  const { data } = await api.get();
`)
	got := SLP097{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for data destructure")
	}
}

func TestSLP097_FiresOnFetchWithoutOkCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  fetch("/api/items").then(res => res.json())
`)
	got := SLP097{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for fetch without ok check")
	}
}

func TestSLP097_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.test.ts b/api.test.ts
--- a/api.test.ts
+++ b/api.test.ts
@@ -1,1 +1,3 @@
+  const { data } = await api.get();
`)
	got := SLP097{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for test file, got %d", len(got))
	}
}

func TestSLP097_IgnoresNonJSTS(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.go b/api.go
--- a/api.go
+++ b/api.go
@@ -1,1 +1,3 @@
+  const { data } = await api.get();  // nonsense in Go
`)
	got := SLP097{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for .go file, got %d", len(got))
	}
}

func TestSLP097_Description(t *testing.T) {
	r := SLP097{}
	if r.ID() != "SLP097" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
