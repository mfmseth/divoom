package main

import (
	"slices"
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/render"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
	"github.com/dragonpaw/divoom/internal/widget/homeassistant"
)

// "homeassistant" — header (weekday+date) and clock from alwaysOn, then
// a weather row and three room rows (Upstairs / Downstairs / Bedroom),
// per the wallclock-scene design review: one Archivo family, one
// reserved accent (occupancy only), centered throughout. The widget
// emits "<weather>|<icon>|<presence>|<upstairs>|<downstairs>|<bedroom>"
// — presence is fetched but not displayed (pre-existing, unrelated to
// this scene's layout). Each room field is "AREA temp°@FLAGS": the
// "@FLAGS" suffix is stripped before display and used only to decide
// whether that room's occupancy mark shows.
//
// Occupancy used to recolor the whole row; the design review replaces
// that with a small accent-colored square prepended to the room name
// (see haRoomY / OnActivate below) so the accent stays reserved for
// activity instead of doubling as a temperature-reading color. Likewise
// the weather row's icon glyph sits left of its (centered, and
// therefore width-varying) text. A Text element only has one FontColor
// for its whole string and can't host a second glyph, so both the
// occupancy mark and the weather icon are separate Image elements
// repositioned fresh each activation with render.MeasureLabel against
// the text Mounts already resolved that cycle — see
// positionDynamicMarks.
func homeAssistantScene(widgets map[string]widget.Widget) *scene.Scene {
	elements := []frame.DispElement{
		{
			ID: idSceneWeather, Type: "Text",
			StartX: 40, StartY: haWeatherY, Width: 720, Height: haRowHeight,
			Align: 2, FontSize: haTextSize, FontID: fontArchivoSemiBold,
			FontColor: cHaTextAccent, BgColor: cHaNeutral900,
		},
		{
			// Url/StartX are placeholders; OnActivate sets both every
			// activation from the widget's icon field and the weather
			// text's measured width (see positionDynamicMarks).
			ID: idSceneWeatherIcon, Type: "Image",
			StartX: -haWeatherIconSize, StartY: haWeatherY + (haRowHeight-haWeatherIconSize)/2,
			Width: haWeatherIconSize, Height: haWeatherIconSize,
			Url: haIconSunPath, ImgLocalFlag: 1,
			FontSize: 1, FontID: fontArchivoSemiBold,
			FontColor: cHaNeutral100, BgColor: cHaNeutral900,
		},
	}
	for i, y := range haRoomY {
		elements = append(elements, frame.DispElement{
			ID: idSceneRoomBase + i, Type: "Text",
			StartX: 40, StartY: y, Width: 720, Height: haRowHeight,
			Align: 2, FontSize: haTextSize, FontID: fontArchivoSemiBold,
			FontColor: cHaNeutral100, BgColor: cHaNeutral900,
		})
	}
	for i, y := range haRoomY {
		elements = append(elements, frame.DispElement{
			// Parked off-canvas by default; OnActivate below moves it
			// on-screen only for an occupied room. Font/Color fields
			// are semantically meaningless for Image elements but
			// required anyway -- see docs/api.md "Image DispElements
			// require Font/Color fields even though they're
			// semantically meaningless for images".
			ID: idSceneMarkBase + i, Type: "Image",
			StartX: haMarkOffscreenX, StartY: y + (haRowHeight-haMarkSize)/2,
			Width: haMarkSize, Height: haMarkSize,
			Url: haOccupancyMarkPath, ImgLocalFlag: 1,
			FontSize: 1, FontID: fontArchivoSemiBold,
			FontColor: cHaNeutral100, BgColor: cHaNeutral900,
		})
	}

	// Parked off-canvas; OnActivate shows it with the current phrase's image.
	elements = append(elements, frame.DispElement{
		ID: idSceneWear, Type: "Image",
		StartX: -CanvasW, StartY: haWearY,
		Width: CanvasW, Height: haWearHeight,
		Url: wearPath(wearPhrases[0]), ImgLocalFlag: 1,
		FontSize: 1, FontID: fontArchivoSemiBold,
		FontColor: cHaNeutral100, BgColor: cHaNeutral900,
	})

	// Parked off-canvas; OnActivate brings it on-screen, with the
	// matching image, only while Home Assistant or the internet is down.
	elements = append(elements, frame.DispElement{
		ID: idSceneBanner, Type: "Image",
		StartX: -CanvasW, StartY: haBannerY,
		Width: CanvasW, Height: haBannerHeight,
		Url: haBanners[homeassistant.StatusDown].path, ImgLocalFlag: 1,
		FontSize: 1, FontID: fontArchivoSemiBold,
		FontColor: cHaNeutral100, BgColor: cHaNeutral900,
	})

	mounts := []scene.Mount{
		{ID: idSceneWeather, Format: pipeAt(0)},
	}
	for i := range haRoomY {
		mounts = append(mounts, scene.Mount{ID: idSceneRoomBase + i, Format: roomRow(3 + i)})
	}

	return &scene.Scene{
		Name:       "homeassistant",
		Weight:     WeightInformational,
		BgPath:     bgHomeAssistant,
		Elements:   elements,
		Widget:     widgets["homeassistant"],
		Mounts:     mounts,
		OnActivate: positionDynamicMarks,
	}
}

