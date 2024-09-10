package main

import (
	"testing"
)

func TestParseDeviceOptionConstraints(t *testing.T) {
	tests := []struct {
		input []string
		// Expected results value map
		expectedr map[string][]string
		// Expected default value map
		expectedd map[string]string
	}{
		{
			input: []string{
				"mode 24bit Color[Fast]|Black & White|True Gray|Gray[Error Diffusion] [24bit Color[Fast]]",
				"resolution 100|150|200|300|400|600|1200dpi [100]",
				"source Automatic Document Feeder(left aligned) [Automatic Document Feeder(left aligned)]",
			},
			expectedr: map[string][]string{
				"mode":       {"24bit Color[Fast]", "Black & White", "True Gray", "Gray[Error Diffusion]"},
				"resolution": {"100", "150", "200", "300", "400", "600", "1200dpi"},
				"source":     {"Automatic Document Feeder(left aligned)"},
			},
			expectedd: map[string]string{
				"mode":       "24bit Color[Fast]",
				"resolution": "100",
				"source":     "Automatic Document Feeder(left aligned)",
			},
		},
	}

	for _, test := range tests {
		gotr, gotd := parseDeviceOptionConstraints(test.input)

		if gotr == nil {
			t.Error("nil pointer for map")
		}

		for k, v := range gotr {
			for kk, vv := range test.expectedr {
				if k != kk {
					continue
				}
				if len(v) != len(vv) {
					t.Errorf("string slice len mismatch: got %v, wanted %v", len(v), len(vv))
				}

				for i := range v {
					if v[i] != vv[i] {
						t.Errorf("string slice value mismatch: got %v, wanted %v", v[i], vv[i])
					}
				}
			}
		}

		for k, v := range gotd {
			if v != test.expectedd[k] {
				t.Errorf("expected default values had a mismatch: got %v, wanted %v", v, test.expectedd[k])
			}
		}
	}
}
