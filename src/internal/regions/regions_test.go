package regions

import (
	"testing"
)

func TestRegionTableEmpty(t *testing.T) {
	table := NewTable()

	r := table.RegionAt(0x0800)
	if r != nil {
		t.Errorf("RegionAt(0x0800) on empty table returned %v, want nil", r)
	}
}

func TestRegionTableSet(t *testing.T) {
	table := NewTable()

	// Must cover full range 0x0000-0xFFFF
	table.Set(0x0000, 0xFFFF, RegionData)

	r := table.RegionAt(0x0800)
	if r == nil {
		t.Fatal("RegionAt(0x0800) returned nil")
	}
	if r.Type != RegionData {
		t.Errorf("RegionAt(0x0800).Type = %v, want %v", r.Type, RegionData)
	}
	if r.Start != 0x0000 || r.End != 0xFFFF {
		t.Errorf("Region = %04X-%04X, want 0000-FFFF", r.Start, r.End)
	}
}

func TestRegionTableSplit(t *testing.T) {
	table := NewTable()

	// Set initial data region covering full range
	table.Set(0x0000, 0xFFFF, RegionData)

	// Set code region in middle - should split
	table.Set(0x0900, 0x09FF, RegionCode)

	// Check before code region
	r := table.RegionAt(0x0850)
	if r == nil {
		t.Fatal("RegionAt(0x0850) returned nil")
	}
	if r.Type != RegionData {
		t.Errorf("RegionAt(0x0850).Type = %v, want %v", r.Type, RegionData)
	}
	if r.Start != 0x0000 || r.End != 0x08FF {
		t.Errorf("Data region before = %04X-%04X, want 0000-08FF", r.Start, r.End)
	}

	// Check code region
	r = table.RegionAt(0x0950)
	if r == nil {
		t.Fatal("RegionAt(0x0950) returned nil")
	}
	if r.Type != RegionCode {
		t.Errorf("RegionAt(0x0950).Type = %v, want %v", r.Type, RegionCode)
	}
	if r.Start != 0x0900 || r.End != 0x09FF {
		t.Errorf("Code region = %04X-%04X, want 0900-09FF", r.Start, r.End)
	}

	// Check after code region
	r = table.RegionAt(0x0A50)
	if r == nil {
		t.Fatal("RegionAt(0x0A50) returned nil")
	}
	if r.Type != RegionData {
		t.Errorf("RegionAt(0x0A50).Type = %v, want %v", r.Type, RegionData)
	}
	if r.Start != 0x0A00 || r.End != 0xFFFF {
		t.Errorf("Data region after = %04X-%04X, want 0A00-FFFF", r.Start, r.End)
	}

	// Verify we have 3 regions
	regions := table.Regions()
	if len(regions) != 3 {
		t.Errorf("Expected 3 regions, got %d", len(regions))
	}
}

func TestRegionTableAtOutOfRange(t *testing.T) {
	// Use SetRegions to test partial coverage (bypasses validation)
	table := NewTable()
	table.SetRegions([]Region{
		{Start: 0x0800, End: 0x08FF, Type: RegionData},
	})

	if r := table.RegionAt(0x07FF); r != nil {
		t.Errorf("RegionAt(0x07FF) = %v, want nil", r)
	}
	if r := table.RegionAt(0x0900); r != nil {
		t.Errorf("RegionAt(0x0900) = %v, want nil", r)
	}
}

func TestRegionTableMerge(t *testing.T) {
	table := NewTable()

	// Create initial full coverage
	table.Set(0x0000, 0xFFFF, RegionData)

	// Set code region in middle
	table.Set(0x0900, 0x09FF, RegionCode)

	// Now set code back to data - should merge all into one
	table.Set(0x0900, 0x09FF, RegionData)

	regions := table.Regions()
	if len(regions) != 1 {
		t.Fatalf("Expected 1 merged region, got %d: %v", len(regions), regions)
	}

	if regions[0].Start != 0x0000 || regions[0].End != 0xFFFF || regions[0].Type != RegionData {
		t.Errorf("Merged region = {%04X, %04X, %v}, want {0000, FFFF, data}",
			regions[0].Start, regions[0].End, regions[0].Type)
	}
}

func TestRegionTableSetRegions(t *testing.T) {
	table := NewTable()

	// SetRegions bypasses validation, useful for testing partial states
	regions := []Region{
		{Start: 0x0800, End: 0x08FF, Type: RegionData},
		{Start: 0x0900, End: 0x09FF, Type: RegionCode},
		{Start: 0x0A00, End: 0x0FFF, Type: RegionData},
	}

	table.SetRegions(regions)

	r := table.RegionAt(0x0950)
	if r == nil {
		t.Fatal("RegionAt(0x0950) returned nil")
	}
	if r.Type != RegionCode {
		t.Errorf("RegionAt(0x0950).Type = %v, want %v", r.Type, RegionCode)
	}
}

