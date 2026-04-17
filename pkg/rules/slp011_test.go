package rules

import "testing"

func TestSLP011_TableDriven(t *testing.T) {
	cases := []struct {
		name     string
		diff     string
		wantLen  int
		wantLine int // 0 means don't check line
	}{
		{
			name: "fires on single assertion",
			diff: `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestFoo(t *testing.T) {
+	assert.Equal(t, 1, 1)
+}
`,
			wantLen:  1,
			wantLine: 3,
		},
		{
			name: "fires on multiple assertions with no arrange",
			diff: `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,7 @@
 package a

+func TestBar(t *testing.T) {
+	assert.Equal(t, 1, 1)
+	assert.NotNil(t, nil)
+	require.True(t, true)
+}
`,
			wantLen: 1,
		},
		{
			name: "ignores test with arrange via if-short-declaration",
			diff: `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,6 @@
 package a

+func TestFoo(t *testing.T) {
+	if got := Foo(); assert.Equal(t, 1, got) {
+	}
+}
`,
			wantLen: 0,
		},
		{
			name: "ignores test with separate variable declaration",
			diff: `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,6 @@
 package a

+func TestFoo(t *testing.T) {
+	got := Foo()
+	assert.Equal(t, 1, got)
+}
`,
			wantLen: 0,
		},
		{
			name: "ignores test with t.Errorf",
			diff: `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestFoo(t *testing.T) {
+	if got := Foo(); got != 42 { t.Errorf("got %d", got) }
+}
`,
			wantLen: 0,
		},
		{
			name: "ignores NilSafe safety test",
			diff: `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestNilSafe(t *testing.T) {
+	assert.Nil(t, nil)
+}
`,
			wantLen: 0,
		},
		{
			name: "ignores non-test file",
			diff: `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func TestHelper(t *testing.T) {
+	assert.True(t, true)
+}
`,
			wantLen: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := parseDiff(t, c.diff)
			got := SLP011{}.Check(d)
			if len(got) != c.wantLen {
				t.Fatalf("expected %d findings, got %d: %+v", c.wantLen, len(got), got)
			}
			if c.wantLen > 0 && len(got) > 0 {
				if c.wantLine > 0 && got[0].Line != c.wantLine {
					t.Errorf("line = %d, want %d", got[0].Line, c.wantLine)
				}
				if got[0].File != "a/foo_test.go" {
					t.Errorf("file = %q", got[0].File)
				}
			}
		})
	}
}

func TestSLP011_Description(t *testing.T) {
	r := SLP011{}
	if r.ID() != "SLP011" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
}
