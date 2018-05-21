package animation

// Code to support mapping from logical 'universes' to physical pixel layout.

import (
	"fmt"
	"image/color"
	"math"
)

// {board, strand, pixel} tuple identifying a physical pixel
type location struct {
	board, strand, pixel uint
}

// Mapping captures mapping from logical to physical layer
type Mapping struct {
	// Buffer of data mapping to physical pixels
	// Three levels of indexing:
	// 1. Controller board number
	// 2. Strand number within controller board
	// 3. Pixel number within strand
	physBuf [][][]color.RGBA

	// Mapping from 'universes' (logical view of pixels) to physical pixels.
	// Two levels of indexing:
	// 1. Universe number
	// 2. Pixel number within universe
	universes [][]location

	// Mapping from universe name to universe ID
	uniNameToIndex map[string]int
}

// PhysicalRange defines a range of physical pixels within asingle strand
type PhysicalRange struct {
	Board, Strand, StartPixel, Size uint
}

// NewMapping creates a new Mapping, using the provided dimensions.
// Size of outer array governs the number of controller boards
// Sizes of inner arrays govern the number of strands within each board
// Values in inner array govern the number of pixels in the strand
func NewMapping(dimension [][]int) Mapping {
	// Make the triply-nested physical buffer structure based on the provided dimensions
	// Allocate space for a reasonable number of universes
	m := Mapping{
		physBuf:        make([][][]color.RGBA, len(dimension)),
		universes:      make([][]location, 0, 16),
		uniNameToIndex: make(map[string]int),
	}
	for boardIdx := range dimension {
		m.physBuf[boardIdx] = make([][]color.RGBA, len(dimension[boardIdx]))
		for strandIdx := range dimension[boardIdx] {
			m.physBuf[boardIdx][strandIdx] = make([]color.RGBA, dimension[boardIdx][strandIdx])
		}
	}
	return m
}

// AddUniverse adds a universe mapping with the given name.
// The provided set of physical ranges identifies the set of physical pixels
// corresponding to the universe. The order of physical pixels presented defines
// the logical ordering of the universe, and the size of the universe is equal
// to the number of physical pixels provided
// Returns true if the universe was successfully added; returns false if the
// universe name already exists or a specified physical pixel doesn't exist.
func (m *Mapping) AddUniverse(name string, ranges []PhysicalRange) bool {
	if _, exists := m.uniNameToIndex[name]; exists {
		return false
	}
	// Figure out the size
	size := uint(0)
	for _, r := range ranges {
		size += r.Size
	}
	// Allocate locations array for universe
	locs := make([]location, size)
	// Populate locations array from pixel ranges
	unidx := 0
	for _, r := range ranges {
		for idx := r.StartPixel; idx < r.StartPixel+r.Size; idx++ {
			locs[unidx] = location{r.Board, r.Strand, idx}
			unidx++
		}
	}

	// Add the universe to the structure
	m.universes = append(m.universes, locs)
	m.uniNameToIndex[name] = len(m.universes) - 1
	return true
}

// IDForUniverse gets the internal ID associated with the given universe name.
// Returns error and large invalid ID if universe name is not found
func (m *Mapping) IDForUniverse(universeName string) (uint, error) {
	id, ok := m.uniNameToIndex[universeName]
	if !ok {
		return math.MaxUint32, fmt.Errorf("\"%s\" is not a known universe", universeName)
	}
	return uint(id), nil
}

// UpdateUniverse updates physical pixel color values for pixels corresponding
// to the provided universe.
func (m *Mapping) UpdateUniverse(id uint, rgbData []color.RGBA) (err error) {
	u := m.universes[id]
	for idx, l := range u {
		if idx >= len(rgbData) {
			return fmt.Errorf("RGB values (%d) not long enough for universe %d (%+v)", len(rgbData), id, l)
		}
		m.physBuf[l.board][l.strand][l.pixel] = rgbData[idx]
	}
	return nil
}

// GetStrandData returns color data for a physical strand. The slice returned
// references the master buffer for the strand and so can be changed by further
// calls to UpdateUniverse. If the caller needs to retain the data, a copy
// should be made
// The strand in question is identified by the board and strand indices provided.
// Returns an empty slice and an error if an invalid strand is specified
func (m *Mapping) GetStrandData(board, strand uint) ([]color.RGBA, error) {
	if int(board) >= len(m.physBuf) {
		return nil, fmt.Errorf("%d is an invalid board index", board)
	}
	if int(strand) >= len(m.physBuf[board]) {
		return nil, fmt.Errorf("%d is an invalid strand number for board %d",
			strand, board)
	}
	return m.physBuf[board][strand], nil
}
