# divoom

A Home Assistant dashboard for the Divoom Times Frame (800×1280 portrait) —
a fork of [dragonpaw/divoom](https://github.com/dragonpaw/divoom) stripped
down to a single custom scene. The Times Frame is an Allwinner TinaLinux box
that quietly accepts `adb` pushes and exposes a local JSON HTTP API at
`:9000/divoom_api`; this daemon uses that to show live Home Assistant data
instead of the stock app's locked preset dials.

## What it shows

One screen, sized to be read from across a room:

- **Header** — `WEEKDAY · MM-DD-YYYY`, then a big 12-hour clock
  (`3:04 PM`).
- **Weather** — current condition + temperature from Home Assistant
  (`weather.forecast_home`), e.g. `CLOUDY 61°`, with a sun / rain / snow
  icon to its left picked from today's daily forecast. Long Home Assistant
  states are shortened so they fit (`PT CLOUDY`, `STORMS`, `SLEET`, …).
- **Upstairs / Downstairs / Bedroom** — one row per area, `AREA temp°`.
  A small orange square appears left of the name while that area is
  occupied; the row dims to the accent color when its thermostat is
  offline (`—`).

Weather and room rows share one font size (78), the largest that still fits
`DOWNSTAIRS 76°` plus its occupancy square on the 800px-wide screen.

## Why a fork this stripped-down

The upstream project rotates through ~28 hand-designed scenes (markets,
weather, quotes, NASA APOD, a calendar grid, generative art, …). This fork
deleted all of them — every `scene_*.go` file, their widget packages, the
NASA/cocktail baking machinery, the daily-refresh scheduler — keeping only
the shared plumbing (`scenes.go`'s always-on header, font/color constants,
`buildScenes`) and a new `homeassistant` scene backed by its own widget
package (`internal/widget/homeassistant`) that talks directly to a Home
Assistant instance's REST API.

## Hardware constraints that shaped the layout

- **6 Text elements, total, forever.** The device caps Text-type elements
  at 6 across the *entire* install. The header uses 2 (date line and
  clock), leaving 4 for the scene — exactly what it uses (weather + 3
  rooms). The weather icon and occupancy squares are Image elements, which
  don't count. Exceeding 6 doesn't error; the device silently drops
  whichever Text element lands last in the array — see the commit history
  around 2026-09-18 for the saga of area rows silently vanishing until
  this was understood.
- **The device clips, it doesn't wrap.** Text that overflows its box width
  just gets cut off mid-character. Every row's `FontSize`/`Width` is sized
  against its actual worst-case string, not just the common case.
- **Per-slot property caching.** The device's element cache is keyed on
  `(Type, position-in-DispList)` and only reallocates when the *total*
  array length changes between installs — same length can silently reuse
  stale `FontSize`/`Align`/`Color` from a previous, differently-shaped
  install. `internal/scene/scene.go`'s `Run()` forces a one-element-longer
  "filler" install whenever the length would otherwise repeat, including
  the very first install after any daemon restart (the device's cache
  outlives our process).
- **`adb` over this device's USB is flaky.** Pushes (background images,
  fonts) connect and drop unpredictably; retrying every few seconds until
  one lands is the normal workflow, not a bug to fix.

## Usage

```
go run ./cmd/divoom probe          # discover the frame, print current dial
go run ./cmd/divoom display test   # 30s test layout, then restore
go run ./cmd/divoom render         # write scene JPGs to ./dist/scenes/
go run ./cmd/divoom push           # adb-push the scene backgrounds + fonts
                                    # (USB-attached host only; prereq:
                                    # scripts/download-fonts.sh once)
go run ./cmd/divoom serve          # the dashboard daemon
```

### Environment

Copy `.env.example` to `.env` (gitignored, keep it `chmod 600` — it holds
the HA token). These are the only variables the code reads:

| Variable | Required | Purpose |
|---|---|---|
| `HA_URL` | yes | Base URL of your Home Assistant instance, e.g. `http://10.0.0.7:8123` |
| `HA_TOKEN` | yes | A long-lived access token (HA UI → profile → Security → Long-Lived Access Tokens) |
| `DIVOOM_FRAME_IP` | recommended | Talk to the frame at this LAN IP and skip cloud discovery, whose upstream endpoint (`app.divoom-gz.com`) has a history of being unreachable (see `docs/api.md`) |
| `DIVOOM_FRAME_MAC` | no | Pin cloud discovery to one frame if you have several (ignored when `DIVOOM_FRAME_IP` is set) |
| `TZ` | no | Timezone for the clock and date; defaults to `America/Los_Angeles` |
| `ADB_SERIAL` | no | Which adb device `push` targets when more than one is attached; set in the shell, not `.env` |

The `homeassistant` widget's entity IDs (which areas, which climate/
occupancy/light entities belong to each) are hardcoded in
`internal/widget/homeassistant/homeassistant.go` for this specific home —
not env-configurable.

## Deploy

Runs on the `divoom` LXC (Proxmox CT 103, 10.0.0.235) with the frame
USB-attached, from a checkout at `/opt/divoom`:

```
cd /opt/divoom
git pull
docker compose build && docker compose up -d
docker exec divoom-dashboard divoom push   # after any layout/background change
```

`push` ends by restarting the frame's app (~5s) so it reloads fonts; restart
the daemon afterwards (`docker restart divoom-dashboard`) so the new layout
installs immediately instead of at the next 3-minute scene cycle.

## Architecture

One container does both jobs: `serve` runs forever (LAN HTTP to the frame,
Home Assistant polling, scene install), and the same image carries `adb` +
`/dev/bus/usb` passthrough so `push` can refresh backgrounds/fonts on the
USB-attached frame without a separate dev-box step.

```
  Container (privileged + /dev/bus/usb)                Times Frame
  ┌────────────────────────────────────┐    USB-adb    ┌──────────┐
  │ divoom serve                       │ ───bg/fonts──▶ │/userdata │
  │   ├─ poll Home Assistant REST API  │                │          │
  │   └─ EnterCustomControlMode ──────────LAN HTTP──────▶│ :9000    │
  │ divoom push (manual, USB required) │                │ JSON API │
  └────────────────────────────────────┘                │ 800×1280 │
   ▲                                                     │ IPS LCD  │
   │ Home Assistant REST API (weather.forecast_home,     └──────────┘
   │ climate.*, binary_sensor.*_occupancy_group, light.*)
```

## Docs

- [`docs/api.md`](docs/api.md) — empirical notes on the Times Frame's local
  API: the quirks referenced above (element caps, the position-keyed
  cache, clip-not-wrap), plus pointers into Divoom's own broken-English
  upstream docs. Predates this fork's Home Assistant rewrite but still
  accurate for how the *device* behaves — update it in the same commit as
  any change that exercises new device behavior.
- [`CLAUDE.md`](CLAUDE.md) — engineering philosophy (distilled from
  Kanat-Alexander's *Code Simplicity*) this repo is judged against: reduce
  maintenance over implementation, keep pieces small, no speculative
  generality.

`docs/fonts.md` + `docs/fonts.json` document the Times Frame font
catalog, and `docs/decisions/` holds unrelated repo-wide engineering
decisions — both still accurate and referenced from `docs/api.md`. The
upstream multi-scene project's deploy docs, scene-authoring rules, and
per-scene screenshots (plus the parser scripts that generated the
quote/dictionary widgets) were deleted along with the scenes themselves.
