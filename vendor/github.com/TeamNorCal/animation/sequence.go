package animation

// Sequencing of animation effects across the entire model, to implement
// higher-level transitions requiring model-level animation effects

import (
	"image/color"
	"log"
	"os"
	"time"
)

const (
	// DefaultQueueSize is the initial size of arrays representing queues
	DefaultQueueSize = 10
)

// Step is a sequencing step. Contains information about the effect(s) to
// perform and the universe[s] being targetted.
// Can be gated on completion of another referenced step, and/or delayed by
// an amount of time. If both gating step and delay are specified, the delay
// will be applied after the gating step completes.
type Step struct {
	UniverseID     uint          // The universe to which the step is applied
	Effect         Animation     // The animation effect to play
	OnCompletionOf int           // ID of step that must complete before this step commences
	Delay          time.Duration // Delay before starting step [after prior step completes if set]
	StepID         int           // ID of step, to allow reference by another step. Must be non-0 to be honored
}

// Sequence encapsulates an animation sequence
type Sequence struct {
	Steps []Step
}

/*
 * SequenceRunner
 */

// Encapsulate a step waiting for another step to complete
type stepAndTime struct {
	runAt time.Time
	toRun *Step
}

// Encapsulate a step waiting for another step to complete
type stepAndGatingStep struct {
	waitingOn int   // Step waiting for completion
	toRun     *Step // Step to run
}

// SequenceRunner is responsible for executing a given sequence
type SequenceRunner struct {
	awaitingTime     []stepAndTime       // Queue of steps waiting on a particular time
	awaitingStep     []stepAndGatingStep // Queue of steps waiting on another step to complete
	activeByUniverse [][]*Step           // Queue of steps that can be run on a particular universe. Only head of queue is procssed
	buffers          [][]color.RGBA      // Buffers to hold universe data
}

var logger = log.New(os.Stdout, "(SEQUENCE) ", 0)

// NewSequenceRunner creates a SequenceRunner for the provided sequence with the
// specified universe sizes. These size indicate the number of pixels in each
// universe, with the universe ID being the index into the array. (Universe IDs
// are expected to start at 0 and be consecutive.)
func NewSequenceRunner(universeSizes []uint) *SequenceRunner {
	// Create the framework of the SequenceRunner
	activeByUniverse := make([][]*Step, len(universeSizes))
	buffers := make([][]color.RGBA, len(universeSizes))
	for idx, size := range universeSizes {
		activeByUniverse[idx] = make([]*Step, DefaultQueueSize)
		buffers[idx] = make([]color.RGBA, size)
	}
	sr := &SequenceRunner{
		make([]stepAndTime, DefaultQueueSize),
		make([]stepAndGatingStep, DefaultQueueSize),
		activeByUniverse,
		buffers}

	return sr
}

func findStep(steps []Step, stepID int) *Step {
	for _, step := range steps {
		if step.StepID == stepID {
			return &step
		}
	}
	return nil
}

// InitSequence initializes the SequenceRunner with the provides sequence, to
// start at the provided time. If a sequence is already in process, it will be
// stopped and the SequenceRunner reinitialized.
func (sr *SequenceRunner) InitSequence(seq Sequence, now time.Time) {
	// Clear structures
	for idx := range sr.awaitingTime {
		sr.awaitingTime[idx] = stepAndTime{}
	}
	sr.awaitingTime = sr.awaitingTime[:0]
	for idx := range sr.awaitingStep {
		sr.awaitingStep[idx] = stepAndGatingStep{}
	}
	sr.awaitingStep = sr.awaitingStep[:0]
	for idx := range sr.activeByUniverse {
		for idx2 := range sr.activeByUniverse[idx] {
			sr.activeByUniverse[idx][idx2] = nil
		}
		sr.activeByUniverse[idx] = sr.activeByUniverse[idx][:0]
	}

	// Process the provided sequence steps
	for idx, step := range seq.Steps {
		pstep := &seq.Steps[idx]
		if step.OnCompletionOf != 0 {
			waitingOnStep := findStep(seq.Steps, step.OnCompletionOf)
			if waitingOnStep != nil {
				sr.awaitingStep = append(sr.awaitingStep, stepAndGatingStep{waitingOnStep.StepID, pstep})
			} else {
				logger.Printf("WARNING: Could not find step %d which step %v is waiting on. This step will be ignored",
					step.OnCompletionOf, step)
			}
		} else if step.Delay > 0 {
			sr.scheduleAt(pstep, now.Add(step.Delay))
		} else {
			sr.activeByUniverse[step.UniverseID] = append(sr.activeByUniverse[step.UniverseID], pstep)
		}
	}
}

