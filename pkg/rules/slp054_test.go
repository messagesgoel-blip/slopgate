package rules

import (
	"strings"
	"testing"
)

func TestSLP054_FiresOnPackageMismatch(t *testing.T) {
	d := parseDiff(t, `diff --git a/auth/token.go b/auth/token.go
--- a/auth/token.go
+++ b/auth/token.go
@@ -1,2 +1,3 @@
-package auth
+package token
`)
	got := SLP054{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "token") {
		t.Errorf("expected message about package token, got %q", got[0].Message)
	}
}

func TestSLP054_IgnoresMatchingPackage(t *testing.T) {
	d := parseDiff(t, `diff --git a/auth/token.go b/auth/token.go
--- a/auth/token.go
+++ b/auth/token.go
@@ -1,2 +1,3 @@
-package auth
+package auth
`)
	got := SLP054{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP054_IgnoresMatchPackages(t *testing.T) {
	d := parseDiff(t, `diff --git a/auth/token.go b/auth/token.go
--- a/auth/token.go
+++ b/auth/token.go
@@ -1,2 +1,3 @@
 package auth
+
+package auth
 `)
	got := SLP054{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for matching package, got %d: %+v", len(got), got)
	}
}

func TestSLP054_IgnoresTestFileWithTestSuffix(t *testing.T) {
	d := parseDiff(t, `diff --git a/auth/token_test.go b/auth/token_test.go
--- a/auth/token_test.go
+++ b/auth/token_test.go
@@ -1,2 +1,3 @@
 package auth
+
+package auth_test
 `)
	got := SLP054{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for _test.go with package_test, got %d: %+v", len(got), got)
	}
}

func TestSLP054_IgnoresNonGoFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,2 +1,3 @@
+const x = 1
`)
	got := SLP054{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for non-Go, got %d: %+v", len(got), got)
	}
}

func TestSLP054_IgnoresPackageMainOnlyForExactCmdSegment(t *testing.T) {
	d := parseDiff(t, `diff --git a/cmd/server/main.go b/cmd/server/main.go
--- a/cmd/server/main.go
+++ b/cmd/server/main.go
@@ -1,1 +1,2 @@
+package main
diff --git a/cmdserver/main.go b/cmdserver/main.go
--- a/cmdserver/main.go
+++ b/cmdserver/main.go
@@ -1,1 +1,2 @@
+package main
`)
	got := SLP054{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected only cmdserver/main.go to fire, got %d: %+v", len(got), got)
	}
	if got[0].File != "cmdserver/main.go" {
		t.Fatalf("expected finding for cmdserver/main.go, got %+v", got[0])
	}
}

func TestSLP054_Description(t *testing.T) {
	r := SLP054{}
	if r.ID() != "SLP054" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
