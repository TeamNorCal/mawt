package animation

import (
	"fmt"
	"image/color"
	"math/rand"
	"strconv"
	"time"

	"github.com/TeamNorCal/animation/model"
	ingressModel "github.com/TeamNorCal/mawt/model"
)

// Enacapsulates a model of a portal from the perpsective of animations.
// Provides an API semantically meaningful to Ingress

const (
	// Number of pixels in each window
	windowSize = 30
	// Number of resonators
	numResos = 8
	// Number of windows in the tower
	numShaftWindows = 16
	// How much do we dim the reso by?
	resoDimRatio = 0.7
	// Length of reso pulse
	resoPulseDuration = 3 * time.Second
)

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
	logger.Println("Initializing...")
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
	frameBuf := make([]model.ChannelData, numResos+numShaftWindows)
	for idx := range frameBuf {
		frameBuf[idx] = model.ChannelData{model.OpcChannel(idx + 1), make([]color.RGBA, windowSize)}
	}
	resoBufs := make([]animCircBuf, 0, numResos)
	for idx := 0; idx < numResos; idx++ {
		resoBufs = append(resoBufs, newAnimCircBuf())
	}
	return &Portal{
		currentStatus: &PortalStatus{NEU, 0.0, make([]ResonatorStatus, numResos)},
		sr:            NewSequenceRunner(sizes),
		seqBuf:        newSeqCircBuf(),
		resonators:    resoBufs,
		frameBuf:      frameBuf,
	}
}

func externalStatusToInternal(external *ingressModel.Status) (status *PortalStatus) {
	status = &PortalStatus{
		Faction:    NEU,
		Level:      external.Level,
		Resonators: make([]ResonatorStatus, numResos),
	}

	switch external.Faction {
	case "E":
		status.Faction = ENL
	case "R":
		status.Faction = RES
	case "N":
	default:
		fmt.Printf("Treating unexpected faction in external status as neutral: '%s'\n", external.Faction)
	}

	resos := make([]ResonatorStatus, numResos)
	numResosInStatus := len(external.Resonators)

	for idx := range resos {
		// TODO: Honor resonator position in status here
		if idx < numResosInStatus {
			status.Resonators[idx] = ResonatorStatus{
				Health: external.Resonators[idx].Health,
				Level:  int(external.Resonators[idx].Level),
			}
		} else {
			// Treat missing reso as undeployed
			status.Resonators[idx] = ResonatorStatus{
				Health: 0.0,
				Level:  0,
			}
		}
	}
	return status
}

// UpdateFromCanonicalStatus updates the animation with the status of the portal,
// using the canonical Status type
func (p *Portal) UpdateFromCanonicalStatus(status *ingressModel.Status) {
	p.UpdateStatus(externalStatusToInternal(status))
}

