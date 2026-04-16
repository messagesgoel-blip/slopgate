package rules

import "testing"

func TestSLP005_FiresOnItDotOnly(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.test.ts b/app.test.ts
--- a/app.test.ts
+++ b/app.test.ts
@@ -1,2 +1,3 @@
 describe("thing", () => {
+  it.only("runs just this", () => expect(1).toBe(1));
 });
`)
	got := SLP005{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP005_FiresOnDescribeDotOnly(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.spec.js b/foo.spec.js
--- a/foo.spec.js
+++ b/foo.spec.js
@@ -1,2 +1,3 @@
 // header
+describe.only("only this", () => {});
`)
	got := SLP005{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP005_FiresOnFdescribeAndFit(t *testing.T) {
	d := parseDiff(t, `diff --git a/x.test.ts b/x.test.ts
--- a/x.test.ts
+++ b/x.test.ts
@@ -1,1 +1,3 @@
 // existing
+fdescribe("focused", () => {});
+fit("focused it", () => {});
`)
	got := SLP005{}.Check(d)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(got))
	}
}

func TestSLP005_IgnoresContextLines(t *testing.T) {
	// Pre-existing .only should not trigger — we only care about the diff.
	d := parseDiff(t, `diff --git a/a.test.ts b/a.test.ts
--- a/a.test.ts
+++ b/a.test.ts
@@ -1,3 +1,4 @@
 it.only("legacy focus", () => {});
 describe("thing", () => {
+  it("new test", () => {});
 });
`)
	got := SLP005{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(got))
	}
}

func TestSLP005_IgnoresNonTestLookalikes(t *testing.T) {
	// `foo.only` is legitimate for any object method named `only`.
	// We match only test-runner prefixes (it, describe, test, context).
	// The fixture MUST use a test-file path so the regex is actually
	// exercised — a non-test path would be filtered before patterns run.
	d := parseDiff(t, `diff --git a/app.test.ts b/app.test.ts
--- a/app.test.ts
+++ b/app.test.ts
@@ -1,1 +1,3 @@
 const x = 1;
+const y = someObject.only;
+const z = filter.only(item);
`)
	got := SLP005{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP005_IgnoresPythonTestsWithFitCall(t *testing.T) {
	// Python data-science code uses fit() for model training.
	// SLP005 must not flag these as focused tests.
	d := parseDiff(t, `diff --git a/tests/test_model.py b/tests/test_model.py
--- a/tests/test_model.py
+++ b/tests/test_model.py
@@ -1,1 +1,5 @@
 import pytest
+def test_training():
+    model = LinearRegression()
+    model.fit(X_train, y_train)
+    assert model.score(X_test, y_test) > 0.8
`)
	got := SLP005{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings on Python fit(), got %d: %+v", len(got), got)
	}
}

func TestSLP005_IgnoresPytestPrefixFiles(t *testing.T) {
	// test_*.py pytest convention should also be excluded.
	d := parseDiff(t, `diff --git a/test_pipeline.py b/test_pipeline.py
--- a/test_pipeline.py
+++ b/test_pipeline.py
@@ -1,1 +1,4 @@
 import sklearn
+def test_fit():
+    pipe = Pipeline([('scaler', StandardScaler())])
+    pipe.fit(data)
`)
	got := SLP005{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings on pytest test_*.py with fit(), got %d: %+v", len(got), got)
	}
}

func TestSLP005_Description(t *testing.T) {
	r := SLP005{}
	if r.ID() != "SLP005" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
}
