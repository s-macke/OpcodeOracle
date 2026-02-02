package main

import (
	"github.com/urfave/cli/v2"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/stateio"
)

// loadState loads a state file.
// Returns the state and any error as a cli.ExitError.
func loadState(stateFile string) (*state.State, error) {
	s, err := stateio.Load(stateFile)
	if err != nil {
		return nil, cli.Exit("error: "+err.Error(), ExitInvalidState)
	}
	return s, nil
}

// saveState saves a state file.
// Returns any error as a cli.ExitError.
func saveState(s *state.State, stateFile string) error {
	if err := stateio.Save(s, stateFile); err != nil {
		return cli.Exit("error: "+err.Error(), ExitIOError)
	}
	return nil
}

// loadAndAnalyze loads a state file and runs analysis with UpdateXRefsOnly.
// Returns the state, analyzer, and any error as a cli.ExitError.
func loadAndAnalyze(stateFile string) (*state.State, *analysis.Analyzer, error) {
	s, err := loadState(stateFile)
	if err != nil {
		return nil, nil, err
	}

	analyzer := analysis.NewAnalyzer(s, analysis.UpdateXRefsOnly)
	if err := analyzer.Analyze(); err != nil {
		return nil, nil, cli.Exit("error: analysis failed: "+err.Error(), ExitAnalysisError)
	}

	return s, analyzer, nil
}
