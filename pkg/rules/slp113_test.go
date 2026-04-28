package rules

import "testing"

func TestSLP113_FiresOnGoSourceWithoutTest(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
new file mode 100644
--- /dev/null
+++ b/handler.go
@@ -0,0 +1,5 @@
+package api
+
+func Handler() {}
`)
	got := SLP113{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for .go source without test")
	}
}

func TestSLP113_NoFireOnGoSourceWithTest(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
new file mode 100644
--- /dev/null
+++ b/handler.go
@@ -0,0 +1,5 @@
+package api
+
+func Handler() {}
diff --git a/handler_test.go b/handler_test.go
new file mode 100644
--- /dev/null
+++ b/handler_test.go
@@ -0,0 +1,5 @@
+package api
+
+func TestHandler(t *testing.T) {}
`)
	got := SLP113{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when test present, got %d", len(got))
	}
}

func TestSLP113_NoFireOnTestFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler_test.go b/handler_test.go
new file mode 100644
--- /dev/null
+++ b/handler_test.go
@@ -0,0 +1,5 @@
+package api
+
+func TestHandler(t *testing.T) {}
`)
	got := SLP113{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for test file, got %d", len(got))
	}
}

func TestSLP113_Description(t *testing.T) {
	r := SLP113{}
	if r.ID() != "SLP113" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
