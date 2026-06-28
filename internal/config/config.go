package config

import "github.com/kelseyhightower/envconfig"

// Config holds runtime configuration from the environment.
type Config struct {
	BotToken  string `envconfig:"BOT_TOKEN" required:"true"`
	MongoURI  string `envconfig:"MONGO_URI" default:"mongodb://localhost:27017"`
	MongoDB   string `envconfig:"MONGO_DB" default:"irregular_verbs"`
	VerbsPath string `envconfig:"VERBS_PATH" default:"data/verbs.json"`
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return Config{}, err
	}
	return c, nil
}
