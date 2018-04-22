package mawt

// This file contains a function that when started will listen for status
// messages for the home portal and will update a data structure that another
// function checks for on a regular basis and uses to update LEDs etc attached
// to one or more fadecandy device(s)

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-stack/stack"
	"github.com/karlmutch/errors"
	// colorful "github.com/lucasb-eyer/go-colorful"

	"github.com/cnf/structhash"

	"github.com/kellydunn/go-opc"
)

type LastStatus struct {
	status *Status
	sync.Mutex
}

type FadeCandy struct {
	oc *opc.Client
}

// This file contains the implementation of a listener for tecthulhu events that will on
// a regular basis lift the last known state of the portal and will update the fade-candy as needed

func StartFadeCandy(server string, subscribeC chan chan *PortalMsg, errorC chan<- errors.Error, quitC <-chan struct{}) (fc *FadeCandy) {

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

	fc = &FadeCandy{
		oc: opc.NewClient(),
	}

	go fc.run(status, server, time.Duration(200*time.Millisecond), errorC, quitC)

	return fc
}

func (fc *FadeCandy) run(status *LastStatus, server string, refresh time.Duration, errorC chan<- errors.Error, quitC <-chan struct{}) {

	last := []byte{}

	if errGo := fc.oc.Connect("tcp", server); errGo != nil {

		err := errors.Wrap(errGo).With("url", server).With("stack", stack.Trace().TrimRuntime())

		select {
		case errorC <- err:
		case <-time.After(100 * time.Millisecond):
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}

	// Start the LED command message pusher
	go fc.RunLoop(errorC, quitC)

	for {
		select {
		case <-time.After(refresh):
			status.Lock()
			copied := status.status.DeepCopy()
			status.Unlock()

			hash := structhash.Md5(copied, 1)
			if bytes.Compare(last, hash) != 0 {
				last = hash
				// TODO Call InitSequence instead of the hard coded test
				if err := test8LED(fc, 0.15, copied); err != nil {
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

func (fc *FadeCandy) Send(m *opc.Message) (err errors.Error) {
	if errGo := fc.oc.Send(m); errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}

func (fc *FadeCandy) RunLoop(errorC chan<- errors.Error, quitC <-chan struct{}) (err errors.Error) {

	defer close(errorC)

	sr, err := GetSeqRunner()
	if err != nil {
		return err
	}

	devices, universes, err := GetUniverses()
	if err != nil {
		return err
	}

	for {
		select {
		case <-time.After(30 * time.Millisecond):
			// Populate the logical buffers
			sr.ProcessFrame(time.Now())

			// Copy the logical buffers into the physical buffers

			for _, id := range universes {
				devices.UpdateUniverse(id, sr.UniverseData(id))
			}

			// Iterate across physical strands sending updates to the
			// FadeCandies, possibly diffing to previous frame to see if necessary
			deviceStrands, errGo := GetStrands()
			if errGo != nil {
				sendErr(errorC, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()))
				continue
			}
			for device, strands := range deviceStrands {
				for strand, strandLen := range strands {
					if strandLen == 0 {
						continue
					}
					// The following gives us an array of RGBA as a linear arrangement of LEDs
					// for the indicated strand
					strandData, errGo := devices.GetStrandData(uint(device), uint(strand))
					if errGo != nil {
						sendErr(errorC, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()))
						continue
					}

					// Prepare a message for this strand that has 3 bytes per LED
					m := opc.NewMessage(0)
					m.SetLength(uint16(len(strandData) * 3))
					for i, rgba := range strandData {
						r, g, b, a := rgba.RGBA()
						if a == 0 {
							r = 0
							g = 0
							b = 0
						}
						m.SetPixelColor(i, uint8(r), uint8(g), uint8(b))
					}
					if err := fc.Send(m); err != nil {
						sendErr(errorC, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()))
					}
				}
			}
		case <-quitC:
			return
		}
	}
}

func sendErr(errorC chan<- errors.Error, err errors.Error) {
	if errorC == nil {
		return
	}
	select {
	case errorC <- err:
	case <-time.After(20 * time.Millisecond):
		fmt.Println(fmt.Sprintf("%+v", err.Error()))
	}
}
