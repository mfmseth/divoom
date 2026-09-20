// Bakes the always-on header (weekday+date, clock) into a rendered
// scene JPG. Used only by `divoom render` so the screenshot tree in
// dist/scenes/ looks like a wall-installed frame — at runtime these
// elements are installed as device Text/Time elements by the daemon
// and live atop an otherwise-empty header.

package render

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// BakeAlwaysOnHeaderJPEG decodes a scene JPG, paints the always-on
// header onto it (weekday+date, clock), and re-encodes. Mirrors the
// layout produced at runtime by `alwaysOn` in cmd/divoom/scenes.go.
func BakeAlwaysOnHeaderJPEG(in []byte, now time.Time) ([]byte, error) {
	src, err := jpeg.Decode(bytes.NewReader(in))
	if err != nil {
		return nil, fmt.Errorf("decode jpeg: %w", err)
	}
	img := image.NewRGBA(src.Bounds())
	draw.Draw(img, img.Bounds(), src, image.Point{}, draw.Src)
	if err := bakeAlwaysOnHeader(img, now); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func bakeAlwaysOnHeader(img *image.RGBA, now time.Time) error {
	semibold, err := LoadFont("Archivo-SemiBold.ttf")
	if err != nil {
		return fmt.Errorf("load semibold: %w", err)
	}

	headerFace, err := opentype.NewFace(semibold, &opentype.FaceOptions{
		Size: 56, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer headerFace.Close()
	header := strings.ToUpper(now.Weekday().String()) + " · " + now.Format("01-02-2006")
	drawLabelCentered(img, header, headerFace, CanvasW/2, 78, haNeutral400)

	extrabold, err := LoadFont("Archivo-ExtraBold.ttf")
	if err != nil {
		return fmt.Errorf("load extrabold: %w", err)
	}
	timeFace, err := opentype.NewFace(extrabold, &opentype.FaceOptions{
		Size: 169, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer timeFace.Close()
	drawLabelCentered(img, now.Format("15:04"), timeFace, CanvasW/2, 330, haNeutral100)

	return nil
}
