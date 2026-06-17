# NavioDeck

A self-hosted homelab start page. Widgets are dragged and resized on a free-form grid on desktop; on mobile they reflow into a scrollable single-column layout in their visual order (top → bottom, left → right).

Built with **SvelteKit 5** (frontend) + **Go** (backend), shipped as a single Docker image.

---

## Features

- **Services widget** — app launcher tiles with icons from [dashboard-icons](https://github.com/walkxcode/dashboard-icons), configurable sizes (small / medium / large)
- **Bookmarks widget** — link list with configurable columns, gap, and icon fallback to favicon
- **Clock widget** — digital clock with optional date, timezone, and 12/24 h format
- **Weather widget** — current weather + 5-day forecast via Open-Meteo (no API key required)
- **News widget** — live Tagesschau headlines (ARD public news API, German)
- **Docker widget** — live container status from a local or remote Docker socket
- **Garbage widget** — upcoming collection dates via any iCal/ICS URL or local file
- **Fuel widget** — live fuel prices via Tankerkönig API (Germany)
- **Calendar widget** — upcoming events from any iCal/ICS source, e.g. Google Calendar (no API key required)
- **Dark / Light mode toggle** — ☀/🌙 button in the header bar, default dark, persisted in browser
- **Drag & resize** — GridStack-powered free layout, saved to YAML automatically
- **Live reload** — dashboard reloads in all open tabs when `dashboard.yaml` changes on disk
- **Theme editor** — colours, glass blur, border radius, font, wallpaper
- **Per-widget styling** — background, border, title colour, opacity, badge colours for news
- **Per-item styling** — icon size, text size, font family, text alignment, colours
- **Item reordering** — up/down arrows and direct position input (type a number to jump)
- **Password protection** — optional, configured in `data/config.yaml` (or `DASHBOARD_PASSWORD` env var)
- **Mobile responsive** — widgets stack in their desktop visual order, panels open full-width

---

## Quick Start

```bash
# Clone or copy docker-compose.yml
mkdir naviodeck && cd naviodeck
curl -O https://raw.githubusercontent.com/youruser/naviodeck/main/docker-compose.yml

# Start (creates ./data/dashboard.yaml on first run)
docker compose up -d

# Open
open http://localhost:3080
```

On first start the container creates `./data/dashboard.yaml` with example widgets. Edit the file directly or use the in-browser editor (click **✏ Bearbeiten**).

---

## docker-compose.yml

```yaml
services:
  naviodeck:
    image: naviodeck:latest
    container_name: naviodeck
    restart: unless-stopped
    ports:
      - "3080:8080"
    volumes:
      - ./data:/data          # dashboard.yaml lives here
    environment:
      - TZ=Europe/Berlin
      # Optional password protection:
      # - DASHBOARD_PASSWORD=my-secret-password
    # Traefik labels (optional):
    #labels:
    #  - "traefik.enable=true"
    #  - "traefik.http.routers.naviodeck.rule=Host(`naviodeck.example.com`)"
    #  - "traefik.http.routers.naviodeck.entrypoints=websecure"
    #  - "traefik.http.routers.naviodeck.tls.certresolver=letsencrypt"
```

---

## Configuration

The dashboard is stored in `/data/dashboard.yaml` (mounted via Docker volume). It is read and written live — no restart needed.

### Minimal example

```yaml
theme:
  background_color: "#0f172a"
  card_color: "rgba(30, 41, 59, 0.85)"
  primary_color: "#818cf8"
  accent_color: "#fb923c"
  text_color: "#f1f5f9"
  glass_effect: true
  border_radius: 14
  font: Inter

widgets:
  - id: clock-1
    type: clock
    title: ""
    position: { x: 0, y: 0, w: 3, h: 3 }
    config:
      format: 24h
      show_date: true
      timezone: Europe/Berlin

  - id: services-1
    type: services
    title: Services
    position: { x: 3, y: 0, w: 9, h: 4 }
    items:
      - name: Portainer
        url: http://localhost:9000
        icon: portainer
        description: Container Management
      - name: Nextcloud
        url: http://localhost:8080
        icon: nextcloud
```

---

## Widget Reference

### Common position fields

| Field | Type | Description |
|-------|------|-------------|
| `x` | int | Column (0–11, grid is 12 columns) |
| `y` | int | Row |
| `w` | int | Width in columns |
| `h` | int | Height in rows (1 row ≈ 80 px) |

---

### `services` — App launcher

```yaml
type: services
title: My Apps
position: { x: 0, y: 0, w: 6, h: 4 }
items:
  - name: Portainer
    url: http://nas:9000
    icon: portainer          # name from dashboard-icons, URL, or /local/path.png
    description: Docker UI
    size: medium             # small | medium (default) | large
    target: _blank           # link target (_blank default)
```

**Per-item style fields** (all optional):

| Field | Description |
|-------|-------------|
| `size` | `small` / `medium` / `large` |
| `item_width` | Fixed tile width in px |
| `item_height` | Min tile height in px |
| `name_font_size` | Title font size in px |
| `desc_font_size` | Description font size in px |
| `font_family` | Font name (e.g. `Roboto`) |
| `text_align` | `left` / `center` / `right` |
| `text_color` | CSS colour |
| `desc_color` | Description text colour |
| `bg_color` | Tile background colour |
| `border_color` | Tile border colour |
| `icon_bg_color` | Icon box background colour |

---

### `bookmarks` — Link list

```yaml
type: bookmarks
title: Links
position: { x: 0, y: 4, w: 3, h: 5 }
config:
  columns: 1         # 1 = single column; 0 = auto-fill grid
  gap: 4             # gap between items in px
  # grid_cols: 3     # fixed columns when columns: 0
  # grid_min_w: 120  # min item width when columns: 0
items:
  - name: GitHub
    url: https://github.com
    icon: github
    description: Code Hosting
    size: medium     # small | medium (default) | large
```

---

### `clock` — Digital clock

```yaml
type: clock
title: ""
position: { x: 0, y: 0, w: 3, h: 3 }
config:
  format: 24h              # 24h | 12h
  show_date: true
  timezone: Europe/Berlin  # IANA timezone
```

---

### `weather` — Current weather + forecast

Uses [Open-Meteo](https://open-meteo.com/) — no API key required.

```yaml
type: weather
title: Wetter
position: { x: 3, y: 4, w: 3, h: 3 }
config:
  city: Berlin             # city name (geocoded automatically)
  units: celsius           # celsius | fahrenheit
  forecast_size: normal    # compact | normal | large
```

---

### `news` — Tagesschau headlines (ARD) or custom RSS/Atom feeds

Two modes, selected via `mode`:

**Tagesschau** (`mode: tagesschau`, default) — live ARD public news:

```yaml
type: news
title: Nachrichten
position: { x: 6, y: 4, w: 6, h: 5 }
config:
  mode: tagesschau
  ressort: ""          # inland | ausland | wirtschaft | sport | … (empty = all)
  regions: []          # region IDs, e.g. [1, 9] (empty = all)
  page_size: 10        # number of articles
  refresh_interval: 10 # refresh every N minutes
```

> ⚠️ **Legal note on the Tagesschau mode.** This calls `tagesschau.de/api2u` — ARD's **own**
> backend that powers the official Tagesschau app/site, but **not an officially published or
> supported public developer API**. It is undocumented by ARD (the documentation comes from the
> community project [bund.dev](https://tagesschau.api.bund.dev/)) and may change or stop working
> at any time. Tagesschau's general content terms allow use only for **private, non-commercial
> purposes** and **prohibit publishing/redistributing** the content, except material explicitly
> under a Creative Commons licence:
>
> > *"Die Nutzung der Inhalte für den privaten, nicht-kommerziellen Gebrauch ist gestattet,
> > die Veröffentlichung hingegen nicht – mit Ausnahme von Angeboten, die explizit unter der
> > CC-Lizenz stehen."* — and *"Es ist unzulässig, mehr als 60 Abrufe pro Stunde zu tätigen."*
>
> What this means for you as a self-hoster:
> - The Tagesschau widget is **off by default** — you opt in by adding a `news` widget in `tagesschau` mode.
> - Keep it to **private, non-commercial** use. For commercial/public displays, use the `rss` mode with feeds you are licensed to use, or content from [tagesschau.de/creativecommons](https://www.tagesschau.de/creativecommons).
> - Each instance fetches directly for its own user; nothing is re-served centrally. Stay under **60 requests/hour** — the default 10-minute refresh (6/h per widget) is well within that; avoid very short refresh intervals or many parallel news widgets.
>
> This project is **not affiliated with or endorsed by ARD/Tagesschau**, and this note is not legal advice.

**RSS/Atom** (`mode: rss`) — any number of feeds, merged and sorted by date:

```yaml
config:
  mode: rss
  rss_sources:
    - { url: "https://www.heise.de/rss/heise.rdf", name: "Heise" }
    - { url: "https://rss.golem.de/rss.php?feed=ATOM1.0", name: "Golem" }
  max_items: 20
  refresh_interval: 10
```

**Widget-level badge styling** (via `style` map):

| Key | Description |
|-----|-------------|
| `topline_color` | Badge text colour |
| `topline_bg` | Badge background colour |
| `topline_font_size` | Badge font size in px |
| `topline_bold` | `"0"` = normal weight, `"1"` = bold (default) |
| `topline_italic` | `"1"` = italic |
| `topline_border` | `"0"` = no border, `"1"` = border (default) |

---

### `docker` — Container status

Displays running/stopped container counts and a per-container list. Reads from Docker's API socket directly — no extra agent needed.

```yaml
type: docker
title: Docker
position: { x: 0, y: 4, w: 3, h: 4 }
config:
  endpoint: unix:///var/run/docker.sock   # or tcp://host:2375
  show_stopped: false                     # include stopped containers
  max_items: 10
```

> **Docker socket access:** Mount `/var/run/docker.sock:/var/run/docker.sock:ro` in your `docker-compose.yml`.

---

### `garbage` — Waste collection calendar

Fetches an iCal/ICS file (URL or local file) and shows upcoming collection dates with auto-detected icons for common waste types.

```yaml
type: garbage
title: Müllkalender
position: { x: 3, y: 4, w: 3, h: 3 }
config:
  source: "https://..."   # https:// URL or filename in /data (e.g. collection.ics)
  days_ahead: 30
  max_items: 10
  show_next_only: false   # true = show only the next event in large format
```

Local ICS files are placed in the `/data` directory of the container (same volume as `dashboard.yaml`).

---

### `calendar` — Calendar events (Google Calendar, iCal)

Shows upcoming events from any iCal/ICS source. Works with Google Calendar, Nextcloud, Apple iCloud, and any standard `.ics` feed — no API key or OAuth required.

```yaml
type: calendar
title: Kalender
position: { x: 6, y: 4, w: 3, h: 4 }
config:
  source: "https://calendar.google.com/calendar/ical/…/basic.ics"
  days_ahead: 30    # how many days ahead to fetch
  max_items: 20     # maximum number of events shown
```

**How to get the Google Calendar ICS URL:**
1. Open Google Calendar → Settings (⚙) → select your calendar on the left
2. Scroll down to **"Kalenderadresse"** → click **"Geheime Adresse im iCal-Format"** → copy the URL

Events are grouped by day. Timed events show their start and end time; all-day events show "Ganztag". Today and tomorrow are highlighted in colour. The location is shown below the title when present. Data is refreshed every 30 minutes.

---

### `fuel` — Fuel prices (Germany)

Live petrol prices via [Tankerkönig](https://creativecommons.tankerkoenig.de/) (free API key).

The key may be left blank here and supplied **once** server-side via `data/config.yaml` (`tankerkoenig_api_key`) — see [Secrets & Server Configuration](#secrets--server-configuration). A per-widget key still takes precedence if set.

```yaml
type: fuel
title: Benzinpreise
position: { x: 9, y: 4, w: 3, h: 4 }
config:
  api_key: ""          # leave empty to use the server key from config.yaml
  location: Berlin     # city or address for initial search
  radius: 5            # search radius in km
  max_stations: 5
  sort: dist           # dist | e5 | e10 | diesel
  show_e5: true
  show_e10: true
  show_diesel: true
```

---

### Per-widget style

Any widget can have a `style` block:

```yaml
style:
  background_color: "rgba(20, 30, 50, 0.9)"
  background_image: "/data/wallpapers/my-bg.jpg"
  border_color: "#818cf8"
  title_color: "#fff"
  text_color: "#e2e8f0"
  opacity: "0.85"
```

---

## UI

### Dark / Light mode

A **☀ / 🌙** toggle sits in the top-right bar and is always visible. The preference is stored in `localStorage` and survives page reloads. Default is dark mode.

### Item ordering

In the widget editor (Services / Bookmarks items), each entry has **↑ / ↓** arrows and a small **position number field** between them. Type any number and press Enter (or click away) to jump the item directly to that position — no need to click the arrow repeatedly.

---

## Icons

Icon names come from the [walkxcode/dashboard-icons](https://github.com/walkxcode/dashboard-icons) collection (PNG).  
You can also use:
- A full URL: `https://example.com/icon.png`
- A path starting with `/`: `/data/icons/custom.png`

---

## Development

Requirements: Node 22+, Go 1.23+

```bash
# Frontend (http://localhost:5173 — proxies API to :8080)
cd frontend
npm install
npm run dev

# Backend (serves API on :8080, uses ./data for config)
cd backend
go run . --data ../data --static ../frontend/dist
```

> Go is only needed for the backend. The Docker build compiles it inside the container, so Go does **not** need to be installed on the host to run the container.

### Build image locally

```bash
docker build -t naviodeck:latest .
```

---

## Architecture

```
┌─────────────────────────────────────────┐
│  Browser                                │
│  SvelteKit 5  ←──WebSocket──┐           │
│  GridStack layout            │           │
└──────────────┬───────────────┘           │
               │ HTTP REST                 │
┌──────────────▼───────────────────────── ┐│
│  Go backend (:8080)          │          ││
│  ├─ /api/config  (GET/PUT)   │          ││
│  ├─ /api/widgets             │          ││
│  ├─ /api/weather             │          ││
│  ├─ /api/news                │          ││
│  ├─ /api/docker              │          ││
│  ├─ /api/garbage             │          ││
│  ├─ /api/calendar            │          ││
│  ├─ /api/fuel                │          ││
│  ├─ /ws  ──────── hub ───────┘          ││
│  └─ /* (SvelteKit static build)         ││
│                                         ││
│  /data/dashboard.yaml  ◄── file watcher ┘│
└──────────────────────────────────────────┘
```

Config is stored as YAML. The file watcher reloads the config on external changes and broadcasts it to all connected WebSocket clients — edit the YAML in your editor and the dashboard updates live in every tab.

---

## Secrets & Server Configuration

Server-side settings and secrets (API keys, password) live in **`/data/config.yaml`** — a file in the mounted data volume, so it is never baked into the image and is easy to edit per host. Copy the template and fill it in:

```bash
cp config.yaml.example data/config.yaml
```

```yaml
# data/config.yaml
tankerkoenig_api_key: ""   # used by the Fuel widget when no per-widget key is set
dashboard_password: ""     # empty = open dashboard
```

Changes take effect after a container restart (`docker compose restart`). Both values can alternatively be supplied as environment variables (`TANKERKOENIG_API_KEY`, `DASHBOARD_PASSWORD`) — the env var is used as a fallback when the `config.yaml` field is empty.

> **Note:** `data/` is git-ignored. Your config, secrets, uploaded icons/wallpapers and `dashboard.yaml` stay on the host and are never committed.

### Password protection

When `dashboard_password` (or `DASHBOARD_PASSWORD`) is set, a login screen is shown and the session is kept in an HttpOnly cookie. When empty (default), the dashboard is open.

## Security model

NavioDeck is designed for a **trusted local network / homelab**. Keep this in mind before exposing it:

- **Open by default.** Set a password (above) before making it reachable from untrusted networks.
- **Server-side fetchers.** Widgets like Calendar, Garbage, RSS, Weather, Fuel and Docker make the backend fetch a URL/endpoint you provide. With no password set these endpoints are unauthenticated — do **not** expose the dashboard directly to the internet. Put it behind a reverse proxy with authentication, or use a private overlay network (Tailscale/WireGuard).
- **Docker widget.** Mount the Docker socket read-only (`/var/run/docker.sock:/var/run/docker.sock:ro`) and never expose an unauthenticated dashboard that can reach a Docker API.

---

## License
MIT
