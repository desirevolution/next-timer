# NextTimer

Countdown timers with a web app, terminal client and optional AWTRIX NG display. Run several timers, group favorites and control server timers through the REST API. A standalone PWA runs entirely in the browser.

## Quick start

```sh
cp .env.example .env
docker compose up -d
```

Open <http://127.0.0.1:8080>. The Compose file uses `ghcr.io/desirevolution/next-timer:latest` and persists favorites and timer history in a Docker volume. To build locally instead:

```sh
docker compose -f compose.dev.yaml up -d --build
```

Don't start both Compose variants at once; they share the same port and volume. Set `NEXTTIMER_BIND_ADDRESS` and `NEXTTIMER_PORT` in `.env` if needed.

## Web app and standalone

Enter `5m Kaffee`, `1h30m Workshop` or `#1` (an assigned favorite). Server favorites are shared; **Browser only** timers stay on the current device and don't trigger server integrations. NextTimer Local is a standalone, installable PWA with browser-local timers, favorites and JSON backup. Build it with:

```sh
python3 scripts/build-static.py --output dist/static --version dev
```

Host the generated `dist/static/` directory on any static web server. GitHub Pages is deployed only when a **regular GitHub Release** is published (not on pushes or pre-releases). Release builds show their tag beside the app name; development builds show their build label.

The main display follows the next expiring timer unless you select another. Paused timers stay selectable; stop and delete actions ask for confirmation in the UI. Favorites can be reordered by dragging their handles. Assign optional unique shortcut numbers `#1`–`#0` in the favorite editor.

| Key | Action |
| --- | --- |
| `Enter` in input | Start a new timer or numbered favorite |
| `N` | Focus timer input |
| `Esc` | Clear input, then leave it; cancel a dialog |
| `←` / `→` | Switch displayed timer (outside forms/dialogs) |
| `P` / `S` | Pause/resume or confirm stop of displayed timer |
| `R` | Repeat the last expired or stopped timer |
| `?` | Shortcut help |

The clock icon in the input opens the last **five distinct started configurations**; selecting one repeats it immediately. `↑`/`↓` navigate that history in an empty focused input. Separately, `R` repeats the **most recently expired or stopped** timer, server-wide for server timers. The repeat button appears in the empty main timer area. Completion history survives server restarts; active server timers do not.

**Sound:** Settings → Sounds has separate optional progress cues (25/50/75%) and last-ten-seconds ticking, one volume slider, and previews. Progress cues require a configurable minimum duration of 1, 2, 5 or 10 minutes. Sound preferences are per browser, off by default, and excluded from backups. The final alarm is separate. Background audio depends on the browser; the terminal client is silent for progress cues. Language can be switched between Automatic, Deutsch and English in Settings.

## REST API

The server API is usable from Node-RED or `curl` over a trusted internal network. There is **no built-in authentication**; protect externally reachable instances using a reverse proxy and access controls.

### Quick entry

`POST /api/timers/quick` accepts the same timer syntax as the web input, including server-side numbered favorites. Both content types work:

```sh
# Text body
curl -i -X POST http://localhost:8080/api/timers/quick \
  -H 'Content-Type: text/plain' --data-binary '5m Kaffee'

# JSON body (only a string field named "input")
curl -i -X POST http://localhost:8080/api/timers/quick \
  -H 'Content-Type: application/json' -d '{"input":"#1"}'
```

Success returns `201 Created` with the existing timer object (including `id`). Each request starts a new timer; **there is no automatic retry or duplicate protection**. `#1`–`#0` resolve only server-side favorites; browser-only favorites and Standalone data aren't accessible to REST. Normal timers use the existing server defaults, numbered favorites use their stored options.

### Stop the next running timer

```sh
curl -i -X POST http://localhost:8080/api/timers/next/stop
```

Stops the **soonest-expiring running server timer**, ignoring paused timers and any timer that has already expired. Expirations are applied first; target selection and stop are atomic. This does **not** target a timer manually selected in a browser. Returns `200 OK` with the stopped timer object, or `204 No Content` when nothing is running. As with the existing ID-based stop endpoint, the returned object's `status` reflects its last state *before* removal. To stop a particular timer instead, use its ID.

