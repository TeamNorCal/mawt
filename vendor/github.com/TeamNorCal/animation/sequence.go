package animation

// Sequencing of animation effects across the entire model, to implement
// higher-level transitions requiring model-level animation effects

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"sync"
	"time"
)

// Operation is an operation to apply in order to orchestrate a sequence
type Operation struct {
	StepName string        // The name of the step to run next
	Delay    time.Duration // Optional delay to apply before starting next step
}

// Step is a sequencing step. Contains information about the effect(s) to
// perform and the universe[s] being targetted.
// Can be gated on completion of another referenced step, and/or delayed by
// an amount of time. If both gating step and delay are specified, the delay
// will be applied after the gating step completes.
type Step struct {
	UniverseID uint        // The universe to which the step is applied
	Effect     Animation   // The animation effect to play
	Next       []Operation // An optional list of operations to apply after completion of this step
}

// ThenDo adds a 'next' operation to a step
func (s *Step) ThenDo(stepName string, delay time.Duration) *Step {
	if s.Next == nil {
		s.Next = make([]Operation, 0)
	}
	s.Next = append(s.Next, Operation{stepName, delay})
	return s
}

// ThenDoImmediately adds a 'next' operation to a step, running immediately
func (s *Step) ThenDoImmediately(stepName string) *Step {
	return s.ThenDo(stepName, 0)
}

// Sequence encapsulates an animation sequence
type Sequence struct {
	steps             map[string]*Step // Steps by name
	initialOperations []Operation      // An initial set of operations to perform upon sequence start
}

// NewSequence creates a new, empty sequence
func NewSequence() *Sequence {
	return &Sequence{make(map[string]*Step), make([]Operation, 0)}
}

// AddStep adds a step to the sequence
func (seq *Sequence) AddStep(name string, step *Step) *Sequence {
	seq.steps[name] = step
	return seq
}

// AddInitialOperation adds an initial operation to the sequence
func (seq *Sequence) AddInitialOperation(operation Operation) *Sequence {
	seq.initialOperations = append(seq.initialOperations, operation)
	return seq
}

// AddInitialStep is a convenience method to add a new step and schedule it
// to run immediately
func (seq *Sequence) AddInitialStep(name string, step *Step) *Sequence {
	seq.AddStep(name, step)
	seq.AddInitialOperation(Operation{name, 0})
	return seq
}

// CreateStepCycle creates a cycle of steps by adding an appropriate 'next'
// operation to each step pointing at the next step, circularly
func (seq *Sequence) CreateStepCycle(names ...string) *Sequence {
	for idx := 0; idx < len(names); idx++ {
		step, isPresent := seq.steps[names[idx]]
		if !isPresent {
			logger.Printf("Can't create cycle from step %s, as it can't be found. Ignoring.\n", names[idx])
			continue
		}
		step.ThenDoImmediately(names[(idx+1)%len(names)])
	}
	return seq
}

/*
 * SequenceRunner
 */

// Encapsulate a step waiting for another step to complete
type stepAndTime struct {
	runAt time.Time
	toRun *Step
}

// SequenceRunner is responsible for executing a given sequence
type SequenceRunner struct {
	awaitingTime     []stepAndTime    // Queue of steps waiting on a particular time
	activeByUniverse map[uint][]*Step // Queue of steps that can be run on a particular universe. Only head of queue is processed
	buffers          [][]color.RGBA   // Buffers to hold universe data
	currSeq          Sequence         // Reference to currently-running sequence
	sync.Mutex
}

var logger = log.New(os.Stdout, "(SEQUENCE) ", 0)

// NewSequenceRunner creates a SequenceRunner for the provided sequence with the
// specified universe sizes. These size indicate the number of pixels in each
// universe, with the universe ID being the index into the array. (Universe IDs
// are expected to start at 0 and be consecutive.)
func NewSequenceRunner(universeSizes []uint) (sr *SequenceRunner) {

	sr = &SequenceRunner{
		// Slices by default are initialized with a length of 0 and a capacity of 8 preventing
		// downstream extensions
		awaitingTime:     make([]stepAndTime, 0, 8),
		activeByUniverse: make(map[uint][]*Step, 16),
		buffers:          make([][]color.RGBA, len(universeSizes)),
	}

	for i, size := range universeSizes {
		sr.activeByUniverse[uint(i)] = make([]*Step, 0, 8)
		// Create a slice filled with zero values
		sr.buffers[i] = make([]color.RGBA, size)
	}

	return sr
}

func (sr *SequenceRunner) startStep(step *Step) {
	if _, isPresent := sr.activeByUniverse[step.UniverseID]; !isPresent {
		sr.activeByUniverse[step.UniverseID] = make([]*Step, 0, 8)
	}
	step.Effect.Start(time.Now())
	sr.activeByUniverse[step.UniverseID] = append(sr.activeByUniverse[step.UniverseID], step)
}

