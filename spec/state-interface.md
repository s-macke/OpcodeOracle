# State Interface Specification

This document specifies the Go interface for interacting with OpcodeOracle state files.

## Related Documents

| Document                                   | Description                            |
|--------------------------------------------|----------------------------------------|
| [state-file.md](state-file.md)             | JSON file format specification         |
| [binary.md](binary.md)                     | Binary struct and read methods         |
| [symbol-table.md](symbol-table.md)         | Symbol types, struct, and interface    |
| [annotation-table.md](annotation-table.md) | Annotation types and interface         |
| [regions-table.md](regions-table.md)       | Region types, struct, and interface    |
| [xref-table.md](xref-table.md)             | Cross-reference types and interface    |
| [entrypoint-table.md](entrypoint-table.md) | Entry point methods                    |

## Overview

The state interface provides programmatic access to load, save, query, and modify reverse engineering session data. All CLI commands (`new`, `info`, `export`) use this interface to interact with state files.

## Package Structure

State management is split across two packages:

| Package          | Purpose                                        |
|------------------|------------------------------------------------|
| `internal/state` | State and Metadata types, `NewState()` constructor |
| `internal/stateio` | File I/O: `Load()`, `Save()`, JSON serialization |

## State Struct

The primary struct for working with state data (defined in `internal/state`).

```go
type State struct {
    Version     string
    Metadata    Metadata
    Binary      Binary
    EntryPoints []uint16
    Symbols     *SymbolTable
    Annotations *AnnotationTable
    Regions     *RegionTable
    XRefs       *XRefTable
}
```

### State Fields

| Field         | Type               | Description                    |
|---------------|--------------------|--------------------------------|
| `Version`     | `string`           | Schema version (e.g., "1.0")   |
| `Metadata`    | `Metadata`         | Project metadata               |
| `Binary`      | `Binary`           | Binary data with read methods  |
| `EntryPoints` | `[]uint16`         | Code entry point addresses     |
| `Symbols`     | `*SymbolTable`     | Symbol table (labels, names)   |
| `Annotations` | `*AnnotationTable` | Comments and notes             |
| `Regions`     | `*RegionTable`     | Code/data region boundaries    |
| `XRefs`       | `*XRefTable`       | Cross-references (not persisted) |

## Package Functions

### state package

```go
// NewState creates an empty state with the given binary data
func NewState(data []byte, origin uint16, sourceFile string) *State

// CurrentVersion is the current schema version
const CurrentVersion = "1.0"
```

### stateio package

```go
// Load reads a state from a JSON file
func Load(path string) (*state.State, error)

// Save writes the state to a JSON file
func Save(s *state.State, path string) error

// Errors
var ErrUnsupportedVersion = errors.New("unsupported version")
var ErrMissingRequired    = errors.New("missing required field")
```

## Usage Examples

### Creating a New State

```go
import (
    "opcodeoracle/internal/state"
    "opcodeoracle/internal/stateio"
    "opcodeoracle/internal/symbols"
)

// Read binary file
data, err := os.ReadFile("game.prg")
if err != nil {
    return err
}

// Create state
s := state.NewState(data, 0x0801, "game.prg")
s.EntryPoints = append(s.EntryPoints, 0x0810)

// Add auto-generated symbol
s.Symbols.Add(0x0810, symbols.Symbol{
    Name:   "L_0810",
    Type:   symbols.SymbolLabel,
    Source: symbols.SourceAuto,
})

// Save to file
err = stateio.Save(s, "game.orc")
```

### Loading and Querying State

```go
import (
    "opcodeoracle/internal/stateio"
    "opcodeoracle/internal/regions"
)

s, err := stateio.Load("game.orc")
if err != nil {
    return err
}

// Get metadata
fmt.Printf("Source: %s\n", s.Metadata.SourceFile)

// Check region type at address
region := s.Regions.At(0x0900)
if region.Type == regions.RegionData {
    fmt.Println("Address is in data section")
}

// Mark address range as code
s.Regions.Set(0x0900, 0x09FF, regions.RegionCode)
```

### Adding User Annotations

```go
import (
    "opcodeoracle/internal/stateio"
    "opcodeoracle/internal/symbols"
    "opcodeoracle/internal/annotations"
)

s, _ := stateio.Load("game.orc")

// Add user symbol
s.Symbols.Add(0x1000, symbols.Symbol{
    Name:   "main_loop",
    Type:   symbols.SymbolSubroutine,
    Source: symbols.SourceUser,
})

// Add comment
s.Annotations.Add(0x1000, annotations.AnnotationInline, "Main game loop - runs every frame", "user")

// Save changes
stateio.Save(s, "game.orc")
```

## Error Handling

| Error Condition                            | Behavior     |
|--------------------------------------------|--------------|
| File not found (Load)                      | Return error |
| Invalid JSON (Load)                        | Return error |
| Version mismatch (Load)                    | Return error |
| Write permission denied (Save)             | Return error |
| Address outside binary (ReadByte/ReadWord) | Return error |
| Index out of range (Remove annotation)     | Return error |
