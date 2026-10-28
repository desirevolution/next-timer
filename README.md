# NextTimer

A small countdown timer service with a web interface and a terminal client.

Run multiple timers, save frequently used durations as favorites, and control timers from a browser or the command line. The timer due to finish next is shown as the main timer.

The project provides two programs:

- `nexttimer-server` — HTTP API, web app, optional AWTRIX NG display and alarm webhook.
- `nexttimer` — CLI and interactive terminal UI.

## Docker

Copy the example configuration and start the server:

```sh
cp .env.example .env
docker compose pull
docker compose up -d
```

Open <http://127.0.0.1:8080>.

The default image is `ghcr.io/desirevolution/next-timer:0.21.0`. Change `NEXTTIMER_VERSION` in `.env` to use a different release. The image must be published to GHCR before it can be pulled.

To build from source instead:

```sh
docker compose -f compose.dev.yaml up -d --build
```

Both Compose files use the same settings and persistent Docker volume. Use one or the other, not both at once.

## Web app

The web app lets you start and control timers, save favorites, and arrange favorites into groups. It can also be installed as a PWA.

Timers marked **Browser only** run locally and do not use the server, AWTRIX, or the webhook. Browser notifications are available when permitted, but a local timer cannot guarantee an alarm while the app is closed.

## Terminal client

Download the `nexttimer` binary for your architecture from [GitHub Releases](https://github.com/desirevolution/next-timer/releases), or build it from source.

Set the server address once:

```sh
nexttimer config set-url http://127.0.0.1:8080
```

Typical commands:

```sh
nexttimer                         # Open the TUI
nexttimer 10m "Coffee"            # Start a timer in full-screen mode
nexttimer start "Pomodoro"        # Start a saved favorite
nexttimer ls                      # List timers
nexttimer watch                   # Watch the current main timer
nexttimer --local 5m "Break"      # Run a timer only in this client
nexttimer --detach 10m "Coffee"   # Start on the server and exit
nexttimer --keep-running 10m      # Leave the server timer running when quitting
```

A timer started directly from the CLI stops when you exit its full-screen view, unless you pass `--keep-running`. With `--detach`, no TUI is opened and the server timer continues. Local CLI timers require the client to stay open.

The client URL is stored in `~/.config/nexttimer/client.json` (or under `$XDG_CONFIG_HOME`). You can override it with `TIMER_URL` or `--url`.

## Configuration

All Docker settings are listed in [`.env.example`](.env.example). The most useful ones are:

| Variable | Purpose |
| --- | --- |
| `NEXTTIMER_VERSION` | GHCR image tag |
| `NEXTTIMER_BIND_ADDRESS` / `NEXTTIMER_PORT` | Host address and port |
| `AWTRIX_URL` | Enable AWTRIX NG countdown notifications |
| `AWTRIX_FINISH_EFFECT` | Effect shown when a timer finishes |
| `TIMER_ALARM_WEBHOOK_URL` | HTTP POST when a server timer finishes |

AWTRIX and the webhook are disabled when their URLs are empty. AWTRIX effect speed, duration, and timing offset can also be set in `.env`.

Favorites are stored in a persistent Docker volume. Running server timers are not restored after a server restart.

**Security:** NextTimer has no built-in authentication. The default Compose configuration binds to localhost. If you expose it over a network, put it behind a reverse proxy with appropriate access controls.

## Build from source

Requires Go 1.25 or newer.

```sh
go build -o nexttimer-server ./cmd/nexttimer-server
go build -o nexttimer ./cmd/nexttimer
```

For Docker-based Linux AMD64 and ARM64 builds:

```sh
./scripts/build-binaries-docker.sh
```

Tagged releases publish the server image to [GHCR](https://github.com/desirevolution/next-timer/pkgs/container/next-timer) and attach both binaries to the [GitHub release](https://github.com/desirevolution/next-timer/releases).