// InitSequence initializes the SequenceRunner with the provides sequence, to
// start at the provided time. If a sequence is already in process, it will be
// stopped and the SequenceRunner reinitialized.
func (sr *SequenceRunner) InitSequence(seq *Sequence, now time.Time) {
	sr.Lock()
	defer sr.Unlock()

	sr.initSequenceInternal(*seq, now)
}

func (sr *SequenceRunner) initSequenceInternal(seq Sequence, now time.Time) {
	sr.currSeq = seq

	// Clear structures, no need to leave old slice preallocations
	// as all slices when initially created are automatically 8
	// entries from a capacity perspective
	sr.awaitingTime = sr.awaitingTime[:0]
	for k, v := range sr.activeByUniverse {
		sr.activeByUniverse[k] = v[:0]
	}

	// Process the initial operations, scheduling or starting steps
	for _, operation := range seq.initialOperations {
		err := sr.processOperation(operation, now)
		if err != nil {
			logger.Println(err)
		}
	}
}

func (sr *SequenceRunner) processOperation(operation Operation, now time.Time) error {
	step, isPresent := sr.currSeq.steps[operation.StepName]
	if !isPresent {
		return fmt.Errorf("WARNING: Could not find step %s specified by initial operation. This operation will be ignored", operation.StepName)
	}
	if operation.Delay > 0 {
		sr.scheduleAt(step, now.Add(operation.Delay))
	} else {
		sr.startStep(step)
	}

	return nil
}

func deleteStep(a []*Step, i int) []*Step {
	// SliceTricks has some gaps
	if i == 0 && len(a) == 1 {
		return []*Step{}
	}

	copy(a[i:], a[i+1:])
	a[len(a)-1] = nil
	return a[:len(a)-1]
}

func deleteSAT(a []stepAndTime, i int) []stepAndTime {
	// SliceTricks has some gaps
	if i == 0 && len(a) == 1 {
		return []stepAndTime{}
	}

	// Wont leak as the slice is not using a pointer
	return append(a[:i], a[i+1:]...)
}

func (sr *SequenceRunner) scheduleAt(s *Step, runAt time.Time) {
	sr.awaitingTime = append(sr.awaitingTime, stepAndTime{runAt, s})
}

// Check for steps that are waiting on another step to complete.
// 'now' is the time that should be considered to be the current time
//
func (sr *SequenceRunner) handleStepComplete(completed *Step, now time.Time) {
	uniSteps, isPresent := sr.activeByUniverse[completed.UniverseID]
	if isPresent {
		if len(uniSteps) > 0 && uniSteps[0] == completed {
			sr.activeByUniverse[completed.UniverseID] = deleteStep(uniSteps, 0)
		}
	}

	if completed.Next != nil {
		for _, operation := range completed.Next {
			err := sr.processOperation(operation, now)
			if err != nil {
				logger.Println(err)
			}
		}
	}

	// for idx := 0; idx < len(sr.awaitingStep); {
	// 	waiting := sr.awaitingStep[idx]
	// 	if waiting.waitingOn == completed.StepID {
	// 		s := waiting.toRun
	// 		if s.Delay > 0 {
	// 			// Schedule to run after delay
	// 			runAt := now.Add(s.Delay)
	// 			sr.scheduleAt(s, runAt)
	// 		} else {
	// 			// Run immediately
	// 			sr.startStep(s)
	// 		}
	// 		// Delete this from the list of waiting steps (and don't increment index)
	// 		sr.awaitingStep = deleteSAGS(sr.awaitingStep, idx)
	// 	} else {
	// 		idx++
	// 	}
	// }
}

// Check for any tasks that should run at this point
// 'now' is the time that should be considered to be the current time
func (sr *SequenceRunner) checkScheduledTasks(now time.Time) {
	for idx := 0; idx < len(sr.awaitingTime); {
		waiting := sr.awaitingTime[idx]
		if now.After(waiting.runAt) && waiting.toRun != nil {
			// Time to run it!
			s := waiting.toRun
			sr.startStep(s)
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
	effectDone := false

	sr.Lock()
	defer sr.Unlock()

	sr.checkScheduledTasks(now)

	for universeID, universe := range sr.activeByUniverse {
		if len(universe) > 0 {
			// We have an active step on this universe
			s := universe[0]
			// ...so we're not done yet
			done = false
			// Process the animation for the universe, first clear the slice
			if sr.buffers[universeID], effectDone = s.Effect.Frame(sr.buffers[universeID], now); effectDone {
				sr.handleStepComplete(s, now)
			}
		}
	}

	// We are done if we procssed nothing and there are no more queued-up steps
	seqDone := done && len(sr.awaitingTime) == 0

	return seqDone
}

// UniverseData gets current data for the specified universe. This data is
// updated by calling ProcessFrame for the universe
func (sr *SequenceRunner) UniverseData(UniverseID uint) []color.RGBA {
	sr.Lock()
	defer sr.Unlock()

	return sr.buffers[UniverseID]
}
