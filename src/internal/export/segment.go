// Package export provides assembly file generation from disassembled state.
package export

import (
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/symbols"
)

// SegmentType identifies the kind of content in a segment.
type SegmentType string

const (
	SegmentCode SegmentType = "code" // Code without subroutine symbol
	SegmentSub  SegmentType = "sub"  // Subroutine or entry point
	SegmentData SegmentType = "dat"  // Data region
)

// Segment represents a contiguous block of code or data for export.
type Segment struct {
	Start uint16
	End   uint16
	Type  SegmentType
	Name  string // Symbol name if present
}

// identifySegments splits the binary into exportable segments based on regions and symbols.
func (e *Exporter) identifySegments() []Segment {
	var segments []Segment

	// Calculate binary bounds
	binStart := e.state.Binary.Start()
	binEnd := e.state.Binary.End()

	for _, region := range e.state.Regions.Regions() {
		// Skip regions completely outside binary bounds
		if region.End < binStart || region.Start > binEnd {
			continue
		}

		// Clamp region to binary bounds
		start := region.Start
		end := region.End
		if start < binStart {
			start = binStart
		}
		if end > binEnd {
			end = binEnd
		}

		clampedRegion := regions.Region{Start: start, End: end, Type: region.Type}

		if region.Type == regions.RegionData {
			segments = append(segments, Segment{
				Start: start,
				End:   end,
				Type:  SegmentData,
			})
		} else {
			// Code region - split by subroutine/entry symbols
			subSymbols := e.state.Symbols.SubroutinesInRange(start, end)

			if len(subSymbols) == 0 {
				// No subroutines - entire region is code
				segments = append(segments, Segment{
					Start: start,
					End:   end,
					Type:  SegmentCode,
				})
			} else {
				// Split by subroutine symbols
				segments = append(segments, e.splitBySubroutines(clampedRegion, subSymbols)...)
			}
		}
	}

	return segments
}

// splitBySubroutines splits a code region into segments at subroutine boundaries.
func (e *Exporter) splitBySubroutines(region regions.Region, subSymbols []symbols.AddressedSymbol) []Segment {
	var segments []Segment
	addr := region.Start

	for i, sym := range subSymbols {
		if addr < sym.Addr {
			// Code before this subroutine
			segments = append(segments, Segment{
				Start: addr,
				End:   sym.Addr - 1,
				Type:  SegmentCode,
			})
		}

		// Find end address (next symbol or region end)
		var endAddr uint16
		if i+1 < len(subSymbols) {
			endAddr = subSymbols[i+1].Addr - 1
		} else {
			endAddr = region.End
		}

		segments = append(segments, Segment{
			Start: sym.Addr,
			End:   endAddr,
			Type:  SegmentSub,
			Name:  sym.Symbol.Name,
		})

		addr = endAddr + 1
	}

	return segments
}
