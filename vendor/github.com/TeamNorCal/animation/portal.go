package animation

import (
	"encoding/json"
	"image/color"
	"strconv"
	"time"
)

// Enacapsulates a model of a portal from the perpsective of animations.
// Provides an API semantically meaningful to Ingress

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

const windowSize = 30
const numResos = 8
const numShaftWindows = 16

// Effect contants

const (
	// How much do we dim the reso by?
	resoDimRatio = 0.7
	// Length of reso pulse
	resoPulseDuration = 3 * time.Second
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

func (cb *animCircBuf) enqueueAnim(effect Animation) {
	cb.buf[cb.tail] = effect
	cb.tail = (cb.tail + 1) % len(cb.buf)
}

func (cb *animCircBuf) dequeueAnim() Animation {
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

func (cb *animCircBuf) currAnim() Animation {
	if cb.head == cb.tail {
		return nil
	}
	return cb.buf[cb.head]
}

func (cb *animCircBuf) size() int {
	bufLen := len(cb.buf)
	return (cb.tail + bufLen - cb.head) % bufLen
}

// Portal encapsulates the animation status of the entire portal. This will probably be a singleton
// object, but the fields are encapsulated into a struct to allow for something different
type Portal struct {
	currentStatus *PortalStatus   // The cached current status of the portal
	sr            *SequenceRunner // SequenceRunner for portal portion
	resonators    []animCircBuf   // Animations for resonators
	frameBuf      []ChannelData   // Frame buffers by universe
}

// resonatorLevelColors is an array of colors of resonators of various levels, 0-8
var resonatorLevelColors = []uint32{
	0x000000, // L0
	0xEE8800, // L1
	0xFF6600, // L2
	0xCC3300, // L3
	0x990000, // L4
	0xFF0033, // L5
	0xCC0066, // L6
	0x990066, //0x660066, // L7
	0x660066, //0x330033, // L8
}

func init() {
	// Set up universes
	Universes = make(map[string]Universe)
	idx := 0
	// The resonators at the base
	for reso := 1; reso <= 8; reso++ {
		name := "base" + strconv.Itoa(reso)
		Universes[name] = Universe{
			Index: idx,
			Size:  windowSize,
		}
		idx++
	}

	// The, umm, shaft
	for level := 1; level <= 8; level++ {
		for window := 1; window <= 2; window++ {
			name := "towerLevel" + strconv.Itoa(level) + "Window" + strconv.Itoa(window)
			Universes[name] = Universe{
				Index: idx,
				Size:  windowSize,
			}
			idx++
		}
	}
}

// NewPortal creates a new portal structure
func NewPortal() *Portal {
	sizes := make([]uint, numShaftWindows)
	for idx := range sizes {
		sizes[idx] = windowSize
	}
	frameBuf := make([]ChannelData, numResos+numShaftWindows)
	for idx := range frameBuf {
		frameBuf[idx] = ChannelData{OpcChannel(idx + 1), make([]color.RGBA, windowSize)}
	}
	resoBufs := make([]animCircBuf, 0, numResos)
	for idx := 0; idx < numResos; idx++ {
		resoBufs = append(resoBufs, newAnimCircBuf())
	}
	return &Portal{
		currentStatus: &PortalStatus{NEU, 0.0, make([]ResonatorStatus, numResos)},
		sr:            NewSequenceRunner(sizes),
		resonators:    resoBufs,
		frameBuf:      frameBuf,
	}
}

// UpdateStatus updates the status of the portal from an animation perspective
func (p *Portal) UpdateStatus(status *PortalStatus) {
	newStatus := status.deepCopy()
	if p.currentStatus.Faction != newStatus.Faction || p.currentStatus.Level != newStatus.Level {
		p.updatePortal(status.Faction)
	}

	for idx, status := range newStatus.Resonators {
		if status != p.currentStatus.Resonators[idx] {
			p.updateResonator(idx, &status)
		}
	}

	p.currentStatus = newStatus.deepCopy()
}

// GetFrame gets frame data for the portal, returning an array of frame data
// for each universe in the portal. Indices into this array are specified in the
// Universes map
// The returned buffers will typically be reused between frames, so callers
// should not hold onto references to them nor modify them!
func (p *Portal) GetFrame(frameTime time.Time) []ChannelData {
	// Update resonators
	for idx := 0; idx < numResos; idx++ {
		p.getResoFrame(idx, frameTime)
	}
	p.sr.ProcessFrame(frameTime)
	for idx := 0; idx < numShaftWindows; idx++ {
		p.frameBuf[numResos+idx].Data = p.sr.UniverseData(uint(idx))
	}
	return p.frameBuf
}

func (msg *PortalStatus) deepCopy() (cpy *PortalStatus) {
	cpy = &PortalStatus{}

	byt, _ := json.Marshal(msg)
	json.Unmarshal(byt, cpy)
	return cpy
}

func (p *Portal) updatePortal(newFaction Faction) {

}

func (p *Portal) updateResonator(index int, newStatus *ResonatorStatus) {
	currStatus := p.currentStatus.Resonators[index]
	if currStatus.Level != newStatus.Level {
		// A change
		if newStatus.Level == 0 {
			// Clear current animations, then fade to black and stay there
			p.resonators[index].clear()
			p.resonators[index].enqueueAnim(NewInterpolateToHexRGB(0x000000, time.Second))
			p.resonators[index].enqueueAnim(NewSolid(RGBAFromRGBHex(0x000000)))
		} else {
			// Clear current animations, then fade to new nominal reso color and pulse
			resoColor := resonatorLevelColors[newStatus.Level]
			p.resonators[index].clear()
			logger.Printf("Enqueuing 2 animations for index %d\n", index)
			p.resonators[index].enqueueAnim(NewInterpolateToHexRGB(resoColor, time.Second))
			p.resonators[index].enqueueAnim(NewDimmingPulse(RGBAFromRGBHex(resoColor), resoDimRatio, resoPulseDuration))
		}
		p.resonators[index].currAnim().Start(time.Now())
	}
}

// getResoFrame updates the frame buffer for the specified resonator with data
// for the current frame, with specified frame time
func (p *Portal) getResoFrame(index int, frameTime time.Time) {
	currAnim := p.resonators[index].currAnim()
	if currAnim == nil {
		return
	}
	buf, done := currAnim.Frame(p.frameBuf[index].Data, frameTime)
	// applyBrightness(p.frameBuf[index].Data, p.currentStatus.Resonators[index].Health/100.0)
	p.frameBuf[index].Data = buf
	// Resonator animations run in a continuous loop, so restart if done
	if done {
		if p.resonators[index].size() == 1 {
			// Only 1 animation - restart it
			currAnim.Start(frameTime)
		} else {
			// More animations queued - move on to the Next
			// We know it's > 1 because we returned early if there were 0
			logger.Printf("Dequeuing animation for index %d\n", index)
			p.resonators[index].dequeueAnim()
			p.resonators[index].currAnim().Start(frameTime)
		}
	}
}

func applyBrightness(buf []color.RGBA, brightness float32) {
	for idx := range buf {
		buf[idx].R = uint8(float32(buf[idx].R) * brightness)
		buf[idx].G = uint8(float32(buf[idx].G) * brightness)
		buf[idx].B = uint8(float32(buf[idx].B) * brightness)
	}
}