func deleteStep(a []*Step, i int) []*Step {
	copy(a[i:], a[i+1:])
	a[len(a)-1] = nil
	return a[:len(a)-1]
}

func deleteSAGS(a []stepAndGatingStep, i int) []stepAndGatingStep {
	copy(a[i:], a[i+1:])
	a[len(a)-1] = stepAndGatingStep{}
	return a[:len(a)-1]
}

func deleteSAT(a []stepAndTime, i int) []stepAndTime {
	copy(a[i:], a[i+1:])
	a[len(a)-1] = stepAndTime{}
	return a[:len(a)-1]
}

func (sr *SequenceRunner) scheduleAt(s *Step, runAt time.Time) {
	sr.awaitingTime = append(sr.awaitingTime, stepAndTime{runAt, s})
}

// Check for steps that are waiting on another step to complete.
// 'now' is the time that should be considered to be the current time
func (sr *SequenceRunner) handleStepComplete(completed *Step, now time.Time) {
	uniSteps := sr.activeByUniverse[completed.UniverseID]
	if len(uniSteps) > 0 && uniSteps[0] == completed {
		sr.activeByUniverse[completed.UniverseID] = deleteStep(uniSteps, 0)
	}
	for idx := 0; idx < len(sr.awaitingStep); {
		waiting := sr.awaitingStep[idx]
		if waiting.waitingOn == completed.StepID {
			s := waiting.toRun
			if s.Delay > 0 {
				// Schedule to run after delay
				runAt := now.Add(s.Delay)
				sr.scheduleAt(s, runAt)
			} else {
				// Run immediately
				sr.activeByUniverse[s.UniverseID] = append(sr.activeByUniverse[s.UniverseID], s)
			}
			// Delete this from the list of waiting steps (and don't increment index)
			sr.awaitingStep = deleteSAGS(sr.awaitingStep, idx)
		} else {
			idx++
		}
	}
}

// Check for any tasks that should run at this point
// 'now' is the time that should be considered to be the current time
func (sr *SequenceRunner) checkScheduledTasks(now time.Time) {
	for idx := 0; idx < len(sr.awaitingTime); {
		waiting := sr.awaitingTime[idx]
		if now.After(waiting.runAt) {
			// Time to run it!
			s := waiting.toRun
			sr.activeByUniverse[s.UniverseID] = append(sr.activeByUniverse[s.UniverseID], s)
			// Delete this from the list of waiting steps (and don't increment index)
			sr.awaitingTime = deleteSAT(sr.awaitingTime, idx)
		} else {
			idx++
		}
	}
}

// ProcessFrame generates frame data corresponding to the specified time (which
// should be monotonically increasing with each call)
// Return value indicates whether the sequence is complete.
func (sr *SequenceRunner) ProcessFrame(now time.Time) (done bool) {
	done = true
	sr.checkScheduledTasks(now)
	for universeID, universe := range sr.activeByUniverse {
		if len(universe) > 0 {
			// We have an active step on this universe
			s := universe[0]
			// ...so we're not done yet
			done = false
			// Process the animation for the universe
			effectDone := s.Effect.Frame(sr.buffers[universeID], now)
			if effectDone {
				sr.handleStepComplete(s, now)
			}
		}
	}

	// We are done if we procssed nothing and there are no more queued-up steps
	done = done && len(sr.awaitingStep) == 0 && len(sr.awaitingTime) == 0
	return
}

// UniverseData gets current data for the specified universe. This data is
// updated by calling ProcessFrame for the universe
func (sr *SequenceRunner) UniverseData(UniverseID uint) []color.RGBA {
	return sr.buffers[UniverseID]
}
