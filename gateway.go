package mawt

// This module implements a sound effects generator that tracks the state of portal(s)
// based on portal state messages events that are sent to it, it then
// in turn queues up sounds effects to match.

import (
	"github.com/karlmutch/errors"
)

type Gateway struct {
}

func (*Gateway) Start(server string, errorC chan<- errors.Error, quitC <-chan struct{}) (tectC chan *PortalMsg, subscribeC chan chan *PortalMsg) {

	tectC, subscribeC = startFanOut(quitC)

	// After creating the broadcast channel we add a listener
	// for the sounds effects so that it can process detected
	// state changes etc
	//
	go StartSFX(subscribeC, errorC, quitC)

	StartFadeCandy(server, subscribeC, errorC, quitC)

	return tectC, subscribeC
}
