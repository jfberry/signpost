# Signpost
Signpost serves as a lightweight alternative to URL shorteners for Pokémon Go mappers. Its purpose is to make it more challenging for individuals to scrape and redistribute your data through coordinate feeds.

The functionality of Signpost revolves around handling requests for Pokémon data and redirecting clients to navigation URLs (such as Google Maps, Apple Maps, or Waze). To achieve this, it utilizes the "encounter ID" in the link, which allows users to obscure the coordinates of Pokémon data within the navigation URL.

By default, Pokémon notifications in [Poracle](https://github.com/KartulUdus/PoracleJS) are displayed in plaintext, revealing coordinates in URLs like the following example:
```
https://maps.google.com/maps?q=51.50150352191488,-0.14220178361437658
```

The link instead could look like: 
```
https://signpost.yourmap.com/pokemon/1782929313465823/google
```

# Requirements

* [go 1.25](https://go.dev/doc/install)
* [Golbat](https://github.com/UnownHash/Golbat)

# Installation

1. Git clone the repo `git clone https://github.com/jfberry/signpost.git`
2. `cp config.toml.example config.toml` & adjust config.toml accordingly.
3. `go build .`
4. `pm2 start ./signpost --name signpost`

⚠️ Signpost is a web server so you will need to self host it and put it behind a reverse proxy such as Caddy or Nginx.

# Docker

Prebuilt multi-arch (`amd64` + `arm64`) images are published to the GitHub Container Registry on every push to `master`:

```
ghcr.io/jfberry/signpost:latest
```

You can pin to a specific build using its commit tag, e.g. `ghcr.io/jfberry/signpost:sha-abc1234`.

### docker compose

1. `cp config.toml.example config.toml` & adjust `config.toml` accordingly.
2. A ready-to-use [`docker-compose.yml`](docker-compose.yml) is included in this repo:

```yaml
services:
  signpost:
    container_name: SignPost
    image: ghcr.io/jfberry/signpost:latest
    ports:
      - "3035:3035"
    volumes:
      - ./config.toml:/usr/src/app/config.toml
    restart: unless-stopped
```

3. `docker compose up -d`

To update to the latest image: `docker compose pull && docker compose up -d`.

⚠️ Signpost is a web server so you will still need to put it behind a reverse proxy such as Caddy or Nginx.

# Updating
1.  `pm2 stop signpost`
1. `git pull`
3. `go build .`
3. `pm2 restart signpost`

# Poracle DTS Changes
Update your DTS templates to use Signpost. The links in your templates should look something like this:

Pokemon:
```
[Google](<https://signpost.yourmap.com/pokemon/{{{encounter_id}}}/google>)
[Apple](<https://signpost.yourmap.com/pokemon/{{{encounter_id}}}/apple>)
[Waze](<https://signpost.yourmap.com/pokemon/{{{encounter_id}}}/waze>)
```

Pokestops:
```
[Google](<https://signpost.yourmap.com/pokestop/{{{pokestop_id}}}/google>)
[Apple](<https://signpost.yourmap.com/pokestop/{{{pokestop_id}}}/apple>)
[Waze](<https://signpost.yourmap.com/pokestop/{{{pokestop_id}}}/waze>)
```

Gyms:
```
[Google](<https://signpost.yourmap.com/gym/{{{gymId}}}/google>)
[Apple](<https://signpost.yourmap.com/gym/{{{gymId}}}/apple>)
[Waze](<https://signpost.yourmap.com/gym/{{{gymId}}}/waze>)
```
