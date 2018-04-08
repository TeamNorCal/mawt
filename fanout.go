package mawt

import (
	"fmt"
	"sync"
	"time"
)

var (
	subs = &Subs{
		subs: []chan *PortalMsg{},
	}
)

type Subs struct {
	subs []chan *PortalMsg
	sync.Mutex
}

// startFanOut implement a broadcast mechanisim for accepting portal state messages
// and relaying then to subscribers.  The function returns a single channel
// to which portal update messages get sent and, a channel that can be used to add
// listeners
//
func startFanOut(quitC <-chan struct{}) (inC chan *PortalMsg, subC chan chan *PortalMsg) {

	inC = make(chan *PortalMsg, 1)
	subC = make(chan chan *PortalMsg, 1)

	go func(quitC <-chan struct{}) {
		defer fmt.Println("fanout stopped")
		for {
			select {
			case <-quitC:
				return
			case sub := <-subC:
				if nil != sub {
					subs.Lock()
					subs.subs = append(subs.subs, sub)
					subs.Unlock()
					fmt.Println("subscription added")
				}
			case msg := <-inC:
				// The subscriptions are notified of a message and are groomed out
				// on unrecoverable failures using https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
				subs.Lock()
				fmt.Printf("tecthulhu status (%d)\n", len(subs.subs))
				newSubs := subs.subs[:0]
				for _, ch := range subs.subs {
					func() {
						defer func() {
							if r := recover(); r == nil {
								newSubs = append(newSubs, ch)
								return
							}
							fmt.Println("subscription dropped failed to send")
						}()
					}()
					select {
					case ch <- msg:
					case <-time.After(250 * time.Millisecond):
						fmt.Println("subscription failed to send")
					}
				}
				subs.subs = newSubs
				subs.Unlock()
			}
		}
	}(quitC)

	return inC, subC
}
