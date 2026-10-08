package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeedVerbsUpsertsEach(t *testing.T) {
	vr := newFakeVerbRepo()
	err := SeedVerbs(context.Background(), SeedVerbsParams{
		Repo: vr,
		Verbs: []Verb{
			{
				Base: "go",
			},
			{
				Base: "be",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, vr.calls["go"], "calls = %v, want each verb upserted once", vr.calls)
	require.Equal(t, 1, vr.calls["be"], "calls = %v, want each verb upserted once", vr.calls)
}

type fakeVerbRepo struct {
	calls map[string]int
}

func newFakeVerbRepo() *fakeVerbRepo {
	return &fakeVerbRepo{
		calls: map[string]int{},
	}
}

func (s *fakeVerbRepo) Upsert(_ context.Context, v *Verb) error {
	s.calls[v.Base]++
	return nil
}
