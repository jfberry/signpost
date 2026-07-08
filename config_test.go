package main

import (
	"path/filepath"
	"testing"
)

// pointNoConfigFile makes Load look at a path that does not exist, so only
// the embedded defaults (and any env) apply.
func pointNoConfigFile(t *testing.T) {
	t.Helper()
	t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "absent.toml"))
}

func TestLoad_DefaultsOnly_HasTemplatesAndDefaultPort(t *testing.T) {
	pointNoConfigFile(t)

	// The env layer lands in Task 3, so validation may fail here for lack of a
	// Golbat URL; this test only asserts the embedded defaults were loaded.
	cfg, _ := Load()
	if cfg.Port != 3035 {
		t.Errorf("default port = %d, want 3035", cfg.Port)
	}
	for _, tc := range []struct {
		name string
		defs []templateDefinition
	}{{"pokemon", cfg.Pokemon}, {"pokestop", cfg.Pokestop}, {"gym", cfg.Gym}} {
		if len(tc.defs) != 3 {
			t.Errorf("%s templates = %d, want 3", tc.name, len(tc.defs))
		}
	}
}

func TestLoad_MissingGolbatURL_Errors(t *testing.T) {
	pointNoConfigFile(t)
	t.Setenv("GOLBAT_URL", "") // explicitly empty == missing

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when GOLBAT_URL is empty, got nil")
	}
}
