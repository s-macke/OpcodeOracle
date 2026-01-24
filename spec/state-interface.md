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

## State Interface

The primary interface for working with state files.

```go
type State struct {
    // Persistence
    Load(path string) error
    Save(path string) error

    // Metadata
    Metadata() Metadata
    SetDescription(desc string)

    // Binary data (accessed as field)
    Binary Binary

    // Tables (accessed as fields)
    Symbols     SymbolTable
    Annotations AnnotationTable
    Regions     RegionTable
    XRefs       XRefTable
    EntryPoints EntryPointTable
}
```

### State Methods

| Method                       | Description                    |
|------------------------------|--------------------------------|
| `Load(path string) error`    | Load state from JSON file      |
| `Save(path string) error`    | Save state to JSON file        |
| `Metadata() Metadata`        | Get project metadata           |
| `SetDescription(desc string)`| Update project description     |

### State Fields

| Field         | Type              | Description                    |
|---------------|-------------------|--------------------------------|
| `Binary`      | `Binary`          | Binary data with read methods  |
| `Symbols`     | `SymbolTable`     | Symbol table (labels, names)   |
| `Annotations` | `AnnotationTable` | Comments and notes             |
| `Regions`     | `RegionTable`     | Code/data region boundaries    |
| `XRefs`       | `XRefTable`       | Cross-references               |
| `EntryPoints` | `EntryPointTable` | Code entry points              |

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
state.EntryPoints.Add(0x0810)

// Add auto-generated symbol
state.Symbols.Add(0x0810, Symbol{
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

// Check region type at address
region := state.Regions.At(0x0900)
if region.Type == RegionData {
    fmt.Println("Address is in data section")
}

// Mark address range as code
state.Regions.Set(0x0900, 0x09FF, RegionCode)
```

### Adding User Annotations

```go
state, _ := LoadState("game.orc")

// Add user symbol
state.Symbols.Add(0x1000, Symbol{
    Name:   "main_loop",
    Type:   SymbolSubroutine,
    Source: SourceUser,
})

// Add comment
state.Annotations.Add(0x1000, AnnotationInline, "Main game loop - runs every frame", "user")

// Save changes
state.Save("game.orc")
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
