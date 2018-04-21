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
	// individual strands with the length of each being the value.  This can contain 0 length
	// strands
	cfgStrands = [][]int{}

	deviceMap = animation.Mapping{}

	// In order to track the information needed to drive the animation library
	// add data structures are able to track what was pushed into it during
	// the setup phase that can be used when invoking library calls later
	// without hard coded values and loops sitting around.
	universes = map[string][]animation.PhysicalRange{
		"test": []animation.PhysicalRange{
			animation.PhysicalRange{
				Board:      0,
				Strand:     0,
				StartPixel: 0,
				Size:       8,
			},
		},
	}

	// The animation relies on a sequence runner that contains an
	// array of universes and a set of pixels for each universe
	// that is operated on as a single ribbon of consecutive
	// LEDs.  The universe ribbon lengths can be calculated during
	// the setup phase and are kept as a global variable for use
	// by clients of the animation library when ever they need
	// a sequence runner
	universeSizes = []uint{}
)

func init() {
	// Calculate the maximum extent of every strand on all boards from
	// the logical viewpoint master configuration
	//
	// This go map is board major, and strand minor with maximum extents
	// as the inner most value
	//
	boards := map[uint]map[uint]int{}
	for _, uni := range universes {
		for _, physRange := range uni {
			if _, isPresent := boards[physRange.Board]; !isPresent {
				boards[physRange.Board] = map[uint]int{}
			}
			if extent, isPresent := boards[physRange.Board][physRange.Strand]; !isPresent {
				boards[physRange.Board][physRange.Strand] = int(physRange.StartPixel + physRange.Size)
			} else {
				if extent < int(physRange.StartPixel+physRange.Size) {
					boards[physRange.Board][physRange.Strand] = int(physRange.Size)
				}
			}
		}
	}

	// Now for every board get the length of its strand map and use that to initial the arrays needed
	// for physical boards, and stand lengths
	cfgStrands = make([][]int, len(boards))
	for i, board := range boards {
		// Add the strand array using the length of the individual board maps
		cfgStrands[i] = make([]int, len(board))
		// Now within the board map visit each known strand and places its length into
		// the indexed slice for the physical view
		for strand, strandLen := range board {
			cfgStrands[i][strand] = strandLen
		}
	}

	// Everything was placed into a map to prevent complex slice extensions so now go through
	// and get an appropriately sized array for all boards and their stands

	// Now that we have the mountain of complexity behind us we can create a physical viewpoint
	// across devices which is a summary of the universe viewed from a physical perspective
	deviceMap = animation.NewMapping(cfgStrands)

	// Now add the universes from our logical representation into an array of lengths
	//
	// The animation library uses an implied assumption that universes are added
	// with IDs that related to positions in a slice as each universe is added,
	// this assumption is exploited here so be careful in the future with any changes
	universeSizes = make([]uint, 0, len(universes))
	for k, v := range universes {
		deviceMap.AddUniverse(k, v)
		size := uint(0)
		for _, aRange := range v {
			size += uint(aRange.Size)
		}
		universeSizes = append(universeSizes, size)
	}
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

func GetSeqRunner() (sr *animation.SequenceRunner, err errors.Error) {
	return animation.NewSequenceRunner(universeSizes), nil
}

func GetUniverses() (devices animation.Mapping, uniIds []uint, err errors.Error) {
	uniIds = make([]uint, len(universes))
	for i := uint(0); i < uint(len(uniIds)); i++ {
		uniIds[i] = i
	}
	return deviceMap, uniIds, nil
}

func GetStrands() (deviceStrands [][]int, err errors.Error) {
	return cfgStrands, nil
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
