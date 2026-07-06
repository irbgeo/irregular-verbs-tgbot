package service

import "testing"

func TestMatchForm(t *testing.T) {
	// multi-variant forms: any single variant OR all of them together passes.
	if !matchForm("was", []string{"was", "were"}) {
		t.Error("was/were: a single variant should pass")
	}
	if !matchForm("were", []string{"was", "were"}) {
		t.Error("was/were: the other single variant should pass")
	}
	if !matchForm("was were", []string{"was", "were"}) {
		t.Error("was/were: both variants together must pass")
	}
	if !matchForm("were was", []string{"was", "were"}) {
		t.Error("was/were: order within does not matter")
	}
	// other alternatives: any one — or both — suffices
	if !matchForm("burnt", []string{"burnt", "burned"}) {
		t.Error("alt: burnt should pass")
	}
	if !matchForm("burned", []string{"burnt", "burned"}) {
		t.Error("alt: burned should pass")
	}
	if !matchForm("burnt burned", []string{"burnt", "burned"}) {
		t.Error("alt: both variants together should pass")
	}
	// single form
	if !matchForm("went", []string{"went"}) {
		t.Error("single form should pass")
	}
	if matchForm("nope", []string{"went"}) {
		t.Error("wrong form should fail")
	}
}
