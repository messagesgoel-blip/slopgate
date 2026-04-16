package rules

import (
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// fakeRule is a rule double used to test the registry.
type fakeRule struct {
	id       string
	desc     string
	severity Severity
	findings []Finding
}

func (f fakeRule) ID() string                { return f.id }
func (f fakeRule) Description() string       { return f.desc }
func (f fakeRule) DefaultSeverity() Severity { return f.severity }
func (f fakeRule) Check(*diff.Diff) []Finding {
	return f.findings
}

func TestRegistry_RegisterAndList(t *testing.T) {
	r := NewRegistry()
	a := fakeRule{id: "SLP999", desc: "dummy", severity: SeverityBlock}
	r.Register(a)
	got := r.All()
	if len(got) != 1 || got[0].ID() != "SLP999" {
		t.Errorf("expected SLP999 in registry, got %+v", got)
	}
}

func TestRegistry_RejectsDuplicateID(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeRule{id: "SLP999"})
	defer func() {
		if recover() == nil {
			t.Errorf("expected panic on duplicate rule ID")
		}
	}()
	r.Register(fakeRule{id: "SLP999"})
}

func TestRegistry_RunCollectsAllFindings(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeRule{
		id: "SLP100",
		findings: []Finding{
			{RuleID: "SLP100", Message: "first"},
		},
	})
	r.Register(fakeRule{
		id: "SLP101",
		findings: []Finding{
			{RuleID: "SLP101", Message: "second"},
			{RuleID: "SLP101", Message: "third"},
		},
	})
	found := r.Run(&diff.Diff{})
	if len(found) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(found))
	}
	// Findings should come out in registration order so reporters can
	// present them deterministically.
	if found[0].RuleID != "SLP100" || found[1].RuleID != "SLP101" || found[2].RuleID != "SLP101" {
		t.Errorf("unexpected finding order: %+v", found)
	}
}

func TestSeverity_String(t *testing.T) {
	cases := map[Severity]string{
		SeverityInfo:  "info",
		SeverityWarn:  "warn",
		SeverityBlock: "block",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("Severity(%d).String() = %q, want %q", s, got, want)
		}
	}
}
