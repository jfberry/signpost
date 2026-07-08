package main

import (
	"os"
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

// writeConfig writes a temp config file and points CONFIG_FILE at it.
func writeConfig(t *testing.T, body string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", p)
}

func TestLoad_FileOverridesScalars(t *testing.T) {
	writeConfig(t, `
port = 8080
[golbat]
url = "http://file:1234"
api_password = "filepw"
`)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 || cfg.Golbat.Url != "http://file:1234" || cfg.Golbat.ApiPassword != "filepw" {
		t.Errorf("scalars not applied from file: %+v", cfg)
	}
}

func TestLoad_FileReplacesOnlyDefinedCategory(t *testing.T) {
	writeConfig(t, `
[golbat]
url = "http://file:1234"

[[pokemon]]
name = "google"
url = "http://custom/{{.lat}}"
`)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Pokemon) != 1 || cfg.Pokemon[0].Url != "http://custom/{{.lat}}" {
		t.Errorf("pokemon should be replaced by file, got %+v", cfg.Pokemon)
	}
	if len(cfg.Pokestop) != 3 || len(cfg.Gym) != 3 {
		t.Errorf("untouched categories should keep defaults: pokestop=%d gym=%d", len(cfg.Pokestop), len(cfg.Gym))
	}
}

func TestLoad_MalformedFile_Errors(t *testing.T) {
	writeConfig(t, "port = = broken")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for malformed config file")
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	writeConfig(t, `
port = 8080
[golbat]
url = "http://file:1234"
api_password = "filepw"
`)
	t.Setenv("GOLBAT_URL", "http://env:9999")
	t.Setenv("PORT", "7000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Golbat.Url != "http://env:9999" {
		t.Errorf("GOLBAT_URL not applied: %q", cfg.Golbat.Url)
	}
	if cfg.Port != 7000 {
		t.Errorf("PORT not applied: %d", cfg.Port)
	}
	if cfg.Golbat.ApiPassword != "filepw" {
		t.Errorf("password should fall back to file value: %q", cfg.Golbat.ApiPassword)
	}
}

func TestLoad_PasswordFile(t *testing.T) {
	pointNoConfigFile(t)
	t.Setenv("GOLBAT_URL", "http://golbat:9001")
	pw := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(pw, []byte("s3cret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOLBAT_API_PASSWORD_FILE", pw)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Golbat.ApiPassword != "s3cret" {
		t.Errorf("password from file = %q, want %q", cfg.Golbat.ApiPassword, "s3cret")
	}
}

func TestLoad_PlainPasswordWinsOverFile(t *testing.T) {
	pointNoConfigFile(t)
	t.Setenv("GOLBAT_URL", "http://golbat:9001")
	t.Setenv("GOLBAT_API_PASSWORD", "plain")
	t.Setenv("GOLBAT_API_PASSWORD_FILE", "/does/not/matter")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Golbat.ApiPassword != "plain" {
		t.Errorf("plain password should win, got %q", cfg.Golbat.ApiPassword)
	}
}

func TestLoad_InvalidPort_Errors(t *testing.T) {
	pointNoConfigFile(t)
	t.Setenv("GOLBAT_URL", "http://golbat:9001")
	t.Setenv("PORT", "not-a-number")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for non-numeric PORT")
	}
}
