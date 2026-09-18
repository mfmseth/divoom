package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/dragonpaw/divoom/internal/adb"
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

	if err := driver.Run(ctx); err != nil {
		slog.Error("scene driver returned", "err", err)
	}
	return nil
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

// pushSceneBackgrounds renders the homeassistant bg JPG and adb-pushes
// it to the device. Done once at startup; the device references this
// path via BackgroundImageLocalFlag: 1 in the scene layout.
func pushSceneBackgrounds(ctx context.Context) error {
	data, err := render.SceneBackground(render.SceneHomeAssistant, render.FormatJPEG, time.Now())
	if err != nil {
		return fmt.Errorf("render %s bg: %w", bgHomeAssistant, err)
	}
	if err := pushBytes(ctx, data, bgHomeAssistant); err != nil {
		return fmt.Errorf("push %s: %w", bgHomeAssistant, err)
	}
	return nil
}

func pushBytes(ctx context.Context, data []byte, devicePath string) error {
	tmp, err := os.CreateTemp("", "wallclock-bg-*.jpg")
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
