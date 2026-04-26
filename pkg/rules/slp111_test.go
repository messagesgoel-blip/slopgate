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
	if len(got) == 0 {
		t.Fatal("expected findings for binary file")
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
	if len(got) == 0 {
		t.Fatal("expected findings for .class file")
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
	if len(got) == 0 {
		t.Fatal("expected findings even with gitignore — binary may already be tracked")
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
	if len(got) == 0 {
		t.Fatal("expected finding for new extensionless binary file build/app")
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
