# NextTimer – Server + Terminal-Client (Bubble Tea v2 / Lip Gloss v2)

> v20: Schlanke Vollbildansicht ohne Kopfzeile und X-Shortcut; direkt per CLI gestartete Timer schließen nach Abschluss ohne FERTIG-Zwischenanzeige. Dauer/Typ erscheinen dezent hinter dem Timernamen. AWTRIX NG sendet nun zeitlich auf die Countdown-Sekunden abgestimmte Updates.

Zwei unabhängige Go-Binaries: `nexttimer-server` betreibt Timer-Engine, REST API, Web-UI/PWA, AWTRIX NG und optional den Alarm-Webhook. `nexttimer` ist ein schlanker CLI/TUI-Client, der über HTTP auf den Server zugreift. Mehrere Timer können parallel laufen; der noch laufende Timer mit der **frühesten Endzeit** ist automatisch Haupttimer.

## v21: Favoriten-Gruppen in der Web-UI

Favoriten können beim Speichern oder Bearbeiten einer Gruppe zugeordnet werden.
Ein vorhandener Gruppenname kann über Vorschläge gewählt oder frei eingegeben werden.
Leere Auswahl bedeutet **Standard**. Sobald mindestens eine benannte Gruppe
existiert, zeigt die Web-UI Gruppenüberschriften. Gruppen erscheinen in ihrer
Erstellreihenfolge, **Standard zuletzt**; eine leere Standardgruppe wird nicht
gezeigt. Besteht nur die Standardgruppe, entfallen die Gruppenüberschriften.

Im Modus **Favoriten bearbeiten** kann man eine Gruppe umbenennen. Dabei werden
alle zugehörigen Favoriten aktualisiert. Ein Favorit wechselt über sein
Bearbeitungsformular die Gruppe. Sortieren mit Griff/Pfeilen ist ausschließlich
innerhalb derselben Gruppe möglich. Leere Gruppen verschwinden automatisch;
es gibt keine zusätzliche Gruppenverwaltung. Die TUI zeigt die gewohnte flache
Favoritenliste und bleibt unverändert. Gruppen beeinflussen weder Timer
noch AWTRIX, Webhooks oder Benachrichtigungen.

Die Gruppenzuordnung und deren Erstellreihenfolge liegen in `templates.json`,
daher wird keine weitere Konfigurationsdatei benötigt. Vorhandene Favoriten
ohne Gruppe landen automatisch in **Standard**.

Neue API-Felder: `group` und `group_order` bei Favoriten, `groups` im
`GET /api/state`-Snapshot. Umbenennen: `PUT /api/groups/{name}` mit
`{"name":"Neuer Name"}`. `PUT /api/templates/order` akzeptiert nur eine
Sortierung, die die Gruppengrenzen respektiert.

**Entfernt:** „Ton testen“ und das optionale Unicode-/Emoji-Symbol, einschließlich
zugehöriger Felder im Datenmodell und der API. Vorhandene Icon-Werte werden
nicht übernommen oder migriert; mit der nächsten Speicherung werden sie nicht
mehr in `templates.json` ausgegeben. Der Signalton selbst bleibt verfügbar.

---

## v20: Countdown-Synchronisation mit AWTRIX NG

Bisher aktualisierte AWTRIX den Countdown über einen **unabhängigen Ein-Sekunden-Ticker**. Das konnte zu nahezu einer Sekunde sichtbarem Versatz zur TUI führen. Die Anzeige wird jetzt lokal alle 40 ms geprüft, aber nur dann an AWTRIX gesendet, wenn die dargestellte Sekunde wechselt, der Timer geändert wird oder die Benachrichtigung eine Auffrischung benötigt. Start/Pause/Fortsetzen/Zeitänderung lösen ein unmittelbares Update aus. Die lokale Überprüfung erzeugt **keine** zusätzlichen HTTP-Anfragen (weiterhin ungefähr eine pro Sekunde).

Optional kann eine kleine Übertragungslatenz kompensiert werden (Standard: **100ms**):

```yaml
AWTRIX_SYNC_LEAD: "100ms" # 0..500ms; z.B. 150ms, wenn AWTRIX noch hinterherhinkt
```

