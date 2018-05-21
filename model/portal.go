package model

// This module defines implementation neutral portal state information
// data structures

import (
	"encoding/json"
	"fmt"

	"github.com/TeamNorCal/animation"
)

type Resonator struct {
	Position string  `json:"position"`
	Level    float32 `json:"level"`
	Health   float32 `json:"health"`
	Owner    string  `json:"owner"`
}

type Mod struct {
	Owner  string  `json:"owner"`
	Slot   float32 `json:"slot"`
	Type   string  `json:"type"`   // 'FA'mp, 'HS'ink, 'LA'mp, 'SBUL'ink, 'MH'ack, 'PS'hield, 'AXA' Aegis Shield, 'T'urret
	Rarity string  `json:"rarity"` // 'C'ommon, 'R'are, 'VR'  Very Rare
}

type Status struct {
	Title         string      `json:"Title"`
	Description   string      `json:"description"`
	CoverImageURL string      `json:"coverImageUrl"`
	Owner         string      `json:"owner"`
	Level         float32     `json:"level"`
	Health        float32     `json:"health"`
	Faction       string      `json:"controllingFaction"` // Will be 'E'nlightened, 'R'esistance, or 'N'eutral
	Mods          []Mod       `json:"mods"`
	Resonators    []Resonator `json:"resonators"`
}

type PortalStatus struct {
	Status Status `json:"externalApiPortal"`
}

type PortalMsg struct {
	Home   bool   `json:"home"`
	Status Status `json:"externalApiPortal"`
}

// DeepCopy deepcopies a to b using json marshaling
func (msg *PortalMsg) DeepCopy() (cpy *PortalMsg) {
	cpy = &PortalMsg{}

	byt, _ := json.Marshal(msg)
	json.Unmarshal(byt, cpy)
	return cpy
}

// DeepCopy deepcopies a to b using json marshaling
func (status *Status) DeepCopy() (cpy *Status) {
	cpy = &Status{}

	byt, _ := json.Marshal(status)
	json.Unmarshal(byt, cpy)
	return cpy
}

// The following needs to be moved into the animation library
const numResos = 8

func StatusToAnimation(status *Status) *animation.PortalStatus {
	var faction animation.Faction
	switch status.Faction {
	case "E":
		faction = animation.ENL
	case "R":
		faction = animation.RES
	case "N":
		faction = animation.NEU
	default:
		panic(fmt.Sprintf("Unexpected faction in external status: %s", status.Faction))
	}

	resos := make([]animation.ResonatorStatus, numResos)
	if len(status.Resonators) != numResos {
		panic(fmt.Sprintf("Number of resonators in external status is %d, not the expected %d", len(status.Resonators), numResos))
	}

	for idx := range resos {
		resos[idx] = animation.ResonatorStatus{
			Health: status.Resonators[idx].Health,
			Level:  int(status.Resonators[idx].Level),
		}
	}

	return &animation.PortalStatus{
		Faction:    faction,
		Level:      status.Level,
		Resonators: resos,
	}
}
