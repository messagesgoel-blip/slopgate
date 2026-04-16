package rules

import (
	"strings"
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

func parseDiff(t *testing.T, s string) *diff.Diff {
	t.Helper()
	d, err := diff.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("diff parse: %v", err)
	}
	return d
}

func TestSLP012_FiresOnNewTODO(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,2 +1,4 @@
 package a

+// TODO: handle retry
+func Do() {}
`)
	got := SLP012{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "a.go" {
		t.Errorf("file: %q", got[0].File)
	}
	if got[0].Line != 3 {
		t.Errorf("line: %d, want 3", got[0].Line)
	}
	if !strings.Contains(got[0].Message, "TODO") {
		t.Errorf("message should mention TODO: %q", got[0].Message)
	}
}

func TestSLP012_FiresOnFIXMEAndHACK(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+// FIXME: broken
+// HACK: temporary workaround
+// XXX: revisit
`)
	got := SLP012{}.Check(d)
	if len(got) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(got))
	}
}

func TestSLP012_IgnoresExistingTODOInContext(t *testing.T) {
	// Pre-existing TODO as context is OK — we only flag what the diff adds.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,3 +1,4 @@
 // TODO: old one already there
 package a
+func New() {}
 var x = 1
`)
	got := SLP012{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP012_IgnoresTODOInsideStringLiteral(t *testing.T) {
	// A TODO inside a string literal (like a user-facing message or a
	// test fixture) is not slop. We only fire on comment-style TODOs.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+var msg = "TODO list feature"
+const help = "Use TODO.md to track items"
`)
	got := SLP012{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (string literals), got %d: %+v", len(got), got)
	}
}

func TestSLP012_IgnoresMarkdownAndTextFiles(t *testing.T) {
	// TODO markers in README.md / CHANGELOG.md / docs are not slop.
	d := parseDiff(t, `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,1 +1,2 @@
 # Project
+- TODO: add install instructions
`)
	got := SLP012{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in README.md, got %d", len(got))
	}
}

func TestSLP012_IgnoresTrackedTODOWithTicketRef(t *testing.T) {
	// A TODO with a parenthetical ticket reference is tracked work, not
	// slop. Bare TODO: still fires; TODO(TICKET-123): does not.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+// TODO(DSH-015): register monaco-toml community grammar
+// TODO(backend#42): implement gRPC DeliverEvent
`)
	got := SLP012{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for tracked TODOs, got %d: %+v", len(got), got)
	}
}

func TestSLP012_FiresOnBareTODO(t *testing.T) {
	// Still fires on TODO without a ticket reference.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+// TODO: fix this later
+// TODO : also this
`)
	got := SLP012{}.Check(d)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings for bare TODOs, got %d", len(got))
	}
}

func TestSLP012_Description(t *testing.T) {
	r := SLP012{}
	if r.ID() != "SLP012" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
