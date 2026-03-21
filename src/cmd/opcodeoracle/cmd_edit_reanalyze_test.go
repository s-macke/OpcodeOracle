package main

import (
	"os"
	"path/filepath"
	"testing"

	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/stateio"
)

func TestCmdEditReanalyzeRebuildsAutoRegionsFromEntryPoints(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "project.opcodeoracle.json")

	// 0800: RTS
	// 0801: JSR $0804
	// 0804: RTS
	s := state.NewState([]byte{0x60, 0x20, 0x04, 0x08, 0x60}, 0x0800, []uint16{0x0800}, "project.prg")
	s.Regions.Set(0x0801, 0x0804, regions.RegionCode)
	s.Regions.SetWithSource(0x0801, 0x0804, regions.RegionCode, regions.RegionSourceUser)

	if err := stateio.Save(s, statePath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	if err := cmdEditReanalyze(statePath); err != nil {
		t.Fatalf("cmdEditReanalyze() error = %v", err)
	}

	loaded, err := stateio.Load(statePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	r := loaded.Regions.RegionAt(0x0801)
	if r == nil || r.Type != regions.RegionCode || r.Source != regions.RegionSourceUser {
		t.Fatalf("region at 0x0801 = %+v, want preserved user code", r)
	}
	if loaded.Regions.At(0x0804) != regions.RegionCode {
		t.Fatalf("expected 0x0804 to remain within preserved manual code region")
	}
	if _, ok := loaded.Symbols.At(0x0804); ok {
		t.Fatal("expected no auto symbol at 0x0804 after strict entry-point-only reanalysis")
	}
}