func TestRegionTableOverlappingSet(t *testing.T) {
	table := NewTable()
	table.Set(0x0000, 0xFFFF, RegionData)

	// Set overlapping region
	table.Set(0x0850, 0x0A50, RegionCode)

	regions := table.Regions()

	if len(regions) != 3 {
		t.Fatalf("Regions() returned %d regions, want 3", len(regions))
	}

	if regions[0].Start != 0x0000 || regions[0].End != 0x084F || regions[0].Type != RegionData {
		t.Errorf("regions[0] = {%04X, %04X, %v}, want {0000, 084F, data}",
			regions[0].Start, regions[0].End, regions[0].Type)
	}
	if regions[1].Start != 0x0850 || regions[1].End != 0x0A50 || regions[1].Type != RegionCode {
		t.Errorf("regions[1] = {%04X, %04X, %v}, want {0850, 0A50, code}",
			regions[1].Start, regions[1].End, regions[1].Type)
	}
	if regions[2].Start != 0x0A51 || regions[2].End != 0xFFFF || regions[2].Type != RegionData {
		t.Errorf("regions[2] = {%04X, %04X, %v}, want {0A51, FFFF, data}",
			regions[2].Start, regions[2].End, regions[2].Type)
	}
}

func TestRegionTableInvalidRange(t *testing.T) {
	table := NewTable()
	table.Set(0x0000, 0xFFFF, RegionData)

	// Set with start > end should be ignored
	table.Set(0x0900, 0x0800, RegionCode)

	regions := table.Regions()
	if len(regions) != 1 {
		t.Errorf("Expected 1 region after invalid Set, got %d", len(regions))
	}
}

func TestRegionTableAdjacentDifferentTypes(t *testing.T) {
	table := NewTable()

	// Start with full coverage, then carve out a different type
	table.Set(0x0000, 0xFFFF, RegionData)
	table.Set(0x8000, 0xFFFF, RegionCode)

	regions := table.Regions()
	if len(regions) != 2 {
		t.Fatalf("Expected 2 regions, got %d", len(regions))
	}
}

func TestRegionTableAdjacentSameTypes(t *testing.T) {
	table := NewTable()

	// Start with code, then set second half to data, then set first half to data
	// This tests that adjacent same-type regions merge
	table.Set(0x0000, 0xFFFF, RegionCode)
	table.Set(0x8000, 0xFFFF, RegionData)
	table.Set(0x0000, 0x7FFF, RegionData)

	regions := table.Regions()
	if len(regions) != 1 {
		t.Fatalf("Expected 1 merged region, got %d: %v", len(regions), regions)
	}
	if regions[0].Start != 0x0000 || regions[0].End != 0xFFFF {
		t.Errorf("Merged region = {%04X, %04X}, want {0000, FFFF}",
			regions[0].Start, regions[0].End)
	}
}

func TestRegionTableValidateOK(t *testing.T) {
	table := NewTable()
	// Start with full coverage, then carve out regions
	table.Set(0x0000, 0xFFFF, RegionData)
	table.Set(0x8000, 0xBFFF, RegionCode)

	if err := table.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	regions := table.Regions()
	if len(regions) != 3 {
		t.Fatalf("Expected 3 regions, got %d", len(regions))
	}
}

func TestRegionTableValidateOverlap(t *testing.T) {
	table := NewTable()
	table.SetRegions([]Region{
		{Start: 0x0800, End: 0x08FF, Type: RegionData},
		{Start: 0x08F0, End: 0x09FF, Type: RegionCode},
	})

	if err := table.Validate(); err == nil {
		t.Fatal("Validate returned nil for overlapping regions")
	}
}

func TestRegionTableValidateGap(t *testing.T) {
	table := NewTable()
	table.SetRegions([]Region{
		{Start: 0x0800, End: 0x08FF, Type: RegionData},
		{Start: 0x0901, End: 0x09FF, Type: RegionCode},
	})

	if err := table.Validate(); err == nil {
		t.Fatal("Validate returned nil for gapped regions")
	}
}

func TestRegionTableValidateUnmerged(t *testing.T) {
	table := NewTable()
	table.SetRegions([]Region{
		{Start: 0x0800, End: 0x08FF, Type: RegionData},
		{Start: 0x0900, End: 0x09FF, Type: RegionData},
	})

	if err := table.Validate(); err == nil {
		t.Fatal("Validate returned nil for unmerged regions")
	}
}

func TestRegionTableValidateEmpty(t *testing.T) {
	table := NewTable()

	if err := table.Validate(); err == nil {
		t.Fatal("Validate returned nil for empty table")
	}
}

func TestRegionTableValidateStartEndMismatch(t *testing.T) {
	table := NewTable()
	table.SetRegions([]Region{
		{Start: 0x0001, End: 0x0FFF, Type: RegionData},
		{Start: 0x1000, End: 0xFFFE, Type: RegionCode},
	})

	if err := table.Validate(); err == nil {
		t.Fatal("Validate returned nil for mismatched coverage range")
	}
}
