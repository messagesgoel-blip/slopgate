package rules

import "testing"

func TestSLP104_FiresOnMakeByte(t *testing.T) {
	d := parseDiff(t, `diff --git a/parser.go b/parser.go
--- a/parser.go
+++ b/parser.go
@@ -1,1 +1,3 @@
+  buf := make([]byte, 4096)
`)
	got := SLP104{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for make([]byte, 4096)")
	}
}

func TestSLP104_FiresOnBufioReaderSize(t *testing.T) {
	d := parseDiff(t, `diff --git a/reader.go b/reader.go
--- a/reader.go
+++ b/reader.go
@@ -1,1 +1,3 @@
+  r := bufio.NewReaderSize(f, 65536)
`)
	got := SLP104{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for NewReaderSize")
	}
}

func TestSLP104_FiresOnNestedBufioReaderSize(t *testing.T) {
	d := parseDiff(t, `diff --git a/reader.go b/reader.go
--- a/reader.go
+++ b/reader.go
@@ -1,1 +1,3 @@
+  r := bufio.NewReaderSize(io.LimitReader(f, 1234), 65536)
`)
	got := SLP104{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for nested NewReaderSize")
	}
}

func TestSLP104_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/parser.test.go b/parser.test.go
--- a/parser.test.go
+++ b/parser.test.go
@@ -1,1 +1,3 @@
+  buf := make([]byte, 4096)
`)
	got := SLP104{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for test file, got %d", len(got))
	}
}

func TestSLP104_IgnoresZeroLengthMake(t *testing.T) {
	d := parseDiff(t, `diff --git a/parser.go b/parser.go
--- a/parser.go
+++ b/parser.go
@@ -1,1 +1,3 @@
+  buf := make([]byte, 0)
`)
	got := SLP104{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for make([]byte, 0), got %d", len(got))
	}
}

func TestSLP104_FiresOnMakeThreeArg(t *testing.T) {
	d := parseDiff(t, `diff --git a/parser.go b/parser.go
--- a/parser.go
+++ b/parser.go
@@ -1,1 +1,3 @@
+  buf := make([]byte, 4096, 8192)
`)
	got := SLP104{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for make([]byte, 4096, 8192)")
	}
}

func TestSLP104_FiresOnMakeZeroLenNonZeroCap(t *testing.T) {
	d := parseDiff(t, `diff --git a/parser.go b/parser.go
--- a/parser.go
+++ b/parser.go
@@ -1,1 +1,3 @@
+  buf := make([]byte, 0, 4096)
+`)
	got := SLP104{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for make([]byte, 0, 4096)")
	}
}

func TestSLP104_FiresOnLiteralBufferSize(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,3 @@
+  bufferSize := 4096
+`)
	got := SLP104{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for bufferSize := 4096")
	}
}

func TestSLP104_DoesNotFireOnIdentifierContainingLimit(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,3 @@
+  bufferLimitless := 4096
`)
	got := SLP104{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for bufferLimitless identifier, got %d", len(got))
	}
}

func TestSLP104_DoesNotFireOnBareLimitIdentifier(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,3 @@
+  limit := 4096
`)
	got := SLP104{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for bare limit identifier, got %d", len(got))
	}
}

func TestSLP104_Description(t *testing.T) {
	r := SLP104{}
	if r.ID() != "SLP104" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
