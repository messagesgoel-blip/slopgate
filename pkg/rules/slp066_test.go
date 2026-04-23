package rules

import (
	"strings"
	"testing"
)

func TestSLP066_FiresOnMapWithGoroutineNoMutex(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,7 @@
 package worker
+
+var cache = map[string]int{}
+
+func Run() {
+	go updateCache()
+	cache["key"] = 1
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "worker.go" {
		t.Errorf("file: %q", got[0].File)
	}
	if !strings.Contains(got[0].Message, "map") {
		t.Errorf("message should mention concurrent map: %q", got[0].Message)
	}
}

func TestSLP066_NoFireWithMutex(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,11 @@
package worker
+
+var cache = map[string]int{}
+var mu sync.Mutex
+
+func Run() {
+	go updateCache()
+	mu.Lock()
+	cache["key"] = 1
+	mu.Unlock()
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
+	cache["key"] = 1
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

func TestSLP066_NoFireWithSyncMap(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,7 @@
 package worker
+
+var cache sync.Map
+
+func Run() {
+	go updateCache()
+	cache.Store("key", 1)
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings with sync.Map, got %d: %+v", len(got), got)
	}
}

func TestSLP066_SyncMapDoesNotMaskRegularMap(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,9 @@
 package worker
+
+var safe sync.Map
+var cache = map[string]int{}
+
+func Run() {
+	go updateCache()
+	cache["key"] = 1
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for regular map even when sync.Map exists, got %d: %+v", len(got), got)
	}
}

func TestSLP066_ShortMutexNameDoesNotMatchBySubstring(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,15 @@
 package worker
+
+var summary = map[string]int{}
+var pad1 int
+var pad2 int
+var pad3 int
+var pad4 int
+var pad5 int
+var pad6 int
+var mu sync.Mutex
+
+func Run() {
+	go updateSummary()
+	summary["key"] = 1
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for short mutex-name substring mismatch, got %d: %+v", len(got), got)
	}
}

func TestSLP066_IgnoresGoInComments(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,6 @@
 package worker
+
+var cache = map[string]int{}
+// TODO: go fix this later
+func Run() {
+	cache["key"] = 1
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when only comments contain 'go ', got %d: %+v", len(got), got)
	}
}

func TestSLP066_IgnoresSliceIndexing(t *testing.T) {
	d := parseDiff(t, `diff --git a/worker.go b/worker.go
--- a/worker.go
+++ b/worker.go
@@ -1,1 +1,7 @@
 package worker
+
+var items = []int{1, 2, 3}
+
+func Run(i int) {
+	go work()
+	_ = items[i]
+}
`)
	got := SLP066{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for slice indexing, got %d: %+v", len(got), got)
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
