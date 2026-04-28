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

func TestSLP114_FiresOnNonErrIfStatement(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,4 @@
 package main
+func do() {
+    if ready { file.Close() }
+}
`)
	got := SLP114{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for error-returning call inside non-error if block")
	}
}

func TestSLP114_FiresOnErrPrefixedCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,4 @@
 package main
+func do() {
+    errWrap(data)
+}
`)
	got := SLP114{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for err-prefixed function called as statement")
	}
}

func TestSLP114_FiresOnErrUppercasePrefixedCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,3 @@
 package main
+ErrOpen(data)
`)
	got := SLP114{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for Err-prefixed function called as statement")
	}
}

func TestSLP114_FiresOnPackageQualifiedCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,3 @@
 package main
+errors.New("something failed")
`)
	got := SLP114{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for errors.New package-qualified call")
	}
}

func TestSLP114_FiresOnMultipleCallsPerLine(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,3 @@
 package main
+file.Close(); db.Exec(query)
`)
	got := SLP114{}.Check(d)
	if len(got) < 2 {
		t.Fatalf("expected 2 findings for multiple calls, got %d", len(got))
	}
}

func TestSLP114_FiresOnInlineIfBody(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,3 @@
 package main
+if err := setup(); err != nil { db.Exec(query) }
`)
	got := SLP114{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for error-returning call inside inline if body")
	}
}

func TestSLP114_NoFireOnNewReader(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,3 @@
 package main
+NewReader(data)
`)
	got := SLP114{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for NewReader (non-error constructor), got %d", len(got))
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
