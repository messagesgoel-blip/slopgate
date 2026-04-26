package rules

import "testing"

func TestSLP092_FiresOnDoubleUnwrap(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.test.ts b/api.test.ts
--- a/api.test.ts
+++ b/api.test.ts
@@ -1,1 +1,3 @@
+  const result = res.data.data.items;
`)
	got := SLP092{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for double unwrap")
	}
}

func TestSLP092_FiresOnArrowObjectMockWithoutEnvelope(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.test.ts b/api.test.ts
--- a/api.test.ts
+++ b/api.test.ts
@@ -1,1 +1,5 @@
+  mockImplementation(() => ({ items }));
+  const { ok, data } = await getItems();
`)
	got := SLP092{}.Check(d)
	if len(got) == 0 {
		t.Fatalf("expected findings for envelope mismatch, got %d", len(got))
	}
}

func TestSLP092_IgnoresEnvelopeObjectMock(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.test.ts b/api.test.ts
--- a/api.test.ts
+++ b/api.test.ts
@@ -1,1 +1,5 @@
+  mockResolvedValue({ ok: true, data: items });
+  const { ok, data } = await getItems();
`)
	got := SLP092{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for envelope-shaped mock, got %d", len(got))
	}
}

func TestSLP092_FiresOnDataOnlyMockShape(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.test.ts b/api.test.ts
--- a/api.test.ts
+++ b/api.test.ts
@@ -1,1 +1,5 @@
+  mockResolvedValue({ data: items });
+  const { ok, data } = await getItems();
`)
	got := SLP092{}.Check(d)
	if len(got) == 0 {
		t.Fatalf("expected findings for data-only mock shape, got %d", len(got))
	}
}

func TestSLP092_IgnoresNonTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  const result = res.data.data.items;
`)
	got := SLP092{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for non-test file, got %d", len(got))
	}
}

func TestSLP092_Description(t *testing.T) {
	r := SLP092{}
	if r.ID() != "SLP092" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
