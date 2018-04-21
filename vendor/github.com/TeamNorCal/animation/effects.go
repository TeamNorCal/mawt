package animation

/*
Contains definitions of various animation effects that can be applied. These
effects should implement interface Animation
*/

import (
	"image/color"
	"time"
)

// InterpolateSolid transitions from one solid color (applied to all elements)
// to another solid color
type InterpolateSolid struct {
	startColor, endColor color.RGBA
	duration             time.Duration
	startTime            time.Time
}

// NewInterpolateSolid creates an InterpolateSolid effect
func NewInterpolateSolid(startColor, endColor color.RGBA,
	duration time.Duration) InterpolateSolid {
	return InterpolateSolid{startColor, endColor, duration, time.Now()}
}

// Frame generates an animation frame
func (effect *InterpolateSolid) Frame(buf []color.RGBA, frameTime time.Time) bool {
	return true
}
