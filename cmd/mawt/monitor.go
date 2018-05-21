package main

import (
	"fmt"

	"github.com/TeamNorCal/mawt/model"
)

// This file implements a monitor that subscribe to and displays
// the tecthulhu events using event subscription

func runMonitoring(subscribeC chan chan *model.PortalMsg, quitC <-chan struct{}) {

	statusC := make(chan *model.PortalMsg, 1)
	defer close(statusC)
	subscribeC <- statusC

	for {
		select {
		case msg := <-statusC:
			logger.Debug(fmt.Sprintf("%+v", msg))
		case <-quitC:
			return
		}
	}
}
