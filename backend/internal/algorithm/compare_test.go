package algorithm

import "testing"

func TestCompareBaselineTable(t *testing.T) {
	tests := []struct {
		name              string
		baseline, current []ComparableEvent
		tolerance, loss   float64
		kinds             []string
	}{
		{"new event", []ComparableEvent{{ID: 1, DistanceM: 100, InsertionLossDB: .2, Confidence: .9}}, []ComparableEvent{{ID: 2, DistanceM: 101, InsertionLossDB: .2, Confidence: .9}, {ID: 3, DistanceM: 300, InsertionLossDB: 1.1, Confidence: .8}}, 10, .5, []string{"new"}},
		{"disappeared event", []ComparableEvent{{ID: 1, DistanceM: 100, InsertionLossDB: .2, Confidence: .9}}, []ComparableEvent{{ID: 2, DistanceM: 300, InsertionLossDB: .2, Confidence: .9}}, 10, .5, []string{"disappeared", "new"}},
		{"loss increase", []ComparableEvent{{ID: 1, DistanceM: 100, InsertionLossDB: .2, Confidence: .9}}, []ComparableEvent{{ID: 2, DistanceM: 102, InsertionLossDB: 1.0, Confidence: .8}}, 10, .5, []string{"loss_increased"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			differences, err := CompareBaseline(test.baseline, test.current, test.tolerance, test.loss)
			if err != nil {
				t.Fatalf("compare: %v", err)
			}
			if len(differences) != len(test.kinds) {
				t.Fatalf("got %d differences, want %d: %#v", len(differences), len(test.kinds), differences)
			}
			for i, kind := range test.kinds {
				if differences[i].Kind != kind {
					t.Fatalf("difference %d kind = %s, want %s", i, differences[i].Kind, kind)
				}
			}
		})
	}
}
