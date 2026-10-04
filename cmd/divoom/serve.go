package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/dragonpaw/divoom/internal/adb"
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/render"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
	"github.com/dragonpaw/divoom/internal/widget/homeassistant"
)

// runServe installs the scene background, then rotates forever. Time +
// Date + DoW are always on top; the body area shows homeassistant.
func runServe(ctx context.Context) error {
	client, _, err := connectToFrame(ctx)
	if err != nil {
		return err
	}

	haURL, haToken := os.Getenv("HA_URL"), os.Getenv("HA_TOKEN")
	if haURL == "" || haToken == "" {
		return fmt.Errorf("HA_URL and HA_TOKEN must both be set")
	}
	widgets := map[string]widget.Widget{
		"homeassistant": homeassistant.New(haURL, haToken),
	}
	slog.Info("homeassistant scene enabled", "url", haURL)

	scenes := buildScenes(widgets)

	driver := &scene.Driver{
		Client:   client,
		AlwaysOn: alwaysOn,
		Scenes:   scenes,
	}
	logStartup(driver)
	go tickClock(ctx, client)

	if err := driver.Run(ctx); err != nil {
		slog.Error("scene driver returned", "err", err)
	}
	return nil
}

// tickClock keeps the always-on clock and date current between scene
// reinstalls (which only happen every scene.SceneDuration): at each
// minute boundary it patches their Text elements in place. Failures are
// expected while the frame reboots or a reinstall is in flight, and the
// next tick or install corrects the text, so they're only debug-logged.
func tickClock(ctx context.Context, client *frame.Client) {
	for {
		now := time.Now()
		select {
		case <-ctx.Done():
			return
		case <-time.After(now.Truncate(time.Minute).Add(time.Minute).Sub(now)):
		}
		updCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := client.UpdateTexts(updCtx, alwaysOnTexts(time.Now()))
		cancel()
		if err != nil {
			slog.Debug("clock update failed", "err", err)
		}
	}
}

// counter is the optional Count() interface implemented by static quote
// sources. Nothing in this fork implements it anymore, but logStartup
// still checks for it defensively.
type counter interface {
	Count() int
}

// logStartup reports the rotation config: one line per scene with its
// weight, share %, and entry count ("live" for HTTP-fetching widgets,
// "—" for scenes with no widget). Operators reading the daemon logs see
// exactly what's wired up without cracking open the source.
func logStartup(d *scene.Driver) {
	slog.Info("scene rotation starting", "scenes", len(d.Scenes), "duration", scene.SceneDuration)
	totalWeight := 0
	for _, s := range d.Scenes {
		totalWeight += s.Weight
	}
	for _, s := range d.Scenes {
		share := 0.0
		if totalWeight > 0 {
			share = float64(s.Weight) / float64(totalWeight) * 100
		}
		entries := "live"
		switch w := s.Widget.(type) {
		case nil:
			entries = "—"
		case counter:
			entries = strconv.Itoa(w.Count())
		default:
			_ = w
		}
		slog.Info("scene configured",
			"name", s.Name,
			"weight", s.Weight,
			"share_pct", fmt.Sprintf("%.0f", share),
			"entries", entries,
		)
	}
}

// pushSceneBackgrounds renders the homeassistant scene's (single, flat)
// background plus its dynamic-Image assets -- the three weather icons,
// the room rows' occupancy mark, and the alert banners -- and
// adb-pushes them all to the device. Run by `divoom push`; the scene's Image elements reference
// these on-device paths directly (see scene_homeassistant.go).
func pushSceneBackgrounds(ctx context.Context) error {
	bg, err := render.SceneHomeAssistantBackground(render.FormatJPEG)
	if err != nil {
		return fmt.Errorf("render %s bg: %w", bgHomeAssistant, err)
	}
	if err := pushBytes(ctx, bg, bgHomeAssistant); err != nil {
		return fmt.Errorf("push %s: %w", bgHomeAssistant, err)
	}

	icons := []struct {
		kind string
		path string
	}{
		{"", haIconSunPath},
		{"rain", haIconRainPath},
		{"snow", haIconSnowPath},
	}
	for _, ic := range icons {
		data, err := render.WeatherIconPNG(ic.kind, haWeatherIconSize)
		if err != nil {
			return fmt.Errorf("render %s icon: %w", ic.path, err)
		}
		if err := pushBytes(ctx, data, ic.path); err != nil {
			return fmt.Errorf("push %s: %w", ic.path, err)
		}
	}

	for _, b := range haBanners {
		data, err := render.AlertBannerPNG(b.text, CanvasW, haBannerHeight)
		if err != nil {
			return fmt.Errorf("render %s banner: %w", b.path, err)
		}
		if err := pushBytes(ctx, data, b.path); err != nil {
			return fmt.Errorf("push %s: %w", b.path, err)
		}
	}

	mark, err := render.OccupancyMarkPNG(haMarkSize)
	if err != nil {
		return fmt.Errorf("render occupancy mark: %w", err)
	}
	if err := pushBytes(ctx, mark, haOccupancyMarkPath); err != nil {
		return fmt.Errorf("push %s: %w", haOccupancyMarkPath, err)
	}
	return nil
}

func pushBytes(ctx context.Context, data []byte, devicePath string) error {
	tmp, err := os.CreateTemp("", "wallclock-asset-*")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()

	pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return adb.Push(pushCtx, tmp.Name(), devicePath)
}
