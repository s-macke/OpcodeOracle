package segments

import (
	"testing"

	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

func TestPlanDataOnly(t *testing.T) {
	s := state.NewState(make([]byte, 256), 0x0800, nil, "test.prg")

	segs := Plan(s)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if segs[0].Type != Data {
		t.Fatalf("expected data segment, got %s", segs[0].Type)
	}
}

func TestPlanCodeWithSubroutines(t *testing.T) {
	s := state.NewState(make([]byte, 0x1000), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0FFF, regions.RegionCode)
	_ = s.Symbols.Add(0x0800, symbols.Symbol{Name: "main", Type: symbols.SymbolEntry, Source: symbols.SourceUser})
	_ = s.Symbols.Add(0x0900, symbols.Symbol{Name: "sub1", Type: symbols.SymbolSubroutine, Source: symbols.SourceAuto})
	_ = s.Symbols.Add(0x0A00, symbols.Symbol{Name: "sub2", Type: symbols.SymbolSubroutine, Source: symbols.SourceAuto})

	segs := Plan(s)

	var subSegs []Segment
	for _, seg := range segs {
		if seg.Type == Sub {
			subSegs = append(subSegs, seg)
		}
	}
	if len(subSegs) != 3 {
		t.Fatalf("expected 3 sub segments, got %d", len(subSegs))
	}
	if subSegs[0].Start != 0x0800 || subSegs[0].End != 0x08FF {
		t.Fatalf("segment 0 bounds = %04X-%04X, want 0800-08FF", subSegs[0].Start, subSegs[0].End)
	}
	if subSegs[2].Start != 0x0A00 || subSegs[2].End != 0x0FFF {
		t.Fatalf("segment 2 bounds = %04X-%04X, want 0A00-0FFF", subSegs[2].Start, subSegs[2].End)
	}
}

func TestPlanNoOverlaps(t *testing.T) {
	s := state.NewState(make([]byte, 0x200), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x09FF, regions.RegionCode)
	_ = s.Symbols.Add(0x0900, symbols.Symbol{Name: "sub1", Type: symbols.SymbolSubroutine, Source: symbols.SourceAuto})

	segs := Plan(s)
	for i := 1; i < len(segs); i++ {
		if segs[i-1].End >= segs[i].Start {
			t.Fatalf("segments overlap at index %d: %04X-%04X and %04X-%04X", i, segs[i-1].Start, segs[i-1].End, segs[i].Start, segs[i].End)
		}
	}
}

func TestFilterIntersecting(t *testing.T) {
	segs := []Segment{
		{Start: 0x0800, End: 0x08FF, Type: Code},
		{Start: 0x0900, End: 0x09FF, Type: Sub},
		{Start: 0x0A00, End: 0x0AFF, Type: Data},
	}
	filtered := FilterIntersecting(segs, 0x08F0, 0x0905)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered segments, got %d", len(filtered))
	}
	if filtered[0].Start != 0x0800 || filtered[1].Start != 0x0900 {
		t.Fatalf("unexpected filtered segments: %+v", filtered)
	}
}
