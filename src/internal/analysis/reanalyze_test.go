package analysis

import (
	"testing"

	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/symbols"
)

func TestReanalyzeFromScratchClearsAutoArtifacts(t *testing.T) {
	// 0800: JSR $0804 ; RTS ; NOP ; RTS
	data := []byte{0x20, 0x04, 0x08, 0x60, 0xEA, 0x60}
	s := newTestState(data, 0x0800, []uint16{0x0800})

	// Pre-populate stale artifacts that should be cleared.
	s.Regions.Set(0x0800, 0x0805, regions.RegionCode)
	_ = s.Symbols.Add(0x0804, symbols.Symbol{Name: "STALE_AUTO", Type: symbols.SymbolSubroutine, Source: symbols.SourceAuto})
	_ = s.Symbols.Add(0x0802, symbols.Symbol{Name: "USER_KEEP", Type: symbols.SymbolLabel, Source: symbols.SourceUser})
	s.XRefs.Add(0x1234, 0x5678, "call")

	analyzer, err := ReanalyzeFromScratch(s)
	if err != nil {
		t.Fatalf("ReanalyzeFromScratch error = %v, want nil", err)
	}
	if analyzer == nil {
		t.Fatal("analyzer = nil, want non-nil")
	}

	// User symbol survives.
	if _, ok := s.Symbols.At(0x0802); !ok {
		t.Fatal("expected user symbol to persist")
	}
	// Auto symbol should be replaced by newly generated one, not stale name.
	sym, ok := s.Symbols.At(0x0804)
	if !ok {
		t.Fatal("expected subroutine symbol at 0x0804")
	}
	if sym.Name == "STALE_AUTO" {
		t.Fatalf("stale auto symbol survived: %+v", sym)
	}
	if len(s.XRefs.From(0x1234)) != 0 {
		t.Fatal("expected stale xrefs to be cleared")
	}
}

func TestAnalyzeSkipsLockedDataRegion(t *testing.T) {
	// 0800: JMP $0803 ; 0803: RTS
	data := []byte{0x4C, 0x03, 0x08, 0x60}
	s := newTestState(data, 0x0800, []uint16{0x0800})
	s.Regions.SetWithSource(0x0803, 0x0803, regions.RegionData, regions.RegionSourceUser)

	analyzer := NewAnalyzer(s, UpdateAll)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze error = %v, want nil", err)
	}

	if analyzer.IsInstructionAt(0x0803) {
		t.Fatal("expected forced-data address 0x0803 to remain undecoded")
	}
	r := s.Regions.RegionAt(0x0803)
	if r == nil || r.Type != regions.RegionData {
		t.Fatalf("region at 0x0803 = %+v, want data", r)
	}
}

func TestReanalyzeFromScratchPreservesNonAutoRegions(t *testing.T) {
	// 0800: NOP, RTS
	data := []byte{0xEA, 0x60}
	s := newTestState(data, 0x0800, []uint16{0x0800})

	s.Regions.SetWithSource(0x0801, 0x0801, regions.RegionData, regions.RegionSourceUser)

	_, err := ReanalyzeFromScratch(s)
	if err != nil {
		t.Fatalf("ReanalyzeFromScratch error = %v, want nil", err)
	}

	r := s.Regions.RegionAt(0x0801)
	if r == nil || r.Type != regions.RegionData || r.Source != regions.RegionSourceUser {
		t.Fatalf("region at 0x0801 = %+v, want preserved user data", r)
	}
}

func TestReanalyzeFromEntryPointsDoesNotSeedFromManualCodeRegion(t *testing.T) {
	// 0800: RTS
	// 0801: JSR $0804
	// 0804: RTS
	data := []byte{0x60, 0x20, 0x04, 0x08, 0x60}
	s := newTestState(data, 0x0800, []uint16{0x0800})

	// Manual code region remains preserved, but should not seed strict reanalysis.
	s.Regions.SetWithSource(0x0801, 0x0804, regions.RegionCode, regions.RegionSourceUser)

	analyzer, err := ReanalyzeFromEntryPoints(s)
	if err != nil {
		t.Fatalf("ReanalyzeFromEntryPoints error = %v, want nil", err)
	}

	if analyzer.IsInstructionAt(0x0801) {
		t.Fatal("expected manual code region at 0x0801 to remain unseeded during strict reanalysis")
	}
	r := s.Regions.RegionAt(0x0801)
	if r == nil || r.Type != regions.RegionCode || r.Source != regions.RegionSourceUser {
		t.Fatalf("region at 0x0801 = %+v, want preserved user code", r)
	}
	if _, ok := s.Symbols.At(0x0804); ok {
		t.Fatal("expected no auto symbol at 0x0804 when manual code region is not used as a seed")
	}
}

func TestReanalyzeFromEntryPointsClearsStaleAutoCodeOutsideEntries(t *testing.T) {
	// 0800: RTS
	// 0801: NOP
	data := []byte{0x60, 0xEA}
	s := newTestState(data, 0x0800, []uint16{0x0800})

	// Stale auto code should be removed because only 0x0800 is an entry point.
	s.Regions.Set(0x0801, 0x0801, regions.RegionCode)
	_ = s.Symbols.Add(0x0801, symbols.Symbol{Name: "STALE_AUTO", Type: symbols.SymbolLabel, Source: symbols.SourceAuto})

	_, err := ReanalyzeFromEntryPoints(s)
	if err != nil {
		t.Fatalf("ReanalyzeFromEntryPoints error = %v, want nil", err)
	}

	r := s.Regions.RegionAt(0x0801)
	if r == nil || r.Type != regions.RegionData || r.Source != regions.RegionSourceAuto {
		t.Fatalf("region at 0x0801 = %+v, want auto data after strict reanalysis", r)
	}
	if _, ok := s.Symbols.At(0x0801); ok {
		t.Fatal("expected stale auto symbol at 0x0801 to be cleared")
	}
}

func TestReanalyzeFromEntryPointsIncludesExtraCodeAddresses(t *testing.T) {
	// 0800: RTS (entry point)
	// 0801: NOP (extra code address)
	data := []byte{0x60, 0xEA}
	s := newTestState(data, 0x0800, []uint16{0x0800})
	s.ExtraCodeAddresses = []uint16{0x0801}

	analyzer, err := ReanalyzeFromEntryPoints(s)
	if err != nil {
		t.Fatalf("ReanalyzeFromEntryPoints error = %v, want nil", err)
	}

	if !analyzer.IsInstructionAt(0x0801) {
		t.Fatal("expected extra code address 0x0801 to seed strict reanalysis")
	}
	if sym, ok := s.Symbols.At(0x0801); ok && sym.Type == symbols.SymbolEntry {
		t.Fatalf("unexpected entry symbol at extra code address: %+v", sym)
	}
}
