package stateio

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	if js.Version != "1.1" {
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
	if s.Metadata.ArchiveOnSave {
		if err := archiveExistingStateFile(path); err != nil {
			return err
		}
	}

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

func archiveExistingStateFile(path string) error {
	existingData, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading existing state for archive: %w", err)
	}

	dir := filepath.Dir(path)
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("creating archive directory: %w", err)
	}

	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	timestamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	baseName := fmt.Sprintf("%s_%s", base, timestamp)

	for i := 0; ; i++ {
		name := baseName
		if i > 0 {
			name = fmt.Sprintf("%s_%d", baseName, i)
		}
		archivePath := filepath.Join(archiveDir, name+".opcodeoracle.json")
		if _, err := os.Stat(archivePath); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking archive filename: %w", err)
		}

		if err := os.WriteFile(archivePath, existingData, 0644); err != nil {
			return fmt.Errorf("writing archive file: %w", err)
		}
		return nil
	}
}
