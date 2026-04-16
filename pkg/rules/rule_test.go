package rules

import (
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/config"
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
	found := r.Run(&diff.Diff{}, nil)
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

func TestRegistry_RunConfigIgnoresRule(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeRule{id: "SLP100", severity: SeverityBlock, findings: []Finding{
		{RuleID: "SLP100", Message: "should be skipped"},
	}})
	r.Register(fakeRule{id: "SLP101", severity: SeverityBlock, findings: []Finding{
		{RuleID: "SLP101", Message: "should appear"},
	}})
	cfg := &config.Config{
		Rules: map[string]config.RuleConfig{
			"SLP100": {Ignore: true},
		},
	}
	found := r.Run(&diff.Diff{}, cfg)
	if len(found) != 1 {
		t.Fatalf("expected 1 finding (SLP100 ignored), got %d", len(found))
	}
	if found[0].RuleID != "SLP101" {
		t.Errorf("expected SLP101, got %s", found[0].RuleID)
	}
}

func TestRegistry_RunConfigOverridesSeverity(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeRule{id: "SLP200", severity: SeverityBlock, findings: []Finding{
		{RuleID: "SLP200", Severity: SeverityBlock, Message: "downgrade me"},
	}})
	cfg := &config.Config{
		Rules: map[string]config.RuleConfig{
			"SLP200": {Severity: "warn"},
		},
	}
	found := r.Run(&diff.Diff{}, cfg)
	if len(found) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(found))
	}
	if found[0].Severity != SeverityWarn {
		t.Errorf("expected warn severity, got %s", found[0].Severity)
	}
}
