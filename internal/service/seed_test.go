package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeVerbRepo struct {
	calls map[string]int
}

func (f *fakeVerbRepo) Upsert(_ context.Context, v Verb) error {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls[v.Base]++
	return nil
}

func TestSeedVerbsUpsertsEach(t *testing.T) {
	vr := &fakeVerbRepo{}
	err := SeedVerbs(context.Background(), vr, []Verb{{Base: "go"}, {Base: "be"}})
	require.NoError(t, err)
	require.Equal(t, 1, vr.calls["go"], "calls = %v, want each verb upserted once", vr.calls)
	require.Equal(t, 1, vr.calls["be"], "calls = %v, want each verb upserted once", vr.calls)
}
