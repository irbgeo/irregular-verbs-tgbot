package service

// beVerb is a shared test fixture (irregular verb with a multi-form past).
func beVerb() Verb {
	return Verb{
		Base:         "be",
		Past:         map[string][]string{"gb": {"was", "were"}, "us": {"was", "were"}},
		Participle:   map[string][]string{"gb": {"been"}, "us": {"been"}},
		Translations: []string{"быть", "являться"},
	}
}