// haRoomY is the top y-coordinate of each room row, in display order
// (Upstairs / Downstairs / Bedroom) — index i's pipe field is at
// position 3+i in the widget's raw string.
var haRoomY = [3]int{625, 785, 945}

// haTextSize is shared by the weather and room rows: as large as
// "DOWNSTAIRS 76°" can go and still fit, with its occupancy mark, in
// the 800px-wide canvas (the device clips text, it doesn't wrap), so
// the readings are legible from across the room.
const (
	haTextSize  = 78
	haRowHeight = 95
	haWeatherY  = 385
)

// The what-to-wear line sits directly under the weather row.
const (
	haWearY      = haWeatherY + haRowHeight + 5
	haWearHeight = 70
)

// wearPhrases is every state the "What to Wear" template helper in Home
// Assistant (sensor.what_to_wear) can produce: a clothing tier plus an
// optional rain/snow add-on. Each is pre-rendered and pushed as its own
// image, since the device can't render new text into an Image; a state
// not listed here hides the row. Keep in sync with that helper's template.
var wearPhrases = func() []string {
	var out []string
	for _, base := range []string{"SHORTS", "T-SHIRT", "LONG SLEEVES", "HOODIE", "JACKET", "WARM COAT"} {
		for _, extra := range []string{"", " + UMBRELLA", " + BOOTS"} {
			out = append(out, base+extra)
		}
	}
	return out
}()

// wearPath is the on-device path of phrase's pre-pushed label image.
func wearPath(phrase string) string {
	slug := strings.NewReplacer(" + ", "_", " ", "_", "-", "").Replace(strings.ToLower(phrase))
	return "/userdata/wallclock_wear_" + slug + ".png"
}

// roomRow returns a Mount.Format closure that picks segment i (one of
// the widget's "AREA temp°@FLAGS" strings) and strips the hidden
// "@FLAGS" suffix before display. An offline room's temp field is
// already "—" (see fetchArea's zero value), which is what drives the
// dimmer accent text color here.
func roomRow(i int) func(raw string) (text, color string) {
	return func(raw string) (text, color string) {
		field := weatherPipeField(raw, i)
		if field == "" {
			return "", ""
		}
		text, _, _ = strings.Cut(field, "@")
		if strings.HasSuffix(text, " —") {
			return text, cHaTextAccent
		}
		return text, cHaNeutral100
	}
}

// Occupancy-mark geometry. The mark sits haMarkGap px left of the
// room's centered text, vertically centered in the row; haMarkOffscreenX
// parks it off the 800px canvas when the room is unoccupied.
const (
	haMarkSize       = 36
	haMarkGap        = 24
	haMarkOffscreenX = -haMarkSize
	haRoomBoxCenterX = CanvasW / 2
)

// Weather-icon geometry: same idea as the occupancy mark, but for the
// weather row's sun/rain/snow glyph, which sits haWeatherIconGap px
// left of the (centered) weather text.
const (
	haWeatherIconSize = 64
	haWeatherIconGap  = 14
)

// The alert banner sits in the otherwise-empty space below the last
// room row (which ends at haRoomY[2]+haRowHeight = 1005), so showing it
// doesn't cover or move anything.
const (
	haBannerY      = 1085
	haBannerHeight = 120
)

// haBanners maps each widget status that warrants an alert to its
// banner's text and pre-pushed on-device image (see render.AlertBannerPNG
// and pushSceneBackgrounds).
var haBanners = map[string]struct{ text, path string }{
	homeassistant.StatusDown:    {"HOME ASSISTANT DOWN", "/userdata/wallclock_banner_ha_down.png"},
	homeassistant.StatusISPDown: {"INTERNET DOWN", "/userdata/wallclock_banner_isp_down.png"},
}