Mit `0ms` lässt sich die Vorverlegung abschalten. Die Einstellung beeinflusst **nur die Display-Anzeige**, nicht das tatsächliche Timer-Ende, den Webhook oder die Fertig-Notification. Da das Display über HTTP und den eigenen Bildschirm-Refresh arbeitet, ist millisekundengenaue Synchronität nicht garantiert. Der Terminal-Client berücksichtigt beim Abgleich zusätzlich ungefähr die halbe HTTP-Umlaufzeit.

## v20: Reduzierter Fokusmodus

* **Direkte CLI-Starts** (`nexttimer 5m Kaffee`, `nexttimer start Kaffee`, auch mit `--local`) beenden den Client, sobald der Countdown als abgeschlossen erkannt wird. Es gibt keine 2,5 Sekunden lange „FERTIG“-Einblendung mehr. Bei Server-Timern wird auf den bestätigten `finished`-Status gewartet, damit der automatische Cleanup nicht unbeabsichtigt den serverseitigen Fertig-Webhook verhindert.
* `Esc`, `q` und `Ctrl+C` verlassen den direkten CLI-Fokus weiterhin; standardmäßig wird nur der selbst gestartete Server-Timer gestoppt (`--keep-running` lässt ihn weiterlaufen). `--detach` startet ohne TUI.
* Im **Vollbild** entfällt `x` (kein Stop-Dialog mehr). In der **Übersicht** bleibt `x` zum gezielten Stoppen vorhandener Timer mit Bestätigung verfügbar.
* Die Zeile „NEXTTIMER / FOKUS“ und die Statusmeldung „LETZTE 10 SEKUNDEN“ entfallen. Es steht einfach „LÄUFT“ oder „PAUSIERT“ dort. Die rote Warnfarbe der Ziffern und des Fortschritts bleibt erhalten.
* Der Timername erhält einen dezenten Zusatz, z. B. `Kaffee (5 Min.)` bzw. `Kaffee (5 Min. · lokal)`. Angezeigt wird die aktuell konfigurierte Gesamtdauer, die sich bei einer Zeitänderung entsprechend mitändern kann.
* Wenn ein Timer **aus dem Dashboard** nur zum Beobachten geöffnet wird, bleibt dessen bisheriges Fertig-/Weiterverfolgen-Verhalten bestehen.

## v19: Termdown-artige Großziffern

Die Fokusansicht und `nexttimer watch` verwenden nun **einen eigenen, Termdown-inspirierten Vektor-Renderer** für `0` bis `9` und `:`. Anders als die bisherigen eckigen, vergrößerten 5×5-Blockziffern sind die Ziffern gezeichnete Kurven mit runden Enden (besonders `0`, `2`, `3`, `6`, `7` und `9`). Das Raster wird mit Unicode-Viertelblockzeichen dargestellt und automatisch an die Terminalgröße angepasst. Bei zu kleinen Fenstern wird die Zeit normal als Text ausgegeben.

* **Keine zusätzliche Laufzeitabhängigkeit**, kein Python, kein externes FIGlet-Programm und keine mitgelieferte Fontdatei.
* Die bestehende rote 10-Sekunden-Phase, die Fortschrittsanzeige, Shortcuts und die `--keep-running`/`--detach`-Regeln bleiben unverändert.
* Darstellungsdetails hängen wie bei anderen Terminalgrafiken von Terminalschriftart und Zeichenbreite ab. Eine Vorschau ist separat als PNG enthalten.

Der Renderer ist keine 1:1-Kopie eines Termdown-Fonts. Er nimmt sich den typografischeren, großflächigen Countdown als Vorbild, lässt aber die Go-App eigenständig und ohne Fremdcode.

## Docker Compose: Release und Entwicklung

`compose.yaml` verwendet das fertige **GHCR-Release-Image**; `compose.dev.yaml`
baut den Server für lokale Entwicklung aus dem Quellcode. Beide Dateien
verwenden denselben Service-Namen `nexttimer`, Port und ein persistentes
Docker-Volume `nexttimer_data` für Favoriten. Sie sind alternative Startarten;
bitte nicht gleichzeitig mit demselben Compose-Projekt starten.

### Release-Image verwenden (Standard)

```sh
cp .env.example .env             # Einmalig; anschließend .env anpassen
docker compose pull              # ghcr.io/desirevolution/next-timer:0.21.0
docker compose up -d
```

Die Image-Version steht als `NEXTTIMER_VERSION=0.21.0` in `.env` und kann
für neue Releases angepasst werden. `latest` ist ebenfalls möglich, aber ein
fester Versions-Tag ist reproduzierbarer. **Das Image wird erst verfügbar sein,
nachdem der Release-Workflow v0.21.0 erfolgreich veröffentlicht hat.**

