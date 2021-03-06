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

	animationModel "github.com/TeamNorCal/animation/model"
	"github.com/TeamNorCal/mawt/model"
	"github.com/go-stack/stack"
	"github.com/karlmutch/errors"

	"github.com/cnf/structhash"

	"github.com/kellydunn/go-opc"
)

var (
	updating sync.Mutex
)

type LastStatus struct {
	status *model.Status
	sync.Mutex
}

type FadeCandy struct {
	oc  *opc.Client
	nop bool // Used to set the server into a test mode with no fcserver present
}

// This file contains the implementation of a listener for tecthulhu events that will on
// a regular basis lift the last known state of the portal and will update the fade-candy as needed

func StartFadeCandy(server string, subscribeC chan chan *model.PortalMsg, debug bool, errorC chan<- errors.Error, quitC <-chan struct{}) (fc *FadeCandy) {

	statusC := make(chan *model.PortalMsg, 1)
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
		nop: server == "/dev/null",
	}

	go fc.run(status, server, time.Duration(200*time.Millisecond), debug, errorC, quitC)

	return fc
}

func (fc *FadeCandy) run(status *LastStatus, server string, refresh time.Duration,
	debug bool, errorC chan<- errors.Error, quitC <-chan struct{}) {

	last := []byte{}

	if !fc.nop {
		if fc.oc == nil {
			fc.oc = opc.NewClient()
		}

		if errGo := fc.oc.Connect("tcp", server); errGo != nil {

			fc.oc = nil

			err := errors.Wrap(errGo).With("url", server).With("stack", stack.Trace().TrimRuntime())

			select {
			case errorC <- err:
			case <-time.After(100 * time.Millisecond):
				fmt.Fprintln(os.Stderr, err.Error())
			}
		}
	}

	sink := NewSink()

	for {
		// Start the LED command message pusher
		go fc.RunLoop(sink, debug, errorC, quitC)

		tick := time.NewTicker(refresh)
		defer tick.Stop()

		select {
		case <-tick.C:
			status.Lock()
			copied := status.status.DeepCopy()
			status.Unlock()

			// Portal status not yet available
			if copied.Faction == "" {
				continue
			}

			hash := structhash.Md5(copied, 1)
			if bytes.Compare(last, hash) != 0 {
				last = hash
				sink.UpdateStatus(copied)
			}
		case <-quitC:
			return
		}
	}
}

func (fc *FadeCandy) Send(m *opc.Message) (err errors.Error) {
	if fc.nop {
		return nil
	}
	if fc.oc == nil {
		return errors.New("fadecandy server not online").With("stack", stack.Trace().TrimRuntime())
	}

	if m == nil {
		return errors.New("invalid message").With("stack", stack.Trace().TrimRuntime())
	}

	if errGo := fc.oc.Send(m); errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}

func (fc *FadeCandy) RunLoop(sink *statusSink, debug bool, errorC chan<- errors.Error, quitC <-chan struct{}) (err errors.Error) {

	defer close(errorC)

	refresh := time.Duration(30 * time.Millisecond)
	tick := time.NewTicker(refresh)
	defer tick.Stop()

	opcError := errors.New("")

	for {
		select {
		case <-tick.C:
			updating.Lock()
			// Populate the logical buffers
			frameData := sink.GetFrame(time.Now())

			// Copy the logical buffers into the physical buffers

			// for _, id := range universes {
			// 	if errGo := devices.UpdateUniverse(id, sr.UniverseData(id)); errGo != nil {
			// 		sendErr(errorC, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()))
			// 	}
			// }
			//
			// // Iterate across physical strands sending updates to the
			// // FadeCandies, possibly diffing to previous frame to see if necessary
			// deviceStrands, errGo := GetStrands()
			// if errGo != nil {
			// 	sendErr(errorC, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()))
			// 	updating.Unlock()
			// 	continue
			// }

			newRefresh := refresh
			if opcError = fc.updateStrands(frameData, debug, errorC); opcError != nil {
				newRefresh = time.Duration(250 * time.Millisecond)
			} else {
				newRefresh = time.Duration(30 * time.Millisecond)
			}
			updating.Unlock()

			if newRefresh != refresh {
				refresh = newRefresh
				tick.Stop()
				tick = time.NewTicker(refresh)
			}

		case <-quitC:
			return
		}
	}
}

var (
	headingOnce sync.Once

	onceBody = func() {
		fmt.Printf("\x1b[1;0H\x1b[0J     ")
		for i := 1; i != 10; i++ {
			fmt.Printf("         %d", i)
		}
		fmt.Printf("\n     ")
		for i := 1; i != 10; i++ {
			fmt.Print("1234567890")
		}
		fmt.Printf("\x1b[20;0H")
	}
)

func (fc *FadeCandy) updateStrands(data []animationModel.ChannelData, debug bool, errorC chan<- errors.Error) (err errors.Error) {
	if debug {
		headingOnce.Do(onceBody)
		fmt.Printf("\x1b[3;0H")
	}
	for _, channelData := range data {
		// The OPC protocol assigns a channel per LED strand, and supports a maximum of
		// 255 strands per server.  Channel 0 is a broadcast channel.
		channel := uint8(channelData.ChannelNum)
		strip := fmt.Sprintf("\x1b[%d;0H%02d → ", channel+3, channel)

		// Prepare a message for this strand that has 3 bytes per LED
		m := opc.NewMessage(channel)
		m.SetLength(uint16(len(channelData.Data) * 3))
		for i, rgba := range channelData.Data {
			r, g, b, a := rgba.RGBA()
			if a == 0 {
				r = 0
				g = 0
				b = 0
			}
			strip += fmt.Sprintf("\x1b[38;2;%d;%d;%dm█\x1b[0m", uint8(r), uint8(g), uint8(b))
			m.SetPixelColor(i, uint8(r), uint8(g), uint8(b))
		}
		if err = fc.Send(m); err != nil {
			sendErr(errorC, err)
		}
		if debug {
			fmt.Println(strip)
			fmt.Printf("\x1b[32;0H")
		}
	}
	return err
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
