package service

import (
	"context"
	"testing"
)

func TestToggleMyWordsMovesSections(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)

	// go is study -> tap moves to skipped (draft only)
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusSkipped {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}
	if u.Words["go"].Status != StatusStudy {
		t.Fatal("words must not change before commit")
	}
	// tap go again -> back to original study -> draft entry removed
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	if _, ok := u.State.List.Draft["go"]; ok {
		t.Fatalf("draft should be cleared, got %+v", u.State.List.Draft)
	}
}

func TestToggleWordListStudy(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)

	// be is learned (effective != study) -> tap sets study
	_, _ = svc.ListToggle(ctx, 7, "be")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["be"] != StatusStudy {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}
	// "build" is new -> tap -> study; tap again -> new -> draft cleared
	_, _ = svc.ListToggle(ctx, 7, "build")
	u, _ = repo.Get(ctx, 7)
	if u.State.List.Draft["build"] != StatusStudy {
		t.Fatalf("build draft = %+v", u.State.List.Draft)
	}
	_, _ = svc.ListToggle(ctx, 7, "build")
	u, _ = repo.Get(ctx, 7)
	if _, ok := u.State.List.Draft["build"]; ok {
		t.Fatalf("build draft should clear, got %+v", u.State.List.Draft)
	}
	// go is study -> tap study; tap again -> back to study (stored), draft cleared
	_, _ = svc.ListToggle(ctx, 7, "go")
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	if _, ok := u.State.List.Draft["go"]; ok {
		t.Fatalf("go draft should clear, got %+v", u.State.List.Draft)
	}
}

func TestCommitAppliesDraft(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ListToggle(ctx, 7, "build") // new -> study
	_, _ = svc.ListToggle(ctx, 7, "go")    // study -> new

	v, err := svc.CommitList(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenMainMenu {
		t.Fatalf("screen = %s", v.Screen)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List != nil {
		t.Fatal("list state must be cleared after commit")
	}
	if u.Words["build"].Status != StatusStudy {
		t.Fatalf("words = %+v", u.Words)
	}
	if _, ok := u.Words["go"]; ok {
		t.Fatalf("go should be deleted, words = %+v", u.Words)
	}
	// study sets status only; box stays 0
	if u.Words["build"].Box != 0 || u.Words["build"].Mode != 0 {
		t.Fatalf("build progress = %+v", u.Words["build"])
	}
}

func TestCommitNewDeletes(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ListToggle(ctx, 7, "go") // study -> new (toggle off)
	_, _ = svc.CommitList(ctx, 7)
	u, _ := repo.Get(ctx, 7)
	if _, ok := u.Words["go"]; ok {
		t.Fatalf("go should be deleted, got %+v", u.Words["go"])
	}
}

func TestCancelDiscards(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)
	_, _ = svc.ListToggle(ctx, 7, "go")
	v, _ := svc.CancelList(ctx, 7)
	if v.Screen != ScreenMainMenu {
		t.Fatalf("screen = %s", v.Screen)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List != nil || u.Words["go"].Status != StatusStudy {
		t.Fatalf("cancel should discard; state=%+v words=%+v", u.State.List, u.Words)
	}
}
