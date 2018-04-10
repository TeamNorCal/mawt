package mawt

// This module implements a sound effects generator that tracks the state of portal(s)
// based on portal state messages events that are sent to it, it then
// in turn queues up sounds effects to match.

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/karlmutch/errors"
)

type SFXState struct {
	current *Status
	last    *Status

	ambientC chan string
	sfxC     chan []string

	sync.Mutex
}

func (sfx *SFXState) process(msg *PortalMsg) (err errors.Error) {
	if msg == nil {
		return nil
	}

	sfx.Lock()
	current := msg.Status.DeepCopy()
	lastState := sfx.current
	sfx.Unlock()

	state := msg.Status.DeepCopy()

	// If there is no history add the fresh state as the previous state
	//

	forceAmbient := false
	if lastState == nil {
		lastState = current
		forceAmbient = true
	}

	// Sounds effects that are gathered as a result of state
	// and played back later
	sfxs := []string{}

	factionChange := lastState.Faction != state.Faction

	if factionChange {

		// e-loss, r-loss, n-loss
		faction := strings.ToLower(lastState.Faction)
		effect := faction + "-loss"
		sfxs = append(sfxs, effect)

		// e-capture, r-capture, n-capture
		faction = strings.ToLower(state.Faction)
		effect = faction + "-capture"
		sfxs = append(sfxs, effect)
	} else {
		// If the new state was not a change of faction did the number
		// of resonators change
	}

	if factionChange || forceAmbient {
		ambient := ""
		faction := strings.ToLower(state.Faction)
		ambient = faction + "-ambient"
		forceAmbient = false
		go func() {
			select {
			case sfx.ambientC <- ambient:
			case <-time.After(time.Second):
			}
		}()
	}

	// Check for sound effects that need to be played
	if len(sfxs) != 0 {
		go func() {
			select {
			case sfx.sfxC <- sfxs:
			case <-time.After(time.Second):
			}
		}()
	}

	// Save the new state as the last known state
	sfx.Lock()
	sfx.last = sfx.current
	sfx.current = current
	sfx.Unlock()

	return nil
}

// StartSFX will add itself to the subscriptions for portal messages
func StartSFX(subscribeC chan chan *PortalMsg, errorC chan<- errors.Error, quitC <-chan struct{}) {

	sfx := &SFXState{
		ambientC: make(chan string, 3),
		sfxC:     make(chan []string, 3),
	}

	if err := InitAudio(sfx.ambientC, sfx.sfxC, errorC, quitC); err != nil {
		select {
		case errorC <- err:
		case <-time.After(100 * time.Millisecond):
			fmt.Fprintf(os.Stderr, err.Error())
		}
	}

	// Allow a lot of messages to queue up as we will only process the last one anyway
	updateC := make(chan *PortalMsg, 10)
	defer close(updateC)

	// Subscribe to portal events
	subscribeC <- updateC

	// Attempt to set the default audio effects
	select {
	case sfx.ambientC <- "n-ambient":
	case <-time.After(100 * time.Millisecond):
		fmt.Fprintf(os.Stderr, "unable to start the neutral ambient SFX")
	}

	// Now listen to the subscribed portal events
	for {
		lastMsg := &PortalMsg{}

		select {
		case msg := <-updateC:
			// Only process the most recent portal status msg for the Home portal in
			// the channel, if we are backed up
			if msg.Home {
				lastMsg = msg.DeepCopy()
			}

			if len(updateC) == 0 && lastMsg != nil {
				if err := sfx.process(lastMsg); err != nil {
					select {
					case errorC <- err:
					case <-time.After(20 * time.Millisecond):
						fmt.Fprintf(os.Stderr, err.Error())
					}
				}
				lastMsg = nil
			}
		case <-quitC:
			return
		}
	}
}
