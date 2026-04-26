package rules

import "testing"

func TestSLP102_FiresOnAsyncFunctionWithoutAwait(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,5 @@
+async function getItems() {
+    return [];
+}
`)
	got := SLP102{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for async without await")
	}
}

func TestSLP102_NoFireOnAsyncWithAwait(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,5 @@
+async function getItems() {
+    const items = await fetch("/api/items");
+    return items.json();
+}
`)
	got := SLP102{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for async with await, got %d", len(got))
	}
}

func TestSLP102_FiresOnMultilineAsyncDeclarationWithoutAwait(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,6 @@
+const getItems = async () =>
+{
+    return [];
+}
`)
	got := SLP102{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for multiline async declaration without await")
	}
}

func TestSLP102_NoFireWhenAddedAsyncHasContextAwait(t *testing.T) {
	// The "async" keyword is added to an existing function whose body already
	// contains "await" in a context (unchanged) line.  The function has real
	// async work, so SLP102 must not fire.
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,4 +1,4 @@
-function getItems() {
+async function getItems() {
     const items = await cache.get("items");
     return items;
 }
`)
	got := SLP102{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when added async function body already contains await in context, got %d", len(got))
	}
}

func TestSLP102_FiresWhenClosingBraceIsContextLine(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,3 +1,3 @@
+async function getItems() {
+    return [];
 }
`)
	got := SLP102{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings when closing brace is a context line")
	}
}

func TestSLP102_IgnoresNonJSTS(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.go b/api.go
--- a/api.go
+++ b/api.go
@@ -1,1 +1,3 @@
+  async function getItems() {}  // not valid Go
`)
	got := SLP102{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for .go file, got %d", len(got))
	}
}

func TestSLP102_FiresForOneLineAsyncArrowNearAwait(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,4 @@
+const getItems = async () => []
+async function fetchData() {
+    const result = await somePromise;
+}
`)
	got := SLP102{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for one-line async arrow with unrelated await nearby, got %d", len(got))
	}
}

func TestSLP102_Description(t *testing.T) {
	r := SLP102{}
	if r.ID() != "SLP102" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
