package algorithm

import (
	"fmt"
	"sort"
)

func MovingMedian(points []float64, window int) ([]float64, error) {
	if len(points) < 3 {
		return nil, fmt.Errorf("at least three samples are required")
	}
	if window < 1 || window > 31 {
		return nil, fmt.Errorf("window must be between 1 and 31")
	}
	if window%2 == 0 {
		window++
	}
	half := window / 2
	filtered := make([]float64, len(points))
	for i := range points {
		start, end := i-half, i+half+1
		if start < 0 {
			start = 0
		}
		if end > len(points) {
			end = len(points)
		}
		bucket := append([]float64(nil), points[start:end]...)
		sort.Float64s(bucket)
		middle := len(bucket) / 2
		if len(bucket)%2 == 0 {
			filtered[i] = (bucket[middle-1] + bucket[middle]) / 2
		} else {
			filtered[i] = bucket[middle]
		}
	}
	return filtered, nil
}

func MovingAverage(points []float64, window int) ([]float64, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("samples are empty")
	}
	if window < 1 || window > len(points) {
		return nil, fmt.Errorf("invalid average window")
	}
	out := make([]float64, len(points))
	var sum float64
	for i, point := range points {
		sum += point
		if i >= window {
			sum -= points[i-window]
		}
		count := i + 1
		if count > window {
			count = window
		}
		out[i] = sum / float64(count)
	}
	return out, nil
}
