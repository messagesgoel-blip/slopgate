package rules

import "testing"

func TestSLP099_FiresOnResponseFieldWithoutTestUpdate(t *testing.T) {
	d := parseDiff(t, `diff --git a/response.go b/response.go
--- a/response.go
+++ b/response.go
@@ -1,5 +1,7 @@
 type ItemResponse struct {
     ID   int    ` + "`json:\"id\"`" + `
     Name string ` + "`json:\"name\"`" + `
+    Slug string ` + "`json:\"slug\"`" + `
 }
`)
	got := SLP099{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for response field without test")
	}
}

func TestSLP099_NoFireWhenTestAlsoModified(t *testing.T) {
	d := parseDiff(t, `diff --git a/response.go b/response.go
--- a/response.go
+++ b/response.go
@@ -1,3 +1,4 @@
 type ItemResponse struct {
     ID   int    ` + "`json:\"id\"`" + `
+    Slug string ` + "`json:\"slug\"`" + `
 }
diff --git a/response_test.go b/response_test.go
--- a/response_test.go
+++ b/response_test.go
@@ -1,1 +1,3 @@
+  func TestItemResponse(t *testing.T) {
 `)
	got := SLP099{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when test also modified, got %d", len(got))
	}
}

func TestSLP099_IgnoresNonResponseFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/utils.go b/utils.go
--- a/utils.go
+++ b/utils.go
@@ -1,1 +1,3 @@
+    Helper string ` + "`json:\"helper\"`" + `
`)
	got := SLP099{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for non-response file, got %d", len(got))
	}
}

func TestSLP099_Description(t *testing.T) {
	r := SLP099{}
	if r.ID() != "SLP099" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
