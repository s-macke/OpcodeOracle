# Annotation Specification

This document specifies the annotation system for OpcodeOracle.

See [state-file.md](state-file.md) for the JSON file format.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

Annotations provide comments and notes attached to memory addresses. They allow users to document their understanding of the code during reverse engineering.

## Struct Definition

### Annotation

```go
type Annotation struct {
    Comment string
    Author  string
}
```

| Field     | Type   | Description                        |
|-----------|--------|------------------------------------|
| `Comment` | string | The annotation text                |
| `Author`  | string | Who created the annotation         |

## AnnotationTable Interface

```go
type AnnotationTable interface {
    At(addr uint16) []Annotation
    Add(addr uint16, comment, author string)
    Remove(addr uint16, index int) error
    Clear(addr uint16)
    All() map[uint16][]Annotation
}
```

| Method                                     | Description                                        |
|--------------------------------------------|----------------------------------------------------|
| `At(addr uint16) []Annotation`             | Get all annotations at address                     |
| `Add(addr uint16, comment, author string)` | Add annotation at address                          |
| `Remove(addr uint16, index int) error`     | Remove annotation by index (error if out of range) |
| `Clear(addr uint16)`                       | Remove all annotations at address                  |
| `All() map[uint16][]Annotation`            | Get all annotations as address map                 |

## Multiple Annotations Per Address

Multiple annotations can exist at the same address for:
- Multiple comments describing different aspects
- Comments from different authors (user vs auto)
- Progressive documentation as understanding improves

All annotations at an address are displayed in output.
