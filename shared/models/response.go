package models

import (
	"encoding/json"
	"fmt"
)

type APIResponse struct {
	Players  []json.RawMessage `json:"players"`
	Caches   int               `json:"caches"`
	Requests int               `json:"requests"`
}

// ParsePlayer parses a single player from the positional array format:
// [name, source_id, vrp_id, position, vehicle, job, history]
// Every field may be missing or null — defensive parsing required.
func ParsePlayer(raw json.RawMessage) (*Player, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("player is not an array: %w", err)
	}
	if len(arr) < 3 {
		return nil, fmt.Errorf("player array too short: %d elements", len(arr))
	}

	p := &Player{}

	// [0] name (string)
	if len(arr) > 0 && string(arr[0]) != "null" {
		var name string
		if err := json.Unmarshal(arr[0], &name); err == nil {
			p.Name = name
		}
	}

	// [1] source_id (int)
	if len(arr) > 1 && string(arr[1]) != "null" {
		var sid float64
		if err := json.Unmarshal(arr[1], &sid); err == nil {
			p.SourceID = int(sid)
		}
	}

	// [2] vrp_id (int)
	if len(arr) > 2 && string(arr[2]) != "null" {
		var vid float64
		if err := json.Unmarshal(arr[2], &vid); err == nil {
			p.VrpID = int(vid)
		}
	}

	// [3] position ({x, y, z})
	if len(arr) > 3 && string(arr[3]) != "null" {
		if err := json.Unmarshal(arr[3], &p.Position); err != nil {
			// non-fatal: position might be malformed
		}
	}

	// [4] vehicle (object)
	if len(arr) > 4 && string(arr[4]) != "null" {
		if err := json.Unmarshal(arr[4], &p.Vehicle); err != nil {
			// non-fatal
		}
	}

	// [5] job (object)
	if len(arr) > 5 && string(arr[5]) != "null" {
		if err := json.Unmarshal(arr[5], &p.Job); err != nil {
			// non-fatal
		}
	}

	// [6] history (array of [index, x, y, z])
	if len(arr) > 6 && string(arr[6]) != "null" {
		var histRaw []json.RawMessage
		if err := json.Unmarshal(arr[6], &histRaw); err == nil {
			for _, h := range histRaw {
				var vals []float64
				if err := json.Unmarshal(h, &vals); err == nil && len(vals) >= 4 {
					p.History = append(p.History, HistoryPoint{
						Index: int(vals[0]),
						X:     vals[1],
						Y:     vals[2],
						Z:     vals[3],
					})
				}
			}
		}
	}

	if p.VrpID == 0 {
		return nil, fmt.Errorf("player has no vrp_id")
	}

	return p, nil
}
