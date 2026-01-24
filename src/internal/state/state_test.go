package state

import (
	"testing"
)

func TestNewState(t *testing.T) {
	data := []byte{0xA9, 0x00, 0x8D, 0x20, 0xD0}
	origin := uint16(0x0801)
	sourceFile := "test.prg"

	s := NewState(data, origin, sourceFile)

	if s.Version != CurrentVersion {
		t.Errorf("Version = %q, want %q", s.Version, CurrentVersion)
	}
	if s.Metadata.SourceFile != sourceFile {
		t.Errorf("SourceFile = %q, want %q", s.Metadata.SourceFile, sourceFile)
	}
	if len(s.Binary.Data) != len(data) {
		t.Errorf("Binary.Data length = %d, want %d", len(s.Binary.Data), len(data))
	}
	if s.Binary.Origin != origin {
		t.Errorf("Binary.Origin = %04X, want %04X", s.Binary.Origin, origin)
	}
	if len(s.EntryPoints) != 1 || s.EntryPoints[0] != origin {
		t.Errorf("EntryPoints = %v, want [%04X]", s.EntryPoints, origin)
	}
	if s.Symbols == nil {
		t.Error("Symbols is nil")
	}
	if s.Annotations == nil {
		t.Error("Annotations is nil")
	}
	if s.Regions == nil {
		t.Error("Regions is nil")
	}
	if s.XRefs == nil {
		t.Error("XRefs is nil")
	}
	if err := s.Regions.Validate(); err != nil {
		t.Errorf("Regions.Validate() = %v, want nil", err)
	}
}
