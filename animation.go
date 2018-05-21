package mawt

// This file contains the interfaces to the TeamNorCal animation suite
//
// This package will request sets of frames from the animation library that
// represent actions occuring within the portal and will play these back
// to the fadecandy server interface

import (
	"math"

	"github.com/TeamNorCal/animation"

	"github.com/lucasb-eyer/go-colorful"
)

// Data structure to hold local information about the universes
type universe struct {
	id     uint
	ranges []animation.PhysicalRange
}

var animPortal = animation.NewPortal()

type Color struct {
	R, G, B uint8
}

var (
	enlHealth = [101]colorful.Color{}
	resHealth = [101]colorful.Color{}

	// Go epsilon can be determined for a specific platform based on
	// advice in, https://github.com/golang/go/issues/966
	epsilon = math.Nextafter(1, 2) - 1
)

func init() {
	// Gradient values for health from 0 -> Enlightened green full strength
	c1, _ := colorful.Hex("#0A3306")
	c2, _ := colorful.Hex("#36FF1F")
	for i := 0; i != len(enlHealth); i++ {
		enlHealth[i] = c1.BlendLab(c2, float64(i)/float64(len(enlHealth)))
	}

	// Gradient values for health from 0 -> Resistance blue full strength
	c1, _ = colorful.Hex("#00066B")
	c2, _ = colorful.Hex("#000FFF")
	for i := 0; i != len(resHealth); i++ {
		resHealth[i] = c1.BlendLab(c2, float64(i)/float64(len(enlHealth)))
	}
}
