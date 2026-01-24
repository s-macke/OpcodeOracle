# Region Specification

This document specifies the region system for OpcodeOracle.

See [state-file.md](state-file.md) for the JSON file format.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

Regions define contiguous memory areas with a specific type classification (code or data). They help the disassembler and exporter understand how to process different parts of the binary.

## Type Definition

### RegionType

```go
type RegionType string

const (
    RegionCode     RegionType = "code"
    RegionData     RegionType = "data"
)
```

| Value      | Description                              |
|------------|------------------------------------------|
| `code`     | Executable instructions                  |
| `data`     | Generic data bytes                       |

## Struct Definition

### Region

```go
type Region struct {
    Start uint16
    End   uint16
    Type  RegionType
}
```

| Field   | Type       | Description                          |
|---------|------------|--------------------------------------|
| `Start` | uint16     | Start address (inclusive)            |
| `End`   | uint16     | End address (inclusive)              |
| `Type`  | RegionType | Classification of the memory region  |

## RegionTable Interface

```go
type RegionTable interface {
    At(addr uint16) *Region
    Set(start uint16, end uint16, t RegionType)
}
```

| Method                                      | Description                                   |
|---------------------------------------------|-----------------------------------------------|
| `At(addr uint16) *Region`                   | Get region containing address                 |
| `Set(start, end uint16, t RegionType)`      | Set region type for range, splitting overlaps |

## Initialization

When a binary is loaded, the entire address range is initialized as a single `data` region:

```
Origin to Origin+Size-1: data
```

For example, a binary loaded at $0800 with size 2048 bytes:
```
$0800-$0FFF: data
```

## Set() Behavior

The `Set()` method carves out a new region:
- **Partial overlap**: Splits the overlapped region at the boundary
- **Full overlap**: Replaces the covered region entirely

### Example

Initial state (binary at $0800-$0FFF):
```
$0800-$0FFF: data
```

After `Set($0900, $09FF, code)` - splits data region:
```
$0800-$08FF: data
$0900-$09FF: code
$0A00-$0FFF: data
```

After `Set($0900, $09FF, data)` - replaces code region entirely:
```
$0800-$08FF: data
$0900-$09FF: data
$0A00-$0FFF: data
```

After `Set($0850, $0A50, code)` - splits first, replaces middle, splits last:
```
$0800-$084F: data
$0850-$0A50: code
$0A51-$0FFF: data
```

## Region Rules

- The entire binary is always fully covered by regions (no gaps)
- `Set()` automatically handles splitting, merging, and removal
- Regions are stored sorted by start address

## Implementation Notes

Internally, regions are stored as a byte array where each byte in the binary maps to a `RegionType`. This makes `Set()` a trivial fill operation and eliminates complex splitting logic.

For JSON serialization, the array is expanded into a list of `Region` structs by scanning for type boundaries. When loading from JSON, the region list is expanded back into the byte array.
