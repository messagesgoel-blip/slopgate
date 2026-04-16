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
	r.Register(SLP010{})
	r.Register(SLP012{})
	r.Register(SLP013{})
	r.Register(SLP014{})
	return r
}
