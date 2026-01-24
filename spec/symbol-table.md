# Symbol Specification

This document specifies the symbol system for OpcodeOracle.

See [state-file.md](state-file.md) for the JSON file format.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

Symbols provide human-readable names for memory addresses. They are central to reverse engineering, replacing raw addresses with meaningful identifiers in disassembly output.

## Type Definitions

### SymbolType

```go
type SymbolType string

const (
    SymbolSubroutine SymbolType = "subroutine"
    SymbolLabel      SymbolType = "label"
    SymbolByte       SymbolType = "byte"
    SymbolWord       SymbolType = "word"
    SymbolEntry      SymbolType = "entry"
)
```

| Type         | Description                            |
|--------------|----------------------------------------|
| `subroutine` | Subroutine entry point (target of JSR) |
| `label`      | Code label (target of JMP/branch)      |
| `byte`       | Single byte data (1 byte)              |
| `word`       | Word data (2 bytes, little-endian)     |
| `entry`      | Program entry point                    |

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

| Source   | Description                               |
|----------|-------------------------------------------|
| `user`   | Manually defined by the user              |
| `auto`   | Auto-generated during disassembly         |
| `c64rom` | Imported from C64 ROM/Kernal symbol table |
| `import` | Imported from external symbol file        |

## Struct Definition

### Symbol

```go
type Symbol struct {
    Name   string
    Type   SymbolType
    Source SymbolSource
}
```

| Field    | Type         | Description                    |
|----------|--------------|--------------------------------|
| `Name`   | string       | Symbol name (valid identifier) |
| `Type`   | SymbolType   | Classification of the symbol   |
| `Source` | SymbolSource | Origin of the symbol           |

## SymbolTable Interface

```go
type SymbolTable interface {
    At(addr uint16) []Symbol
    Add(addr uint16, sym Symbol)
    Remove(addr uint16, name string)
}
```

| Method                                         | Description                                      |
|------------------------------------------------|--------------------------------------------------|
| `At(addr uint16) []Symbol`                     | Get all symbols at address (empty slice if none) |
| `Add(addr uint16, sym Symbol)`                 | Add symbol at address                            |
| `Remove(addr uint16, name string)`             | Remove symbol by name at address                 |

## Symbol Naming Rules

Valid symbol names must:
- Start with a letter or underscore
- Contain only alphanumeric characters and underscores
- Not be a reserved assembler keyword

Auto-generated symbol naming conventions:
- Labels: `L_XXXX` (e.g., `L_0810`)
- Subroutines: `SUB_XXXX` (e.g., `SUB_1000`)
- Data bytes: `DAT_XXXX`
- Data words: `WORD_XXXX`

## Multiple Symbols Per Address

Multiple symbols can exist at the same address for:
- User-defined name alongside auto-generated name
- Multiple sources (e.g., C64 ROM symbol and user alias)
- Alternative interpretations during analysis

All symbols at an address are displayed in output.
