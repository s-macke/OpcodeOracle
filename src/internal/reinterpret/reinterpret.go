package reinterpret

import (
	"fmt"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
)

// AsCode forces a single address to be treated as code seed and reanalyzes from scratch.
func AsCode(s *state.State, addr uint16, source regions.RegionSource) (*analysis.Analyzer, error) {
	// One-byte high-priority code seed; analyzer grows flow from this address.
	s.Regions.SetWithSource(addr, addr, regions.RegionCode, source)

	analyzer, err := analysis.ReanalyzeFromScratch(s)
	if err != nil {
		return nil, err
	}
	return analyzer, nil
}

// AsData forces an address range to remain data and reanalyzes from scratch.
func AsData(s *state.State, start, end uint16, source regions.RegionSource) (*analysis.Analyzer, error) {
	if start > end {
		return nil, fmt.Errorf("start address ($%04X) is greater than end address ($%04X)", start, end)
	}
	s.Regions.SetWithSource(start, end, regions.RegionData, source)
	return analysis.ReanalyzeFromScratch(s)
}
