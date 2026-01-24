package regions

import (
	"fmt"
	"sort"
)

type RegionType string

const (
	RegionCode RegionType = "code"
	RegionData RegionType = "data"
)

type Region struct {
	Start uint16
	End   uint16
	Type  RegionType
}

type Table struct {
	regions []Region
}

func NewTable() *Table {
	return &Table{
		regions: make([]Region, 0),
	}
}

// RegionAt returns the region containing the given address.
func (t *Table) RegionAt(addr uint16) *Region {
	for i := range t.regions {
		if addr >= t.regions[i].Start && addr <= t.regions[i].End {
			return &t.regions[i]
		}
	}
	return nil
}

// At returns the region type at the given address.
// Returns empty string if no region contains the address.
func (t *Table) At(addr uint16) RegionType {
	r := t.RegionAt(addr)
	if r == nil { // If everything works, that should never happen
		return RegionData
	}
	return r.Type
}

// Set sets the region type for the given address range.
// Handles splitting and merging of existing regions.
func (t *Table) Set(start, end uint16, regionType RegionType) {
	if start > end {
		return
	}

	var result []Region

	for _, r := range t.regions {
		// Region is completely before new range
		if r.End < start {
			result = append(result, r)
			continue
		}
		// Region is completely after new range
		if r.Start > end {
			result = append(result, r)
			continue
		}
		// Region overlaps - split if needed
		if r.Start < start {
			result = append(result, Region{Start: r.Start, End: start - 1, Type: r.Type})
		}
		if r.End > end {
			result = append(result, Region{Start: end + 1, End: r.End, Type: r.Type})
		}
	}

	// Add the new region
	result = append(result, Region{Start: start, End: end, Type: regionType})

	// Sort by start address
	sort.Slice(result, func(i, j int) bool {
		return result[i].Start < result[j].Start
	})

	// Merge adjacent regions of the same type
	t.regions = merge(result)

	if err := t.Validate(); err != nil {
		panic(err)
	}
}

// Validate checks that regions are non-overlapping, merged, and fully cover the table's address range.
func (t *Table) Validate() error {
	if len(t.regions) == 0 {
		return fmt.Errorf("no regions defined")
	}

	first := t.regions[0]
	if first.Start != 0x0000 {
		return fmt.Errorf("region coverage starts at %04X, want 0000", first.Start)
	}
	if first.Start > first.End {
		return fmt.Errorf("invalid region %04X-%04X", first.Start, first.End)
	}

	prev := first
	for i := 1; i < len(t.regions); i++ {
		curr := t.regions[i]
		if curr.Start > curr.End {
			return fmt.Errorf("invalid region %04X-%04X", curr.Start, curr.End)
		}
		if curr.Start <= prev.End {
			return fmt.Errorf("regions overlap or are out of order at %04X-%04X and %04X-%04X",
				prev.Start, prev.End, curr.Start, curr.End)
		}
		if uint32(curr.Start) != uint32(prev.End)+1 {
			return fmt.Errorf("gap between regions at %04X-%04X and %04X-%04X",
				prev.Start, prev.End, curr.Start, curr.End)
		}
		if curr.Type == prev.Type {
			return fmt.Errorf("adjacent regions not merged at %04X-%04X and %04X-%04X",
				prev.Start, prev.End, curr.Start, curr.End)
		}
		prev = curr
	}

	if prev.End != 0xFFFF {
		return fmt.Errorf("region coverage ends at %04X, want FFFF", prev.End)
	}

	return nil
}

func merge(regions []Region) []Region {
	if len(regions) == 0 {
		return regions
	}

	var merged []Region
	current := regions[0]

	for i := 1; i < len(regions); i++ {
		// Check if adjacent and same type
		if regions[i].Start == current.End+1 && regions[i].Type == current.Type {
			current.End = regions[i].End
		} else {
			merged = append(merged, current)
			current = regions[i]
		}
	}
	merged = append(merged, current)

	return merged
}

// Regions returns all regions.
func (t *Table) Regions() []Region {
	return t.regions
}

// SetRegions replaces all regions.
func (t *Table) SetRegions(regions []Region) {
	t.regions = regions
}
