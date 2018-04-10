package mawt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-stack/stack"
	"github.com/karlmutch/errors"
)

// This module implements a module to handle communications
// with the tecthulhu device.  These devices can provide a WiFi
// like capability, however the original documentation appears
// to indicate a serial like communications peripheral

type tResonator struct {
	Position string `json:"position"`
	Level    int    `json:"level"`
	Health   int    `json:"health"`
	Owner    string `json:"owner"`
}

type tStatus struct {
	Title      string       `json:"title"`
	Owner      string       `json:"owner"`
	Level      int          `json:"level"`
	Health     int          `json:"health"`
	Faction    string       `json:"controllingFaction"`
	Mods       []string     `json:"mods"`
	Resonators []tResonator `json:"resonators"`
}

type tPortalStatus struct {
	State tStatus `json:"status"`
}

type PortalMon interface {
	Run(quitC <-chan struct{})
}

type tecthulhu struct {
	url     url.URL
	home    bool
	statusC chan<- *PortalMsg
	errorC  chan<- errors.Error
}

func NewTecthulu(url url.URL, home bool, statusC chan<- *PortalMsg, errorC chan<- errors.Error) (tec *tecthulhu) {
	return &tecthulhu{
		url:     url,
		home:    home,
		statusC: statusC,
		errorC:  errorC,
	}
}

func (tec *tPortalStatus) status() (state *portalStatus) {
	state = &portalStatus{
		Status: Status{
			Title:      tec.State.Title,
			Owner:      tec.State.Owner,
			Level:      float32(tec.State.Level),
			Health:     float32(tec.State.Health),
			Faction:    tec.State.Faction,
			Mods:       []Mod{},
			Resonators: []Resonator{},
		},
	}
	for _, res := range tec.State.Resonators {
		state.Status.Resonators = append(state.Status.Resonators,
			Resonator{
				Position: res.Position,
				Level:    float32(res.Level),
				Health:   float32(res.Health),
				Owner:    res.Owner,
			})
	}
	switch tec.State.Faction {
	case "1":
		state.Status.Faction = "E" // Wnlightened
	case "2":
		state.Status.Faction = "R" // Resistance
	default:
		state.Status.Faction = "N" // Neutral

	}
	for i, modStr := range tec.State.Mods {
		newMod := Mod{Slot: float32(i)}
		modParts := strings.Split(modStr, "-")
		if len(modParts) == 2 {
			// 'C'ommon, 'R'are, 'VR' very rare
			newMod.Rarity = modParts[1]
		}
		newMod.Type = modParts[0]
		state.Status.Mods = append(state.Status.Mods, newMod)
	}
	return state
}

// checkPortal can be used to extract status information from the portal
//
func (tec *tecthulhu) checkPortal() (status *portalStatus, err errors.Error) {

	body := []byte{}

	switch tec.url.Scheme {
	case "http":
		resp, errGo := http.Get(tec.url.String())
		if errGo != nil {
			return nil, errors.Wrap(errGo).With("url", tec.url).With("stack", stack.Trace().TrimRuntime())
		}

		body, errGo = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if errGo != nil {
			return nil, errors.Wrap(errGo).With("url", tec.url).With("stack", stack.Trace().TrimRuntime())
		}

	case "serial":
		errGo := fmt.Errorf("Unknown scheme %s for the tecthulhu device is not yet implemented", tec.url.Scheme)
		return nil, errors.Wrap(errGo).With("url", tec.url).With("stack", stack.Trace().TrimRuntime())

	default:
		errGo := fmt.Errorf("Unknown scheme %s for the tecthulhu device URI", tec.url.Scheme)
		return nil, errors.Wrap(errGo).With("url", tec.url).With("stack", stack.Trace().TrimRuntime())
	}

	// Parse into the tecthulhu specific format and then convert to
	// the canonical format used by the concentrator which we assume
	// is a reference format for portal data and meta data
	//
	tecStatus := &tPortalStatus{}

	errGo := json.Unmarshal(body, &tecStatus)
	if errGo != nil {
		return nil, errors.Wrap(errGo).With("url", tec.url).With("body", string(body)).With("stack", stack.Trace().TrimRuntime())
	}
	status = tecStatus.status()
	return status, err
}

func (tec *tecthulhu) sendStatus() {
	// Perform a regular status check with the portal
	// and return the received results  to listeners using
	// the channel
	//
	// Use  a TCP and USB Serial handler function
	status, err := tec.checkPortal()

	if err != nil {
		go func(err errors.Error) {
			select {
			case tec.errorC <- err:
			case <-time.After(500 * time.Millisecond):
				fmt.Fprintf(os.Stderr, "could not send error for portal status update %s\n", err.Error())
			}
		}(err)
		return
	}

	msg := &PortalMsg{
		Status: status.Status,
		Home:   tec.home,
	}

	select {
	case tec.statusC <- msg:
	case <-time.After(750 * time.Millisecond):
		go func() {
			err := errors.New("portal status dropped").With("url", tec.url).With("stack", stack.Trace().TrimRuntime())
			select {
			case tec.errorC <- err:
			case <-time.After(2 * time.Second):
				fmt.Fprintf(os.Stderr, "could not send error for portal status update %s\n", err.Error())
			}
		}()
	}
}

// startPortal listens to a tecthulhu device and returns
// regular reports on the status of the portal with which it
// is associated
//
func (tec *tecthulhu) Run(quitC <-chan struct{}) {

	poll := time.NewTicker(time.Second)
	defer poll.Stop()

	for {
		select {
		case <-poll.C:
			tec.sendStatus()

		case <-quitC:
			return
		}
	}
}
