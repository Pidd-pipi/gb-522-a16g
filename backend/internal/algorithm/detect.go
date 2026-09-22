package algorithm

import (
	"fmt"
	"math"
	"sort"

	"fiber-otdr-fault-localization/backend/internal/constants"
)

type Peak struct {
	Index     int
	Magnitude float64
	Direction float64
}

type DetectedEvent struct {
	Index           int                 `json:"index"`
	DistanceM       float64             `json:"distance_m"`
	Type            constants.EventType `json:"event_type"`
	InsertionLossDB float64             `json:"insertion_loss_db"`
	ReflectanceDB   float64             `json:"reflectance_db"`
	Confidence      float64             `json:"confidence"`
}

func EstimateNoiseFloor(points []float64) (float64, error) {
	if len(points) < 16 {
		return 0, fmt.Errorf("at least 16 samples are required for noise estimation")
	}
	start := len(points) * 4 / 5
	tail := append([]float64(nil), points[start:]...)
	sort.Float64s(tail)
	mid := len(tail) / 2
	if len(tail)%2 == 0 {
		return (tail[mid-1] + tail[mid]) / 2, nil
	}
	return tail[mid], nil
}

func DerivativePeaks(points []float64, threshold float64) ([]Peak, error) {
	if len(points) < 3 {
		return nil, fmt.Errorf("at least three samples are required")
	}
	if threshold <= 0 {
		return nil, fmt.Errorf("threshold must be positive")
	}
	peaks := make([]Peak, 0)
	for i := 1; i < len(points); i++ {
		delta := points[i] - points[i-1]
		if math.Abs(delta) >= threshold {
			peaks = append(peaks, Peak{Index: i, Magnitude: math.Abs(delta), Direction: math.Copysign(1, delta)})
		}
	}
	return peaks, nil
}

func MergePeaks(peaks []Peak, window int) ([]Peak, error) {
	if window < 1 {
		return nil, fmt.Errorf("merge window must be positive")
	}
	if len(peaks) == 0 {
		return []Peak{}, nil
	}
	merged := make([]Peak, 0, len(peaks))
	current := peaks[0]
	for _, peak := range peaks[1:] {
		if peak.Index-current.Index <= window {
			if peak.Magnitude > current.Magnitude {
				current = peak
			}
			continue
		}
		merged = append(merged, current)
		current = peak
	}
	return append(merged, current), nil
}

func ClassifyPeak(peak Peak, points []float64, noiseFloor, threshold float64) (constants.EventType, float64, float64, float64) {
	before, after := points[peak.Index-1], points[peak.Index]
	loss := math.Max(0, before-after)
	reflectance := math.Min(before, after) - noiseFloor
	eventType := constants.EventSplice
	if peak.Direction > 0 && peak.Magnitude >= threshold*1.5 {
		eventType = constants.EventConnector
	}
	if peak.Direction < 0 && peak.Magnitude >= threshold*2.5 {
		eventType = constants.EventBreak
	}
	if peak.Direction < 0 && peak.Magnitude < threshold*1.5 {
		eventType = constants.EventBend
	}
	if peak.Index >= len(points)-3 {
		eventType = constants.EventEnd
	}
	confidence := math.Min(0.99, 0.5+peak.Magnitude/(threshold*10))
	return eventType, round(loss), round(reflectance), round(confidence)
}

func Detect(points []float64, threshold float64, mergeWindow int, sampleIntervalNS, refractiveIndex, routeLength float64) ([]DetectedEvent, int, error) {
	noise, err := EstimateNoiseFloor(points)
	if err != nil {
		return nil, 0, err
	}
	peaks, err := DerivativePeaks(points, threshold)
	if err != nil {
		return nil, 0, err
	}
	peaks, err = MergePeaks(peaks, mergeWindow)
	if err != nil {
		return nil, 0, err
	}
	events, rejected := make([]DetectedEvent, 0, len(peaks)), 0
	for _, peak := range peaks {
		distance, err := SampleDistance(peak.Index, sampleIntervalNS, refractiveIndex)
		if err != nil {
			return nil, rejected, err
		}
		if !ValidRouteDistance(distance, routeLength) {
			rejected++
			continue
		}
		typeValue, loss, reflectance, confidence := ClassifyPeak(peak, points, noise, threshold)
		events = append(events, DetectedEvent{peak.Index, distance, typeValue, loss, reflectance, confidence})
	}
	return events, rejected, nil
}

func round(value float64) float64 { return math.Round(value*1000) / 1000 }
