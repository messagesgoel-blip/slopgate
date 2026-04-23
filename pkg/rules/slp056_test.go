package rules

import (
	"strings"
	"testing"
)

func TestSLP056_FiresOnAPIKey(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,2 +1,3 @@
 package main
+
+var apiKey = "abc123secret"
`)
	got := SLP056{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "config.go" {
		t.Errorf("file: %q", got[0].File)
	}
	if !strings.Contains(got[0].Message, "hardcoded secret pattern detected") {
		t.Errorf("message: %q", got[0].Message)
	}
}

func TestSLP056_FiresOnPassword(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,2 +1,3 @@
 def main():
+
+    password = 'hunter2'
`)
	got := SLP056{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP056_FiresOnToken(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,1 +1,2 @@
 const x = 1
+
+const token = "bearertok123"
`)
	got := SLP056{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP056_FiresOnAWSKey(t *testing.T) {
	d := parseDiff(t, `diff --git a/creds.go b/creds.go
--- a/creds.go
+++ b/creds.go
@@ -1,1 +1,2 @@
 package creds
+
+aws_access_key_id=AKIAIOSFODNN7ABC123
`)
	got := SLP056{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP056_FiresOnPrivateKey(t *testing.T) {
	d := parseDiff(t, `diff --git a/key.go b/key.go
--- a/key.go
+++ b/key.go
@@ -1,1 +1,2 @@
 package key
+
+private_key =
`)
	got := SLP056{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP056_SkipsExample(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,2 @@
 package main
+
+api_key = "example_key_here"
`)
	got := SLP056{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for example line, got %d: %+v", len(got), got)
	}
}

func TestSLP056_SkipsDummyAndTest(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,3 @@
 package main
+
+password = "dummy"
+secret = "test_secret"
`)
	got := SLP056{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for dummy/test lines, got %d: %+v", len(got), got)
	}
}

func TestSLP056_SkipsPlaceholder(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,2 @@
 package main
+
+token = "placeholder_value"
`)
	got := SLP056{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for placeholder, got %d: %+v", len(got), got)
	}
}

func TestSLP056_DoesNotSkipRealSecretWithInlineTodo(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,2 @@
 package main
+
+password = "hunter2" // TODO rotate after testing
`)
	got := SLP056{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding when inline TODO accompanies a real secret, got %d: %+v", len(got), got)
	}
}

func TestSLP056_IgnoresAWSKeyFromEnv(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,2 @@
 package main
+
+aws_access_key_id = os.Getenv("AWS_ACCESS_KEY_ID")
`)
	got := SLP056{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for env-backed AWS key, got %d: %+v", len(got), got)
	}
}

func TestSLP056_IgnoresPrivateKeyLoadedFromFunction(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,2 @@
 package main
+
+private_key = read_file("id_rsa")
`)
	got := SLP056{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for function-loaded private key, got %d: %+v", len(got), got)
	}
}

func TestSLP056_Description(t *testing.T) {
	r := SLP056{}
	if r.ID() != "SLP056" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
