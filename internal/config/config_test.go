package config

import (
	"os"
	"testing"
)

// unsetEnv removes key for the duration of the test, restoring it afterwards.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	if orig, ok := os.LookupEnv(key); ok {
		t.Cleanup(func() { os.Setenv(key, orig) })
	}
	os.Unsetenv(key)
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("BOT_TOKEN", "abc")
	unsetEnv(t, "MONGO_URI")
	unsetEnv(t, "MONGO_DB")
	unsetEnv(t, "VERBS_PATH")
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

func TestLoadReadsValues(t *testing.T) {
	t.Setenv("BOT_TOKEN", "abc")
	t.Setenv("MONGO_URI", "mongodb://db:27017")
	t.Setenv("MONGO_DB", "custom")
	t.Setenv("VERBS_PATH", "/tmp/verbs.json")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.BotToken != "abc" || c.MongoURI != "mongodb://db:27017" ||
		c.MongoDB != "custom" || c.VerbsPath != "/tmp/verbs.json" {
		t.Fatalf("config = %+v", c)
	}
}

func TestLoadRequiresToken(t *testing.T) {
	unsetEnv(t, "BOT_TOKEN")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when BOT_TOKEN is missing")
	}
}
