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
		"SLP036": false,
		"SLP037": false,
		"SLP038": false,
		"SLP039": false,
		"SLP040": false,
		"SLP041": false,
		"SLP042": false,
		"SLP043": false,
		"SLP044": false,
		"SLP045": false,
		"SLP046": false,
		"SLP047": false,
		"SLP048": false,
		"SLP049": false,
		"SLP050": false,
		"SLP051": false,
		"SLP052": false,
		"SLP053": false,
		"SLP054": false,
		"SLP055": false,
		"SLP056": false,
		"SLP057": false,
		"SLP058": false,
		"SLP059": false,
		"SLP060": false,
		"SLP061": false,
		"SLP062": false,
		"SLP063": false,
		"SLP064": false,
		"SLP065": false,
		"SLP066": false,
		"SLP067": false,
		"SLP068": false,
		"SLP069": false,
		"SLP070": false,
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

	// Check count first
	wantCount := 77
	if got := len(r.All()); got != wantCount {
		t.Errorf("Default registry has %d rules, want %d", got, wantCount)
	}

	// Build set of all registered rule IDs
	ruleIDs := make(map[string]bool)
	for _, rule := range r.All() {
		ruleIDs[rule.ID()] = true
	}

	// Verify SLP081-SLP090 are present
	newRules := []string{"SLP081", "SLP082", "SLP083", "SLP084", "SLP085", "SLP086", "SLP087", "SLP088", "SLP089", "SLP090"}
	for _, id := range newRules {
		if !ruleIDs[id] {
			t.Errorf("new rule %s not registered in Default()", id)
		}
	}
}
