# Annotation Specification

This document specifies the annotation system for OpcodeOracle.

See [state-file.md](state-file.md) for the JSON file format.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

Annotations provide comments and notes attached to memory addresses. They allow users to document their understanding of the code during reverse engineering.

## Struct Definition

### AnnotationType

```go
type AnnotationType int

const (
    AnnotationInline   AnnotationType = iota // Comment to the right of instruction
    AnnotationHeadline                       // Block comment above address
)
```

| Type                 | Description                                                  |
|----------------------|--------------------------------------------------------------|
| `AnnotationInline`   | Displayed to the right of disassembly (default)              |
| `AnnotationHeadline` | Displayed as a block comment above the address               |

### Annotation

```go
type Annotation struct {
    Type    AnnotationType
    Comment string
    Author  string
}
```

| Field     | Type             | Description                        |
|-----------|------------------|------------------------------------|
| `Type`    | AnnotationType   | How the annotation is displayed    |
| `Comment` | string           | The annotation text                |
| `Author`  | string           | Who created the annotation         |

## AnnotationTable Interface

```go
type AnnotationTable interface {
    At(addr uint16) []Annotation
    Add(addr uint16, typ AnnotationType, comment, author string)
    Remove(addr uint16, index int) error
    Clear(addr uint16)
}
```

| Method                                                        | Description                                        |
|---------------------------------------------------------------|----------------------------------------------------|
| `At(addr uint16) []Annotation`                                | Get all annotations at address                     |
| `Add(addr uint16, typ AnnotationType, comment, author string)`| Add annotation at address with display type        |
| `Remove(addr uint16, index int) error`                        | Remove annotation by index (error if out of range) |
| `Clear(addr uint16)`                                          | Remove all annotations at address                  |

## Rendering

See [disassembler.md](disassembler.md) for formatting details.

**Inline annotations** appear as `;` comments to the right of the instruction:
```
             LDA #$00          ; Initialize border
```

**Headline annotations** appear as block comments above the address, preceded by a blank line:
```
; --------------------------------------------------------
; Main loop - waits forever
; --------------------------------------------------------
L_0815:      JMP L_0815
```

## Multiple Annotations Per Address

Multiple annotations can exist at the same address for:
- Multiple comments describing different aspects
- Comments from different authors (user vs auto)
- Progressive documentation as understanding improves

All annotations at an address are displayed in output.
