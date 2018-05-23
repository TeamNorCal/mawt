package mawt

// This module implements a sound effects generator that tracks the state of portal(s)
// based on portal state messages events that are sent to it, it then
// in turn queues up sounds effects to match.

import (
	"github.com/TeamNorCal/mawt/model"
	"github.com/karlmutch/errors"
)

type Gateway struct {
}

func (*Gateway) Start(server string, debug bool, errorC chan<- errors.Error, quitC <-chan struct{}) (tectC chan *model.PortalMsg, subscribeC chan chan *model.PortalMsg) {

	tectC, subscribeC = startFanOut(quitC)

	// After creating the broadcast channel we add a listener
	// for the sounds effects so that it can process detected
	// state changes etc
	//
	go StartSFX(subscribeC, errorC, quitC)

	StartFadeCandy(server, subscribeC, debug, errorC, quitC)

	return tectC, subscribeC
}