### Lokale Entwicklungsvariante

```sh
cp .env.example .env             # falls noch nicht vorhanden
docker compose -f compose.dev.yaml up -d --build
# Logs ansehen:
docker compose -f compose.dev.yaml logs -f nexttimer
```

Für beide Varianten gilt: Web-UI unter `http://127.0.0.1:8080` (standardmäßig
nur lokal erreichbar). Mit `NEXTTIMER_BIND_ADDRESS=0.0.0.0` kannst du den
Port für andere Geräte freigeben; **dann brauchst du aber einen geeigneten
Zugriffsschutz**, denn die Timer-API hat keine eingebaute Authentifizierung.
Bei Caddy als Reverse Proxy empfiehlt sich die Default-Bind-Adresse.

Der Container führt ausschließlich `nexttimer-server` aus; der Terminal-Client
`nexttimer` wird separat als Binary veröffentlicht. `TIMER_DATA` liegt im
Container immer unter `/data/templates.json`, das benannte Docker-Volume
speichert die Favoriten. Laufende Timer bleiben flüchtig.

### Konfiguration über `.env`

Alle unterstützten Optionen mit Kommentaren stehen in `.env.example`.
Wenn `AWTRIX_URL` oder `TIMER_ALARM_WEBHOOK_URL` leer bleiben, ist die jeweilige
Integration deaktiviert. Netzwerkadressen für den Webhook müssen aus dem
Container erreichbar sein, nicht nur vom Host-Rechner.

Die beiden Compose-Dateien verwenden dieselbe `.env` und dasselbe Volume.
Bei einem Wechsel zwischen Dev und Release genügt ein `docker compose ... up -d`
mit der gewünschten Datei; achte bei bereits bestehenden Installationen
auf einen unveränderten Compose-Projektnamen, damit das Datenvolume wieder
verwendet wird.

## AWTRIX NG ohne Custom Apps und Berry-Skripte

`AWTRIX_URL` aktiviert den **Notification-only**-Adapter. Er nutzt ausschließlich:

- `POST /api/v1/notifications` für den Haupttimer (`name: nexttimer-live`, `stack: false`, Restzeit und Fortschrittsbalken). Etwa einmal pro Sekunde eine zur Server-Endzeit synchronisierte Notification. Die letzte 10-Sekunden-Phase wird rot.
- `POST /api/v1/notifications` mit konfigurierbarer Dauer (standardmäßig `durationMs: 5000`) und optionalem `Fireworks`-Effekt für einen fertig gewordenen Timer. Weitere gleichzeitig fertig werdende Timer werden dahinter eingereiht.
- `DELETE /api/v1/notifications/nexttimer-live` für Bereinigung, wenn kein Timer mehr läuft, beim Start und beim Herunterfahren.

Eine laufende Countdown-Notification hat `durationMs: 2500`, **kein `hold: true`**: Falls NextTimer unerwartet beendet wird, verschwindet die Anzeige von selbst nach etwa 2,5 Sekunden. Im Normalbetrieb wird sie sekündlich ersetzt. Das kann bei AWTRIX NG sichtbare Übergänge/Flackern verursachen; die Anzeige auf echter Hardware ist noch zu überprüfen. Andere Notifications können durch `stack:false` überlagert werden, wie bei der gewünschten Vordergrundanzeige vorgesehen.

Kein `/api/custom`, kein `/api/switch`, keine Skriptinstallation, keine Firmware-Erkennung, keine MQTT-Pflicht.

```yaml
# Compose-Umgebung (per .env.example konfigurierbar)
environment:
  AWTRIX_URL: "http://192.168.178.50"
  AWTRIX_FINISH_EFFECT: "ColorWaves" # oder "none", "Matrix", ...
  AWTRIX_FINISH_EFFECT_SPEED: "0.3" # 0.1–10; kleiner = langsamer
  AWTRIX_FINISH_DURATION: "10s" # 1s–5m
  AWTRIX_SYNC_LEAD: "100ms" # 0..500ms, optional
```

AWTRIX NG validiert Effektnamen. Ein nicht vorhandener Effekt wird mit HTTP 422 abgelehnt und im NextTimer-Log einschließlich Fehlerantwort angezeigt. Gültige Effekte: `GET http://AWTRIX-IP/api/v1/capabilities`.

