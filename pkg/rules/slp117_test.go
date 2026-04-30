package rules

import "testing"

func TestSLP117_FiresOnUnanchoredRegex(t *testing.T) {
	d := parseDiff(t, `diff --git a/validate.go b/validate.go
--- a/validate.go
+++ b/validate.go
@@ -1,1 +1,3 @@
 package main
+
+var re = regexp.MustCompile("\\d+")
`)
	got := SLP117{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for unanchored regex")
	}
}

func TestSLP117_NoFireOnAnchoredRegex(t *testing.T) {
	d := parseDiff(t, `diff --git a/validate.go b/validate.go
--- a/validate.go
+++ b/validate.go
@@ -1,1 +1,3 @@
 package main
+
+var re = regexp.MustCompile("^\\d+$")
`)
	got := SLP117{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for anchored regex, got %d", len(got))
	}
}

func TestSLP117_NoFireOnPlainArrayOrClassNameString(t *testing.T) {
	d := parseDiff(t, `diff --git a/Component.tsx b/Component.tsx
--- a/Component.tsx
+++ b/Component.tsx
@@ -1,1 +1,4 @@
 export function Component() {
+  const classes = ["grid", "gap-4", "md:grid-cols-2"].join(" ")
+  return <div className={classes}>ok</div>
 }
`)
	got := SLP117{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for non-regex strings, got %d: %+v", len(got), got)
	}
}

func TestSLP117_FiresOnUnanchoredJSRegExp(t *testing.T) {
	d := parseDiff(t, `diff --git a/validate.ts b/validate.ts
--- a/validate.ts
+++ b/validate.ts
@@ -1,1 +1,3 @@
+const slugPattern = new RegExp("[a-z0-9-]+")
`)
	got := SLP117{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected finding for unanchored RegExp")
	}
}

func TestSLP117_NoFireOnPlainPatternIdentifier(t *testing.T) {
	d := parseDiff(t, `diff --git a/ui.ts b/ui.ts
--- a/ui.ts
+++ b/ui.ts
@@ -1,1 +1,3 @@
+const pattern = "foo"
`)
	got := SLP117{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for plain pattern string, got %d: %+v", len(got), got)
	}
}

func TestSLP117_Description(t *testing.T) {
	r := SLP117{}
	if r.ID() != "SLP117" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
