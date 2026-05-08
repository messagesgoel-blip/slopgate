package rules

import (
	"fmt"
	"strings"
	"testing"
)

func TestSLP070_FiresOnManyTopLevelDirs(t *testing.T) {
	diffs := []string{
		"diff --git a/auth/login.go b/auth/login.go",
		"diff --git a/db/query.go b/db/query.go",
		"diff --git a/ui/render.go b/ui/render.go",
		"diff --git a/api/handler.go b/api/handler.go",
		"diff --git a/store/model.go b/store/model.go",
		"diff --git a/utils/helper.go b/utils/helper.go",
		"diff --git a/config/settings.go b/config/settings.go",
	}
	hunk := "\n--- a/%s\n+++ b/%s\n@@ -1,1 +1,2 @@\n package %s\n+\n+func Foo() {}\n"
	var combined string
	for i, d := range diffs {
		if i > 0 {
			combined += "\n"
		}
		pkg := []string{"auth", "db", "ui", "api", "store", "utils", "config"}[i]
		file := []string{"auth/login.go", "db/query.go", "ui/render.go", "api/handler.go", "store/model.go", "utils/helper.go", "config/settings.go"}[i]
		combined += d + fmt.Sprintf(hunk, file, file, pkg)
	}
	d := parseDiff(t, combined)
	got := SLP070{}.Check(d)
	if len(got) == 0 {
		t.Fatalf("expected >= 1 finding for 7 top-level dirs, got 0")
	}
	for _, g := range got {
		if !strings.Contains(g.Message, "diff touches") {
			t.Errorf("expected directory message, got %q", g.Message)
		}
	}
}

func TestSLP070_NoFireBelowThreshold(t *testing.T) {
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
diff --git a/api/handler.go b/api/handler.go
--- a/api/handler.go
+++ b/api/handler.go
@@ -1,1 +1,2 @@
 package api
+
+func Handle() {}
diff --git a/store/model.go b/store/model.go
--- a/store/model.go
+++ b/store/model.go
@@ -1,1 +1,2 @@
 package store
+
+func Model() {}
`)
	got := SLP070{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for 5 top-level dirs (below threshold), got %d: %+v", len(got), got)
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
