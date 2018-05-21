/*
Package animation contains implementations of animation routines, generating
buffers representing LED color values for consecutive frames.
*/
package animation

import (
	"image/color"
	"time"
)

// Animation is an interface for types that support generation of animation
// frames
type Animation interface {
	// Start the effect with the provided start time. All frame times should be at
	// this time or later
	Start(startTime time.Time)

	// Generate a frame appropriate for the given Time
	// buf is a buffer into which the frame should be generated. The buffer size
	// determines the number of LEDs to generate a frame for. Values are RGB color
	// values; the alpha channel is unused (or could be used for a white channel)
	// Returns true if the current animation completed a cycle; false otherwise
	Frame(buf []color.RGBA, frameTime time.Time) (output []color.RGBA, endSeq bool)
}
