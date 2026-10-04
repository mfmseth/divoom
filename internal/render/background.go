// Package render builds the static and semi-static images we ship to the
// Times Frame as background / Image elements.
package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// Canvas dimensions are fixed by the device: backgrounds MUST be 800x1280.
const (
	CanvasW = 800
	CanvasH = 1280
)

// Gruvbox dark hard palette. We anchor everything to these.
var (
	GruvBgHard   = color.RGBA{0x1d, 0x20, 0x21, 0xff}
	GruvBgDarker = color.RGBA{0x3c, 0x38, 0x36, 0xff}
	GruvFgDark   = color.RGBA{0xa8, 0x99, 0x84, 0xff}
	GruvFg       = color.RGBA{0xeb, 0xdb, 0xb2, 0xff}
	GruvRed      = color.RGBA{0xfb, 0x49, 0x34, 0xff}
	GruvGreen    = color.RGBA{0xb8, 0xbb, 0x26, 0xff}
	GruvYellow   = color.RGBA{0xfa, 0xbd, 0x2f, 0xff}
	GruvBlue     = color.RGBA{0x83, 0xa5, 0x98, 0xff}
	GruvPurple   = color.RGBA{0xd3, 0x86, 0x9b, 0xff}

	GruvAqua   = color.RGBA{0x8e, 0xc0, 0x7b, 0xff}
	GruvOrange = color.RGBA{0xfe, 0x80, 0x19, 0xff}
)

// Format selects an output encoding for TestBackground.
type Format int

const (
	FormatPNG Format = iota
	FormatJPEG
)

// TestBackground returns a gruvbox-bg-hard field with small fg registration
// dots in each corner, an aqua cross at the canvas midpoint, and an accent-
// color swatch band along the bottom. Use to eyeball orientation, scaling,
// and color reproduction.
func TestBackground(format Format) ([]byte, error) {
	img := buildTestImage()
	var buf bytes.Buffer
	switch format {
	case FormatPNG:
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	case FormatJPEG:
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format %d", format)
	}
	return buf.Bytes(), nil
}

// HeroBackground returns a scene-neutral background: gruvbox bg, hairline
// divider, year-progress bar, no glyph. Useful as a preview/fallback.
func HeroBackground(format Format, now time.Time) ([]byte, error) {
	return encodeImage(buildHeroImage(now), format)
}

// Design tokens for the wallclock-scene homeassistant redesign (mirrors
// cmd/divoom/scenes.go's cHaXxx constants, which the device Text/Time
// elements use — this package can't import cmd/divoom, so the values
// are duplicated here rather than shared). haTextAccent is the brighter
// "pop of color" swapped in for the design review's original flat
// neutral-400 gray on the header/weather/dim text, per user feedback.
var (
	haNeutral900 = color.RGBA{0x2d, 0x2b, 0x2b, 0xff}
	haNeutral700 = color.RGBA{0x60, 0x5d, 0x5d, 0xff}
	haNeutral100 = color.RGBA{0xf8, 0xf4, 0xf4, 0xff}
	haAccent500  = color.RGBA{0xff, 0x56, 0x3c, 0xff}
	haTextAccent = color.RGBA{0xff, 0x97, 0x83, 0xff}
	haAlertRed   = color.RGBA{0xd3, 0x2f, 0x2f, 0xff}
)

// Divider-rule geometry. haDivider2Y sits in the gap between the
// weather row and the first room row (cmd/divoom/scene_homeassistant.go's
// haWeatherY / haRoomY).
const (
	haDividerX0    = 40
	haDividerX1    = CanvasW - 40
	haDividerThick = 5
	haDivider1Y    = 122
	haDivider2Y    = 522
)

// SceneHomeAssistantBackground bakes the homeassistant scene's bg: a
// flat neutral-900 fill plus the two divider rules that mark off the
// header/clock and weather/rooms bands. The weather icon and room
// occupancy marks are no longer baked here — they're separate Image
// elements repositioned live each activation (see
// cmd/divoom/scene_homeassistant.go's positionDynamicMarks), which is
// also what fixed the old baked icon overlapping longer weather text: a
// fixed bake-time position can't track the actual rendered text width.
func SceneHomeAssistantBackground(format Format) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, CanvasW, CanvasH))
	draw.Draw(img, img.Bounds(), &image.Uniform{haNeutral900}, image.Point{}, draw.Src)

	draw.Draw(img, image.Rect(haDividerX0, haDivider1Y, haDividerX1, haDivider1Y+haDividerThick),
		&image.Uniform{haNeutral700}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(haDividerX0, haDivider2Y, haDividerX1, haDivider2Y+haDividerThick),
		&image.Uniform{haNeutral700}, image.Point{}, draw.Src)

	return encodeImage(img, format)
}

