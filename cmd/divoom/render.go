package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/dragonpaw/divoom/internal/render"
)

// runRender writes every known scene background to <outDir>/scenes/<name>.jpg.
func runRender(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	out := fs.String("out", "dist", "output directory (a scenes/ subdir will be created)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	scenesDir := filepath.Join(*out, "scenes")
	if err := os.MkdirAll(scenesDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", scenesDir, err)
	}

	// Hardcoded to keep every screenshot reproducible — year-progress
	// bar, time/day-of-week header and all other time-dependent baking
	// line up across the scene set. Wednesday 2026-05-27 12:34 local
	// (-07:00).
	now := time.Date(2026, time.May, 27, 12, 34, 0, 0,
		time.FixedZone("local", -7*3600))
	scenes := []struct {
		name   string
		render func() ([]byte, error)
	}{
		// Smoke-test pattern with corner dots + midline cross + bottom swatches.
		{name: "test", render: func() ([]byte, error) {
			return render.TestBackground(render.FormatJPEG)
		}},
		// Scene-neutral preview — just the gruvbox frame, no glyph.
		{name: "hero", render: func() ([]byte, error) {
			return render.HeroBackground(render.FormatJPEG, now)
		}},
		// The one scene the daemon pushes via adb.
		{name: "scene-homeassistant", render: func() ([]byte, error) {
			return render.SceneBackground(render.SceneHomeAssistant, render.FormatJPEG, now)
		}},
	}

	if len(scenes) == 0 {
		return errors.New("no scenes defined")
	}

	for _, s := range scenes {
		data, err := s.render()
		if err != nil {
			return fmt.Errorf("render %q: %w", s.name, err)
		}
		// Bake the always-on header (day name, time, footer,
		// weekend status) on top of every scene so the dist/
		// screenshots look like the device with the daemon's
		// Text/Time/Week elements installed. Skipped for the
		// neutral test pattern, which has no scene chrome.
		if s.name != "test" {
			data, err = render.BakeAlwaysOnHeaderJPEG(data, now)
			if err != nil {
				return fmt.Errorf("bake header %q: %w", s.name, err)
			}
		}
		path := filepath.Join(scenesDir, s.name+".jpg")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		slog.Info("rendered scene", "name", s.name, "path", path, "bytes", len(data))
	}
	slog.Info("render complete", "scenes", len(scenes), "out", scenesDir)
	return nil
}
