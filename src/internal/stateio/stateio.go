package stateio

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"opcodeoracle/internal/state"
)

var (
	ErrUnsupportedVersion = errors.New("unsupported version")
	ErrMissingRequired    = errors.New("missing required field")
)

// Load reads a state from a JSON file.
func Load(path string) (*state.State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var js jsonState
	if err := json.Unmarshal(data, &js); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	// Validate version
	if js.Version == "" {
		return nil, fmt.Errorf("%w: version", ErrMissingRequired)
	}
	if js.Version != "1.0" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedVersion, js.Version)
	}

	// Validate required fields
	if js.Metadata.Created == "" {
		return nil, fmt.Errorf("%w: metadata.created", ErrMissingRequired)
	}
	if js.Metadata.Modified == "" {
		return nil, fmt.Errorf("%w: metadata.modified", ErrMissingRequired)
	}
	if js.Binary.Data == nil {
		return nil, fmt.Errorf("%w: binary.data", ErrMissingRequired)
	}
	if js.Binary.Origin == "" {
		return nil, fmt.Errorf("%w: binary.origin", ErrMissingRequired)
	}
	if js.EntryPoints == nil {
		return nil, fmt.Errorf("%w: entryPoints", ErrMissingRequired)
	}

	return jsonToState(&js)
}

// Save writes the state to a JSON file.
func Save(s *state.State, path string) error {
	s.Metadata.Modified = time.Now().UTC()

	js := stateToJSON(s)

	data, err := json.MarshalIndent(js, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	return nil
}
