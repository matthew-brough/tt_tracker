package models

import "time"

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Vehicle struct {
	Type  string `json:"vehicle_type,omitempty"`
	Name  string `json:"vehicle_name,omitempty"`
	Label string `json:"vehicle_label,omitempty"`
	Class int    `json:"vehicle_class,omitempty"`
	Spawn string `json:"vehicle_spawn,omitempty"`
	Model int    `json:"vehicle_model,omitempty"`
}

type Job struct {
	Group string `json:"group,omitempty"`
	Name  string `json:"name,omitempty"`
}

type HistoryPoint struct {
	Index int
	X     float64
	Y     float64
	Z     float64
}

type Player struct {
	Name     string
	SourceID int
	VrpID    int
	Position Position
	Vehicle  Vehicle
	Job      Job
	History  []HistoryPoint
}

type PositionRow struct {
	Ts          time.Time
	VrpID       int
	X           float64
	Y           float64
	Z           float64
	JobGroup    *string
	JobName     *string
	VehicleType *string
	VehicleName *string
}

type PlayerState struct {
	VrpID       int       `json:"vrp_id"`
	Name        string    `json:"name"`
	X           float64   `json:"x"`
	Y           float64   `json:"y"`
	Z           float64   `json:"z"`
	JobGroup    string    `json:"job_group,omitempty"`
	JobName     string    `json:"job_name,omitempty"`
	VehicleType string    `json:"vehicle_type,omitempty"`
	VehicleName string    `json:"vehicle_name,omitempty"`
	Trail       []Position `json:"trail,omitempty"`
}

type HexBin struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Count int    `json:"count"`
	Edge float64 `json:"edge"`
}
