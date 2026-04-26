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
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding for data destructure")
	}
	if got[0].RuleID != "SLP097" || got[0].Severity != SeverityWarn {
		t.Errorf("unexpected finding metadata: RuleID=%q Severity=%v", got[0].RuleID, got[0].Severity)
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
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding for fetch without ok check")
	}
	if got[0].RuleID != "SLP097" || got[0].Severity != SeverityWarn {
		t.Errorf("unexpected finding metadata: RuleID=%q Severity=%v", got[0].RuleID, got[0].Severity)
	}
}

func TestSLP097_IgnoresMismatchedJsonReceiver(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  fetch("/api/items").then(res => response.json())
`)
	got := SLP097{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for mismatched callback receiver, got %d", len(got))
	}
}

func TestSLP097_IgnoresPrefetchHelper(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  prefetch("/api/items").then(res => res.json())
`)
	got := SLP097{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for prefetch helper, got %d", len(got))
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

func TestSLP097_FiresOnFetchWithTypedParam(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  fetch("/api/items").then((res: Response) => res.json())
`)
	got := SLP097{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding for fetch with typed parameter without ok check")
	}
	if got[0].RuleID != "SLP097" || got[0].Severity != SeverityWarn {
		t.Errorf("unexpected finding metadata: RuleID=%q Severity=%v", got[0].RuleID, got[0].Severity)
	}
}

func TestSLP097_FiresOnNestedFetch(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  fetch("/a").then(res => fetch("/b").then(r => r.json()))
`)
	got := SLP097{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding for nested fetch without ok check")
	}
	if got[0].RuleID != "SLP097" || got[0].Severity != SeverityWarn {
		t.Errorf("unexpected finding metadata: RuleID=%q Severity=%v", got[0].RuleID, got[0].Severity)
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
