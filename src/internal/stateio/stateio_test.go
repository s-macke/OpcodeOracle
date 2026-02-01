package stateio

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"opcodeoracle/internal/annotations"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

func TestSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.orc")

	// Create state with data
	original := state.NewState([]byte{0xA9, 0x00, 0x4C, 0x00, 0x08}, 0x0801, []uint16{0x0801, 0x1000}, "test.prg")
	original.Metadata.Description = "Test description"

	original.Symbols.Add(0x0801, symbols.Symbol{
		Name:   "start",
		Type:   symbols.SymbolEntry,
		Source: symbols.SourceUser,
	})
	original.Symbols.Add(0x1000, symbols.Symbol{
		Name:   "loop",
		Type:   symbols.SymbolLabel,
		Source: symbols.SourceAuto,
	})

	original.Annotations.Set(0x0801, annotations.AnnotationInline, "Entry point", annotations.AuthorUser)
	original.Annotations.Set(0x0801, annotations.AnnotationHeadline, "Program Start", annotations.AuthorAssistant)

	original.Regions.Set(0x0801, 0x0900, regions.RegionCode)

	// Save
	if err := Save(original, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify
	if loaded.Version != original.Version {
		t.Errorf("Version = %q, want %q", loaded.Version, original.Version)
	}
	if loaded.Metadata.SourceFile != original.Metadata.SourceFile {
		t.Errorf("SourceFile = %q, want %q", loaded.Metadata.SourceFile, original.Metadata.SourceFile)
	}
	if loaded.Metadata.Description != original.Metadata.Description {
		t.Errorf("Description = %q, want %q", loaded.Metadata.Description, original.Metadata.Description)
	}
	if len(loaded.Binary.Data) != len(original.Binary.Data) {
		t.Errorf("Binary.Data length = %d, want %d", len(loaded.Binary.Data), len(original.Binary.Data))
	}
	if loaded.Binary.Origin != original.Binary.Origin {
		t.Errorf("Binary.Origin = %04X, want %04X", loaded.Binary.Origin, original.Binary.Origin)
	}
	if len(loaded.EntryPoints) != len(original.EntryPoints) {
		t.Errorf("EntryPoints length = %d, want %d", len(loaded.EntryPoints), len(original.EntryPoints))
	}

	// Check symbols
	syms := loaded.Symbols.At(0x0801)
	if len(syms) != 1 {
		t.Errorf("Symbols at 0x0801 = %d, want 1", len(syms))
	} else if syms[0].Name != "start" {
		t.Errorf("Symbol name = %q, want %q", syms[0].Name, "start")
	}

	// Check annotations - should have 2 (one user, one assistant)
	anns := loaded.Annotations.At(0x0801)
	if len(anns) != 2 {
		t.Errorf("Annotations at 0x0801 = %d, want 2", len(anns))
	}

	userAnn := loaded.Annotations.Get(0x0801, annotations.AuthorUser)
	if userAnn == nil {
		t.Error("User annotation at 0x0801 is nil")
	} else if userAnn.Comment != "Entry point" {
		t.Errorf("User annotation comment = %q, want %q", userAnn.Comment, "Entry point")
	}

	assistantAnn := loaded.Annotations.Get(0x0801, annotations.AuthorAssistant)
	if assistantAnn == nil {
		t.Error("Assistant annotation at 0x0801 is nil")
	} else if assistantAnn.Comment != "Program Start" {
		t.Errorf("Assistant annotation comment = %q, want %q", assistantAnn.Comment, "Program Start")
	}

	// Check regions
	if err := loaded.Regions.Validate(); err != nil {
		t.Errorf("Regions.Validate() = %v, want nil", err)
	}
	reg := loaded.Regions.RegionAt(0x0850)
	if reg == nil || reg.Type != regions.RegionCode {
		t.Errorf("Region at 0x0850 = %v, want code region", reg)
	}
}

