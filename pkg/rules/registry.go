package rules

// Default returns a Registry pre-populated with every rule shipped in
// the current slopgate version. The CLI calls this; tests that need
// custom rule sets should use NewRegistry directly.
func Default() *Registry {
	r := NewRegistry()
	r.Register(SLP001{})
	r.Register(SLP002{})
	r.Register(SLP003{})
	r.Register(SLP005{})
	r.Register(SLP006{})
	r.Register(SLP007{})
	r.Register(SLP008{})
	r.Register(SLP009{})
	r.Register(SLP010{})
	r.Register(SLP011{})
	r.Register(SLP012{})
	r.Register(SLP013{})
	r.Register(SLP014{})
	r.Register(SLP015{})
	return r
}
