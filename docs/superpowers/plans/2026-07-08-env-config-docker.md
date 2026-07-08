# Env-driven Config & First-Class Docker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let Signpost run under Docker with no config file — driven by `GOLBAT_URL`, `GOLBAT_API_PASSWORD`, and `PORT` env vars — with the link templates shipped as built-in defaults, while existing `config.toml` deployments keep identical behaviour.

**Architecture:** A layered config loader in `config.go` (`Load`) merges three sources in increasing precedence: embedded `defaults.toml` → optional `config.toml` → environment variables. `main.go` calls `Load`, fails fast on a clear error, adds a `/healthz` route + `-healthcheck` self-probe (distroless-friendly), and shuts down gracefully on `SIGTERM`.

**Tech Stack:** Go 1.25, gin, `github.com/pelletier/go-toml/v2`, `//go:embed` (stdlib). No new dependencies.

## Global Constraints

- Go module directive is `go 1.25.0`; the image builds with `CGO_ENABLED=0` (distroless static). Applies to every task.
- No new Go dependencies — env parsing uses the standard library only.
- The template blocks in `defaults.toml` must be **byte-for-byte identical** to the current `config.toml.example` template blocks (the `/google`, `/waze`, `/apple` links the README documents must not change).
- Backwards compatibility: a deployment with a full `config.toml` (port + golbat + all templates) must behave exactly as before.
- The healthcheck must not require a shell or curl (distroless has neither) — it is a mode of the `signpost` binary itself.
- Single Go package (`package main`); tests live in the same package.
- Precedence rules: env overrides `config.toml` overrides embedded defaults. For the password specifically: `GOLBAT_API_PASSWORD` > `GOLBAT_API_PASSWORD_FILE` > `config.toml` > empty.

---

### Task 1: Embedded defaults + `Load` (defaults layer) + validation

**Files:**
- Create: `defaults.toml`
- Modify: `config.go`
- Test: `config_test.go`

**Interfaces:**
- Consumes: nothing (first task).
- Produces:
  - `func Load() (Config, error)` — builds effective config; returns the populated `Config` even when the error is a validation failure.
  - `func (c *Config) validate() error`
  - `func getenvDefault(key, def string) string`
  - `Config` gains struct tag `toml:"-"` on `TimestampFormat`.

- [ ] **Step 1: Create the embedded defaults file**

Create `defaults.toml` (templates copied verbatim from `config.toml.example`; no `[golbat]` section; default port):

```toml
port = 3035

[[pokemon]]
name = "google"
url = "https://maps.google.com/maps?q={{.lat}},{{.lon}}&z=17"

[[pokemon]]
name = "waze"
url = "https://www.waze.com/ul?ll={{.lat}},{{.lon}}&navigate=yes&zoom=17"

[[pokemon]]
name = "apple"
url = "https://maps.apple.com/place?coordinate={{.lat}},{{.lon}}"

[[pokestop]]
name = "google"
url = "https://maps.google.com/maps?q={{.lat}},{{.lon}}&z=17"

[[pokestop]]
name = "waze"
url = "https://www.waze.com/ul?ll={{.lat}},{{.lon}}&navigate=yes&zoom=17"

[[pokestop]]
name = "apple"
url = "https://maps.apple.com/place?coordinate={{.lat}},{{.lon}}"

[[gym]]
name = "google"
url = "https://maps.google.com/maps?q={{.lat}},{{.lon}}&z=17"

[[gym]]
name = "waze"
url = "https://www.waze.com/ul?ll={{.lat}},{{.lon}}&navigate=yes&zoom=17"

[[gym]]
name = "apple"
url = "https://maps.apple.com/place?coordinate={{.lat}},{{.lon}}"
```

- [ ] **Step 2: Write the failing test**

Add to a new `config_test.go`:

```go
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
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./... -run TestLoad -v`
Expected: FAIL — `undefined: Load` / `undefined: templateDefinition` compile errors.

- [ ] **Step 4: Implement the defaults layer + validation**

Edit `config.go`. Add imports and the tag change, and append the loader:

```go
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
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./... -run TestLoad -v`
Expected: PASS (both tests).

- [ ] **Step 6: Commit**

```bash
git add defaults.toml config.go config_test.go
git commit -m "Add embedded default config and Load() defaults layer"
```

---

### Task 2: Optional `config.toml` overlay via explicit `merge`

**Files:**
- Modify: `config.go`
- Test: `config_test.go`

