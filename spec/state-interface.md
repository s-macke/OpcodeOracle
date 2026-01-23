# State Interface Specification

This document specifies the Go interface for interacting with OpcodeOracle state files.

See [state.md](state.md) for the JSON file format specification.
See [architecture.md](architecture.md) for type and struct definitions.

## Overview

The state interface provides programmatic access to load, save, query, and modify reverse engineering session data. All CLI commands (`new`, `info`, `export`) use this interface to interact with state files.

## State Interface

The primary interface for working with state files.

```go
type State interface {
    // Persistence
    Load(path string) error
    Save(path string) error

    // Metadata
    Metadata() Metadata
    SetDescription(desc string)

    // Binary data
    Binary() Binary
    ReadByte(addr uint16) (byte, error)
    ReadWord(addr uint16) (uint16, error)

    // Entry points
    EntryPoints() []uint16
    AddEntryPoint(addr uint16)
    RemoveEntryPoint(addr uint16)
    HasEntryPoint(addr uint16) bool

    // Sub-tables
    Symbols() SymbolTable
    Annotations() AnnotationTable
    Segments() SegmentTable
    XRefs() XRefTable
}
```

### State Methods

| Method                                  | Description                                |
|-----------------------------------------|--------------------------------------------|
| `Load(path string) error`               | Load state from JSON file                  |
| `Save(path string) error`               | Save state to JSON file                    |
| `Metadata() Metadata`                   | Get project metadata                       |
| `SetDescription(desc string)`           | Update project description                 |
| `Binary() Binary`                       | Get binary data and parameters             |
| `ReadByte(addr uint16) (byte, error)`   | Read byte at virtual address               |
| `ReadWord(addr uint16) (uint16, error)` | Read little-endian word at virtual address |
| `EntryPoints() []uint16`                | Get list of entry point addresses          |
| `AddEntryPoint(addr uint16)`            | Add entry point (no-op if exists)          |
| `RemoveEntryPoint(addr uint16)`         | Remove entry point (no-op if not exists)   |
| `HasEntryPoint(addr uint16) bool`       | Check if address is an entry point         |
| `Symbols() SymbolTable`                 | Get symbol table interface                 |
| `Annotations() AnnotationTable`         | Get annotation table interface             |
| `Segments() SegmentTable`               | Get segment table interface                |
| `XRefs() XRefTable`                     | Get cross-reference table interface        |

## SymbolTable Interface

Manages symbols (labels, subroutine names, data labels) at memory addresses.

```go
type SymbolTable interface {
    At(addr uint16) []Symbol
    Add(addr uint16, sym Symbol)
    Remove(addr uint16, name string)
    All() map[uint16][]Symbol
    ByType(t SymbolType) map[uint16][]Symbol
    BySource(s SymbolSource) map[uint16][]Symbol
}
```

### SymbolTable Methods

| Method                                         | Description                                      |
|------------------------------------------------|--------------------------------------------------|
| `At(addr uint16) []Symbol`                     | Get all symbols at address (empty slice if none) |
| `Add(addr uint16, sym Symbol)`                 | Add symbol at address                            |
| `Remove(addr uint16, name string)`             | Remove symbol by name at address                 |
| `All() map[uint16][]Symbol`                    | Get all symbols as address map                   |
| `ByType(t SymbolType) map[uint16][]Symbol`     | Filter symbols by type                           |
| `BySource(s SymbolSource) map[uint16][]Symbol` | Filter symbols by source                         |

## XRefTable Interface

Manages cross-reference data (which addresses reference which).

```go
type XRefTable interface {
    To(addr uint16) []XRef
    From(addr uint16) []XRef
    Add(from, to uint16, refType XRefType)
    Remove(from, to uint16)
    All() []XRef
}
```
[
### XRefTable Methods]()

| Method                                   | Description                     |
|------------------------------------------|---------------------------------|
| `To(addr uint16) []XRef`                 | Get all references to address   |
| `From(addr uint16) []XRef`               | Get all references from address |
| `Add(from, to uint16, refType XRefType)` | Add cross-reference             |
| `Remove(from, to uint16)`                | Remove cross-reference          |
| `All() []XRef`                           | Get all cross-references        |

## Constructor Functions

```go
// NewState creates an empty state with the given binary data
func NewState(data []byte, origin uint16, sourceFile string) State

// LoadState loads a state from a JSON file
func LoadState(path string) (State, error)
```

## Usage Examples

### Creating a New State

```go
// Read binary file
data, err := os.ReadFile("game.prg")
if err != nil {
    return err
}

// Create state
state := NewState(data, 0x0801, "game.prg")
state.AddEntryPoint(0x0810)

// Add auto-generated symbol
state.Symbols().Add(0x0810, Symbol{
    Name:   "L_0810",
    Type:   SymbolLabel,
    Source: SourceAuto,
})

// Save to file
err = state.Save("game.orc")
```

### Loading and Querying State

```go
state, err := LoadState("game.orc")
if err != nil {
    return err
}

// Get metadata
meta := state.Metadata()
fmt.Printf("Source: %s\n", meta.SourceFile)

// Iterate symbols
for addr, symbols := range state.Symbols().All() {
    for _, sym := range symbols {
        fmt.Printf("$%04X: %s (%s)\n", addr, sym.Name, sym.Type)
    }
}

// Check segments
segment := state.Segments().At(0x0900)
if segment != nil && segment.Type == SegmentData {
    fmt.Println("Address is in data section")
}
```

### Adding User Annotations

```go
state, _ := LoadState("game.orc")

// Add user symbol
state.Symbols().Add(0x1000, Symbol{
    Name:   "main_loop",
    Type:   SymbolSubroutine,
    Source: SourceUser,
})

// Add comment
state.Annotations().Add(0x1000, "Main game loop - runs every frame", "user")

// Save changes
state.Save("game.orc")
```

## Error Handling

| Error Condition | Behavior |
|-----------------|----------|
| File not found (Load) | Return error |
| Invalid JSON (Load) | Return error |
| Version mismatch (Load) | Return error |
| Write permission denied (Save) | Return error |
| Address outside binary (ReadByte/ReadWord) | Return error |
| Segment overlap (Add segment) | Return error |
| Index out of range (Remove annotation) | Return error |
