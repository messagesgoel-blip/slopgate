package rules

import (
	"strings"
	"testing"
)

func TestSLP070_FiresOnThreeTopLevelDirs(t *testing.T) {
	d := parseDiff(t, `diff --git a/auth/login.go b/auth/login.go
--- a/auth/login.go
+++ b/auth/login.go
@@ -1,1 +1,2 @@
 package auth
+
+func Login() {}
diff --git a/db/query.go b/db/query.go
--- a/db/query.go
+++ b/db/query.go
@@ -1,1 +1,2 @@
 package db
+
+func Query() {}
diff --git a/ui/render.go b/ui/render.go
--- a/ui/render.go
+++ b/ui/render.go
@@ -1,1 +1,2 @@
 package ui
+
+func Render() {}
`)
	got := SLP070{}.Check(d)
	if len(got) != 3 {
		t.Fatalf("expected 3 findings, got %d: %+v", len(got), got)
	}
	for _, g := range got {
		if !strings.Contains(g.Message, "diff touches") {
			t.Errorf("expected directory message, got %q", g.Message)
		}
	}
}

func TestSLP070_NoFireOnTwoDirs(t *testing.T) {
	d := parseDiff(t, `diff --git a/auth/login.go b/auth/login.go
--- a/auth/login.go
+++ a/auth/login.go
@@ -1,1 +1,2 @@
 package auth
+
+func Login() {}
diff --git a/db/query.go b/db/query.go
--- a/db/query.go
+++ b/db/query.go
@@ -1,1 +1,2 @@
 package db
+
+func Query() {}
`)
	got := SLP070{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP070_Meta(t *testing.T) {
	r := SLP070{}
	if r.ID() != "SLP070" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