**Interfaces:**
- Consumes: `Load`, `Config`, `getenvDefault` from Task 1.
- Produces: `func (c *Config) merge(o Config)` — scalars overwrite when non-zero; each template category replaces the default set only when the file provides entries for it.

- [ ] **Step 1: Write the failing tests**

Add to `config_test.go`:

```go
import "os" // add alongside existing imports

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
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./... -run TestLoad_File -v`
Expected: FAIL — file contents are ignored (no overlay yet); `TestLoad_FileOverridesScalars` sees default port 3035.

- [ ] **Step 3: Implement the file overlay + merge**

In `config.go`, extend imports with `"io/fs"` and update `Load` to insert the file layer before returning, and add `merge`:

```go
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

	cfg.TimestampFormat = "2006-01-02 15:04:05"
	return cfg, cfg.validate()
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
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./... -run TestLoad -v`
Expected: PASS (Task 1 + Task 2 tests).

- [ ] **Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "Overlay optional config.toml onto defaults via explicit merge"
```

---

### Task 3: Environment-variable overlay (`applyEnv`)

**Files:**
- Modify: `config.go`, `config_test.go`

**Interfaces:**
- Consumes: `Load`, `merge`, `validate` from Tasks 1–2.
- Produces: `func (c *Config) applyEnv() error` — applies `GOLBAT_URL`, `GOLBAT_API_PASSWORD` / `GOLBAT_API_PASSWORD_FILE`, `PORT`; returns an error on unreadable password file or unparseable `PORT`.

- [ ] **Step 1: Write the failing tests**

Add to `config_test.go`:

```go
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
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./... -run TestLoad -v`
Expected: FAIL — env is not applied yet (`GOLBAT_URL`/`PORT`/password-file assertions fail).

- [ ] **Step 3: Implement `applyEnv` and wire it into `Load`**

In `config.go`, add `"strconv"` and `"strings"` imports, insert the env layer into `Load` (before setting `TimestampFormat`), and add `applyEnv`:

```go
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
```

(Delete the old `return cfg, cfg.validate()` tail from Task 2's `Load` so there is exactly one return path as shown.)

- [ ] **Step 4: Run to verify pass + full suite**

Run: `go test ./... -v`
Expected: PASS — all `TestLoad_*` plus the existing `TestGetJson_*`.

- [ ] **Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "Apply GOLBAT_URL/PASSWORD/PORT env overrides in config loader"
```

---

### Task 4: Wire `main.go` to `Load` with fail-fast

**Files:**
- Modify: `main.go`

**Interfaces:**
- Consumes: `Load` from Task 3; global `config`.
- Produces: `main` populates the global `config` from `Load` and `log.Fatal`s on error instead of panicking on a missing file.

- [ ] **Step 1: Replace the config-loading block**

In `main.go`, replace the `os.Open("config.toml")` → `toml.Unmarshal` → `TimestampFormat` block (roughly lines 20–34) with:

```go
	cfg, err := Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	config = cfg
```

Update the import block: remove `"io"`, `"os"`, and `"github.com/pelletier/go-toml/v2"` if now unused; add `"log"`. Keep `"fmt"`, `"net/http"`, `"text/template"`, `"time"`, and gin.

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: builds clean (no unused-import errors).

- [ ] **Step 3: Verify fail-fast with no config and no env**

Run:
```bash
env -u GOLBAT_URL CONFIG_FILE=/nonexistent ./signpost 2>&1 | head -1
```
(Build first with `go build -o signpost .`.)
Expected: `config: GOLBAT_URL is required (set the env var or golbat.url in config.toml)` and a non-zero exit.

- [ ] **Step 4: Verify it starts from env only**

Run:
```bash
CONFIG_FILE=/nonexistent GOLBAT_URL=http://localhost:9001 PORT=3999 ./signpost &
sleep 1
curl -s -o /dev/null -w "%{http_code}\n" "http://127.0.0.1:3999/pokemon/123/google"
kill %1
```
Expected: server logs "Starting server on port 3999"; the curl prints a redirect status (`301`) or `404`, proving the route is served without any config file.

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "Load config via Load() and fail fast with a clear message"
```

---

### Task 5: `/healthz` route and `-healthcheck` self-probe

**Files:**
- Modify: `main.go`

**Interfaces:**
- Consumes: `Load`, global `config`.
- Produces: a `GET /healthz` route returning `200`; a `-healthcheck` flag that probes `http://127.0.0.1:<port>/healthz` and exits `0`/`1`; `func runHealthcheck(port int) int`.

