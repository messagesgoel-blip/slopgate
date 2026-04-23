package rules

import (
	"strings"
	"testing"
)

func TestSLP061_FiresOnNewFactoryWithOneFieldStruct(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,7 @@
 package foo
+type Config struct {
+	Host string
+}
+func NewConfig() Config {
+	return Config{Host: "localhost"}
+}
`)
	got := SLP061{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "over-engineered") {
		t.Errorf("message should mention over-engineered: %q", got[0].Message)
	}
	if got[0].Line != 5 {
		t.Errorf("line: %d, want 5", got[0].Line)
	}
}

func TestSLP061_NoFireForThreeFieldStruct(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,10 @@
 package foo
+type Config struct {
+	Host string
+	Port int
+	TLS  bool
+}
+func NewConfig() Config {
+	return Config{Host: "localhost"}
+}
`)
	got := SLP061{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP061_FiresOnBuildFactory(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,7 @@
 package foo
+type Options struct {
+	Timeout int
+}
+func BuildOptions() Options {
+	return Options{Timeout: 10}
+}
`)
	got := SLP061{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP061_NoFireWhenStructNotInDiff(t *testing.T) {
	// Struct definition is not added in this diff, so we can't count fields.
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,4 @@
 package foo
+func NewConfig() Config {
+	return Config{Host: "localhost"}
+}
`)
	got := SLP061{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP061_IgnoresNonGoFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.js b/foo.js
--- a/foo.js
+++ b/foo.js
@@ -1,1 +1,4 @@
 function newBuilder() { return {}; }
+function NewConfig() { return {}; }
`)
	got := SLP061{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP061_FiresOnPointerReturn(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,8 @@
 package foo
+type Client struct {
+	URL string
+}
+func NewClient() *Client {
+	return &Client{URL: ""}
+}
`)
	got := SLP061{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Line != 5 {
		t.Errorf("line: %d, want 5", got[0].Line)
	}
}

func TestSLP061_FiresWhenFieldTypeContainsComma(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,8 @@
 package foo
+type Config struct {
+	Handler func(a, b string)
+	Mode    string
+}
+func NewConfig() Config {
+	return Config{}
+}
`)
	got := SLP061{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding when comma appears inside a field type, got %d: %+v", len(got), got)
	}
}

func TestSLP061_FiresOnMultilineSignaturePointerReturn(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,10 @@
 package foo
+type User struct {
+	Name string
+}
+func NewUser(
+) *User {
+	return &User{
+		Name: "sanjay",
+	}
+}
`)
	got := SLP061{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for multiline pointer-return factory, got %d: %+v", len(got), got)
	}
}

func TestSLP061_Description(t *testing.T) {
	r := SLP061{}
	if r.ID() != "SLP061" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