### Other useful endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/timers` | Start with JSON `name`, `duration`, or `template_id` and options |
| `POST` | `/api/timers/{id}/pause` | Pause a specific timer |
| `POST` | `/api/timers/{id}/resume` | Resume a specific timer |
| `POST` | `/api/timers/{id}/stop` | Stop a specific timer |
| `POST` | `/api/timers/{id}/adjust` | Adjust with JSON `{"delta":"-1m"}` |
| `POST` | `/api/repeat` | Repeat last finished server timer |
| `GET` | `/api/state` | Snapshot (including `focus_id`, timers, favorites, history) |
| `GET` | `/api/events` | Server-sent state events (SSE) |

Favorites and group management also have `/api/templates` and `/api/groups` routes. For quick-entry errors, the API responds with JSON `{"code":"...","error":"..."}`: `422` for invalid input, `404 FAVORITE_NOT_FOUND` for an unassigned/server-inaccessible `#` number, `415` for unsupported content types, and `413` for oversized bodies. Other existing API endpoints retain their response formats.

### Node-RED example

Use an **Inject → Function → HTTP Request → Debug** flow. In the Function node:

```javascript
msg.method = 'POST';
msg.url = 'http://nexttimer:8080/api/timers/quick';
msg.headers = {'Content-Type': 'text/plain'};
msg.payload = '5m Kaffee';
return msg;
```

Set the HTTP Request node to take its method and URL from `msg.method` and `msg.url`, and parse the JSON response. To stop the next timer, send `POST` to `http://nexttimer:8080/api/timers/next/stop` with no body. Use the returned timer `id` and `/api/timers/{id}/stop` when targeting a specific timer. The hostname `nexttimer` works when Node-RED shares the Docker network; otherwise use an internally reachable address. Sequence requests in Node-RED if their order matters.

## Terminal client

Download `nexttimer` from [GitHub Releases](https://github.com/desirevolution/next-timer/releases) or build it from source. Configure the server once with `nexttimer config set-url http://127.0.0.1:8080` (or use `TIMER_URL` / `--url`).

```sh
nexttimer                         # Interactive TUI
nexttimer 10m "Coffee"            # Start with full-screen timer
nexttimer start "Pomodoro"        # Start a saved favorite
nexttimer ls                      # List timers
nexttimer watch                   # Watch the next server timer
nexttimer --detach 10m "Coffee"   # Start on server and exit
nexttimer --keep-running 10m      # Keep running after leaving the TUI
nexttimer --local 5m "Break"      # Client-only timer
```

Direct CLI full-screen timers stop on exit unless `--keep-running` is set; `--detach` doesn't open the TUI. Press `r` in the TUI to repeat the last completed timer (server-wide or current local session).

## Configuration and build

See [`.env.example`](.env.example) for all Docker settings. Main options:

| Variable | Purpose |
| --- | --- |
| `NEXTTIMER_BIND_ADDRESS` / `NEXTTIMER_PORT` | Published host address and port (default localhost:8080) |
| `TIMER_DATA` | Path for favorites and persistent completion/history files (server setting) |
| `TIMER_FINISHED_TTL` | How long expired server timers remain visible |
| `AWTRIX_URL` | Optional AWTRIX NG integration |
| `AWTRIX_FINISH_EFFECT` | Effect displayed when a timer expires |
| `TIMER_ALARM_WEBHOOK_URL` | Optional POST webhook for `timer.finished` |

AWTRIX and webhook are disabled when their URLs are empty. The webhook fires for regular expiry, not manual stop. To build binaries, use **Go 1.25+**:

```sh
go test ./...
go build -o nexttimer-server ./cmd/nexttimer-server
go build -o nexttimer ./cmd/nexttimer
```

Or build Linux AMD64/ARM64 binaries with `./scripts/build-binaries-docker.sh`.

## About this project

This project was built with AI assistance. I'm a lazy dev.

## License

[MIT](LICENSE).
