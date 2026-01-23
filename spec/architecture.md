# Architecture Specification

This document specifies the core Go type definitions used throughout OpcodeOracle.

See [state.md](state.md) for the JSON file format specification.

## Directory Structure

```
opcodeoracle/
├── cmd/
│   └── opcodeoracle/       # CLI entry point
│       └── main.go
├── internal/
│   ├── state/              # State types and file operations
│   ├── symbols/            # Symbol table implementation
│   ├── segments/           # Segment table implementation
│   ├── annotations/        # Annotation table implementation
│   ├── disasm/             # Disassembly engine
│   ├── loader/             # Binary file loaders
│   └── export/             # Assembly output formatter
├── spec/                   # Specification documents
├── testdata/               # Test binaries and fixtures
├── go.mod
└── README.md
```

| Directory                | Purpose                                      |
|--------------------------|----------------------------------------------|
| `cmd/opcodeoracle/`      | Main entry point, CLI argument parsing       |
| `internal/state/`        | State struct, JSON serialization, validation |
| `internal/symbols/`      | Symbol types and SymbolTable interface       |
| `internal/segments/`     | Segment types and SegmentTable interface     |
| `internal/annotations/`  | Annotation types and AnnotationTable interface |
| `internal/disasm/`       | Flow-following disassembler, opcode decoding |
| `internal/loader/`       | Binary file reading, PRG format handling     |
| `internal/export/`       | Assembly file generation, formatting         |
| `spec/`                  | Project specifications (this document)       |
| `testdata/`              | Test binaries and expected outputs           |

## Type Definitions

### SymbolType

```go
type SymbolType string

const (
    SymbolSubroutine SymbolType = "subroutine"
    SymbolLabel      SymbolType = "label"
    SymbolByte       SymbolType = "byte"
    SymbolWord       SymbolType = "word"
    SymbolConstant   SymbolType = "constant"
    SymbolEntry      SymbolType = "entry"
)
```

### SymbolSource

```go
type SymbolSource string

const (
    SourceUser   SymbolSource = "user"
    SourceAuto   SymbolSource = "auto"
    SourceC64ROM SymbolSource = "c64rom"
    SourceImport SymbolSource = "import"
)
```

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

### XRefType

```go
type XRefType string

const (
    XRefCall   XRefType = "call"   // JSR target
    XRefJump   XRefType = "jump"   // JMP target
    XRefBranch XRefType = "branch" // Conditional branch target
    XRefRead   XRefType = "read"   // Memory read
    XRefWrite  XRefType = "write"  // Memory write
)
```

## Struct Definitions

### Symbol

```go
type Symbol struct {
    Name   string
    Type   SymbolType
    Source SymbolSource
}
```

### Annotation

```go
type Annotation struct {
    Comment string
    Author  string
}
```

### Segment

```go
type Segment struct {
    Start uint16
    End   uint16
    Type  SegmentType
}
```

### XRef

```go
type XRef struct {
    From uint16
    To   uint16
    Type XRefType
}
```

### Metadata

```go
type Metadata struct {
    Created     time.Time
    Modified    time.Time
    SourceFile  string
    Description string
}
```

### Binary

```go
type Binary struct {
    Data     []byte
    Origin   uint16
    Size     int
    Checksum string
}
```