func TestLoadMinimal(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "minimal.orc")

	// Write minimal JSON
	content := `{
  "version": "1.0",
  "metadata": {
    "created": "2025-01-22T10:30:00Z",
    "modified": "2025-01-22T10:30:00Z"
  },
  "binary": {
    "data": [169, 0, 141, 32, 208],
    "origin": "0x0801"
  },
  "entryPoints": ["0x0801"]
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if s.Version != "1.0" {
		t.Errorf("Version = %q, want %q", s.Version, "1.0")
	}
	if len(s.Binary.Data) != 5 {
		t.Errorf("Binary.Data length = %d, want 5", len(s.Binary.Data))
	}
	if s.Binary.Origin != 0x0801 {
		t.Errorf("Binary.Origin = %04X, want 0801", s.Binary.Origin)
	}

	// Should default to full data region
	if err := s.Regions.Validate(); err != nil {
		t.Errorf("Regions.Validate() = %v, want nil", err)
	}
	reg := s.Regions.RegionAt(0x5000)
	if reg == nil || reg.Type != regions.RegionData {
		t.Errorf("Region at 0x5000 = %v, want data region", reg)
	}
}

func TestLoadFull(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "full.orc")

	content := `{
  "version": "1.0",
  "metadata": {
    "created": "2025-01-22T10:30:00Z",
    "modified": "2025-01-22T14:45:00Z",
    "sourceFile": "game.prg",
    "description": "Space shooter game analysis"
  },
  "binary": {
    "data": [169, 0, 141, 32, 208, 76, 21, 8],
    "origin": "0x0801"
  },
  "entryPoints": ["0x0801", "0x1000"],
  "symbols": {
    "0x0801": [
      {"name": "start", "type": "entry", "source": "user"}
    ],
    "0xD020": [
      {"name": "BORDER", "type": "byte", "source": "c64rom"}
    ]
  },
  "annotations": {
    "0x0801": {
      "user": {"type": "inline", "comment": "Program entry"},
      "assistant": {"type": "headline", "comment": "Main section"}
    }
  },
  "regions": [
    {"start": "0x0000", "end": "0x0800", "type": "data"},
    {"start": "0x0801", "end": "0x0FFF", "type": "code"},
    {"start": "0x1000", "end": "0xFFFF", "type": "data"}
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if s.Metadata.Description != "Space shooter game analysis" {
		t.Errorf("Description = %q, want %q", s.Metadata.Description, "Space shooter game analysis")
	}
	if len(s.EntryPoints) != 2 {
		t.Errorf("EntryPoints length = %d, want 2", len(s.EntryPoints))
	}

	// Check symbols
	syms := s.Symbols.At(0x0801)
	if len(syms) != 1 || syms[0].Name != "start" {
		t.Errorf("Symbols at 0x0801 = %v, want [start]", syms)
	}
	syms = s.Symbols.At(0xD020)
	if len(syms) != 1 || syms[0].Name != "BORDER" {
		t.Errorf("Symbols at 0xD020 = %v, want [BORDER]", syms)
	}

	// Check annotations
	anns := s.Annotations.At(0x0801)
	if len(anns) != 2 {
		t.Errorf("Annotations at 0x0801 length = %d, want 2", len(anns))
	}

	userAnn := s.Annotations.Get(0x0801, annotations.AuthorUser)
	if userAnn == nil || userAnn.Comment != "Program entry" {
		t.Errorf("User annotation = %v, want 'Program entry'", userAnn)
	}

	assistantAnn := s.Annotations.Get(0x0801, annotations.AuthorAssistant)
	if assistantAnn == nil || assistantAnn.Comment != "Main section" {
		t.Errorf("Assistant annotation = %v, want 'Main section'", assistantAnn)
	}

	// Check regions
	reg := s.Regions.RegionAt(0x0850)
	if reg == nil || reg.Type != regions.RegionCode {
		t.Errorf("Region at 0x0850 = %v, want code region", reg)
	}
	reg = s.Regions.RegionAt(0x0500)
	if reg == nil || reg.Type != regions.RegionData {
		t.Errorf("Region at 0x0500 = %v, want data region", reg)
	}
}

func TestLoadInvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.orc")

	content := `{
  "version": "2.0",
  "metadata": {
    "created": "2025-01-22T10:30:00Z",
    "modified": "2025-01-22T10:30:00Z"
  },
  "binary": {
    "data": [169],
    "origin": "0x0801"
  },
  "entryPoints": ["0x0801"]
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadMissingRequired(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing version",
			content: `{
  "metadata": {"created": "2025-01-22T10:30:00Z", "modified": "2025-01-22T10:30:00Z"},
  "binary": {"data": [169], "origin": "0x0801"},
  "entryPoints": ["0x0801"]
}`,
		},
		{
			name: "missing created",
			content: `{
  "version": "1.0",
  "metadata": {"modified": "2025-01-22T10:30:00Z"},
  "binary": {"data": [169], "origin": "0x0801"},
  "entryPoints": ["0x0801"]
}`,
		},
		{
			name: "missing binary data",
			content: `{
  "version": "1.0",
  "metadata": {"created": "2025-01-22T10:30:00Z", "modified": "2025-01-22T10:30:00Z"},
  "binary": {"origin": "0x0801"},
  "entryPoints": ["0x0801"]
}`,
		},
		{
			name: "missing entry points",
			content: `{
  "version": "1.0",
  "metadata": {"created": "2025-01-22T10:30:00Z", "modified": "2025-01-22T10:30:00Z"},
  "binary": {"data": [169], "origin": "0x0801"}
}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tc.name+".orc")
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			_, err := Load(path)
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
		})
	}
}