// On-device paths for the pre-pushed occupancy-mark and weather-icon
// PNGs (see render.OccupancyMarkPNG, render.WeatherIconPNG, and
// pushSceneBackgrounds).
const (
	haOccupancyMarkPath = "/userdata/wallclock_mark_occupied.png"
	haIconSunPath       = "/userdata/wallclock_icon_sun.png"
	haIconRainPath      = "/userdata/wallclock_icon_rain.png"
	haIconSnowPath      = "/userdata/wallclock_icon_snow.png"
)

// iconPathFor maps the widget's icon field ("rain", "snow", or "" for
// sunny) to its on-device asset path.
func iconPathFor(icon string) string {
	switch icon {
	case "rain":
		return haIconRainPath
	case "snow":
		return haIconSnowPath
	default:
		return haIconSunPath
	}
}

// positionDynamicMarks is the homeassistant scene's OnActivate. It runs
// after Mounts have resolved this cycle's text (Driver.activate
// guarantees the order), and re-derives the position of every Image
// element that has to sit next to a centered, width-varying Text
// element:
//
//   - The weather icon always shows, swapped to match the widget's
//     icon field and placed left of the weather text.
//   - Each room's occupancy mark shows only when that room's raw field
//     carries the "OCC" flag; otherwise it's parked off-canvas.
//   - The what-to-wear line shows the image for the helper's current
//     phrase, or is parked off-canvas if the phrase isn't one we pushed.
//   - The red alert banner shows only while the widget reports Home
//     Assistant or the internet down; otherwise it's parked off-canvas.
func positionDynamicMarks(_ time.Time, raw string, elements []frame.DispElement) {
	bannerX := -CanvasW
	if b, ok := haBanners[weatherPipeField(raw, 6)]; ok {
		setElementURL(elements, idSceneBanner, b.path)
		bannerX = 0
	}
	setElementX(elements, idSceneBanner, bannerX)

	wearX := -CanvasW
	if wear := weatherPipeField(raw, 7); slices.Contains(wearPhrases, wear) {
		setElementURL(elements, idSceneWear, wearPath(wear))
		wearX = 0
	}
	setElementX(elements, idSceneWear, wearX)

	setElementURL(elements, idSceneWeatherIcon, iconPathFor(weatherPipeField(raw, 1)))
	setElementX(elements, idSceneWeatherIcon,
		leftOfCenteredText(elements, idSceneWeather, haTextSize, haWeatherIconSize, haWeatherIconGap))

	for i := range haRoomY {
		markX := haMarkOffscreenX
		if roomOccupied(raw, 3+i) {
			markX = leftOfCenteredText(elements, idSceneRoomBase+i, haTextSize, haMarkSize, haMarkGap)
		}
		setElementX(elements, idSceneMarkBase+i, markX)
	}
}

// leftOfCenteredText measures the current TextMessage of the element
// with the given ID (rendered at fontSize in Archivo SemiBold) and
// returns the StartX an elementSize-wide glyph needs to sit gap px to
// its left, given both are centered as one group in the room/weather
// box (see haRoomBoxCenterX). Falls back to parking off-canvas if the
// text can't be measured.
func leftOfCenteredText(elements []frame.DispElement, textID int, fontSize float64, elementSize, gap int) int {
	text := elementText(elements, textID)
	w, err := render.MeasureLabel(text, archivoSemiBoldFile, fontSize)
	if err != nil {
		return -elementSize
	}
	return haRoomBoxCenterX - w/2 - gap - elementSize
}

// roomOccupied reports whether pipe field i carries the "OCC" flag.
func roomOccupied(raw string, i int) bool {
	field := weatherPipeField(raw, i)
	_, flags, _ := strings.Cut(field, "@")
	return flags == "OCC"
}

// elementText returns the TextMessage of the element with the given ID,
// or "" if not found.
func elementText(elements []frame.DispElement, id int) string {
	for _, e := range elements {
		if e.ID == id {
			return e.TextMessage
		}
	}
	return ""
}

// setElementX sets StartX on the element with the given ID, in place.
func setElementX(elements []frame.DispElement, id, x int) {
	for i := range elements {
		if elements[i].ID == id {
			elements[i].StartX = x
			return
		}
	}
}

// setElementURL sets Url on the element with the given ID, in place.
func setElementURL(elements []frame.DispElement, id int, url string) {
	for i := range elements {
		if elements[i].ID == id {
			elements[i].Url = url
			return
		}
	}
}

// bgHomeAssistant is the on-device path for this scene's (single, flat)
// background, pre-pushed at startup.
const bgHomeAssistant = "/userdata/wallclock_bg_homeassistant.jpg"
