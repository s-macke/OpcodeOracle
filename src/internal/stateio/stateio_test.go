package stateio

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"opcodeoracle/internal/author"
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

	original.Annotations.Set(0x0801, "Entry point", author.User)
	original.Headlines.Set(0x0801, "Program Start", author.Assistant)

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

	// Verify version is updated to 1.1
	if loaded.Version != "1.1" {
		t.Errorf("Version = %q, want %q", loaded.Version, "1.1")
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
	sym, ok := loaded.Symbols.At(0x0801)
	if !ok {
		t.Error("Symbol at 0x0801 should exist")
	} else if sym.Name != "start" {
		t.Errorf("Symbol name = %q, want %q", sym.Name, "start")
	}

	// Check annotations (inline only)
	anns := loaded.Annotations.At(0x0801)
	if len(anns) != 1 {
		t.Errorf("Annotations at 0x0801 = %d, want 1", len(anns))
	}

	userAnn := loaded.Annotations.Get(0x0801, author.User)
	if userAnn == nil {
		t.Error("User annotation at 0x0801 is nil")
	} else if userAnn.Comment != "Entry point" {
		t.Errorf("User annotation comment = %q, want %q", userAnn.Comment, "Entry point")
	}

	// Check headlines
	hdls := loaded.Headlines.At(0x0801)
	if len(hdls) != 1 {
		t.Errorf("Headlines at 0x0801 = %d, want 1", len(hdls))
	}

	assistantHdl := loaded.Headlines.Get(0x0801, author.Assistant)
	if assistantHdl == nil {
		t.Error("Assistant headline at 0x0801 is nil")
	} else if assistantHdl.Comment != "Program Start" {
		t.Errorf("Assistant headline comment = %q, want %q", assistantHdl.Comment, "Program Start")
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

	// Write minimal JSON (new format v1.1)
	content := `{
  "version": "1.1",
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

	if s.Version != "1.1" {
		t.Errorf("Version = %q, want %q", s.Version, "1.1")
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

func TestLoadNewFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "new_format.orc")

	content := `{
  "version": "1.1",
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
      "user": {"comment": "Program entry"}
    }
  },
  "headlines": {
    "0x0801": {
      "assistant": {"comment": "Main section"}
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
	sym, ok := s.Symbols.At(0x0801)
	if !ok || sym.Name != "start" {
		t.Errorf("Symbol at 0x0801 = %v, want start", sym)
	}
	sym, ok = s.Symbols.At(0xD020)
	if !ok || sym.Name != "BORDER" {
		t.Errorf("Symbol at 0xD020 = %v, want BORDER", sym)
	}

	// Check annotations (inline)
	userAnn := s.Annotations.Get(0x0801, author.User)
	if userAnn == nil || userAnn.Comment != "Program entry" {
		t.Errorf("User annotation = %v, want 'Program entry'", userAnn)
	}

	// Check headlines
	assistantHdl := s.Headlines.Get(0x0801, author.Assistant)
	if assistantHdl == nil || assistantHdl.Comment != "Main section" {
		t.Errorf("Assistant headline = %v, want 'Main section'", assistantHdl)
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
  "version": "1.1",
  "metadata": {"modified": "2025-01-22T10:30:00Z"},
  "binary": {"data": [169], "origin": "0x0801"},
  "entryPoints": ["0x0801"]
}`,
		},
		{
			name: "missing binary data",
			content: `{
  "version": "1.1",
  "metadata": {"created": "2025-01-22T10:30:00Z", "modified": "2025-01-22T10:30:00Z"},
  "binary": {"origin": "0x0801"},
  "entryPoints": ["0x0801"]
}`,
		},
		{
			name: "missing entry points",
			content: `{
  "version": "1.1",
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
  "version": "1.1",
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
      "user": {"comment": "User only comment"}
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

	userAnn := s.Annotations.Get(0x0801, author.User)
	if userAnn == nil || userAnn.Comment != "User only comment" {
		t.Errorf("User annotation = %v, want 'User only comment'", userAnn)
	}

	assistantAnn := s.Annotations.Get(0x0801, author.Assistant)
	if assistantAnn != nil {
		t.Errorf("Assistant annotation = %v, want nil", assistantAnn)
	}
}

func TestSaveNewFormatStructure(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.orc")

	// Create state with both annotations and headlines
	s := state.NewState([]byte{0xA9, 0x00}, 0x0801, []uint16{0x0801}, "test.prg")
	s.Annotations.Set(0x0801, "Inline comment", author.User)
	s.Headlines.Set(0x0801, "Section header", author.Assistant)

	if err := Save(s, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Read raw JSON to verify structure
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(data)

	// Check version is 1.1
	if !contains(content, `"version": "1.1"`) {
		t.Error("JSON should contain version 1.1")
	}

	// Check annotations section exists
	if !contains(content, `"annotations"`) {
		t.Error("JSON should contain annotations key")
	}

	// Check headlines section exists
	if !contains(content, `"headlines"`) {
		t.Error("JSON should contain headlines key")
	}

	// Verify annotations don't have type field
	if contains(content, `"type": "inline"`) || contains(content, `"type": "headline"`) {
		t.Error("New format should not have type field in annotations")
	}
}

func TestSaveLoadArchiveOnSave(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "archive_flag.orc")

	s := state.NewState([]byte{0xEA}, 0x0800, []uint16{0x0800}, "test.prg")
	s.Metadata.ArchiveOnSave = true

	if err := Save(s, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !loaded.Metadata.ArchiveOnSave {
		t.Fatal("ArchiveOnSave should persist through save/load")
	}
}

func TestSaveArchiveOnFirstSaveNoArchiveFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "first_save.orc")

	s := state.NewState([]byte{0xEA}, 0x0800, []uint16{0x0800}, "test.prg")
	s.Metadata.ArchiveOnSave = true

	if err := Save(s, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	archiveDir := filepath.Join(tmpDir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if err == nil && len(entries) != 0 {
		t.Fatalf("expected no archive files on first save, got %d", len(entries))
	}
}

func TestSaveArchiveOnOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "overwrite.orc")

	s := state.NewState([]byte{0xEA}, 0x0800, []uint16{0x0800}, "test.prg")
	s.Metadata.ArchiveOnSave = true
	s.Metadata.Description = "first"
	if err := Save(s, path); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	beforeOverwrite, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	s.Metadata.Description = "second"
	if err := Save(s, path); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}

	archiveDir := filepath.Join(tmpDir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 archive file, got %d", len(entries))
	}

	archivePath := filepath.Join(archiveDir, entries[0].Name())
	archived, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile(archive) error = %v", err)
	}

	if string(archived) != string(beforeOverwrite) {
		t.Fatal("archive file does not match pre-overwrite state")
	}
}

func TestSaveArchiveDisabledNoArchive(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "disabled.orc")

	s := state.NewState([]byte{0xEA}, 0x0800, []uint16{0x0800}, "test.prg")
	s.Metadata.ArchiveOnSave = false
	if err := Save(s, path); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	s.Metadata.Description = "changed"
	if err := Save(s, path); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}

	archiveDir := filepath.Join(tmpDir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if err == nil && len(entries) != 0 {
		t.Fatalf("expected no archive files when disabled, got %d", len(entries))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
