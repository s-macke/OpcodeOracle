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
  "regions": [ ... ]
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

### Metadata Object

| Field         | Type   | Required  | Description                            |
|---------------|--------|-----------|----------------------------------------|
| `created`     | string | Yes       | ISO8601 timestamp of creation          |
| `modified`    | string | Yes       | ISO8601 timestamp of last modification |
| `sourceFile`  | string | No        | Original binary filename               |
| `description` | string | No        | Project description                    |

### Binary Object

| Field      | Type     | Required  | Description                            |
|------------|----------|-----------|----------------------------------------|
| `data`     | array    | Yes       | Binary data as array of byte values    |
| `origin`   | string   | Yes       | Load address in hex (e.g., `"0x0801"`) |

Example:
```json
"binary": {
  "data": [169, 0, 141, 32, 208, 76, 21, 8],
  "origin": "0x0801"
}
```

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
| `entry`     | Program entry point                      |

#### Symbol Sources

| Source    | Description                                |
|-----------|--------------------------------------------|
| `user`    | Manually defined by the user               |
| `auto`    | Auto-generated during disassembly          |
| `c64rom`  | Imported from C64 ROM/Kernal symbol table  |
| `import`  | Imported from external symbol file         |

### Annotations Object

Maps addresses to arrays of comments (multiple annotations per address supported):

```json
"annotations": {
  "0x0810": [
    {"type": "inline", "comment": "Initialize screen color", "author": "user"},
    {"type": "inline", "comment": "Set to black", "author": "user"}
  ],
  "0x0815": [
    {"type": "headline", "comment": "Main loop - runs every frame", "author": "user"},
    {"type": "inline", "comment": "Infinite loop", "author": "auto"}
  ]
}
```

#### Annotation Definition

| Field     | Type   | Required | Description                          |
|-----------|--------|----------|--------------------------------------|
| `type`    | string | Yes      | Display type: `inline` or `headline` |
| `comment` | string | Yes      | The annotation text                  |
| `author`  | string | Yes      | Who created the annotation           |

#### Annotation Types

| Type       | Description                                    |
|------------|------------------------------------------------|
| `inline`   | Displayed to the right of disassembly (default)|
| `headline` | Displayed as block comment above the address   |

### Regions Array

Defines memory region classifications. Regions must cover the entire 64KB address space (0x0000-0xFFFF) without gaps or overlaps. Adjacent regions of the same type should be merged.

```json
"regions": [
  {
    "start": "0x0000",
    "end": "0x0800",
    "type": "data"
  },
  {
    "start": "0x0801",
    "end": "0x0FFF",
    "type": "code"
  },
  {
    "start": "0x1000",
    "end": "0xFFFF",
    "type": "data"
  }
]
```

**Note:** If regions are omitted or empty, the loader will default to a single data region covering 0x0000-0xFFFF.

| Type       | Description             |
|------------|-------------------------|
| `code`     | Executable instructions |
| `data`     | Generic data            |

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
    "data": [169, 0, 141, 32, 208, 76, 21, 8],
    "origin": "0x0801"
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
    "data": [169, 0, 141, 32, 208, 76, 21, 8],
    "origin": "0x0801"
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
      {"type": "inline", "comment": "Program entry - initialization", "author": "user"}
    ],
    "0x1000": [
      {"type": "headline", "comment": "Main game loop", "author": "user"}
    ]
  },
  "regions": [
    {"start": "0x0000", "end": "0x0800", "type": "data"},
    {"start": "0x0801", "end": "0x17FF", "type": "code"},
    {"start": "0x1800", "end": "0xFFFF", "type": "data"}
  ]
}
```

## Versioning

The `version` field follows semantic versioning:
- **Major**: Breaking changes to required fields
- **Minor**: New optional fields added
- **Patch**: Documentation or validation changes

Current version: `1.0`
