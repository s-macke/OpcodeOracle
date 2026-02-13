package analysis

import (
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
	"opcodeoracle/internal/xrefs"
)

// ReanalyzeFromScratch rebuilds analysis artifacts from a clean analysis state.
// It preserves user/assistant edits while clearing auto-generated analysis output.
func ReanalyzeFromScratch(s *state.State) (*Analyzer, error) {
	preserved := preservedNonAutoRegions(s)

	s.Regions.SetRegions([]regions.Region{
		{Start: 0x0000, End: 0xFFFF, Type: regions.RegionData, Source: regions.RegionSourceAuto},
	})
	for _, r := range preserved {
		s.Regions.SetWithSource(r.Start, r.End, r.Type, r.Source)
	}
	s.XRefs = xrefs.NewTable()
	s.Symbols.RemoveBySource(symbols.SourceAuto)

	analyzer := NewAnalyzer(s, UpdateAll)
	if err := analyzer.Analyze(); err != nil {
		return nil, err
	}
	return analyzer, nil
}

func preservedNonAutoRegions(s *state.State) []regions.Region {
	var out []regions.Region
	for _, r := range s.Regions.Regions() {
		if r.Source == regions.RegionSourceAssistant || r.Source == regions.RegionSourceUser {
			out = append(out, r)
		}
	}
	return out
}
