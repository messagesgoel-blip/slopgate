package rules

import "testing"

func TestDefault_RegistersAllV001Rules(t *testing.T) {
	r := Default()
	want := map[string]bool{
		"SLP001": false,
		"SLP002": false,
		"SLP003": false,
		"SLP005": false,
		"SLP006": false,
		"SLP007": false,
		"SLP008": false,
		"SLP009": false,
		"SLP010": false,
		"SLP011": false,
		"SLP012": false,
		"SLP013": false,
		"SLP014": false,
		"SLP015": false,
		"SLP016": false,
		"SLP017": false,
		"SLP018": false,
		"SLP019": false,
		"SLP020": false,
		"SLP021": false,
		"SLP022": false,
		"SLP023": false,
		"SLP024": false,
		"SLP025": false,
		"SLP026": false,
		"SLP027": false,
		"SLP030": false,
		"SLP031": false,
		"SLP032": false,
		"SLP033": false,
		"SLP034": false,
		"SLP035": false,
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
	if got := len(r.All()); got != 32 {
		t.Errorf("Default registry has %d rules, want 32", got)
	}
}
