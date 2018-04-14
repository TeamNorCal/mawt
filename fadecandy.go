package mawt

// This file contains a function that when started will listen for status
// messages for the home portal and will update a data structure that another
// function checks for on a regular basis and uses to update LEDs etc attached
// to one or more fadecandy device(s)

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"os"
	"sync"
	"time"

	"github.com/go-stack/stack"
	"github.com/karlmutch/errors"

	"github.com/cnf/structhash"

	"github.com/kellydunn/go-opc"

	"github.com/lucasb-eyer/go-colorful"
)

type LastStatus struct {
	status *Status
	sync.Mutex
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

// This file contains the implementation of a listener for tecthulhu events that will on
// a regular basis lift the last known state of the portal and will update the fade-candy as needed

func StartFadeCandy(server string, subscribeC chan chan *PortalMsg, errorC chan<- errors.Error, quitC <-chan struct{}) {

	statusC := make(chan *PortalMsg, 1)
	subscribeC <- statusC

	status := &LastStatus{}

	go func() {
		defer close(statusC)
		for {
			select {
			case msg := <-statusC:
				if nil == msg {
					continue
				}
				if msg.Home {
					status.Lock()
					status.status = msg.Status.DeepCopy()
					status.Unlock()
				}
			case <-quitC:
				return
			}
		}
	}()

	go runFadeCandyOPC(status, server, time.Duration(200*time.Millisecond), errorC, quitC)
}

// test8LED is used to send an 8 LED test pattern based on the simple resonator
// patterns seen on the portal
//
// brightness can be used to scale the brightness, 0 = off, 0.01 1% brightness
// 1.0 and above 100%
//
func test8LED(oc *opc.Client, brightness float64, status *Status) (err errors.Error) {

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

	if errGo := oc.Send(m); errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}

func runFadeCandyOPC(status *LastStatus, server string, refresh time.Duration, errorC chan<- errors.Error, quitC <-chan struct{}) {

	last := []byte{}

	oc := opc.NewClient()
	if errGo := oc.Connect("tcp", server); errGo != nil {

		err := errors.Wrap(errGo).With("url", server).With("stack", stack.Trace().TrimRuntime())

		select {
		case errorC <- err:
		case <-time.After(100 * time.Millisecond):
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}

	for {
		select {
		case <-time.After(refresh):
			status.Lock()
			copied := status.status.DeepCopy()
			status.Unlock()

			hash := structhash.Md5(copied, 1)
			if bytes.Compare(last, hash) != 0 {
				last = hash
				if err := test8LED(oc, 0.15, copied); err != nil {
					select {
					case errorC <- err.With("url", server):
					case <-time.After(100 * time.Millisecond):
						fmt.Fprintln(os.Stderr, err.Error())
					}
				}
			}
		case <-quitC:
			return
		}
	}
}