- [ ] **Step 1: Add the flag, probe, and route**

At the top of `main`, before `Load`:

```go
	healthcheck := flag.Bool("healthcheck", false, "probe /healthz on the local server and exit 0 (healthy) or 1")
	flag.Parse()
```

Immediately after computing `cfg, err := Load()`, handle healthcheck mode before the fatal check so a probe never needs a valid Golbat URL to report unhealthy cleanly:

```go
	if *healthcheck {
		if err != nil {
			os.Exit(1)
		}
		os.Exit(runHealthcheck(cfg.Port))
	}
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	config = cfg
```

Register the route alongside the others:

```go
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
```

Add `runHealthcheck` at file scope:

```go
func runHealthcheck(port int) int {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
```

Re-add `"flag"` and `"os"` to the imports (os was removed in Task 4; it is needed again here).

- [ ] **Step 2: Build**

Run: `go build -o signpost .`
Expected: builds clean.

- [ ] **Step 3: Verify healthz + probe end-to-end**

Run:
```bash
CONFIG_FILE=/nonexistent GOLBAT_URL=http://localhost:9001 PORT=3999 ./signpost &
sleep 1
curl -s -w " -> %{http_code}\n" http://127.0.0.1:3999/healthz
CONFIG_FILE=/nonexistent GOLBAT_URL=http://localhost:9001 PORT=3999 ./signpost -healthcheck; echo "probe exit: $?"
kill %1
```
Expected: `ok -> 200`, then `probe exit: 0`. Kill the server and run the probe again → `probe exit: 1`.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "Add /healthz route and -healthcheck self-probe"
```

---

### Task 6: Graceful shutdown on SIGTERM/SIGINT

**Files:**
- Modify: `main.go`

**Interfaces:**
- Consumes: the `srv *http.Server` built in `main`.
- Produces: server runs in a goroutine; `main` blocks on `signal.NotifyContext` and calls `srv.Shutdown` with a timeout.

- [ ] **Step 1: Replace the blocking `ListenAndServe`**

Replace the final `srv.ListenAndServe()` call with:

```go
	go func() {
		fmt.Printf("%s [] Starting server on port %d\n", time.Now().Format(config.TimestampFormat), config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	stop()

	fmt.Printf("%s [] Shutting down\n", time.Now().Format(config.TimestampFormat))
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
```

Remove the earlier standalone `fmt.Printf("... Starting server ...")` line (it now lives in the goroutine). Add imports `"context"`, `"os/signal"`, `"syscall"`.

- [ ] **Step 2: Build**

Run: `go build -o signpost .`
Expected: builds clean.

- [ ] **Step 3: Verify graceful shutdown**

Run:
```bash
CONFIG_FILE=/nonexistent GOLBAT_URL=http://localhost:9001 PORT=3999 ./signpost &
sleep 1
kill -TERM %1
wait %1; echo "exit: $?"
```
Expected: logs "Shutting down" and exits `0` (clean), not killed.

- [ ] **Step 4: Run the full test suite**

Run: `go test ./... && go vet ./...`
Expected: all pass, vet clean.

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "Shut the server down gracefully on SIGTERM/SIGINT"
```

---

### Task 7: Dockerfile HEALTHCHECK + `.dockerignore`

**Files:**
- Create: `.dockerignore`
- Modify: `Dockerfile`

**Interfaces:**
- Consumes: the `-healthcheck` flag from Task 5.
- Produces: image with a `HEALTHCHECK`; a lean build context.

- [ ] **Step 1: Create `.dockerignore`**

```
.git
.github
.idea
docs
config.toml
signpost
shortlink
*_test.go
```

- [ ] **Step 2: Add HEALTHCHECK to the runtime stage**

In `Dockerfile`, after the `EXPOSE 3035` line and before `ENTRYPOINT`:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD ["/usr/src/app/signpost", "-healthcheck"]
```

- [ ] **Step 3: Verify the image builds (if a Docker daemon is available)**

Run: `docker build -t signpost:healthcheck-test .`
Expected: builds successfully; the final stage is distroless. If no daemon is running locally, note that the branch's PR CI builds the image on both arches — rely on that.

- [ ] **Step 4: Commit**

```bash
git add .dockerignore Dockerfile
git commit -m "Add distroless HEALTHCHECK and .dockerignore"
```

---

### Task 8: docker-compose, `config.toml.example`, README

**Files:**
- Modify: `docker-compose.yml`, `config.toml.example`, `README.md`

**Interfaces:**
- Consumes: env-var contract from Tasks 1–3.
- Produces: user-facing docs and compose that need no config file.

- [ ] **Step 1: Env-drive `docker-compose.yml`**

Replace the service body so it needs no volume:

```yaml
services:
  signpost:
    container_name: SignPost
    image: ghcr.io/jfberry/signpost:latest
    environment:
      GOLBAT_URL: "http://golbat:9001"
      GOLBAT_API_PASSWORD: "golbat"
      # PORT: "3035"   # optional, defaults to 3035
    ports:
      - "3035:3035"
    restart: unless-stopped
    # To override the default links, mount a config.toml:
    # volumes:
    #   - ./config.toml:/usr/src/app/config.toml
```

- [ ] **Step 2: Slim `config.toml.example`**

Replace its contents with the golbat section plus a note that templates are optional overrides:

```toml
# Only these are needed; everything else has built-in defaults.
# All values can also be set via env vars: PORT, GOLBAT_URL, GOLBAT_API_PASSWORD.
port = 3035

[golbat]
url = "http://127.0.0.1:9001"
api_password = "golbat"

# Optional: override the default map links per entity type. Defining any
# [[pokemon]] blocks replaces the built-in pokemon links entirely (same for
# [[pokestop]] and [[gym]]). Defaults are google / waze / apple.
#
# [[pokemon]]
# name = "google"
# url = "https://maps.google.com/maps?q={{.lat}},{{.lon}}&z=17"
```

- [ ] **Step 3: Add a Configuration section to `README.md`**

Insert before the `# Docker` section:

```markdown
# Configuration

Signpost reads configuration from three layers, each overriding the one before:

1. **Built-in defaults** — the google / waze / apple links and port `3035`, baked into the binary.
2. **`config.toml`** — optional; only needed to override defaults (e.g. custom links). Path overridable with `CONFIG_FILE`.
3. **Environment variables** — take precedence over everything:

| Variable | Default | Purpose |
|---|---|---|
| `GOLBAT_URL` | — (required) | Golbat API base URL |
| `GOLBAT_API_PASSWORD` | empty | Golbat API secret |
| `GOLBAT_API_PASSWORD_FILE` | — | read the secret from a file (Docker/K8s secrets) |
| `PORT` | `3035` | listening port |

Because the links have defaults, a container needs only `GOLBAT_URL` (and usually `GOLBAT_API_PASSWORD`) to run — no config file required.
```

Then update the `# Docker` section's compose block to the env-driven one from Step 1 (drop the volume; mention mounting `config.toml` only for custom links).

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml config.toml.example README.md
git commit -m "Document env-driven config; compose needs no config file"
```

---

### Task 9: Bump remaining workflow action versions

**Files:**
- Modify: `.github/workflows/docker-publish.yml`

**Interfaces:**
- Consumes: nothing.
- Produces: no remaining Node 20 deprecation warnings on `master` runs.

- [ ] **Step 1: Bump the artifact actions**

In `.github/workflows/docker-publish.yml`:
- `actions/upload-artifact@v4` → `actions/upload-artifact@v7`
- `actions/download-artifact@v4` → `actions/download-artifact@v8`

- [ ] **Step 2: Lint the workflow**

Run: `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/docker-publish.yml; echo "exit: $?"`
Expected: `exit: 0`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/docker-publish.yml
git commit -m "Bump artifact actions to Node 24 majors (v7/v8)"
```

---

## Verification (whole feature)

- [ ] `go test ./...` — all green (config loader + json).
- [ ] `go vet ./...` — clean.
- [ ] `go build -o signpost .` then run with only `GOLBAT_URL` set → server starts, `/healthz` returns 200, `SIGTERM` shuts down cleanly.
- [ ] Push the branch; PR CI builds the distroless image (with HEALTHCHECK) on amd64 + arm64 and shows no Node 20 annotations.
- [ ] Open a PR; on merge, the first `master` run publishes and the healthcheck-enabled image is live.

## Notes for the reviewer

- The `config` global stays (handlers in `pokemon.go`/`pokestop.go`/`gym.go` read it); only its *population* moves into `Load`.
- The healthcheck and merge/publish job only run for real on a `master` push, so the runtime image behaviour (nonroot reading env, healthcheck) is first fully exercised after merge — call this out in the PR.
