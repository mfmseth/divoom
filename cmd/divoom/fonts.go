package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dragonpaw/divoom/internal/adb"
)

// Per docs/api.md "Custom font workflow": each TTF is pushed to the
// device's font cache as `<catalog_id+1>.bin`, then the matching
// `font_list.cfg` registers the catalog ID so divoom_app finds it on
// startup. One custom font, hardcoded -- the scene's only non-stock
// typography (single-family per the Modernist-pairing design review;
// see scripts/download-fonts.sh). Reuses catalog ID 7 -- previously
// Iosevka's slot -- rather than registering a new ID: this codebase
// has already hit one hardware bug from a novel, never-before-used ID
// (see the commit history around 2026-09-18, an *element* ID that
// silently broke rendering), and font catalog IDs are the same kind of
// device-side-cached, easy-to-get-wrong identifier.
type customFont struct {
	src     string // basename under ./fonts/
	devSlot string // absolute path on device
	fontID  int    // catalog ID referenced from scenes.go
}

var customFonts = []customFont{
	{src: "ArchivoBlack-Regular.ttf", devSlot: "/usr/share/divoom_app/divoom/21/8.bin", fontID: 7},
}

const (
	// Pre-built font_list.cfg with the custom entry spliced in.
	// Checked into the repo so we never mutate the on-device cfg in
	// place (see feedback_device_init_modifications: build the exact
	// bytes locally, push them).
	fontListLocal  = "device-files/divoom-config/system/font_list.cfg"
	fontListDevice = "/divoom-config/system/font_list.cfg"
)

// runPush is the `divoom push` subcommand. Backgrounds first (fast, no
// device restart); fonts last (slow, ends with a daemon crash-restart).
// Order matters: if we restarted before pushing bgs, the device would
// still be coming back up while adb tried to push JPGs.
func runPush(ctx context.Context) error {
	if err := pushSceneBackgrounds(ctx); err != nil {
		return err
	}
	return pushFonts(ctx)
}

// pushFonts installs the custom TTF and the matching
// font_list.cfg, then triggers the daemon-reload crash-restart so
// divoom_app re-reads the cfg. The frame restarts at the end — `push`
// is the slow path, run from the USB-attached dev box only.
func pushFonts(ctx context.Context) error {
	for _, f := range customFonts {
		local := filepath.Join("fonts", f.src)
		if _, err := os.Stat(local); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("missing %s — run scripts/download-fonts.sh first", local)
			}
			return fmt.Errorf("stat %s: %w", local, err)
		}
	}
	if _, err := os.Stat(fontListLocal); err != nil {
		return fmt.Errorf("stat %s: %w", fontListLocal, err)
	}

	for _, f := range customFonts {
		local := filepath.Join("fonts", f.src)
		pushCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		err := adb.Push(pushCtx, local, f.devSlot)
		cancel()
		if err != nil {
			return fmt.Errorf("push %s -> %s: %w", local, f.devSlot, err)
		}
		slog.Info("font installed", "src", f.src, "slot", f.devSlot, "font_id", f.fontID)
	}

	pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err := adb.Push(pushCtx, fontListLocal, fontListDevice)
	cancel()
	if err != nil {
		return fmt.Errorf("push %s: %w", fontListDevice, err)
	}

	if err := reloadFontsViaCrashRestart(ctx); err != nil {
		return fmt.Errorf("reload fonts: %w", err)
	}
	slog.Info("fonts installed; frame will restart in ~5s to load new font_list.cfg")
	return nil
}

// reloadFontsViaCrashRestart sends `Device/GetTimeDialFontV2` to the
// frame's local API. divoom_app crashes on that command; procd restarts
// it within ~5s, and divoom_system_font_init re-reads font_list.cfg on
// the way up. Ugly, undocumented, fragile — but it's the only known
// way to refresh fonts short of a power cycle. See docs/api.md.
func reloadFontsViaCrashRestart(ctx context.Context) error {
	_, device, err := connectToFrame(ctx)
	if err != nil {
		return err
	}
	ip := os.Getenv("DIVOOM_FRAME_IP")
	if ip == "" && device != nil {
		ip = device.DevicePrivateIP
	}
	if ip == "" {
		return fmt.Errorf("no frame IP resolved for crash-restart")
	}

	url := fmt.Sprintf("http://%s:9000/divoom_api", ip)
	body := []byte(`{"Command":"Device/GetTimeDialFontV2"}`)

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Expect the connection to die mid-flight as divoom_app crashes.
	// Any "connection reset" / timeout here is success, not failure.
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Info("crash-restart triggered (request errored as expected)", "err", err)
		return nil
	}
	resp.Body.Close()
	slog.Info("crash-restart request returned", "status", resp.Status)
	return nil
}
