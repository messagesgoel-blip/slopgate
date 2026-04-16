package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/rules"
)

func TestText_NoFindings(t *testing.T) {
	var buf bytes.Buffer
	WriteText(&buf, nil, false)
	if strings.TrimSpace(buf.String()) != "slopgate: no findings" {
		t.Errorf("unexpected output: %q", buf.String())
	}
}

func TestText_GroupsByFile(t *testing.T) {
	var buf bytes.Buffer
	WriteText(&buf, []rules.Finding{
		{RuleID: "SLP012", Severity: rules.SeverityBlock, File: "a.go", Line: 3, Message: "todo", Snippet: "// TODO: fix"},
		{RuleID: "SLP014", Severity: rules.SeverityBlock, File: "a.go", Line: 7, Message: "print", Snippet: "fmt.Println(x)"},
		{RuleID: "SLP012", Severity: rules.SeverityBlock, File: "b.go", Line: 10, Message: "todo", Snippet: "// TODO: also"},
	}, false)
	out := buf.String()
	if !strings.Contains(out, "a.go") || !strings.Contains(out, "b.go") {
		t.Errorf("expected both files in output: %q", out)
	}
	if !strings.Contains(out, "SLP012") || !strings.Contains(out, "SLP014") {
		t.Errorf("expected rule IDs in output: %q", out)
	}
	if !strings.Contains(out, "a.go:3") || !strings.Contains(out, "a.go:7") || !strings.Contains(out, "b.go:10") {
		t.Errorf("expected file:line refs: %q", out)
	}
}

func TestText_CountsBlockingVsWarn(t *testing.T) {
	var buf bytes.Buffer
	WriteText(&buf, []rules.Finding{
		{RuleID: "SLP012", Severity: rules.SeverityBlock, File: "a.go", Line: 1, Message: "x"},
		{RuleID: "SLP014", Severity: rules.SeverityWarn, File: "a.go", Line: 2, Message: "y"},
	}, false)
	out := buf.String()
	if !strings.Contains(out, "2 findings") {
		t.Errorf("expected total count: %q", out)
	}
	if !strings.Contains(out, "1 block") {
		t.Errorf("expected block count: %q", out)
	}
}
