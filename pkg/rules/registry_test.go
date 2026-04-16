package rules

import "testing"

func TestDefault_RegistersAllV001Rules(t *testing.T) {
	r := Default()
	want := map[string]bool{
		"SLP001": false,
		"SLP005": false,
		"SLP012": false,
		"SLP013": false,
		"SLP014": false,
	}
	for _, rule := range r.All() {
		if _, ok := want[rule.ID()]; ok {
			want[rule.ID()] = true
		}
	}
	for id, found := range want {
		if !found {
			t.Errorf("rule %s not registered in Default()", id)
		}
	}
}

func TestDefault_NoExtraRules(t *testing.T) {
	r := Default()
	if got := len(r.All()); got != 5 {
		t.Errorf("Default registry has %d rules, want 5", got)
	}
}
