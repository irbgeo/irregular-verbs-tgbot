package config

import (
	"fmt"
	"os"
)

// Config holds runtime configuration from the environment.
type Config struct {
	BotToken  string
	MongoURI  string
	MongoDB   string
	VerbsPath string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	c := Config{
		BotToken:  os.Getenv("BOT_TOKEN"),
		MongoURI:  getenv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:   getenv("MONGO_DB", "irregular_verbs"),
		VerbsPath: getenv("VERBS_PATH", "data/verbs.json"),
	}
	if c.BotToken == "" {
		return Config{}, fmt.Errorf("BOT_TOKEN is required")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
