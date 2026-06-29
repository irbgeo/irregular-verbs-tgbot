package service

import "testing"

func TestMatchForm(t *testing.T) {
	// was/were: both required
	if matchForm("was", []string{"was", "were"}) {
		t.Error("was/were: a single form must not pass")
	}
	if !matchForm("was were", []string{"was", "were"}) {
		t.Error("was/were: both forms must pass")
	}
	if !matchForm("were was", []string{"was", "were"}) {
		t.Error("was/were: order within does not matter")
	}
	// other alternatives: any one suffices
	if !matchForm("burnt", []string{"burnt", "burned"}) {
		t.Error("alt: burnt should pass")
	}
	if !matchForm("burned", []string{"burnt", "burned"}) {
		t.Error("alt: burned should pass")
	}
	// single form
	if !matchForm("went", []string{"went"}) {
		t.Error("single form should pass")
	}
	if matchForm("nope", []string{"went"}) {
		t.Error("wrong form should fail")
	}
}
