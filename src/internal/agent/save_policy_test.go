package agent

import "testing"

func TestShouldSaveImmediatelyOnMutation(t *testing.T) {
	tests := []struct {
		name          string
		dryRun        bool
		statePath     string
		archiveOnSave bool
		mutationCount uint64
		lastSaved     uint64
		want          bool
	}{
		{
			name:          "archive mode saves when mutation increased",
			statePath:     "state.json",
			archiveOnSave: true,
			mutationCount: 3,
			lastSaved:     2,
			want:          true,
		},
		{
			name:          "archive mode does not save when unchanged",
			statePath:     "state.json",
			archiveOnSave: true,
			mutationCount: 2,
			lastSaved:     2,
			want:          false,
		},
		{
			name:          "dry run disables immediate save",
			dryRun:        true,
			statePath:     "state.json",
			archiveOnSave: true,
			mutationCount: 3,
			lastSaved:     2,
			want:          false,
		},
		{
			name:          "missing path disables immediate save",
			archiveOnSave: true,
			mutationCount: 3,
			lastSaved:     2,
			want:          false,
		},
		{
			name:          "non archive mode never immediate saves",
			statePath:     "state.json",
			archiveOnSave: false,
			mutationCount: 3,
			lastSaved:     2,
			want:          false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldSaveImmediatelyOnMutation(tc.dryRun, tc.statePath, tc.archiveOnSave, tc.mutationCount, tc.lastSaved)
			if got != tc.want {
				t.Fatalf("shouldSaveImmediatelyOnMutation(...) = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShouldAttemptPeriodicSave(t *testing.T) {
	tests := []struct {
		name          string
		dryRun        bool
		statePath     string
		archiveOnSave bool
		saveInterval  int
		toolCallCount int
		want          bool
	}{
		{
			name:          "periodic save triggers at interval",
			statePath:     "state.json",
			saveInterval:  5,
			toolCallCount: 5,
			want:          true,
		},
		{
			name:          "periodic save does not trigger before interval",
			statePath:     "state.json",
			saveInterval:  5,
			toolCallCount: 4,
			want:          false,
		},
		{
			name:          "archive mode disables periodic save",
			statePath:     "state.json",
			archiveOnSave: true,
			saveInterval:  5,
			toolCallCount: 5,
			want:          false,
		},
		{
			name:          "dry run disables periodic save",
			dryRun:        true,
			statePath:     "state.json",
			saveInterval:  5,
			toolCallCount: 5,
			want:          false,
		},
		{
			name:          "missing path disables periodic save",
			saveInterval:  5,
			toolCallCount: 5,
			want:          false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldAttemptPeriodicSave(tc.dryRun, tc.statePath, tc.archiveOnSave, tc.saveInterval, tc.toolCallCount)
			if got != tc.want {
				t.Fatalf("shouldAttemptPeriodicSave(...) = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShouldAttemptSaveOnAgentError(t *testing.T) {
	tests := []struct {
		name          string
		dryRun        bool
		statePath     string
		archiveOnSave bool
		saveInterval  int
		mutationCount uint64
		lastSaved     uint64
		want          bool
	}{
		{
			name:          "archive mode saves on error when mutation pending",
			statePath:     "state.json",
			archiveOnSave: true,
			saveInterval:  5,
			mutationCount: 2,
			lastSaved:     1,
			want:          true,
		},
		{
			name:          "archive mode skips on error when no pending mutation",
			statePath:     "state.json",
			archiveOnSave: true,
			saveInterval:  5,
			mutationCount: 2,
			lastSaved:     2,
			want:          false,
		},
		{
			name:         "non archive mode follows interval availability",
			statePath:    "state.json",
			saveInterval: 5,
			want:         true,
		},
		{
			name:         "non archive mode disabled when interval off",
			statePath:    "state.json",
			saveInterval: 0,
			want:         false,
		},
		{
			name:          "dry run disables error save",
			dryRun:        true,
			statePath:     "state.json",
			archiveOnSave: true,
			saveInterval:  5,
			mutationCount: 2,
			lastSaved:     1,
			want:          false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldAttemptSaveOnAgentError(
				tc.dryRun,
				tc.statePath,
				tc.archiveOnSave,
				tc.saveInterval,
				tc.mutationCount,
				tc.lastSaved,
			)
			if got != tc.want {
				t.Fatalf("shouldAttemptSaveOnAgentError(...) = %v, want %v", got, tc.want)
			}
		})
	}
}
