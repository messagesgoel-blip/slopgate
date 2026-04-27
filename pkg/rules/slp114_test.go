package rules

import "testing"

func TestSLP114_FiresOnUncheckedErrorReturn(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,4 @@
 package main
+func do() {
+    db.Insert("users", "data")
+}
`)
	got := SLP114{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for unchecked error-returning call")
	}
}

func TestSLP114_NoFireOnCheckedError(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,4 @@
 package main
+func do() error {
+    return db.Insert("users", "data")
+}
`)
	got := SLP114{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when error is returned, got %d", len(got))
	}
}

func TestSLP114_NoFireOnIfCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,5 @@
 package main
+func do() {
+    if err := db.Insert("users", "data"); err != nil {
+    }
+}
`)
	got := SLP114{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when error is checked, got %d", len(got))
	}
}

func TestSLP114_Description(t *testing.T) {
	r := SLP114{}
	if r.ID() != "SLP114" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
