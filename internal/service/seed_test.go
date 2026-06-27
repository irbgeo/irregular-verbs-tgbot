package service

import (
	"context"
	"testing"
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
	if err := SeedVerbs(context.Background(), vr, []Verb{{Base: "go"}, {Base: "be"}}); err != nil {
		t.Fatalf("SeedVerbs: %v", err)
	}
	if vr.calls["go"] != 1 || vr.calls["be"] != 1 {
		t.Fatalf("calls = %v, want each verb upserted once", vr.calls)
	}
}
