package calc

import (
	"math"
	"testing"

	"ice-pos-api-v2/internal/domain"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		v1       domain.FloatVector
		v2       domain.FloatVector
		expected float64
		delta    float64
	}{
		{
			name:     "identical vectors",
			v1:       domain.FloatVector{1.0, 0.0, 0.0},
			v2:       domain.FloatVector{1.0, 0.0, 0.0},
			expected: 1.0,
			delta:    0.0001,
		},
		{
			name:     "orthogonal vectors",
			v1:       domain.FloatVector{1.0, 0.0, 0.0},
			v2:       domain.FloatVector{0.0, 1.0, 0.0},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "opposite vectors",
			v1:       domain.FloatVector{1.0, 0.0},
			v2:       domain.FloatVector{-1.0, 0.0},
			expected: -1.0,
			delta:    0.0001,
		},
		{
			name:     "different lengths",
			v1:       domain.FloatVector{1.0, 0.0},
			v2:       domain.FloatVector{1.0, 0.0, 0.0},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "empty vectors",
			v1:       domain.FloatVector{},
			v2:       domain.FloatVector{},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "zero vector",
			v1:       domain.FloatVector{0.0, 0.0, 0.0},
			v2:       domain.FloatVector{1.0, 2.0, 3.0},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "arbitrary vectors with known similarity",
			v1:       domain.FloatVector{1.0, 2.0, 3.0},
			v2:       domain.FloatVector{2.0, 4.0, 6.0}, // same direction, double magnitude
			expected: 1.0,
			delta:    0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CosineSimilarity(tt.v1, tt.v2)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("CosineSimilarity() = %v, want %v (delta %v)", got, tt.expected, tt.delta)
			}
		})
	}
}
