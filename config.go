package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

//go:embed defaults.toml
var defaultsTOML []byte

type Config struct {
	Port            int                  `toml:"port"`
	Golbat          golbatConfiguration  `toml:"golbat"`
	Pokemon         []templateDefinition `toml:"pokemon"`
	Pokestop        []templateDefinition `toml:"pokestop"`
	Gym             []templateDefinition `toml:"gym"`
	TimestampFormat string               `toml:"-"`
}

type golbatConfiguration struct {
	Url         string `toml:"url"`
	ApiPassword string `toml:"api_password"`
}

type templateDefinition struct {
	Url  string `toml:"url"`
	Name string `toml:"name"`
}

var config Config

// Load builds the effective configuration from embedded defaults, then (later
// tasks) an optional config file and environment overrides, and validates it.
func Load() (Config, error) {
	var cfg Config
	if err := toml.Unmarshal(defaultsTOML, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing embedded defaults: %w", err)
	}

	cfg.TimestampFormat = "2006-01-02 15:04:05"
	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.Golbat.Url == "" {
		return errors.New("GOLBAT_URL is required (set the env var or golbat.url in config.toml)")
	}
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive, got %d", c.Port)
	}
	return nil
}

func getenvDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
