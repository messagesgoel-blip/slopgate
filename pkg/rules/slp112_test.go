package rules

import "testing"

func TestSLP112_FiresOnGeneratedFileWithoutSource(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/api.pb.go b/api/api.pb.go
new file mode 100644
--- /dev/null
+++ b/api/api.pb.go
@@ -0,0 +1,5 @@
+package api
+
+// Generated code
`)
	got := SLP112{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for generated file without source")
	}
}

func TestSLP112_NoFireOnGeneratedWithSource(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/api.pb.go b/api/api.pb.go
new file mode 100644
--- /dev/null
+++ b/api/api.pb.go
@@ -0,0 +1,5 @@
+package api
+
+// Generated code
diff --git a/api/api.proto b/api/api.proto
new file mode 100644
--- /dev/null
+++ b/api/api.proto
@@ -0,0 +1,5 @@
+syntax = "proto3";
`)
	got := SLP112{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when source also present, got %d", len(got))
	}
}

func TestSLP112_IgnoresNonGeneratedFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/handler.go b/api/handler.go
new file mode 100644
--- /dev/null
+++ b/api/handler.go
@@ -0,0 +1,5 @@
+package api
+
+func Handler() {}
`)
	got := SLP112{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for non-generated file, got %d", len(got))
	}
}

func TestSLP112_Description(t *testing.T) {
	r := SLP112{}
	if r.ID() != "SLP112" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
