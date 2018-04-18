package mawt

// This file contains the interfaces to the TeamNorCal animation suite
//
// This package will request sets of frames from the animation library that
// represent actions occuring within the portal and will play these back
// to the fadecandy server interface

import (
	"image/color"
	"math"

	"github.com/karlmutch/errors"

	"github.com/TeamNorCal/animation"

	"github.com/kellydunn/go-opc"
	"github.com/lucasb-eyer/go-colorful"
)

var (
	// cfgStrands represents boards using the first dimension, the integer values represent
	// individual strands with the length of each being the value
	cfgStrands = [][]int{
		[]int{8},
	}

	deviceMap = animation.NewMapping(cfgStrands)
)

func init() {
	deviceMap.AddUniverse("test", []animation.PhysicalRange{
		animation.PhysicalRange{Board: 0, Strand: 0, StartPixel: 0, Size: 8},
	})

	return
}

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

// test8LED is used to send an 8 LED test pattern based on the simple resonator
// patterns seen on the portal
//
// brightness can be used to scale the brightness, 0 = off, 0.01 1% brightness
// 1.0 and above 100%
//
func test8LED(fc *FadeCandy, brightness float64, status *Status) (err errors.Error) {

	clr := colorful.Color{}

	m := opc.NewMessage(0)
	m.SetLength(uint16(8 * 3))

	directions := map[string]int{"E": 0, "NE": 1, "N": 2, "NW": 3, "W": 4, "SW": 5, "S": 6, "SE": 7}
	levels := make([]int, 8, 8)
	for _, res := range status.Resonators {
		if pos, isPresent := directions[res.Position]; isPresent {
			levels[pos] = int(res.Health)
		}
	}

	for i := 0; i < 8; i++ {
		// For now very simple just the faction and presence of the resonator
		switch status.Faction {
		case "E":
			if 0 != levels[i] {
				clr = enlHealth[levels[i]]
			} else {
				clr = colorful.MakeColor(color.Black)
			}
		case "R":
			if 0 != levels[i] {
				clr = resHealth[levels[i]]
			} else {
				clr = colorful.MakeColor(color.Black)
			}
		default:
			clr = colorful.Hsv(0, 0, 5)
		}

		if brightness < 1.0-epsilon {
			if diff := math.Abs(brightness - 1); diff <= epsilon {
				h, c, l := clr.Hcl()
				l = (l * brightness) / 100.0
				clr = colorful.Hcl(h, c, l)
			}
		}
		r, g, b := clr.RGB255()

		m.SetPixelColor(i, r, g, b)
	}

	return fc.Send(m)
}
