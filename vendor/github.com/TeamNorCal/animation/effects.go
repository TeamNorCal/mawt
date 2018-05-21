package animation

/*
Contains definitions of various animation effects that can be applied. These
effects should implement interface Animation
*/

import (
	"image/color"
	"log"
	"math"
	"os"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
)

// RGBAFromRGBHex converts a 24-bit hex color value into a color.RGBA value
func RGBAFromRGBHex(hexColor uint32) color.RGBA {
	return color.RGBA{uint8(hexColor >> 16 & 0xff), uint8(hexColor >> 8 & 0xff), uint8(hexColor & 0xff), 0xff}
}

// InterpolateSolid transitions from one solid color (applied to all elements)
// to another solid color
type InterpolateSolid struct {
	startColor, endColor colorful.Color
	duration             time.Duration
	startTime            time.Time
	startOnCurrent       bool // Capture the current color and use it as the start color?
}

var fxlog = log.New(os.Stdout, "(EFFECT) ", 0)

// NewInterpolateSolidHexRGB creates an InterpolateSolid effect, given hex-encoded 24-bit RGB colors
func NewInterpolateSolidHexRGB(startColor, endColor uint32, duration time.Duration) *InterpolateSolid {
	startRGBA := RGBAFromRGBHex(startColor)
	endRGBA := RGBAFromRGBHex(endColor)
	return &InterpolateSolid{startColor: colorful.MakeColor(startRGBA), endColor: colorful.MakeColor(endRGBA), duration: duration}
}

// NewInterpolateSolid creates an InterpolateSolid effect
func NewInterpolateSolid(startColor, endColor color.RGBA,
	duration time.Duration) *InterpolateSolid {
	return &InterpolateSolid{startColor: colorful.MakeColor(startColor), endColor: colorful.MakeColor(endColor), duration: duration}
}

// NewInterpolateToHexRGB interpolates from the current color of the universe (determined by sampling the first element)
// to the provided end color, specified as a 24-bit RGB hex value
func NewInterpolateToHexRGB(endColor uint32, duration time.Duration) *InterpolateSolid {
	// Create a standard effect with arbitrary start color
	effect := NewInterpolateSolidHexRGB(0x0, endColor, duration)
	// ...then set the magic flag
	effect.startOnCurrent = true
	return effect
}

// Start starts the effect
func (effect *InterpolateSolid) Start(startTime time.Time) {
	fxlog.Printf("Setting start time %v", startTime)
	effect.startTime = startTime
}

// Frame generates an animation frame
func (effect *InterpolateSolid) Frame(buf []color.RGBA, frameTime time.Time) (output []color.RGBA, endSeq bool) {
	//fxlog.Printf("Buf cap: %d len: %d\n", cap(buf), len(buf))
	if frameTime.After(effect.startTime.Add(effect.duration)) {
		fxlog.Printf("Done at time %v (start time %v)\n", frameTime, effect.startTime)
		return buf, true
	}

	// See if we need to find the current universe color and use it as the start color
	if effect.startOnCurrent {
		sc := buf[0]
		sc.A = 0xff // Avoid a 0 transparency (in the case of an uninitialized buffer) which makes go-colorful unhappy
		effect.startColor = colorful.MakeColor(sc)
		effect.startOnCurrent = false // Clear the flag to prevent this from being done again
	}

	elapsed := frameTime.Sub(effect.startTime)
	completion := elapsed.Seconds() / effect.duration.Seconds()
	//fxlog.Printf("Frame at %2.2f%%", completion*100.0)
	//	currColorful := effect.startColor.BlendLab(effect.endColor, completion)
	currColorful := effect.startColor.BlendLuv(effect.endColor, completion)
	currColor := colorfulToRGBA(currColorful)
	for i := 0; i < len(buf); i++ {
		buf[i] = currColor
	}
	return buf, false
}

func colorfulToRGBA(c colorful.Color) color.RGBA {
	r, g, b := c.RGB255()
	return color.RGBA{r, g, b, 0xff}
}

// Pulse is a repeating interpolation between two colors, in a pulsing fashion
type Pulse struct {
	c1        colorful.Color
	c2        colorful.Color
	period    time.Duration
	startTime time.Time
}

// NewDimmingPulse creates a pulse between a color and a dimmer version of itself.
// dimmingRatio determines the amount of dimming - 0.0 means 'black', while 1.0
// means 'color c'. 'period' is the time for a full dimming/brightening cycle
func NewDimmingPulse(c color.Color, dimmingRatio float64, period time.Duration) *Pulse {
	c1 := colorful.MakeColor(c)
	black := colorful.Color{0.0, 0.0, 0.0}
	// c2 := c1.BlendLuv(black, 1.0-dimmingRatio).Clamped()
	c2 := c1.BlendRgb(black, 1.0-dimmingRatio).Clamped()
	fxlog.Printf("Pulse colors: c1=%v, c2=%v\n", c1, c2)
	return &Pulse{
		c1:     c1,
		c2:     c2,
		period: period,
	}
}

// Start sets the start time of the pulse effect
func (effect *Pulse) Start(startTime time.Time) {
	effect.startTime = startTime
}

// Frame generates a frame of the Pulse animation. It will always return 'false' for endSeq. It returns
// the passed-in buffer
func (effect *Pulse) Frame(buf []color.RGBA, frameTime time.Time) (output []color.RGBA, endSeq bool) {
	// Use a sinusoidal pulse
	elapsed := frameTime.Sub(effect.startTime)
	phase := float64(elapsed%effect.period) / float64(effect.period)
	position := 0.5 - (math.Cos(2*math.Pi*phase) / 2.0)
	// color := effect.c1.BlendLuv(effect.c2, position).Clamped()
	color := effect.c1.BlendRgb(effect.c2, position).Clamped()
	rgba := colorfulToRGBA(color)
	for idx := range buf {
		buf[idx] = rgba
	}
	return buf, false
}

// Solid is a simple static solid color
type Solid color.RGBA

// NewSolid creates a Solid effect for the given color
func NewSolid(color color.RGBA) Solid {
	return Solid(color)
}

// Start the Solid effect - NOP
func (effect Solid) Start(startTime time.Time) {
	// NOP
}

// Frame creates a frame of the Solid effect
func (effect Solid) Frame(buf []color.RGBA, frameTime time.Time) (output []color.RGBA, endSeq bool) {
	for idx := range buf {
		buf[idx] = color.RGBA(effect)
	}
	return buf, false
}
