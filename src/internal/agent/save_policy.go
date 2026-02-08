package agent

func shouldSaveImmediatelyOnMutation(dryRun bool, statePath string, archiveOnSave bool, mutationCount, lastSavedMutationCount uint64) bool {
	return !dryRun && statePath != "" && archiveOnSave && mutationCount > lastSavedMutationCount
}

func shouldAttemptPeriodicSave(dryRun bool, statePath string, archiveOnSave bool, saveInterval, toolCallCount int) bool {
	return !dryRun && statePath != "" && !archiveOnSave && saveInterval > 0 && toolCallCount >= saveInterval
}

func shouldAttemptSaveOnAgentError(dryRun bool, statePath string, archiveOnSave bool, saveInterval int, mutationCount, lastSavedMutationCount uint64) bool {
	if dryRun || statePath == "" {
		return false
	}
	if archiveOnSave {
		return mutationCount > lastSavedMutationCount
	}
	return saveInterval > 0
}
