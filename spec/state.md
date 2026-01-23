# Reverse Engineering State File Specification

This document specifies the JSON format for persisting OpcodeOracle reverse engineering sessions.

## Overview

The state file (`.orc` extension - OpcodeOracle Project) contains all data needed to save and restore a reverse engineering session, including the binary data, analysis results, and user annotations.

## File Format

State files are JSON documents with the following structure:

```json
{
  "version": "1.0",
  "metadata": { ... },
  "binary": { ... },
  "entryPoints": [ ... ],
  "symbols": { ... },
  "annotations": { ... },
  "regions": [ ... ],
  "xrefs": { ... }
}
```

## Schema Definition

### Root Object

| Field         | Type   | Required  | Description                             |
|---------------|--------|-----------|-----------------------------------------|
| `version`     | string | Yes       | Schema version (semver format)          |
| `metadata`    | object | Yes       | Project metadata                        |
| `binary`      | object | Yes       | Binary data and load parameters         |
| `entryPoints` | array  | Yes       | List of entry point addresses           |
| `symbols`     | object | No        | User-defined and auto-generated symbols |
| `annotations` | object | No        | Comments and notes                      |
| `regions`     | array  | No        | Memory region classifications           |
| `xrefs`       | object | No        | Cross-reference data                    |

### Metadata Object

| Field         | Type   | Required  | Description                            |
|---------------|--------|-----------|----------------------------------------|
| `created`     | string | Yes       | ISO8601 timestamp of creation          |
| `modified`    | string | Yes       | ISO8601 timestamp of last modification |
| `sourceFile`  | string | No        | Original binary filename               |
| `description` | string | No        | Project description                    |

### Binary Object

| Field      | Type    | Required  | Description                            |
|------------|---------|-----------|----------------------------------------|
| `data`     | string  | Yes       | Base64-encoded binary data             |
| `origin`   | string  | Yes       | Load address in hex (e.g., `"0x0801"`) |
| `size`     | integer | Yes       | Size in bytes (for validation)         |
| `checksum` | string  | No        | SHA256 hash of original binary         |

### Entry Points Array

Array of hex address strings representing known entry points:

```json
"entryPoints": ["0x0801", "0x1000"]
```

### Symbols Object

Maps addresses to arrays of symbol definitions. Multiple symbols per address are supported for aliases, different sources, or alternative interpretations.

```json
"symbols": {
  "0x0801": [
    {
      "name": "main",
      "type": "entry",
      "source": "user"
    },
    {
      "name": "L_0801",
      "type": "entry",
      "source": "auto"
    }
  ],
  "0xD020": [
    {
      "name": "BORDER_COLOR",
      "type": "byte",
      "source": "c64rom"
    },
    {
      "name": "VIC_BORDER",
      "type": "byte",
      "source": "user"
    }
  ]
}
```

#### Symbol Definition

| Field    | Type   | Required | Description                      |
|----------|--------|----------|----------------------------------|
| `name`   | string | Yes      | Symbol name (valid identifier)   |
| `type`   | string | Yes      | Symbol type (see below)          |
| `source` | string | Yes      | Origin of the symbol (see below) |

#### Symbol Types

| Type        | Description                              |
|-------------|------------------------------------------|
| `subroutine`| Subroutine entry point (target of JSR)   |
| `label`     | Code label (target of JMP/branch)        |
| `byte`      | Single byte data (1 byte)                |
| `word`      | Word data (2 bytes, little-endian)       |
| `constant`  | Named constant value                     |
| `entry`     | Program entry point                      |

#### Symbol Sources

| Source    | Description                                |
|-----------|--------------------------------------------|
| `user`    | Manually defined by the user               |
| `auto`    | Auto-generated during disassembly          |
| `c64rom`  | Imported from C64 ROM/Kernal symbol table  |
| `import`  | Imported from external symbol file         |

### Annotations Object (Future)

Maps addresses to arrays of comments (multiple annotations per address supported):

```json
"annotations": {
  "0x0810": [
    {"comment": "Initialize screen color", "author": "user"},
    {"comment": "Set to black", "author": "user"}
  ],
  "0x0815": [
    {"comment": "Main loop", "author": "auto"}
  ]
}
```

### Regions Array (Future)

Defines memory region classifications:

```json
"regions": [
  {
    "start": "0x0801",
    "end": "0x0900",
    "type": "code"
  },
  {
    "start": "0x0900",
    "end": "0x0A00",
    "type": "data",
    "format": "string"
  }
]
```

| Type       | Description             |
|------------|-------------------------|
| `code`     | Executable instructions |
| `data`     | Generic data            |
| `string`   | ASCII/PETSCII text      |
| `table`    | Jump/data table         |
| `graphics` | Sprite/bitmap data      |

### Cross-References Object (Future)

Maps addresses to their references:

```json
"xrefs": {
  "0x1000": {
    "from": ["0x0815", "0x0830"],
    "type": "call"
  }
}
```

## Example: Minimal State File

```json
{
  "version": "1.0",
  "metadata": {
    "created": "2025-01-22T10:30:00Z",
    "modified": "2025-01-22T10:30:00Z",
    "sourceFile": "game.prg"
  },
  "binary": {
    "data": "qQCNINDMFQg=",
    "origin": "0x0801",
    "size": 9
  },
  "entryPoints": ["0x0801"]
}
```

## Example: Full State File

```json
{
  "version": "1.0",
  "metadata": {
    "created": "2025-01-22T10:30:00Z",
    "modified": "2025-01-22T14:45:00Z",
    "sourceFile": "game.prg",
    "description": "Space shooter game analysis"
  },
  "binary": {
    "data": "qQCNINDMFQg=",
    "origin": "0x0801",
    "size": 9,
    "checksum": "a1b2c3d4..."
  },
  "entryPoints": ["0x0801", "0x1000"],
  "symbols": {
    "0x0801": [
      {"name": "start", "type": "entry", "source": "user"}
    ],
    "0x1000": [
      {"name": "game_loop", "type": "subroutine", "source": "user"},
      {"name": "SUB_1000", "type": "subroutine", "source": "auto"}
    ],
    "0xD020": [
      {"name": "BORDER", "type": "byte", "source": "c64rom"}
    ]
  },
  "annotations": {
    "0x0801": [
      {"comment": "Program entry - initialization"}
    ],
    "0x1000": [
      {"comment": "Main game loop"}
    ]
  },
  "regions": [
    {"start": "0x0801", "end": "0x0FFF", "type": "code"},
    {"start": "0x1800", "end": "0x1900", "type": "string"}
  ]
}
```

## Versioning

The `version` field follows semantic versioning:
- **Major**: Breaking changes to required fields
- **Minor**: New optional fields added
- **Patch**: Documentation or validation changes

Current version: `1.0`
