package algorithm

import (
	"fmt"
	"math"
)

const vacuumSpeedMPerSecond = 299792458.0

func SampleDistance(index int, sampleIntervalNS, refractiveIndex float64) (float64, error) {
	if index < 0 {
		return 0, fmt.Errorf("sample index cannot be negative")
	}
	if sampleIntervalNS <= 0 {
		return 0, fmt.Errorf("sample interval must be positive")
	}
	if refractiveIndex < 1.3 || refractiveIndex > 1.7 {
		return 0, fmt.Errorf("refractive index outside 1.3-1.7")
	}
	roundTripSeconds := float64(index) * sampleIntervalNS * 1e-9
	distance := vacuumSpeedMPerSecond * roundTripSeconds / (2 * refractiveIndex)
	return math.Round(distance*100) / 100, nil
}

func ValidRouteDistance(distance, routeLength float64) bool {
	return !math.IsNaN(distance) && !math.IsInf(distance, 0) && distance >= 0 && distance <= routeLength
}

func DistanceUncertainty(sampleIntervalNS, refractiveIndex float64, mergeWindow int) (float64, error) {
	if mergeWindow < 1 {
		return 0, fmt.Errorf("merge window must be positive")
	}
	oneSample, err := SampleDistance(1, sampleIntervalNS, refractiveIndex)
	if err != nil {
		return 0, err
	}
	return math.Round(oneSample*float64(mergeWindow)*100) / 100, nil
}
