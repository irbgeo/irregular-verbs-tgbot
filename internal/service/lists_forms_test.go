package service

import (
	"context"
	"testing"
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
	if be == nil || be.Past != "was/were" || be.Participle != "been" {
		t.Fatalf("my_words be item = %+v", be)
	}

	// word_list elementary pool likewise
	wv := svc.buildWordListView(mustUser(t, repo), "elementary", 0)
	for _, it := range wv.Items {
		if it.Base == "be" && (it.Past != "was/were" || it.Participle != "been") {
			t.Fatalf("word_list be item = %+v", it)
		}
	}
}

func mustUser(t *testing.T, repo *fakeUserRepo) *User {
	t.Helper()
	u, _ := repo.Get(context.Background(), 7)
	return u
}
