# divoom

A Home Assistant dashboard for the Divoom Times Frame (800×1280 portrait) —
a fork of [dragonpaw/divoom](https://github.com/dragonpaw/divoom) stripped
down to a single custom scene. The Times Frame is an Allwinner TinaLinux box
that quietly accepts `adb` pushes and exposes a local JSON HTTP API at
`:9000/divoom_api`; this daemon uses that to show live Home Assistant data
instead of the stock app's locked preset dials.

## What it shows

One scene, four rows, all sized to be read from across a room:

- **Weather** — current condition + temperature from Home Assistant
  (`weather.forecast_home`), plain clock-orange text. A cloud icon (rain
  streaks or snow dots) bakes into the background when today's daily
  forecast calls for it.
- **Upstairs / Downstairs / Bedroom** — one row per area, each showing
  `AREA · temp°`, color-coded so activity/temperature reads at a glance:
  **orange** when that area is occupied or has a light on, otherwise banded
  by comfort (aqua cold, green comfortable, yellow warm, red hot).

The always-on header (shared with every scene, though there's only one now)
has the weekday on the left and today's date (`MM-DD-YYYY`) on the right,
both the same size, plus the big clock and a year-progress bar along the
bottom edge.

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
  at 6 across the *entire* install. The always-on header uses 1 (the date;
  the weekday and clock are the non-Text `Week`/`Time` built-in types,
  free of this cap), leaving 5 for the active scene — exactly what this
  one uses (weather + 3 area rows). Exceeding 6 doesn't error; the device
  silently drops whichever Text element lands last in the array. Every row
  in this scene earned its slot the hard way — see the commit history
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

### Required environment

| Variable | Purpose |
|---|---|
| `HA_URL` | Base URL of your Home Assistant instance, e.g. `http://10.0.0.7:8123` |
| `HA_TOKEN` | A long-lived access token (HA UI → profile → Security → Long-Lived Access Tokens) |

`DIVOOM_FRAME_IP` skips cloud discovery and talks to a known device
directly — recommended, since the upstream cloud discovery endpoint
(`app.divoom-gz.com`) has a documented history of being unreachable (see
`docs/api.md`). `DIVOOM_FRAME_MAC` pins to a specific frame if you have more
than one.

The `homeassistant` widget's entity IDs (which areas, which climate/
occupancy/light entities belong to each) are hardcoded in
`internal/widget/homeassistant/homeassistant.go` for this specific home —
not env-configurable, the same tradeoff the upstream `hnKeywords` list in
`serve.go` made.

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
