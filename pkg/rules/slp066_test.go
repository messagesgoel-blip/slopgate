package rules

import (
	"strings"
	"testing"
)

func TestSLP066_FiresOnMapWithGoroutineNoMutex(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,6 @@
 package worker
+
+var cache = map[string]int{}
+
+func Run() {
+	go updateCache()
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "worker.go" {
		t.Errorf("file: %q", got[0].File)
	}
	if !strings.Contains(got[0].Message, "map accessed concurrently") {
		t.Errorf("message should mention concurrent map: %q", got[0].Message)
	}
}

func TestSLP066_NoFireWithMutex(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,8 @@
 package worker
+
+var cache = map[string]int{}
+var mu sync.Mutex
+
+func Run() {
+	go updateCache()
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP066_FiresOnMapWithWaitGroupNoMutex(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,8 @@
 package worker
+
+var cache = map[string]int{}
+
+func Run() {
+	var wg sync.WaitGroup
+	wg.Add(1)
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP066_NoFireWithoutMap(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,4 @@
 package worker
+
+func Run() {
+	go doWork()
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP066_Meta(t *testing.T) {
	r := SLP066{}
	if r.ID() != "SLP066" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
