# Env-driven config & first-class Docker support

Date: 2026-07-08
Status: Approved (design)

## Goal

Let Signpost run under Docker with **no config file** — only environment
variables for the operational essentials (Golbat URL, Golbat API password,
port). The link templates ship as built-in defaults baked into the binary, and
`config.toml` becomes an optional override rather than a hard requirement.

Existing `config.toml` deployments must keep working with identical behaviour.

## Non-goals

- No change to the redirect logic, templates output, or route shapes.
- No new config framework (Viper etc.); three env vars are read with the stdlib.
- No change to the Golbat 404 handling shipped previously.

## Configuration model

Three layers, lowest to highest precedence:

```
1. built-in defaults   (embedded defaults.toml: google/waze/apple templates + port 3035)
      ⤷ 2. config.toml   (OPTIONAL — overrides only what it specifies)
            ⤷ 3. environment variables   (win over everything)
```

### Environment variables

| Env var | Overrides | Required | Notes |
|---|---|---|---|
| `GOLBAT_URL` | `golbat.url` | yes | fail-fast if empty after all layers |
| `GOLBAT_API_PASSWORD` | `golbat.api_password` | no | |
| `GOLBAT_API_PASSWORD_FILE` | `golbat.api_password` | no | reads the secret from a file (Docker/K8s secrets); ignored if the non-`_FILE` var is set |
| `PORT` | `port` | no | defaults to `3035`; must parse as int or fail-fast |
| `CONFIG_FILE` | config path | no | defaults to `./config.toml` |

Precedence within the password: `GOLBAT_API_PASSWORD` > `GOLBAT_API_PASSWORD_FILE` > config.toml > default (empty).

### Embedded defaults (`defaults.toml`)

A new repo file `defaults.toml`, embedded via `//go:embed`, holding today's
example templates (the identical google/waze/apple set for pokemon/pokestop/gym)
and `port = 3035`. It is always loaded first, is human-readable, lives in git,
and documents the default links. `golbat` fields are left empty in defaults.

`config.toml.example` is slimmed to the `golbat` section plus a commented note
that template blocks are optional overrides.

## `config.Load` design

Extract loading out of `main.go` into a testable function in `config.go`:

```go
//go:embed defaults.toml
var defaultsTOML []byte

// Load builds the effective config from embedded defaults, an optional TOML
// file, then environment overrides, and validates the result.
func Load() (Config, error) {
    var cfg Config
    if err := toml.Unmarshal(defaultsTOML, &cfg); err != nil {
        return cfg, err // programmer error: embedded defaults must parse
    }

    path := getenvDefault("CONFIG_FILE", "config.toml")
    if b, err := os.ReadFile(path); err == nil {
        var file Config
        if err := toml.Unmarshal(b, &file); err != nil {
            return cfg, fmt.Errorf("parsing %s: %w", path, err)
        }
        cfg.merge(file)
    } else if !errors.Is(err, fs.ErrNotExist) {
        return cfg, fmt.Errorf("reading %s: %w", path, err) // exists but unreadable
    }

    if err := cfg.applyEnv(); err != nil {
        return cfg, err
    }
    cfg.TimestampFormat = "2006-01-02 15:04:05"
    return cfg, cfg.validate()
}
```

- `merge(file Config)` — explicit, to avoid ambiguous TOML slice-merge:
  - scalar fields (`Port`, `Golbat.Url`, `Golbat.ApiPassword`): overwrite when the
    file value is non-zero.
  - template categories (`Pokemon`, `Pokestop`, `Gym`): **replace** the whole
    category if the file provides any entries for it; otherwise keep defaults.
    (Matches today's semantics where `config.toml` fully defines a category.)
- `applyEnv() error` — `os.LookupEnv` for each var above; `PORT` parsed with
  `strconv.Atoi`, a parse failure returned as an error from `Load`.
- `validate()` — returns a clear error when `Golbat.Url` is empty, e.g.
  `GOLBAT_URL is required (set the env var or golbat.url in config.toml)`.

`main.go` calls `Load()` and `log.Fatal`s on error instead of panicking on a
missing file.

## Good-Docker-citizen behaviour

### Graceful shutdown
Run the server in a goroutine; use `signal.NotifyContext` for `SIGINT`/`SIGTERM`
and call `srv.Shutdown(ctx)` with a short timeout so `docker stop` drains
in-flight requests instead of hard-killing the process.

### Healthcheck (distroless-friendly)
- Add a `GET /healthz` route returning `200 OK`.
- Add a `-healthcheck` flag: the binary does a local HTTP GET to
  `http://127.0.0.1:<port>/healthz` and exits `0`/`1`. This lets the Dockerfile
  `HEALTHCHECK` work without a shell or curl (distroless has neither):

  ```dockerfile
  HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
    CMD ["/usr/src/app/signpost", "-healthcheck"]
  ```

  The healthcheck reads the same `PORT` resolution so it targets the right port.

### Dockerfile / compose / README
- Dockerfile: keep the multi-stage distroless build; add the `HEALTHCHECK`.
- `docker-compose.yml`: drop the config volume; set `environment:` with
  `GOLBAT_URL` / `GOLBAT_API_PASSWORD` (+ optional `PORT`). Mounting a
  `config.toml` remains documented for template overrides.
- README: new "Configuration" subsection documenting the env vars and the
  layering; update the Docker section to the no-file compose.

## CI: pin remaining action versions
Bump the two artifact actions that still emit the Node 20 warning on `master`
runs: `actions/upload-artifact@v4 → v7`, `actions/download-artifact@v4 → v8`.
The v8 change only affects a new "direct upload" mode, so the digest
upload/download pattern is unaffected; it is only fully exercised on a real
`master` push.

## Backwards compatibility

- A user with an existing `config.toml` (full templates + golbat + port) gets
  identical behaviour: the file overrides the defaults exactly as before.
- The default templates in `defaults.toml` are byte-for-byte the current
  `config.toml.example` templates, so the `/google`, `/waze`, `/apple` links the
  README documents are unchanged.

## Testing plan (TDD)

Unit tests for `config.Load` / `merge` / `applyEnv` / `validate`, using
`t.Setenv` and temp files:

1. Defaults only (no file, no env) → default templates present, port 3035,
   `validate` fails because `GOLBAT_URL` unset.
2. Env only (no file) → `GOLBAT_URL`/`PORT` applied; default templates present.
3. File overrides defaults; env overrides file (precedence).
4. `config.toml` defining only `[[pokemon]]` replaces pokemon set, keeps default
   pokestop/gym (confirms replace-per-category, not append).
5. `GOLBAT_API_PASSWORD_FILE` read from disk; plain var wins when both set.
6. Invalid `PORT` → clear error.
7. Missing file → no error; malformed file → error.

## File-by-file changes

- `defaults.toml` — new, embedded default templates + port.
- `config.go` — `//go:embed`, `Load`, `merge`, `applyEnv`, `validate`, helpers.
- `config_test.go` — new, the tests above.
- `main.go` — call `Load()`; `log.Fatal` on error; add `/healthz`, `-healthcheck`
  flag, graceful shutdown.
- `Dockerfile` — add `HEALTHCHECK`.
- `docker-compose.yml` — env-driven, no volume.
- `config.toml.example` — slimmed to golbat + note.
- `README.md` — Configuration + updated Docker section.
- `.github/workflows/docker-publish.yml` — bump artifact action versions.
- `.dockerignore` — new (exclude `.git`, `config.toml`, local binary).
