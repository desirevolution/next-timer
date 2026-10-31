# NextTimer

A small countdown timer service with a web interface and a terminal client.

Run multiple timers, save frequently used durations as favorites, and control timers from a browser or the command line. The main view automatically shows the timer due to finish next, until you select a different running or paused timer.

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

The web app lets you start and control timers, save favorites, and arrange favorites into groups. The interface is available in English and German: Settings → Language (below Data & backup) offers Automatic (browser language), Deutsch and English. The selection is saved locally and applied without reloading timers. Groups can be reordered in edit mode. Favorite names must be unique within a group, but can be reused in different groups. It can also be installed as a PWA.

When no running or paused timer exists, the empty main timer view cannot be focused or selected. A paused timer remains selectable so it can be resumed. Use the left/right arrows in the main timer view to switch between timers (the position indicator shows which one is selected). A manually selected timer stays visible even while paused; when it stops or finishes, the display returns to the soonest-running timer, or to the first paused timer if all are paused. Clicking **Anzeigen** in the additional-timers list selects that timer too. Stopping any timer asks for confirmation.

### Keyboard controls

The timer entry is focused automatically on desktop. After a timer starts, its entry loses focus so timer shortcuts work immediately; press `N` to type another timer. Ordinary Tab navigation remains available. The **⌨** button opens a shortcut reference.

| Keys | Action |
| --- | --- |
| `Enter` in the entry | Start a timer (e.g. `5m Tea`) or an assigned favorite (e.g. `#1`) |
| `N` | Focus the timer entry from anywhere outside a form/dialog |
| `Esc` | Clear the focused timer entry (if filled); otherwise release the current focus, or focus the entry from the unfocused page. In a dialog, close/cancel it |
| `←` / `→` outside form fields and dialogs | Switch to the previous/next displayed timer (wrap around) |
| `P` outside form fields | Pause/resume only the displayed timer |
| `S` outside form fields | Ask to stop only the displayed timer |
| `?` outside form fields | Open shortcut help |

In favorite edit mode, drag a favorite by its handle to reorder within its group; the layout stays a two-column grid on desktop and becomes a single-column list on narrow mobile screens. The order saves automatically; no separate up/down buttons are required. Each favorite can have one optional, unique `#1`–`#0` assignment, set in its edit form. Assignments remain fixed through favorite reordering and are stored with favorite data (including browser-only backups). Only confirmation dialogs (stop timer, delete favorite, restore backup) use `←` and `→` to focus **Cancel** and **Confirm** respectively; `Enter` activates the focused button, `Esc` cancels. Arrows in edit/settings forms keep their usual cursor and control behavior. Normal text-entry keys are never captured for shortcuts while typing. On mobile the input is **not** auto-focused; dialog action buttons are kept above the software keyboard.

Expired browser alarms appear in one compact scrollable list, with individual acknowledgement and **Acknowledge all** for persistent alarms. Timers marked **Browser only** run locally and do not use the server, AWTRIX, or the webhook. Browser notifications are available when permitted, but a local timer cannot guarantee an alarm while the app is closed.

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

## About this project

This project was built with AI assistance. I'm a lazy dev.

## License

[MIT](LICENSE).


### Repeat recent timers and release versions

The clock icon inside the quick-entry field opens the five most recently
started distinct timer configurations. Clicking an entry immediately starts
another timer with the same duration, name, and alarm settings. In an empty
focused entry, `↑` and `↓` navigate history; `Enter` starts the selected timer.

The server saves its shared history in `<TIMER_DATA>.recent.json` alongside
its favorites file, including timers started by the CLI. Browser-only timer
history remains in that browser's `localStorage`; the standalone edition keeps
its entire history there (and includes it in JSON backups). The history
menu supports clearing the list.

GitHub Pages deploys **only when a regular GitHub Release is published**, not
on pushes or pre-releases. The version shown beside NextTimer comes from the
release tag; development builds show `dev`. To build the static version use
`python3 scripts/build-static.py --version TAG`. Docker builds accept the
`NEXTTIMER_VERSION` build argument.
