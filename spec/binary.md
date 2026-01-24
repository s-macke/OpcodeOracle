# Binary Specification

This document specifies the binary data interface for OpcodeOracle.

See [state-file.md](state-file.md) for the JSON file format.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

The Binary struct holds the raw binary data and provides methods to read bytes and words at virtual addresses. It handles address translation from the virtual address space (based on the origin/load address) to offsets within the binary data.

## Struct Definition

```go
type Binary struct {
    Data   []byte // Raw binary data
    Origin uint16 // Load address
}

func (b *Binary) ReadByte(addr uint16) (byte, error)
func (b *Binary) ReadWord(addr uint16) (uint16, error)
```

### Fields

| Field    | Type     | Description                      |
|----------|----------|----------------------------------|
| `Data`   | `[]byte` | Raw binary data as byte array    |
| `Origin` | `uint16` | Load address of binary in memory |

### Methods

| Method                                  | Description                                |
|-----------------------------------------|--------------------------------------------|
| `ReadByte(addr uint16) (byte, error)`   | Read byte at virtual address               |
| `ReadWord(addr uint16) (uint16, error)` | Read little-endian word at virtual address |

## Address Translation

Virtual addresses are translated to binary offsets:

```
offset = addr - Origin
```

For example, with `Origin = 0x0801`:
- Address `0x0801` reads offset `0`
- Address `0x0810` reads offset `15`

## Error Conditions

| Condition                | Error                  |
|--------------------------|------------------------|
| Address below Origin     | `ErrAddressOutOfRange` |
| Address beyond binary end| `ErrAddressOutOfRange` |

## Usage Example

```go
state, _ := LoadState("game.orc")

// Read a byte
b, err := state.Binary.ReadByte(0x0810)
if err != nil {
    return err
}

// Read a little-endian word
w, err := state.Binary.ReadWord(0x0820)
if err != nil {
    return err
}
fmt.Printf("Word at $0820: $%04X\n", w)
```