## Ein optionaler Alarm-Webhook

Nur beim Übergang eines **Server-Timers** nach `finished` wird ein HTTP-POST ausgelöst. Pausieren, Fortsetzen, Starten, Stoppen und normale Countdown-Updates lösen **keinen** Webhook aus. Wird die Restzeit durch eine Änderung auf 0 gesetzt, ist das ebenfalls ein Timer-Ende.

```yaml
# Compose-Umgebung (per .env.example konfigurierbar)
environment:
  TIMER_ALARM_WEBHOOK_URL: "http://node-red:1880/nexttimer/finished"
```

**Beispiel-Body** (Werte beispielhaft):

```json
{
  "event": "timer.finished",
  "event_id": "3-1793190000000",
  "timestamp": "2026-10-28T12:20:00Z",
  "timer": {
    "id": "3",
    "name": "Kaffee",
    "duration_ms": 300000,
    "remaining_ms": 0,
    "status": "finished",
    "created_at_ms": 1793189700000,
    "finished_at_ms": 1793190000000,
    "sound": true,
    "notify": true
  }
}
```

Das Ereignis ist **best effort**: HTTP-Timeout nach vier Sekunden, HTTP-Fehler werden protokolliert, kein persistentes Retry. Die Timer-Engine wird beim Aufruf nicht blockiert. Bei einer späteren Erweiterung können wir Retry und Warteschlange ergänzen.

Wichtig: Browser-only-Timer laufen ausschließlich lokal in der PWA und erzeugen keinen Server-Webhook. Die Webhook-Auslösung ist unabhängig von den `sound`- und `notify`-Optionen für Browser-Benachrichtigungen.

## Konfiguration

Server-Konfiguration in `${XDG_CONFIG_HOME:-~/.config}/nexttimer/config.json`; die Client-URL ist separat in `client.json` gespeichert; Favoriten in `${XDG_DATA_HOME:-~/.local/share}/nexttimer/templates.json`.

```sh
nexttimer config init
nexttimer config check
nexttimer config show
nexttimer config path
```

```json
{
  "url": "http://127.0.0.1:8080",
  "listen": "127.0.0.1:8080",
  "awtrix_url": "http://192.168.178.50",
  "awtrix_finish_effect": "ColorWaves",
  "awtrix_finish_effect_speed": 0.3,
  "awtrix_finish_duration": "10s",
  "awtrix_sync_lead": "100ms",
  "alarm_webhook_url": "http://127.0.0.1:1880/nexttimer/finished",
  "finished_ttl": "60s"
}
```

Präzedenz: CLI-Flags > Umgebungsvariablen > Konfigurationsdatei > Standardwerte.

| Variable | Flag für `nexttimer-server` | Wirkung |
| --- | --- | --- |
| `AWTRIX_URL` | `--awtrix` | AWTRIX NG HTTP-Basisadresse (optional) |
| `AWTRIX_FINISH_EFFECT` | `--awtrix-finish-effect` | Fertig-Animation (`none` deaktiviert) |
| `AWTRIX_FINISH_EFFECT_SPEED` | `--awtrix-finish-effect-speed` | Effektgeschwindigkeit als Zahl (0.1–10, Standard 1) |
| `AWTRIX_FINISH_DURATION` | `--awtrix-finish-duration` | Anzeigedauer der Fertigmeldung (z.B. `10s`, Standard `5s`) |
| `TIMER_ALARM_WEBHOOK_URL` | `--alarm-webhook` | HTTP-Webhook bei `timer.finished` (optional) |
| `TIMER_LISTEN` | `--listen` | HTTP-Bind-Adresse |
| `TIMER_DATA` | `--data` | Favoriten-Datei |
| `TIMER_FINISHED_TTL` | `--finished-ttl` | Fertige Timer intern aufbewahren |

## CLI-Client und TUI (v15)

Die TUI (`nexttimer` ohne Argumente) hat eine an Lazygit angelehnte Panel-Struktur:
links **Favoriten** und **laufende Timer** als eigene umschaltbare Listen, rechts die
**Countdown-/Detailansicht**. Das ausgewählte Panel besitzt einen hervorgehobenen
Rahmen. `Tab` wechselt Panels, `j/k` oder Pfeiltasten bewegen die Auswahl,
`Enter` startet einen Favoriten oder zeigt einen Timer im Vollbild an.