// UpdateStatus updates the status of the portal from an animation perspective
func (p *Portal) UpdateStatus(status *PortalStatus) {
	newStatus := status.deepCopy()
	if p.currentStatus.Faction != newStatus.Faction || p.currentStatus.Level != newStatus.Level {
		p.updatePortal(status)
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
func (p *Portal) GetFrame(frameTime time.Time) []model.ChannelData {
	// Update resonators
	for idx := 0; idx < numResos; idx++ {
		p.getResoFrame(idx, frameTime)
	}
	seqDone := p.sr.ProcessFrame(frameTime)
	for idx := 0; idx < numShaftWindows; idx++ {
		p.frameBuf[numResos+idx].Data = p.sr.UniverseData(uint(idx))
	}
	if seqDone {
		if nextSeq := p.seqBuf.dequeue(); nextSeq != nil {
			p.sr.InitSequence(nextSeq, frameTime)
		}
	}
	return p.frameBuf
}

func createFadePulseSeq(c color.Color, pulseDuration time.Duration) *Sequence {
	seq := NewSequence()
	for uniID := 0; uniID < numShaftWindows; uniID++ {
		idStr := strconv.Itoa(uniID)

		fadeOut := &Step{
			UniverseID: uint(uniID),
			Effect:     NewInterpolateToHexRGB(0x000000, time.Second),
		}
		seq.AddInitialStep("fadeOut"+idStr, fadeOut)

		pulse := &Step{
			UniverseID: uint(uniID),
			Effect:     NewPulse(RGBAFromRGBHex(0x000000), c, pulseDuration, true),
		}
		seq.AddStep("pulse"+idStr, pulse)
		fadeOut.ThenDoImmediately("pulse" + idStr)
	}
	return seq
}

func (p *Portal) createOwnedPortalSeq(newStatus *PortalStatus) {
	var c uint32
	switch newStatus.Faction {
	case NEU:
		c = 0xffffff
	case ENL:
		c = 0x00ff00
	case RES:
		c = 0x0000ff
	}
	stepMap := make(map[string]*Step)
	for uniID := 0; uniID < numShaftWindows; uniID++ {
		createWindowFadeInOut(stepMap, uniID, c, time.Duration(125.0*newStatus.Level)*time.Millisecond)
	}
	seq := NewSequence()
	for name, step := range stepMap {
		seq.AddStep(name, step)
	}
	// Create dependencies between steps, to create a cycle
	for uniID := 0; uniID < numShaftWindows; uniID++ {
		idStr := strconv.Itoa(uniID)
		// Link the three phases within each universe
		stepMap["in"+idStr].ThenDoImmediately("solid" + idStr)
		stepMap["solid"+idStr].ThenDoImmediately("out" + idStr)
		// Then do cross-universe linking, with fades overlapping
		nextID := (uniID + 2) % numShaftWindows
		stepMap["in"+idStr].ThenDoImmediately("in" + strconv.Itoa(nextID))
	}
	// Add the initial operation to kick it off - two cycles at once
	seq.AddInitialOperation(Operation{StepName: "in0"})
	seq.AddInitialOperation(Operation{StepName: "in1"})
	p.seqBuf.clear() // Clear any still-pending sequences from before
	p.seqBuf.enqueue(seq)
	takeOverPulse := createFadePulseSeq(RGBAFromRGBHex(c), 1500*time.Millisecond)
	p.sr.InitSequence(takeOverPulse, time.Now())
}

func (p *Portal) createNeutralPortalSeq(newStatus *PortalStatus) {
	seq := NewSequence()
	for uniID := 0; uniID < numShaftWindows; uniID++ {
		idStr := strconv.Itoa(uniID)

		fadeOut := &Step{
			UniverseID: uint(uniID),
			Effect:     NewInterpolateToHexRGB(0x000000, time.Second),
		}
		seq.AddInitialStep("fadeOut"+idStr, fadeOut)

		pulseIn := &Step{
			UniverseID: uint(uniID),
			Effect:     NewInterpolateToHexRGB(0xff0000, 250*time.Millisecond),
		}
		seq.AddStep("pulseIn"+idStr, pulseIn)
		fadeOut.ThenDoImmediately("pulseIn" + idStr)

		pulseOut := &Step{
			UniverseID: uint(uniID),
			Effect:     NewInterpolateToHexRGB(0x000000, 1500*time.Millisecond),
		}
		seq.AddStep("pulseOut"+idStr, pulseOut)
		pulseIn.ThenDoImmediately("pulseOut" + idStr)

		fadeIn := &Step{
			UniverseID: uint(uniID),
			Effect:     NewInterpolateToHexRGB(0xaaaaaa, time.Second),
		}
		seq.AddStep("fadeIn"+idStr, fadeIn)
		pulseOut.ThenDo("fadeIn"+idStr, time.Duration(rand.Intn(3000))*time.Millisecond)

		solid := &Step{
			UniverseID: uint(uniID),
			Effect:     NewSolid(RGBAFromRGBHex(0xaaaaaa)),
		}
		seq.AddStep("solid"+idStr, solid)
		fadeIn.ThenDoImmediately("solid" + idStr)
	}
	p.seqBuf.clear()
	p.sr.InitSequence(seq, time.Now())
}

func (p *Portal) updatePortal(newStatus *PortalStatus) {
	if p.currentStatus.Faction != newStatus.Faction {
		// Faction change
		if newStatus.Faction == ENL || newStatus.Faction == RES {
			p.createOwnedPortalSeq(newStatus)
		} else {
			p.createNeutralPortalSeq(newStatus)
		}
	} else if p.currentStatus.Level != newStatus.Level {
		if newStatus.Faction == ENL || newStatus.Faction == RES {
			updateHoldTime(&p.sr.currSeq, time.Duration(125.0*newStatus.Level)*time.Millisecond)
		}
	}
	// applyBrightness(p.frameBuf[index].Data, p.currentStatus.Resonators[index].Health/100.0)
}

func createWindowFadeInOut(stepMap map[string]*Step, uniID int, color uint32, holdTime time.Duration) {
	idStr := strconv.Itoa(uniID)
	in := &Step{
		Effect:     NewInterpolateToHexRGB(color, 250*time.Millisecond),
		UniverseID: uint(uniID),
	}
	stepMap["in"+idStr] = in
	solid := &Step{
		Effect:     NewTimedSolid(RGBAFromRGBHex(color), holdTime),
		UniverseID: uint(uniID),
	}
	stepMap["solid"+idStr] = solid
	out := &Step{
		Effect:     NewInterpolateToHexRGB(0x000000, 500*time.Millisecond),
		UniverseID: uint(uniID),
	}
	stepMap["out"+idStr] = out
}

func updateHoldTime(seq *Sequence, holdTime time.Duration) {
	for uniID := 0; uniID < numShaftWindows; uniID++ {
		if step := seq.steps["solid"+strconv.Itoa(uniID)]; step != nil {
			step.Effect.(*Solid).duration = holdTime
		}
	}
}

func (p *Portal) updateResonator(index int, newStatus *ResonatorStatus) {
	currStatus := p.currentStatus.Resonators[index]
	if currStatus.Level != newStatus.Level {
		// A change
		if newStatus.Level == 0 {
			// Clear current animations, then fade to black and stay there
			p.resonators[index].clear()
			p.resonators[index].enqueue(NewInterpolateToHexRGB(0x000000, time.Second))
			p.resonators[index].enqueue(NewSolid(RGBAFromRGBHex(0x000000)))
		} else {
			// Clear current animations, then fade to new nominal reso color and pulse
			resoColor := resonatorLevelColors[newStatus.Level]
			p.resonators[index].clear()
			logger.Printf("Enqueuing 2 animations for index %d\n", index)
			p.resonators[index].enqueue(NewInterpolateToHexRGB(resoColor, time.Second))
			p.resonators[index].enqueue(NewDimmingPulse(RGBAFromRGBHex(resoColor), resoDimRatio, resoPulseDuration))
		}
		p.resonators[index].peek().Start(time.Now())
	}
}

// getResoFrame updates the frame buffer for the specified resonator with data
// for the current frame, with specified frame time
func (p *Portal) getResoFrame(index int, frameTime time.Time) {
	currAnim := p.resonators[index].peek()
	if currAnim == nil {
		return
	}
	buf, done := currAnim.Frame(p.frameBuf[index].Data, frameTime)
	applyBrightness(p.frameBuf[index].Data, p.currentStatus.Resonators[index].Health/100.0)
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
			p.resonators[index].dequeue()
			p.resonators[index].peek().Start(frameTime)
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
