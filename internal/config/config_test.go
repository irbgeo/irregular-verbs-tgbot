package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	require.Equal(t, "mongodb://localhost:27017", c.MongoURI)
	require.Equal(t, "irregular_verbs", c.MongoDB)
	require.Equal(t, "data/verbs.json", c.VerbsPath)
}

func TestLoadReadsValues(t *testing.T) {
	t.Setenv("BOT_TOKEN", "abc")
	t.Setenv("MONGO_URI", "mongodb://db:27017")
	t.Setenv("MONGO_DB", "custom")
	t.Setenv("VERBS_PATH", "/tmp/verbs.json")
	c, err := Load()
	require.NoError(t, err)
	require.Equal(t, "abc", c.BotToken)
	require.Equal(t, "mongodb://db:27017", c.MongoURI)
	require.Equal(t, "custom", c.MongoDB)
	require.Equal(t, "/tmp/verbs.json", c.VerbsPath)
}

func TestLoadRequiresToken(t *testing.T) {
	unsetEnv(t, "BOT_TOKEN")
	_, err := Load()
	require.Error(t, err, "expected error when BOT_TOKEN is missing")
}