Im Vollbildmodus bleiben die großen Countdown-Ziffern erhalten.
Mit `Esc` zurück zur Übersicht; `Space` pausiert/fortsetzt, `+/-` ändern
jeweils um eine Minute und `x` öffnet eine Stop-Bestätigung (`Enter`/`Esc`).
`1`–`9`, `0` starten Favoriten sofort im Fokusmodus.

### Server-URL des Clients

Der Client verwendet jetzt eine **eigene** XDG-Datei unter
`$XDG_CONFIG_HOME/nexttimer/client.json` (normalerweise
`~/.config/nexttimer/client.json`). Sie enthält nur `url` und wird getrennt
von der Server-Konfiguration verwaltet:

```bash
nexttimer config set-url http://192.168.1.10:8080
nexttimer config url
nexttimer config path
```

Die URL wird in der Reihenfolge `--url` > `TIMER_URL` > `client.json` >
bestehende `config.json` > `http://127.0.0.1:8080` ermittelt.
Der bestehende gemeinsame Konfigurationspfad bleibt als Fallback erhalten.

### Einmalbefehle ohne interaktive TUI

```bash
nexttimer --detach 5m Kaffee             # Timer starten, Meldung ausgeben, beenden
nexttimer --detach start Pomodoro        # Favorit starten, beenden
nexttimer --detach start '#1'            # Ersten Favoriten starten
nexttimer --url http://host:8080 --detach 20s Kurzpause
nexttimer ls                           # Status einmalig abfragen
```

Ohne `--detach` startet ein Daueraufruf (`nexttimer 5m Kaffee` oder
`nexttimer run 5m Kaffee`) bei einem interaktiven Terminal automatisch die
große Countdown-Ansicht. `start <Favorit>` öffnet dagegen wie ein direkter Timerstart
standardmäßig den großen Fokusmodus. Kein dieser Befehle startet selbstständig einen Server: Dieser
läuft unabhängig als `nexttimer-server`.

### Bauen

```bash
go mod tidy
go build -o nexttimer-server ./cmd/nexttimer-server
go build -o nexttimer ./cmd/nexttimer
# Oder: ./scripts/build-binaries-docker.sh  (AMD64 + ARM64)
```

### Tests

Die Timer-API, `--detach`, die XDG-URL-Datei und die TUI-Zustandslogik wurden
getestet. Die neue Paneldarstellung wurde gegen lokale API-Testhüllen für
Bubble Tea/Lip Gloss kompiliert; eine echte Terminal-/Docker-Build-Abnahme
ist noch erforderlich, da die externen Module in der Entwicklungsumgebung
nicht heruntergeladen werden konnten.

## v16: Client-only-Timer und optionale Übersicht in der interaktiven TUI

Mit `--local` wird ein Countdown **ausschließlich im laufenden Terminal-Client**
verwaltet. Der Client muss geöffnet bleiben. Ein lokaler Timer wird nicht an den
NextTimer-Server, AWTRIX NG oder den Alarm-Webhook übertragen. Lokale Timer
überstehen weder Client-Neustart noch das Schließen des Terminals (im Gegensatz
zu Browser-only-Timern der PWA, die lokal im Browser gespeichert werden).

```bash
nexttimer --local 5m Kaffee          # Ohne Server, Fokusansicht
nexttimer --local start Pomodoro     # Vorlage einmal vom Server laden, lokal ausführen
nexttimer --detach 10m Kaffee         # Server-Timer starten und sofort BEENDEN
nexttimer --detach start Pomodoro     # Server-Favorit starten und sofort BEENDEN
```

In der TUI: `n` öffnet die Schnelleingabe. `!5m Kaffee` erstellt einen lokalen
Timer (auch in der normalen Server-TUI). `Enter` startet und öffnet den Fokus,
`Alt+Enter` startet und bleibt in der Übersicht. In der Favoritenliste
starten `Alt+Enter` oder `Alt+1` bis `Alt+0` den ausgewählten Favoriten,
ohne in den Fokusmodus zu wechseln. Favoriten mit aktivem
`browser_only` werden in der TUI lokal gestartet, statt fälschlich über den
Server-Endpunkt angelegt zu werden.

`--local --detach` ist absichtlich nicht erlaubt: Ein lokaler Countdown existiert
nur solange der Client läuft. Für einen Hintergrund-Timer ist `--detach` ohne
`--local` vorgesehen.

