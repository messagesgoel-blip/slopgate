package rules

import "testing"

func TestSLP109_FiresOnDuplicateFunction(t *testing.T) {
	d := parseDiff(t, `diff --git a/handlers.go b/handlers.go
--- a/handlers.go
+++ b/handlers.go
@@ -1,1 +1,18 @@
+func ProcessUser(id string) error {
+    ctx := context.Background()
+    validate(id)
+    log.Printf("processing %s", id)
+    result := db.Insert("users", id)
+    return result
+}
+
+func ProcessItem(id string) error {
+    ctx := context.Background()
+    validate(id)
+    log.Printf("processing %s", id)
+    result := db.Insert("items", id)
+    return result
+}
`)
	got := SLP109{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for duplicate function")
	}
}

func TestSLP109_NoFireOnDifferentFunctions(t *testing.T) {
	d := parseDiff(t, `diff --git a/handlers.go b/handlers.go
--- a/handlers.go
+++ b/handlers.go
@@ -1,1 +1,10 @@
+func ProcessUser(id string) error {
+    return db.Insert("users", id)
+}
+
+func GetMetrics() []Metric {
+    metrics := fetchMetrics()
+    return metrics
+}
`)
	got := SLP109{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for different functions, got %d", len(got))
	}
}

func TestSLP109_Description(t *testing.T) {
	r := SLP109{}
	if r.ID() != "SLP109" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}