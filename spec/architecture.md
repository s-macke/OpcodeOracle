# Architecture Specification

This document specifies the core Go type definitions used throughout OpcodeOracle.

See [state-file.md](state-file.md) for the JSON file format specification.

## Program Flow

```mermaid
flowchart TB
    CLI[CLI Interface]

    subgraph new [new command]
        BIN[Binary File]
        LOADER[Binary Loader]
        DISASM[Disassembly Engine]
    end

    STATE[State File .orc]

    subgraph export [export command]
        FORMATTER[Output Formatter]
        MAIN[Main .asm File]
        SEGMENTS[Segment Files]
    end

    CLI --> new
    CLI --> export
    BIN --> LOADER
    LOADER --> DISASM
    DISASM --> STATE
    STATE --> FORMATTER
    FORMATTER --> MAIN
    FORMATTER --> SEGMENTS
```

## Directory Structure

```
opcodeoracle/
├── src/                        # Go module root
│   ├── go.mod
│   ├── cmd/
│   │   └── opcodeoracle/       # CLI entry point
│   │       └── main.go
│   └── internal/
│       ├── state/              # State and Metadata types
│       ├── stateio/            # State file I/O (Load/Save)
│       ├── binary/             # Binary data and address translation
│       ├── symbols/            # Symbol table implementation
│       ├── regions/            # Region table implementation
│       ├── annotations/        # Annotation table implementation
│       ├── xrefs/              # Cross-reference table implementation
│       ├── disasm/             # Disassembly engine
│       ├── loader/             # Binary file loaders
│       └── export/             # Assembly output formatter
├── spec/                       # Specification documents
├── testdata/                   # Test binaries and fixtures
└── README.md
```

| Directory                    | Purpose                                        |
|------------------------------|------------------------------------------------|
| `src/cmd/opcodeoracle/`      | Main entry point, CLI argument parsing         |
| `src/internal/state/`        | State and Metadata structs, NewState()         |
| `src/internal/stateio/`      | State file I/O: Load(), Save(), JSON encoding  |
| `src/internal/binary/`       | Binary struct and address translation          |
| `src/internal/symbols/`      | Symbol types and SymbolTable interface         |
| `src/internal/regions/`      | Region types and RegionTable interface         |
| `src/internal/annotations/`  | Annotation types and AnnotationTable interface |
| `src/internal/xrefs/`        | XRef types and XRefTable interface             |
| `src/internal/disasm/`       | Flow-following disassembler, opcode decoding   |
| `src/internal/loader/`       | Binary file reading, PRG format handling       |
| `src/internal/export/`       | Assembly file generation, formatting           |
| `spec/`                      | Project specifications (this document)         |
| `testdata/`                  | Test binaries and expected outputs             |

## Struct Definitions

### State

This struct maps directly to the JSON structure in state files (`.orc`). See [state-file.md](state-file.md) for the JSON serialization format.

```go
type State struct {
    Version     string
    Metadata    Metadata
    Binary      Binary
    EntryPoints []uint16
    ExtraCodeAddresses []uint16
    Symbols     map[uint16][]Symbol
    Annotations map[uint16][]Annotation
    Regions     []Region
}
```

| Field         | Type                       | Description                                    |
|---------------|----------------------------|------------------------------------------------|
| `Version`     | string                     | Schema version (semver format)                 |
| `Metadata`    | Metadata                   | Project metadata                               |
| `Binary`      | Binary                     | Binary data and load parameters                |
| `EntryPoints` | []uint16                   | Entry point addresses                          |
| `ExtraCodeAddresses` | []uint16            | Additional code seed addresses without entry symbols |
| `Symbols`     | map[uint16][]Symbol        | See [symbol-table.md](symbol-table.md)         |
| `Annotations` | map[uint16][]Annotation    | See [annotation-table.md](annotation-table.md) |
| `Regions`     | []Region                   | See [regions-table.md](regions-table.md)       |

### Metadata

```go
type Metadata struct {
    Created     time.Time
    Modified    time.Time
    SourceFile  string
    Description string
}
```
