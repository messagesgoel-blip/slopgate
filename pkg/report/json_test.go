package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/rules"
)

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteJSON_PropagatesWriteError(t *testing.T) {
	wantErr := fmt.Errorf("broken pipe")
	err := WriteJSON(failingWriter{err: wantErr}, nil)
	if err == nil {
		t.Fatal("expected error from WriteJSON with failing writer, got nil")
	}
}

func TestWriteJSON_NoFindings(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, nil); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}
	var out JSONReport
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(out.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(out.Findings))
	}
	if out.Summary.Total != 0 {
		t.Errorf("expected total 0, got %d", out.Summary.Total)
	}
}

func TestWriteJSON_WithFindings(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, []rules.Finding{
		{RuleID: "SLP012", Severity: rules.SeverityBlock, File: "a.go", Line: 3, Message: "todo", Snippet: "// TODO: fix"},
		{RuleID: "SLP001", Severity: rules.SeverityWarn, File: "b.go", Line: 7, Message: "test", Snippet: "func TestX(t *testing.T) {"},
	}); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}
	var out JSONReport
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if out.Summary.Total != 2 {
		t.Errorf("total = %d, want 2", out.Summary.Total)
	}
	if out.Summary.Block != 1 {
		t.Errorf("block = %d, want 1", out.Summary.Block)
	}
	if out.Summary.Warn != 1 {
		t.Errorf("warn = %d, want 1", out.Summary.Warn)
	}
	if len(out.Findings) != 2 {
		t.Fatalf("findings len = %d, want 2", len(out.Findings))
	}
	if out.Findings[0].RuleID != "SLP012" {
		t.Errorf("first finding rule = %q", out.Findings[0].RuleID)
	}
	if out.Findings[0].Severity != "block" {
		t.Errorf("first finding severity = %q", out.Findings[0].Severity)
	}
}
