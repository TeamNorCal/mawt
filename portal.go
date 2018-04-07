package mawt

// This module defines implementation neutral portal state information
// data structures

import (
	"encoding/json"
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
	Title              string      `json:"Title"`
	Description        string      `json:"description"`
	CoverImageURL      string      `json:"coverImageUrl"`
	Owner              string      `json:"owner"`
	Level              float32     `json:"level"`
	Health             float32     `json:"health"`
	ControllingFaction string      `json:"controllingFaction"` // Will be 'E'nlightened, 'R'esistance, or 'N'eutral
	Mods               []Mod       `json:"mods"`
	Resonators         []Resonator `json:"resonators"`
}

type portalStatus struct {
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
