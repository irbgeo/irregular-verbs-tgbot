package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListItemsCarryForms(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	_ = repo.Save(ctx, &User{ID: 7, Settings: Settings{Variant: "gb"},
		Words: map[string]WordProgress{"be": {Status: StatusStudy}}})
	svc := New(repo, testCatalog())

	// my_words list contains "be" with its forms
	mv := svc.buildMyWordsView(mustUser(t, repo), 0)
	var be *ListItem
	for i := range mv.Items {
		if mv.Items[i].Base == "be" {
			be = &mv.Items[i]
		}
	}
	require.NotNil(t, be)
	require.Equal(t, "was/were", be.Past)
	require.Equal(t, "been", be.Participle)

	// word_list elementary pool likewise
	wv := svc.buildWordListView(mustUser(t, repo), "elementary", 0)
	for _, it := range wv.Items {
		if it.Base == "be" {
			require.Equal(t, "was/were", it.Past)
			require.Equal(t, "been", it.Participle)
		}
	}
}

func mustUser(t *testing.T, repo *fakeUserRepo) *User {
	t.Helper()
	u, _ := repo.Get(context.Background(), 7)
	return u
}
