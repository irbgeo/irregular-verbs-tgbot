package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type dataset struct {
	SchemaVersion int      `json:"schema_version"`
	Levels        []string `json:"levels"`
	Verbs         []Verb   `json:"verbs"`
}

// LoadVerbs reads and parses the verb dataset from a JSON file.
func LoadVerbs(path string) ([]Verb, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("service: read %s: %w", path, err)
	}
	var ds dataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, fmt.Errorf("service: parse %s: %w", path, err)
	}
	if len(ds.Verbs) == 0 {
		return nil, fmt.Errorf("service: no verbs in %s", path)
	}
	return ds.Verbs, nil
}

// SeedVerbs upserts all verbs through the verb repository.
func SeedVerbs(ctx context.Context, repo VerbRepository, verbs []Verb) error {
	for _, v := range verbs {
		if err := repo.Upsert(ctx, v); err != nil {
			return err
		}
	}
	return nil
}
