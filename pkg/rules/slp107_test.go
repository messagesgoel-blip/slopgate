package rules

import "testing"

func TestSLP107_FiresOnCleanupOnlyInErrorPath(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,5 @@
+  conn, err := net.Dial("tcp", addr)
+  if err != nil {
+      conn.Close()
+      return err
+  }
`)
	got := SLP107{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for cleanup only in error path")
	}
}

func TestSLP107_NoFireOnCleanupInNormalPath(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,3 @@
+  defer conn.Close()
`)
	got := SLP107{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for normal cleanup, got %d", len(got))
	}
}

func TestSLP107_Description(t *testing.T) {
	r := SLP107{}
	if r.ID() != "SLP107" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
