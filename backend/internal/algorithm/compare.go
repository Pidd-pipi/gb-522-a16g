package algorithm

import (
	"fmt"
	"math"
	"sort"
)

type ComparableEvent struct {
	ID                                     uint
	DistanceM, InsertionLossDB, Confidence float64
}

type Difference struct {
	Kind            string  `json:"kind"`
	BaselineEventID *uint   `json:"baseline_event_id,omitempty"`
	CurrentEventID  *uint   `json:"current_event_id,omitempty"`
	DistanceM       float64 `json:"distance_m"`
	LossDeltaDB     float64 `json:"loss_delta_db"`
	Confidence      float64 `json:"confidence"`
}

func CompareBaseline(baseline, current []ComparableEvent, toleranceM, lossIncreaseDB float64) ([]Difference, error) {
	if toleranceM <= 0 {
		return nil, fmt.Errorf("distance tolerance must be positive")
	}
	if lossIncreaseDB <= 0 {
		return nil, fmt.Errorf("loss increase threshold must be positive")
	}
	used := make(map[int]bool)
	differences := make([]Difference, 0)
	for _, base := range baseline {
		match := -1
		best := math.MaxFloat64
		for i, candidate := range current {
			if used[i] {
				continue
			}
			delta := math.Abs(base.DistanceM - candidate.DistanceM)
			if delta <= toleranceM && delta < best {
				match, best = i, delta
			}
		}
		if match < 0 {
			id := base.ID
			differences = append(differences, Difference{Kind: "disappeared", BaselineEventID: &id, DistanceM: base.DistanceM, Confidence: round(base.Confidence * 0.9)})
			continue
		}
		used[match] = true
		candidate := current[match]
		lossDelta := candidate.InsertionLossDB - base.InsertionLossDB
		if lossDelta >= lossIncreaseDB {
			baseID, currentID := base.ID, candidate.ID
			differences = append(differences, Difference{Kind: "loss_increased", BaselineEventID: &baseID, CurrentEventID: &currentID, DistanceM: candidate.DistanceM, LossDeltaDB: round(lossDelta), Confidence: round(math.Min(base.Confidence, candidate.Confidence))})
		}
	}
	for i, candidate := range current {
		if used[i] {
			continue
		}
		id := candidate.ID
		differences = append(differences, Difference{Kind: "new", CurrentEventID: &id, DistanceM: candidate.DistanceM, LossDeltaDB: candidate.InsertionLossDB, Confidence: round(candidate.Confidence)})
	}
	sort.Slice(differences, func(i, j int) bool {
		if differences[i].DistanceM == differences[j].DistanceM {
			return differences[i].Kind < differences[j].Kind
		}
		return differences[i].DistanceM < differences[j].DistanceM
	})
	return differences, nil
}

func PrimaryDifference(differences []Difference) (distance, uncertainty float64, ok bool) {
	if len(differences) == 0 {
		return 0, 0, false
	}
	best := differences[0]
	for _, difference := range differences[1:] {
		if difference.Confidence > best.Confidence || difference.Kind == "loss_increased" && best.Kind != "loss_increased" {
			best = difference
		}
	}
	return best.DistanceM, math.Max(1, 20*(1-best.Confidence)), true
}
