package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("BOT_TOKEN", "abc")
	t.Setenv("MONGO_URI", "")
	t.Setenv("MONGO_DB", "")
	t.Setenv("VERBS_PATH", "")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("MongoURI = %q", c.MongoURI)
	}
	if c.MongoDB != "irregular_verbs" {
		t.Errorf("MongoDB = %q", c.MongoDB)
	}
	if c.VerbsPath != "data/verbs.json" {
		t.Errorf("VerbsPath = %q", c.VerbsPath)
	}
}

func TestLoadRequiresToken(t *testing.T) {
	t.Setenv("BOT_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when BOT_TOKEN is missing")
	}
}