`--detach` startet den Timer ohne TUI und lässt ihn auf dem Server laufen.

**Hinweis zur Technik:** Der Client läuft weiter mit Bubble Tea v2 und Lip Gloss v2.
Die lokalen Countdown-Zustände liegen ausschließlich im Speicher des Clients.
Go- und TUI-Tests wurden mit einer lokalen Testumgebung geprüft; ein echter
Build mit Bubble Tea v2 und der Docker-Build sind in der aktuellen Umgebung
wegen fehlender Go-Modul-Downloads noch nicht vollständig verifiziert.

## v17: Lebensdauer von per CLI gestarteten Timern

Ein direkter Aufruf wie `nexttimer 10m Kaffee`, `nexttimer run 10m Kaffee`
oder `nexttimer start Pomodoro` öffnet **nur den großen Fokusmodus**.
Nach Ablauf beendet sich der Client direkt (ohne „FERTIG“-Zwischenanzeige).
`Esc`, `q` und `Ctrl+C` beenden ebenfalls den Client. Ein direkt gestarteter
**Server-Timer wird dabei standardmäßig gestoppt**; Timer, die andere Nutzer
oder TUI-Ansichten gestartet haben, bleiben unberührt.

```bash
nexttimer 10m Kaffee                  # Fokus, beim Verlassen Timer stoppen
nexttimer --keep-running 10m Kaffee   # Fokus, beim Verlassen weiterlaufen lassen
nexttimer --keep-running start Kaffee # Auch für Server-Favoriten
nexttimer --detach 10m Kaffee         # Keine TUI, Server-Timer läuft weiter
nexttimer --detach start Kaffee       # Favorit ohne TUI starten
nexttimer --local 10m Kaffee          # Lokal im Client, endet beim Verlassen
nexttimer                            # Dashboard (verursacht kein Auto-Stop)
```

`--keep-running` gilt ausschließlich für direkt per CLI gestartete
Server-Timer. Zusammen mit `--detach` oder `--local` ist es nicht sinnvoll und
wird abgewiesen. `--once` und `--no-focus` wurden entfernt. Ohne TTY benötigt
ein direkt gestarteter Server-Timer ausdrücklich `--detach`.

Bei Netzwerkausfällen oder harten Abbrüchen (`SIGKILL`) ist das Stoppen
naturgemäß nicht garantiert. Bei normalem Exit erfolgt es per API-Aufruf.
Die Fokus-Ziffern haben ab v19 eigenständige, geschwungene Konturen statt vergrößerter Blockpixel.

## GitHub Releases (v0.21.0 und spätere Versionen)

Der Workflow `.github/workflows/release.yml` führt bei Pushes auf `main` und
Pull Requests die Go-Tests aus. Beim **Veröffentlichen eines GitHub Releases**
(z. B. `v0.21.0`) erstellt er zusätzlich:

- `ghcr.io/desirevolution/next-timer:0.21.0` sowie für stabile Releases `:latest`
  als Multi-Arch-Image (`linux/amd64` und `linux/arm64`), nur mit dem Server.
- Die vier GitHub-Release-Artefakte `nexttimer-linux-amd64.tar.gz`,
  `nexttimer-linux-arm64.tar.gz`, `nexttimer-server-linux-amd64.tar.gz`,
  `nexttimer-server-linux-arm64.tar.gz` und die Datei `SHA256SUMS`.

**Erst den vollständigen Quellcode einschließlich des Workflows auf `main` pushen,**
und den grünen CI-Test abwarten. Anschließend in GitHub unter Releases
`Draft a new release` wählen, einen neuen Tag `v0.21.0` von `main`
erstellen und das Release **veröffentlichen**. Die Artefakte werden danach vom
Workflow hochgeladen; das Tag allein löst diese Veröffentlichung nicht aus.

Ein Image abrufen:

```sh
docker pull ghcr.io/desirevolution/next-timer:0.21.0
```

Das Image enthält ausschließlich `nexttimer-server`. Der `nexttimer`-CLI/TUI-
Client liegt in den separaten Linux-Release-Archiven.

GitHub Actions benötigt für das Repository die Berechtigung, Pakete zu schreiben
und Release-Assets hochzuladen. Der Workflow verwendet dafür `GITHUB_TOKEN`;
weitere persönliche Tokens sind nicht erforderlich. Für private Pakete kann eine
zusätzliche Paket-Berechtigung beim ersten Publish nötig sein.