func TestSaveUpdatesModified(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.orc")

	// Create state with a Modified time in the past
	s := state.NewState([]byte{0xA9}, 0x0801, []uint16{0x0801}, "test.prg")
	originalModified := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	s.Metadata.Modified = originalModified

	if err := Save(s, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if !s.Metadata.Modified.After(originalModified) {
		t.Errorf("Modified timestamp not updated: %v <= %v", s.Metadata.Modified, originalModified)
	}

	// Load and verify timestamp was persisted
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !loaded.Metadata.Modified.After(originalModified) {
		t.Errorf("Loaded Modified timestamp = %v, want > %v", loaded.Metadata.Modified, originalModified)
	}
}

func TestParseHex(t *testing.T) {
	tests := []struct {
		input   string
		want    uint16
		wantErr bool
	}{
		{"0x0000", 0x0000, false},
		{"0x0801", 0x0801, false},
		{"0xFFFF", 0xFFFF, false},
		{"0X1234", 0x1234, false},
		{"ABCD", 0xABCD, false},
		{"abcd", 0xABCD, false},
		{"", 0, true},
		{"0xGGGG", 0, true},
		{"0x10000", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseHex(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseHex(%q) error = %v, wantErr = %v", tc.input, err, tc.wantErr)
				return
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseHex(%q) = %04X, want %04X", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatHex(t *testing.T) {
	tests := []struct {
		input uint16
		want  string
	}{
		{0x0000, "0x0000"},
		{0x0801, "0x0801"},
		{0xFFFF, "0xFFFF"},
		{0x00FF, "0x00FF"},
	}

	for _, tc := range tests {
		got := formatHex(tc.input)
		if got != tc.want {
			t.Errorf("formatHex(%04X) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestLoadAnnotationsOnlyUser(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "user_only.orc")

	content := `{
  "version": "1.0",
  "metadata": {
    "created": "2025-01-22T10:30:00Z",
    "modified": "2025-01-22T10:30:00Z"
  },
  "binary": {
    "data": [169, 0],
    "origin": "0x0801"
  },
  "entryPoints": ["0x0801"],
  "annotations": {
    "0x0801": {
      "user": {"type": "inline", "comment": "User only comment"}
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	anns := s.Annotations.At(0x0801)
	if len(anns) != 1 {
		t.Errorf("Annotations at 0x0801 = %d, want 1", len(anns))
	}

	userAnn := s.Annotations.Get(0x0801, annotations.AuthorUser)
	if userAnn == nil || userAnn.Comment != "User only comment" {
		t.Errorf("User annotation = %v, want 'User only comment'", userAnn)
	}

	assistantAnn := s.Annotations.Get(0x0801, annotations.AuthorAssistant)
	if assistantAnn != nil {
		t.Errorf("Assistant annotation = %v, want nil", assistantAnn)
	}
}
