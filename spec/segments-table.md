# Segment Specification

This document specifies the segment system for OpcodeOracle.

See [state.md](state.md) for the JSON file format.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

Segments define contiguous memory regions with a specific type classification (code, data, string, etc.). They help the disassembler and exporter understand how to process different parts of the binary.

## Type Definition

### SegmentType

```go
type SegmentType string

const (
    SegmentCode     SegmentType = "code"
    SegmentData     SegmentType = "data"
    SegmentString   SegmentType = "string"
    SegmentTable    SegmentType = "table"
    SegmentGraphics SegmentType = "graphics"
)
```

| Value      | Description                              |
|------------|------------------------------------------|
| `code`     | Executable instructions                  |
| `data`     | Generic data bytes                       |
| `string`   | Text string (PETSCII or ASCII)           |
| `table`    | Lookup table (jump table, data table)    |
| `graphics` | Bitmap or sprite data                    |

## Struct Definition

### Segment

```go
type Segment struct {
    Start uint16
    End   uint16
    Type  SegmentType
}
```

| Field   | Type        | Description                           |
|---------|-------------|---------------------------------------|
| `Start` | uint16      | Start address (inclusive)             |
| `End`   | uint16      | End address (inclusive)               |
| `Type`  | SegmentType | Classification of the memory segment  |

## SegmentTable Interface

```go
type SegmentTable interface {
    At(addr uint16) *Segment
    Add(s Segment) error
    Remove(start uint16)
    Update(start uint16, newType SegmentType) error
    All() []Segment
    ByType(t SegmentType) []Segment
}
```

| Method                                             | Description                                      |
|----------------------------------------------------|--------------------------------------------------|
| `At(addr uint16) *Segment`                         | Get segment containing address (nil if none)     |
| `Add(s Segment) error`                             | Add segment (error if overlaps existing segment) |
| `Remove(start uint16)`                             | Remove segment by start address                  |
| `Update(start uint16, newType SegmentType) error`  | Change segment type (error if not found)         |
| `All() []Segment`                                  | Get all segments sorted by start address         |
| `ByType(t SegmentType) []Segment`                  | Get segments of specific type                    |

## Segment Rules

- Segments cannot overlap
- The entire binary does not need to be covered by segments
- Addresses not in any segment are treated as unclassified
- Segments are sorted by start address in output
