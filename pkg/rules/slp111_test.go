package rules

import "testing"

func TestSLP111_FiresOnBinaryCommitted(t *testing.T) {
	d := parseDiff(t, `diff --git a/build/app.exe b/build/app.exe
new file mode 100755
--- /dev/null
+++ b/build/app.exe
@@ -0,0 +1,0 @@
Binary files differ
`)
	got := SLP111{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding for binary file, got %d", len(got))
	}
	if got[0].RuleID != "SLP111" || got[0].Severity != SeverityBlock {
		t.Errorf("unexpected finding metadata: RuleID=%q Severity=%v", got[0].RuleID, got[0].Severity)
	}
	if got[0].File != "build/app.exe" {
		t.Errorf("expected File=build/app.exe, got %q", got[0].File)
	}
}

func TestSLP111_FiresOnClassFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/Foo.class b/Foo.class
new file mode 100644
--- /dev/null
+++ b/Foo.class
@@ -0,0 +1,0 @@
Binary files differ
`)
	got := SLP111{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding for .class file, got %d", len(got))
	}
	if got[0].RuleID != "SLP111" || got[0].Severity != SeverityBlock {
		t.Errorf("unexpected finding metadata: RuleID=%q Severity=%v", got[0].RuleID, got[0].Severity)
	}
	if got[0].File != "Foo.class" {
		t.Errorf("expected File=Foo.class, got %q", got[0].File)
	}
}

func TestSLP111_FiresWhenBinaryAlreadyTracked(t *testing.T) {
	d := parseDiff(t, `diff --git a/.gitignore b/.gitignore
--- a/.gitignore
+++ b/.gitignore
@@ -1,1 +1,2 @@
+*.exe
diff --git a/app.exe b/app.exe
new file mode 100755
--- /dev/null
+++ b/app.exe
Binary files differ
`)
	got := SLP111{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding even with gitignore, got %d", len(got))
	}
	if got[0].RuleID != "SLP111" || got[0].Severity != SeverityBlock {
		t.Errorf("unexpected finding metadata: RuleID=%q Severity=%v", got[0].RuleID, got[0].Severity)
	}
	if got[0].File != "app.exe" {
		t.Errorf("expected File=app.exe, got %q", got[0].File)
	}
}

func TestSLP111_FiresOnExtensionlessNewFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/build/app b/build/app
new file mode 100755
--- /dev/null
+++ b/build/app
Binary files differ
`)
	got := SLP111{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding for new extensionless binary file, got %d", len(got))
	}
	if got[0].RuleID != "SLP111" || got[0].Severity != SeverityBlock {
		t.Errorf("unexpected finding metadata: RuleID=%q Severity=%v", got[0].RuleID, got[0].Severity)
	}
	if got[0].File != "build/app" {
		t.Errorf("expected File=build/app, got %q", got[0].File)
	}
}

func TestSLP111_AllowsWhitelistedExtensionlessNewFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/Makefile b/Makefile
new file mode 100644
--- /dev/null
+++ b/Makefile
@@ -0,0 +1,3 @@
+all:
+	go build ./...
`)
	got := SLP111{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for whitelisted extensionless file Makefile, got %d", len(got))
	}
}

func TestSLP111_IgnoresSourceFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
new file mode 100644
--- /dev/null
+++ b/main.go
@@ -0,0 +1,5 @@
+package main
+
+func main() {}
`)
	got := SLP111{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for source files, got %d", len(got))
	}
}

func TestSLP111_IgnoresDotfiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/.gitignore b/.gitignore
new file mode 100644
--- /dev/null
+++ b/.gitignore
@@ -0,0 +1,2 @@
+node_modules/
`)
	got := SLP111{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for dotfile, got %d", len(got))
	}
}

func TestSLP111_Description(t *testing.T) {
	r := SLP111{}
	if r.ID() != "SLP111" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
