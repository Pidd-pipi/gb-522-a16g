package algorithm

import (
	"encoding/json"
	"os"
	"testing"
)

type traceFixture struct {
	Raw                    []float64 `json:"raw"`
	Window                 int       `json:"window"`
	Threshold              float64   `json:"threshold"`
	MergeWindow            int       `json:"merge_window"`
	SampleIntervalNS       float64   `json:"sample_interval_ns"`
	RefractiveIndex        float64   `json:"refractive_index"`
	RouteLengthM           float64   `json:"route_length_m"`
	ExpectedEventCount     int       `json:"expected_event_count"`
	ExpectedFirstDistanceM float64   `json:"expected_first_distance_m"`
}

func TestPipelineFixture(t *testing.T) {
	encoded, err := os.ReadFile("testdata/trace_fixture.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture traceFixture
	if err := json.Unmarshal(encoded, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	filtered, err := MovingMedian(fixture.Raw, fixture.Window)
	if err != nil {
		t.Fatalf("denoise fixture: %v", err)
	}
	events, rejected, err := Detect(filtered, fixture.Threshold, fixture.MergeWindow, fixture.SampleIntervalNS, fixture.RefractiveIndex, fixture.RouteLengthM)
	if err != nil {
		t.Fatalf("detect fixture: %v", err)
	}
	if rejected != 0 {
		t.Fatalf("expected no rejected events, got %d", rejected)
	}
	if len(events) != fixture.ExpectedEventCount {
		t.Fatalf("expected %d event, got %d: %#v", fixture.ExpectedEventCount, len(events), events)
	}
	if events[0].DistanceM != fixture.ExpectedFirstDistanceM {
		t.Fatalf("expected distance %.2f, got %.2f", fixture.ExpectedFirstDistanceM, events[0].DistanceM)
	}
}

func TestSampleDistanceTable(t *testing.T) {
	tests := []struct {
		name                 string
		index                int
		interval, refractive float64
		want                 float64
		wantErr              bool
	}{
		{"origin", 0, 100, 1.468, 0, false},
		{"five samples", 5, 100, 1.468, 51.05, false},
		{"negative index", -1, 100, 1.468, 0, true},
		{"invalid refractive index", 2, 100, 2.0, 0, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := SampleDistance(test.index, test.interval, test.refractive)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("distance = %.2f, want %.2f", got, test.want)
			}
		})
	}
}