// OccupancyMarkPNG renders the one small solid accent-colored square
// pushed to the device as the room rows' occupancy-mark Image asset
// (see cmd/divoom/scene_homeassistant.go's haMarkSize/haOccupancyMarkPath).
func OccupancyMarkPNG(size int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{haAccent500}, image.Point{}, draw.Src)
	return encodeImage(img, FormatPNG)
}

// HADownBannerPNG renders the full-width red "HOME ASSISTANT DOWN"
// banner the homeassistant scene shows at the bottom of the screen while
// Home Assistant is unreachable (see cmd/divoom/scene_homeassistant.go).
func HADownBannerPNG(width, height int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{haAlertRed}, image.Point{}, draw.Src)
	f, err := LoadFont("Archivo-ExtraBold.ttf")
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 52, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	defer face.Close()
	m := face.Metrics()
	baseline := height/2 + (m.Ascent-m.Descent).Round()/2
	drawLabelCentered(img, "HOME ASSISTANT DOWN", face, width/2, baseline, haNeutral100)
	return encodeImage(img, FormatPNG)
}

// WeatherIconPNG renders one of the three weather-row glyphs (sun,
// rain, snow) as a standalone asset: a neutral-900 backdrop exactly
// matching the scene's flat background (so it blends in seamlessly
// without needing real alpha transparency) with the glyph painted in
// haTextAccent on top, scaled to fill a size×size box.
func WeatherIconPNG(kind string, size int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{haNeutral900}, image.Point{}, draw.Src)
	g := iconGrid{cx: size / 2, cy: size / 2, scale: float64(size) / iconDesignSize}
	switch kind {
	case "rain":
		drawRainCloud(img, g, haTextAccent)
	case "snow":
		drawSnowCloud(img, g, haTextAccent)
	default:
		drawSunIcon(img, g, haTextAccent)
	}
	return encodeImage(img, FormatPNG)
}

// iconDesignSize is the box size the icon shapes below were drawn for;
// iconGrid scales their pixel offsets to whatever size is requested.
const iconDesignSize = 42

// iconGrid maps the icon shapes' design-size pixel offsets onto the
// actual image: (cx, cy) is the center, scale the size/iconDesignSize ratio.
type iconGrid struct {
	cx, cy int
	scale  float64
}

// px scales a design-size offset or length to real pixels.
func (g iconGrid) px(v int) int {
	return int(math.Round(float64(v) * g.scale))
}

// rect returns the rectangle spanning design-size offsets (x0,y0)-(x1,y1)
// from the center.
func (g iconGrid) rect(x0, y0, x1, y1 int) image.Rectangle {
	return image.Rect(g.cx+g.px(x0), g.cy+g.px(y0), g.cx+g.px(x1), g.cy+g.px(y1))
}

// circle fills a disc of design-size radius r at design-size offset (dx, dy).
func (g iconGrid) circle(img *image.RGBA, dx, dy, r int, c color.RGBA) {
	fillCircle(img, g.cx+g.px(dx), g.cy+g.px(dy), g.px(r), c)
}

// drawSunIcon paints a solid filled sun: a disc plus eight short rays,
// matching the weight/simplicity of drawCloudBody's style rather than
// tracing an SVG path.
func drawSunIcon(img *image.RGBA, g iconGrid, c color.RGBA) {
	g.circle(img, 0, 0, 8, c)
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		g.circle(img, d[0]*15, d[1]*15, 2, c)
	}
}

// drawCloudBody paints the cloud shape shared by drawRainCloud and
// drawSnowCloud -- three overlapping circles plus a base rectangle.
func drawCloudBody(img *image.RGBA, g iconGrid, c color.RGBA) {
	g.circle(img, -8, 0, 7, c)
	g.circle(img, 0, -5, 9, c)
	g.circle(img, 8, 0, 7, c)
	draw.Draw(img, g.rect(-14, 0, 14, 7), &image.Uniform{c}, image.Point{}, draw.Src)
}

// drawRainCloud draws the shared cloud body plus three short vertical
// streaks beneath it, both in c (same slot, same size, same color as
// the text it sits beside — see the design review's icon-swap note).
func drawRainCloud(img *image.RGBA, g iconGrid, c color.RGBA) {
	drawCloudBody(img, g, c)
	for _, dx := range []int{-7, 0, 7} {
		draw.Draw(img, g.rect(dx-1, 9, dx+1, 17), &image.Uniform{c}, image.Point{}, draw.Src)
	}
}

// drawSnowCloud draws the shared cloud body plus three small dots
// beneath it, both in c.
func drawSnowCloud(img *image.RGBA, g iconGrid, c color.RGBA) {
	drawCloudBody(img, g, c)
	for _, dx := range []int{-7, 0, 7} {
		g.circle(img, dx, 13, 2, c)
	}
}

