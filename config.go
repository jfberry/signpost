package main

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

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

// Load builds the effective configuration from embedded defaults, then an
// optional config file, then environment overrides, and validates the result.
func Load() (Config, error) {
	var cfg Config
	if err := toml.Unmarshal(defaultsTOML, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing embedded defaults: %w", err)
	}

	path := getenvDefault("CONFIG_FILE", "config.toml")
	if b, err := os.ReadFile(path); err == nil {
		var file Config
		if err := toml.Unmarshal(b, &file); err != nil {
			return cfg, fmt.Errorf("parsing %s: %w", path, err)
		}
		cfg.merge(file)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return cfg, fmt.Errorf("reading %s: %w", path, err)
	}

	if err := cfg.applyEnv(); err != nil {
		return cfg, err
	}
	cfg.TimestampFormat = "2006-01-02 15:04:05"
	return cfg, cfg.validate()
}

func (c *Config) applyEnv() error {
	if v, ok := os.LookupEnv("GOLBAT_URL"); ok {
		c.Golbat.Url = v
	}
	if v, ok := os.LookupEnv("GOLBAT_API_PASSWORD"); ok {
		c.Golbat.ApiPassword = v
	} else if f, ok := os.LookupEnv("GOLBAT_API_PASSWORD_FILE"); ok {
		b, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("reading GOLBAT_API_PASSWORD_FILE: %w", err)
		}
		c.Golbat.ApiPassword = strings.TrimSpace(string(b))
	}
	if v, ok := os.LookupEnv("PORT"); ok {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid PORT %q: %w", v, err)
		}
		c.Port = p
	}
	return nil
}

func (c *Config) merge(o Config) {
	if o.Port != 0 {
		c.Port = o.Port
	}
	if o.Golbat.Url != "" {
		c.Golbat.Url = o.Golbat.Url
	}
	if o.Golbat.ApiPassword != "" {
		c.Golbat.ApiPassword = o.Golbat.ApiPassword
	}
	if len(o.Pokemon) > 0 {
		c.Pokemon = o.Pokemon
	}
	if len(o.Pokestop) > 0 {
		c.Pokestop = o.Pokestop
	}
	if len(o.Gym) > 0 {
		c.Gym = o.Gym
	}
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
