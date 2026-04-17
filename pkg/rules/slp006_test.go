package rules

import (
	"strings"
	"testing"
)

func TestSLP006_GoPanicNotImplemented(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,3 @@
 package handler
+panic("not implemented")
+func Foo() {}
`)
	got := SLP006{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "handler.go" {
		t.Errorf("file: %q", got[0].File)
	}
	if !strings.Contains(got[0].Message, "stub keyword") {
		t.Errorf("message should mention stub keyword: %q", got[0].Message)
	}
}

func TestSLP006_GoPanicTODO(t *testing.T) {
	d := parseDiff(t, `diff --git a/svc.go b/svc.go
--- a/svc.go
+++ b/svc.go
@@ -1,1 +1,3 @@
 package svc
+panic("TODO: add real logic")
`)
	got := SLP006{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP006_GoPanicNoStringLiteral(t *testing.T) {
	// panic(err) — no string literal, not a stub.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+panic(err)
`)
	got := SLP006{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for panic(err), got %d: %+v", len(got), got)
	}
}

func TestSLP006_GoPanicNoStubKeyword(t *testing.T) {
	// panic("buffer too small") — string literal but no stub keyword.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+panic("buffer too small")
`)
	got := SLP006{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for panic without stub keyword, got %d: %+v", len(got), got)
	}
}

func TestSLP006_GoPanicSegmentationFault(t *testing.T) {
	// panic("segmentation fault recovered") — no stub keyword.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+panic("segmentation fault recovered")
`)
	got := SLP006{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for segfault panic, got %d: %+v", len(got), got)
	}
}

func TestSLP006_JSThrowNotImplemented(t *testing.T) {
	d := parseDiff(t, `diff --git a/svc.ts b/svc.ts
--- a/svc.ts
+++ b/svc.ts
@@ -1,1 +1,3 @@
 import { Foo } from 'bar';
+throw new Error("not implemented");
`)
	got := SLP006{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "svc.ts" {
		t.Errorf("file: %q", got[0].File)
	}
}

func TestSLP006_JSThrowNoStubKeyword(t *testing.T) {
	// throw new Error("database connection failed") — no stub keyword.
	d := parseDiff(t, `diff --git a/db.ts b/db.ts
--- a/db.ts
+++ b/db.ts
@@ -1,1 +1,3 @@
 import { Pool } from 'pg';
+throw new Error("database connection failed");
`)
	got := SLP006{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for throw without stub keyword, got %d: %+v", len(got), got)
	}
}

func TestSLP006_PythonRaiseNotImplementedError(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.py b/api.py
--- a/api.py
+++ b/api.py
@@ -1,1 +1,3 @@
 import os
+raise NotImplementedError
`)
	got := SLP006{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "api.py" {
		t.Errorf("file: %q", got[0].File)
	}
}

func TestSLP006_PythonRaiseNotImplementedErrorWithMessage(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.py b/api.py
--- a/api.py
+++ b/api.py
@@ -1,1 +1,3 @@
 import os
+raise NotImplementedError("streaming not yet supported")
`)
	got := SLP006{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "streaming not yet supported") {
		t.Errorf("message should quote the argument: %q", got[0].Message)
	}
}

func TestSLP006_GoPanicFIXME(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+panic("FIXME: implement this")
`)
	got := SLP006{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for FIXME panic, got %d: %+v", len(got), got)
	}
}

func TestSLP006_GoPanicUnimplemented(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+panic("unimplemented")
`)
	got := SLP006{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for unimplemented panic, got %d: %+v", len(got), got)
	}
}

func TestSLP006_CaseInsensitive(t *testing.T) {
	// Keywords should match case-insensitively.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+panic("TODO: add handler")
+panic("Not Implemented")
`)
	got := SLP006{}.Check(d)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings for case-insensitive stubs, got %d: %+v", len(got), got)
	}
}

func TestSLP006_IgnoresContextLines(t *testing.T) {
	// Pre-existing panic in context should NOT fire.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,2 +1,3 @@
 package a
 panic("not implemented")
+func New() {}
`)
	got := SLP006{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings from context line, got %d: %+v", len(got), got)
	}
}

func TestSLP006_Description(t *testing.T) {
	r := SLP006{}
	if r.ID() != "SLP006" {
		t.Errorf("ID = %q, want SLP006", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}

func TestSLP006_JSThrowTODO(t *testing.T) {
	d := parseDiff(t, `diff --git a/svc.js b/svc.js
--- a/svc.js
+++ b/svc.js
@@ -1,1 +1,3 @@
 const x = 1;
+throw new Error("TODO: implement");
`)
	got := SLP006{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for JS throw TODO, got %d: %+v", len(got), got)
	}
}

func TestSLP006_MultipleFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,2 @@
 package a
+panic("not implemented")
diff --git a/b.py b/b.py
--- a/b.py
+++ b/b.py
@@ -1,1 +1,2 @@
 import os
+raise NotImplementedError
`)
	got := SLP006{}.Check(d)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings across files, got %d: %+v", len(got), got)
	}
}
