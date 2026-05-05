package environments

import (
	"testing"
)

func TestSortPlatformLast(t *testing.T) {
	tests := []struct {
		name            string
		clusters        []string
		platformCluster string
		want            []string
	}{
		{
			name:            "platform in the middle",
			clusters:        []string{"test-onboarding", "test-platform", "test-worker-1"},
			platformCluster: "test-platform",
			want:            []string{"test-onboarding", "test-worker-1", "test-platform"},
		},
		{
			name:            "platform already last",
			clusters:        []string{"test-onboarding", "test-worker-1", "test-platform"},
			platformCluster: "test-platform",
			want:            []string{"test-onboarding", "test-worker-1", "test-platform"},
		},
		{
			name:            "platform first",
			clusters:        []string{"test-platform", "test-onboarding", "test-worker-1"},
			platformCluster: "test-platform",
			want:            []string{"test-worker-1", "test-onboarding", "test-platform"},
		},
		{
			name:            "only platform",
			clusters:        []string{"test-platform"},
			platformCluster: "test-platform",
			want:            []string{"test-platform"},
		},
		{
			name:            "platform not present",
			clusters:        []string{"test-onboarding", "test-worker-1"},
			platformCluster: "test-platform",
			want:            []string{"test-onboarding", "test-worker-1"},
		},
		{
			name:            "empty slice",
			clusters:        []string{},
			platformCluster: "test-platform",
			want:            []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortPlatformLast(tt.clusters, tt.platformCluster)
			if len(tt.clusters) != len(tt.want) {
				t.Fatalf("got %v, want %v", tt.clusters, tt.want)
			}
			for i := range tt.clusters {
				if tt.clusters[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q (full slice: %v)", i, tt.clusters[i], tt.want[i], tt.clusters)
				}
			}
		})
	}
}