func encodeImage(img *image.RGBA, format Format) ([]byte, error) {
	var buf bytes.Buffer
	switch format {
	case FormatPNG:
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	case FormatJPEG:
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format %d", format)
	}
	return buf.Bytes(), nil
}

// fillCircle is a tiny stdlib-only filled-disc rasterizer. The render
// package doesn't depend on x/image, so we roll a small version here
// rather than pulling in a graphics library for one shape.
func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	bounds := img.Bounds()
	r2 := r * r
	for y := cy - r; y <= cy+r; y++ {
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}
		dy := y - cy
		for x := cx - r; x <= cx+r; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X {
				continue
			}
			dx := x - cx
			if dx*dx+dy*dy <= r2 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func buildHeroImage(now time.Time) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, CanvasW, CanvasH))
	draw.Draw(img, img.Bounds(), &image.Uniform{GruvBgHard}, image.Point{}, draw.Src)

	// Hairline divider at y=460 separating the always-on "clock+date" zone
	// above from the rotating scene area below. Inset 60px from each side
	// so the rule reads as a composition mark, not a horizon line.
	draw.Draw(img, image.Rect(60, 460, CanvasW-60, 462),
		&image.Uniform{GruvBgDarker}, image.Point{}, draw.Src)

	// Year-progress bar along the very bottom edge. Track in bg-darker, fill
	// in orange to the elapsed fraction of the year. Subtle ambient marker
	// of where you are in the year.
	const (
		barH       = 4
		barOffsetY = 8
	)
	trackTop := CanvasH - barOffsetY - barH
	trackBot := CanvasH - barOffsetY
	draw.Draw(img, image.Rect(0, trackTop, CanvasW, trackBot),
		&image.Uniform{GruvBgDarker}, image.Point{}, draw.Src)

	yearDays := 365
	if isLeapYear(now.Year()) {
		yearDays = 366
	}
	frac := float64(now.YearDay()-1) / float64(yearDays)
	filledW := int(frac * float64(CanvasW))
	if filledW > 0 {
		draw.Draw(img, image.Rect(0, trackTop, filledW, trackBot),
			&image.Uniform{GruvOrange}, image.Point{}, draw.Src)
	}

	// Baked "> " prompt to the left of the day-of-week. The day name
	// itself renders as a device Week element (built-in type — saves
	// one Text slot, doesn't count against the 6-Text cap). The prompt
	// glyph stays here in cFgDark mono so the always-on header still
	// reads as "> wednesday" rather than just "Wednesday".
	if f, err := LoadFont("Iosevka-Regular.ttf"); err == nil {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: 64, DPI: 72, Hinting: font.HintingFull,
		})
		if err == nil {
			drawLabelLeft(img, "> ", face, 40, 90, GruvFgDark)
			face.Close()
		}
	}

	return img
}

func isLeapYear(y int) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}

func buildTestImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, CanvasW, CanvasH))
	draw.Draw(img, img.Bounds(), &image.Uniform{GruvBgHard}, image.Point{}, draw.Src)

	// 7x7 registration dots inset 20px from each corner.
	for _, p := range []image.Point{
		{20, 20}, {CanvasW - 20, 20},
		{20, CanvasH - 20}, {CanvasW - 20, CanvasH - 20},
	} {
		drawSquare(img, p, 3, GruvFg)
	}

	// Aqua mid-line stripes (horizontal + a short vertical) to spot rotation.
	draw.Draw(img, image.Rect(0, CanvasH/2-1, CanvasW, CanvasH/2+1), &image.Uniform{GruvAqua}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(CanvasW/2-1, CanvasH/2-100, CanvasW/2+1, CanvasH/2+100), &image.Uniform{GruvAqua}, image.Point{}, draw.Src)

	// Gruvbox accent palette swatches along the bottom.
	swatches := []color.RGBA{GruvRed, GruvGreen, GruvYellow, GruvBlue, GruvPurple, GruvAqua, GruvOrange}
	const swH = 12
	swW := CanvasW / len(swatches)
	for i, c := range swatches {
		r := image.Rect(i*swW, CanvasH-swH-20, (i+1)*swW, CanvasH-20)
		draw.Draw(img, r, &image.Uniform{c}, image.Point{}, draw.Src)
	}
	return img
}

// drawSquare paints a (2r+1)×(2r+1) filled square centered on p.
func drawSquare(img *image.RGBA, p image.Point, r int, c color.RGBA) {
	rect := image.Rect(p.X-r, p.Y-r, p.X+r+1, p.Y+r+1)
	draw.Draw(img, rect, &image.Uniform{c}, image.Point{}, draw.Src)
}
