package rules

import (
	"strings"
	"testing"
)

func TestSLP009_GoGetenvWithoutSetenv(t *testing.T) {
	// os.Getenv("FOO") added with no os.Setenv → finding
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,4 @@
 package main

+	val := os.Getenv("FOO")
+	fmt.Println(val)
`)
	got := SLP009{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "main.go" {
		t.Errorf("file: %q, want main.go", got[0].File)
	}
	if !strings.Contains(got[0].Message, "FOO") {
		t.Errorf("message should mention FOO: %q", got[0].Message)
	}
	if got[0].Severity != SeverityInfo {
		t.Errorf("severity: %v, want info", got[0].Severity)
	}
}

func TestSLP009_GoGetenvWithSetenv(t *testing.T) {
	// os.Getenv("FOO") added AND os.Setenv("FOO", "bar") in same diff → NO finding
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,5 @@
 package main

+	os.Setenv("FOO", "bar")
+	val := os.Getenv("FOO")
+	fmt.Println(val)
`)
	got := SLP009{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP009_JSEnvAccessWithoutAssignment(t *testing.T) {
	// process.env.API_KEY added with no assignment → finding
	d := parseDiff(t, `diff --git a/app.ts b/app.ts
--- a/app.ts
+++ b/app.ts
@@ -1,1 +1,3 @@
 const app = express();
+const apiKey = process.env.API_KEY;
+console.log(apiKey);
`)
	got := SLP009{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "app.ts" {
		t.Errorf("file: %q, want app.ts", got[0].File)
	}
	if !strings.Contains(got[0].Message, "API_KEY") {
		t.Errorf("message should mention API_KEY: %q", got[0].Message)
	}
}

func TestSLP009_JSEnvAccessWithAssignment(t *testing.T) {
	// process.env.API_KEY added AND process.env.API_KEY = "test" in same diff → NO finding
	d := parseDiff(t, `diff --git a/app.ts b/app.ts
--- a/app.ts
+++ b/app.ts
@@ -1,1 +1,4 @@
 const app = express();
+process.env.API_KEY = "test";
+const apiKey = process.env.API_KEY;
+console.log(apiKey);
`)
	got := SLP009{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP009_CrossFileGoSetenv(t *testing.T) {
	// os.Getenv("DB_HOST") in file A AND os.Setenv("DB_HOST", ...) in file B → NO finding
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,3 @@
 package config
+func init() { os.Setenv("DB_HOST", "localhost") }
+
diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,4 @@
 package main

+	host := os.Getenv("DB_HOST")
+	fmt.Println(host)
`)
	got := SLP009{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (cross-file setup), got %d: %+v", len(got), got)
	}
}

func TestSLP009_GoLookupEnvCounts(t *testing.T) {
	// os.LookupEnv("FOO") added → no finding because LookupEnv provides
	// the ok boolean, meaning the code handles the missing case.
	// Additionally, os.Getenv("FOO") in the same diff should also not
	// fire because LookupEnv("FOO") is effectively a declaration of
	// intent to handle the missing case.
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,5 @@
 package main

+	val, ok := os.LookupEnv("FOO")
+	if !ok { val = "default" }
+	fmt.Println(val)
`)
	got := SLP009{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP009_JSBracketAccessWithoutAssignment(t *testing.T) {
	// process.env["API_KEY"] added with no assignment → finding
	d := parseDiff(t, `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,1 +1,3 @@
 const app = require("express")();
+const apiKey = process.env["API_KEY"];
+console.log(apiKey);
`)
	got := SLP009{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "API_KEY") {
		t.Errorf("message should mention API_KEY: %q", got[0].Message)
	}
}

func TestSLP009_JSBracketAccessWithAssignment(t *testing.T) {
	// process.env["API_KEY"] = "test" in same diff → NO finding
	d := parseDiff(t, `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,1 +1,4 @@
 const app = require("express")();
+process.env["API_KEY"] = "test";
+const apiKey = process.env["API_KEY"];
+console.log(apiKey);
`)
	got := SLP009{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP009_PythonEnvVarDrift(t *testing.T) {
	// Python file with os.getenv("FOO") — no corresponding setup => finding.
	d := parseDiff(t, `diff --git a/app.py b/app.py
--- a/app.py
+++ b/app.py
@@ -1,1 +1,3 @@
 import os
+val = os.getenv("FOO")
+print(val)
`)
	got := SLP009{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for Python, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "FOO") {
		t.Errorf("message should mention FOO: %q", got[0].Message)
	}
}

func TestSLP009_JavaEnvVarDrift(t *testing.T) {
	// Java file with System.getenv("DB_URL") — no setup => finding.
	d := parseDiff(t, `diff --git a/App.java b/App.java
--- a/App.java
+++ b/App.java
@@ -1,1 +1,2 @@
 // app
+String db = System.getenv("DB_URL");
`)
	got := SLP009{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for Java, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "DB_URL") {
		t.Errorf("message should mention DB_URL: %q", got[0].Message)
	}
}

func TestSLP009_JavaEnvVarWithSetProperty_NoFinding(t *testing.T) {
	// Java: System.getenv("DB_URL") with System.setProperty("DB_URL") => no finding.
	d := parseDiff(t, `diff --git a/App.java b/App.java
--- a/App.java
+++ b/App.java
@@ -1,1 +1,3 @@
 // app
+System.setProperty("DB_URL", "localhost");
+String db = System.getenv("DB_URL");
`)
	got := SLP009{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (env var set), got %d: %+v", len(got), got)
	}
}

func TestSLP009_RustEnvVarDrift(t *testing.T) {
	// Rust file with std::env::var("API_KEY") — no setup => finding.
	d := parseDiff(t, `diff --git a/src/main.rs b/src/main.rs
--- a/src/main.rs
+++ b/src/main.rs
@@ -1,1 +1,2 @@
 // main
+let key = std::env::var("API_KEY");
`)
	got := SLP009{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for Rust, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "API_KEY") {
		t.Errorf("message should mention API_KEY: %q", got[0].Message)
	}
}

func TestSLP009_RustEnvVarWithSetVar_NoFinding(t *testing.T) {
	// Rust: std::env::var("API_KEY") with std::env::set_var("API_KEY") => no finding.
	d := parseDiff(t, `diff --git a/src/main.rs b/src/main.rs
--- a/src/main.rs
+++ b/src/main.rs
@@ -1,1 +1,3 @@
 // main
+std::env::set_var("API_KEY", "test");
+let key = std::env::var("API_KEY");
`)
	got := SLP009{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (env var set), got %d: %+v", len(got), got)
	}
}

func TestSLP009_IgnoresContextLines(t *testing.T) {
	// Pre-existing os.Getenv in context lines should be ignored.
	// Only the added line matters.
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
 val := os.Getenv("EXISTING")
+fmt.Println("new line")
 fmt.Println(val)
`)
	got := SLP009{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (no new env-var reads), got %d", len(got))
	}
}

func TestSLP009_IDAndDescription(t *testing.T) {
	r := SLP009{}
	if r.ID() != "SLP009" {
		t.Errorf("ID = %q, want SLP009", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity = %v, want info", r.DefaultSeverity())
	}
}

func TestSLP009_MultipleEnvVars(t *testing.T) {
	// Multiple env vars read, only one is set → one finding for the unset one.
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,6 @@
 package main

+	os.Setenv("FOO", "bar")
+	a := os.Getenv("FOO")
+	b := os.Getenv("BAZ")
+	fmt.Println(a, b)
`)
	got := SLP009{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (BAZ), got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "BAZ") {
		t.Errorf("message should mention BAZ: %q", got[0].Message)
	}
}
