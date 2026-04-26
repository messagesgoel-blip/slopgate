package rules

import "testing"

func TestSLP093_FiresOnMockWithoutAssertion(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.test.ts b/api.test.ts
--- a/api.test.ts
+++ b/api.test.ts
@@ -1,1 +1,3 @@
+  jest.spyOn(api, 'getItems').mockResolvedValue([]);
`)
	got := SLP093{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for mock without assertion")
	}
}

func TestSLP093_NoFireOnMockWithAssertion(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.test.ts b/api.test.ts
--- a/api.test.ts
+++ b/api.test.ts
@@ -1,1 +1,5 @@
+  jest.spyOn(api, 'getItems').mockResolvedValue([]);
+  expect(api.getItems).toHaveBeenCalled();
`)
	got := SLP093{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for mock with assertion, got %d", len(got))
	}
}

func TestSLP093_Description(t *testing.T) {
	r := SLP093{}
	if r.ID() != "SLP093" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
