package animation

import (
	"encoding/json"
	"fmt"
	"image/color"

	"github.com/TeamNorCal/portalmodel"
)

// OpcChannel represents a channel in Open Pixel Controller parlance. Channel is a logical entity;
// the fcserver config file maps this to pixels on strands on particular FadeCandy boards
// fcserver configuration must honor this enumeration
type OpcChannel int

// List of channels in the portal - 1-8 are resonators, and 9-24 are tower windows
const (
	_ = iota // OPC channels are 1 based, with 0 being broadcast
	Reso1
	Reso2
	Reso3
	Reso4
	Reso5
	Reso6
	Reso7
	Reso8
	Level1_1 // 9
	Level1_2
	Level2_1
	Level2_2
	Level3_1
	Level3_2
	Level4_1
	Level4_2
	Level5_1
	Level5_2
	Level6_1
	Level6_2
	Level7_1
	Level7_2
	Level8_1
	Level8_2
)

// Faction represents an Ingress fasction
type Faction int

const (
	// NEU means Neutral
	NEU Faction = iota
	// ENL means Enlightened!
	ENL
	// RES means Resistance :(
	RES
)

// Universe defines a universe from the perspective of the animation engine.
// It consists of an index in the array of returned frame data, and a size
type Universe struct {
	Index int
	Size  int
}

// Universes defines universe data for the model. It's indexed by logical name,
// with values containing size of the universe and index into the array of
// frame data
var Universes map[string]Universe

// ResonatorStatus is the status of a resonator, you'll be surprised to hear
type ResonatorStatus struct {
	Level  int     // Resonator level, 0-8
	Health float32 // Resonator health, 0-100
}

func (msg *PortalStatus) deepCopy() (cpy *PortalStatus) {
	cpy = &PortalStatus{}

	byt, _ := json.Marshal(msg)
	json.Unmarshal(byt, cpy)
	return cpy
}

// PortalStatus encapsulates the status of the portal
type PortalStatus struct {
	Faction    Faction           // Owning faction
	Level      float32           // Portal level, 0-8 (floating point, because average of resonator levels)
	Resonators []ResonatorStatus // Array of 8 resonators, level 0 (undeployed) - 8
}

// ChannelData defines data for a particular OPC channel for a frame
type ChannelData struct {
	ChannelNum OpcChannel
	Data       []color.RGBA
}

type animCircBuf struct {
	buf  []Animation
	head int
	tail int
}

func newAnimCircBuf() animCircBuf {
	return animCircBuf{
		buf:  make([]Animation, 5),
		head: 0,
		tail: 0,
	}
}

func (cb *animCircBuf) enqueue(effect Animation) {
	cb.buf[cb.tail] = effect
	cb.tail = (cb.tail + 1) % len(cb.buf)
}

func (cb *animCircBuf) dequeue() Animation {
	if cb.head == cb.tail {
		return nil
	}
	ret := cb.buf[cb.head]
	cb.buf[cb.head] = nil
	cb.head = (cb.head + 1) % len(cb.buf)
	return ret
}

func (cb *animCircBuf) clear() {
	for cb.head != cb.tail {
		cb.buf[cb.head] = nil
		cb.head = (cb.head + 1) % len(cb.buf)
	}
}

func (cb *animCircBuf) peek() Animation {
	if cb.head == cb.tail {
		return nil
	}
	return cb.buf[cb.head]
}

func (cb *animCircBuf) size() int {
	bufLen := len(cb.buf)
	return (cb.tail + bufLen - cb.head) % bufLen
}

type seqCircBuf struct {
	buf  []*Sequence
	head int
	tail int
}

func newSeqCircBuf() seqCircBuf {
	return seqCircBuf{
		buf:  make([]*Sequence, 5),
		head: 0,
		tail: 0,
	}
}

func (cb *seqCircBuf) enqueue(seq *Sequence) {
	cb.buf[cb.tail] = seq
	cb.tail = (cb.tail + 1) % len(cb.buf)
}

func (cb *seqCircBuf) dequeue() *Sequence {
	if cb.head == cb.tail {
		return nil
	}
	ret := cb.buf[cb.head]
	cb.buf[cb.head] = nil
	cb.head = (cb.head + 1) % len(cb.buf)
	return ret
}

func (cb *seqCircBuf) clear() {
	for cb.head != cb.tail {
		cb.buf[cb.head] = nil
		cb.head = (cb.head + 1) % len(cb.buf)
	}
}

func (cb *seqCircBuf) peek() *Sequence {
	if cb.head == cb.tail {
		return nil
	}
	return cb.buf[cb.head]
}

func (cb *seqCircBuf) size() int {
	bufLen := len(cb.buf)
	return (cb.tail + bufLen - cb.head) % bufLen
}

// Portal encapsulates the animation status of the entire portal. This will probably be a singleton
// object, but the fields are encapsulated into a struct to allow for something different
type Portal struct {
	currentStatus *PortalStatus   // The cached current status of the portal
	sr            *SequenceRunner // SequenceRunner for portal portion
	seqBuf        seqCircBuf      // Queue of sequences to run on SequenceRunner
	resonators    []animCircBuf   // Animations for resonators
	frameBuf      []ChannelData   // Frame buffers by universe
}

func externalStatusToInternal(external *portalmodel.Status) *PortalStatus {
	var faction Faction
	switch external.Faction {
	case "E":
		faction = ENL
	case "R":
		faction = RES
	case "N":
		faction = NEU
	default:
		panic(fmt.Sprintf("Unexpected faction in external status: %s", external.Faction))
	}

	resos := make([]ResonatorStatus, numResos)
	if len(external.Resonators) != numResos {
		panic(fmt.Sprintf("Number of resonators in external status is %d, not the expected %d", len(external.Resonators), numResos))
	}

	for idx := range resos {
		resos[idx] = ResonatorStatus{
			Health: external.Resonators[idx].Health,
			Level:  int(external.Resonators[idx].Level),
		}
	}

	return &PortalStatus{
		Faction:    faction,
		Level:      external.Level,
		Resonators: resos,
	}
}
