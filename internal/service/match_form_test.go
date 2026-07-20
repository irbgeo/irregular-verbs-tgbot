package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchForm(t *testing.T) {
	// multi-variant forms: any single variant OR all of them together passes.
	require.True(t, matchForm("was", []string{"was", "were"}), "was/were: a single variant should pass")
	require.True(t, matchForm("were", []string{"was", "were"}), "was/were: the other single variant should pass")
	require.True(t, matchForm("was were", []string{"was", "were"}), "was/were: both variants together must pass")
	require.True(t, matchForm("were was", []string{"was", "were"}), "was/were: order within does not matter")
	// other alternatives: any one — or both — suffices
	require.True(t, matchForm("burnt", []string{"burnt", "burned"}), "alt: burnt should pass")
	require.True(t, matchForm("burned", []string{"burnt", "burned"}), "alt: burned should pass")
	require.True(t, matchForm("burnt burned", []string{"burnt", "burned"}), "alt: both variants together should pass")
	// single form
	require.True(t, matchForm("went", []string{"went"}), "single form should pass")
	require.False(t, matchForm("nope", []string{"went"}), "wrong form should fail")
}
