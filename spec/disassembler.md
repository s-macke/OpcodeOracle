# MOS6502 Disassembler Specification

This document specifies the disassembler interface and instruction formatting for OpcodeOracle.

## Overview

The disassembler decodes MOS6502 machine code and outputs formatted assembly text. It:

1. Decodes opcodes and operands for a given address range
2. Formats code regions as assembly instructions
3. Formats data regions as `.BYTE`/`.WORD` directives
4. Resolves symbols and includes annotations as comments

## Disassembler Interface

```go
type Disassembler interface {
    // Disassemble formats the address range [start, end) as assembly text
    Disassemble(start, end uint16) (string, error)
}
```

### Constructor

```go
// NewDisassembler creates a disassembler that reads from the given state
func NewDisassembler(state State) Disassembler
```

### Behavior

- Decodes and formats instructions sequentially from `start` until reaching `end`
- Returns formatted assembly text ready for export
- Includes labels, mnemonics, operands, and comments as specified in Instruction Formatting
- Data regions are formatted as `.BYTE` directives as specified in Data Section Format

### Error Conditions

| Condition                          | Error                  |
|------------------------------------|------------------------|
| Start or end outside binary bounds | `ErrAddressOutOfRange` |
| Invalid/illegal opcode encountered | `ErrIllegalOpcode`     |

## Addressing Modes

```go
type AddressingMode int

const (
    AddrImplied AddressingMode = iota
    AddrAccumulator
    AddrImmediate
    AddrZeroPage
    AddrZeroPageX
    AddrZeroPageY
    AddrAbsolute
    AddrAbsoluteX
    AddrAbsoluteY
    AddrIndirect
    AddrIndexedIndirect  // (zp,X)
    AddrIndirectIndexed  // (zp),Y
    AddrRelative
)
```

### Operand Formatting

| Addressing Mode  | Format    | Example       | Size |
|------------------|-----------|---------------|------|
| Implied          | (none)    | `RTS`         | 1    |
| Accumulator      | `A`       | `ASL A`       | 1    |
| Immediate        | `#$XX`    | `LDA #$00`    | 2    |
| Zero Page        | `$XX`     | `LDA $10`     | 2    |
| Zero Page,X      | `$XX,X`   | `LDA $10,X`   | 2    |
| Zero Page,Y      | `$XX,Y`   | `LDX $10,Y`   | 2    |
| Absolute         | `$XXXX`   | `JMP $1000`   | 3    |
| Absolute,X       | `$XXXX,X` | `LDA $1000,X` | 3    |
| Absolute,Y       | `$XXXX,Y` | `LDA $1000,Y` | 3    |
| Indirect         | `($XXXX)` | `JMP ($FFFC)` | 3    |
| Indexed Indirect | `($XX,X)` | `LDA ($10,X)` | 2    |
| Indirect Indexed | `($XX),Y` | `LDA ($10),Y` | 2    |
| Relative         | `L_XXXX`  | `BNE L_0820`  | 2    |

Note: Relative addressing uses a label when the target is known; otherwise displays the raw offset.

## Instruction Formatting

Code instructions are formatted as:

```
[LABEL]      MNEMONIC OPERAND  [; COMMENT]
```

- **Label**: Shown if a symbol exists at the instruction address; uses the symbol name (e.g., `SUB_XXXX:` for subroutines, `L_XXXX:` for labels). If no symbol exists but the address is a jump/branch target, an auto label `L_XXXX:` is shown.
- **No address column**: The label contains the address information
- **No hex bytes**: Not shown for code instructions (only in data sections)
- **Comment**: Annotations from state file are appended as comments
- **Multiple comments**: Additional annotations appear on new lines, aligned with the first comment

Example:
```
             LDA #$00          ; Initialize border
                               ; Set to black
             STA $D020
L_0815:      JMP L_0815        ; Main loop
```

### Field Specifications

| Field    | Width    | Description                               |
|----------|----------|-------------------------------------------|
| Label    | 12 chars | Symbol name or `L_XXXX:` if auto label, empty otherwise |
| Mnemonic | 3 chars  | Instruction mnemonic                      |
| Operand  | variable | Formatted according to addressing mode    |
| Comment  | variable | Annotation from state file (if present)   |

## Data Section Format

Bytes not identified as code are output as data in rows of 16 bytes:

```
$0900    .BYTE $48,$45,$4C,$4C,$4F,$20,$57,$4F,$52,$4C,$44,$00,$00,$00,$00,$00  ; "HELLO WORLD....."
```

Format:
```
ADDRESS  .BYTE HEX_VALUES  ; ASCII_REPRESENTATION
```

- Non-printable characters (< 0x20 or > 0x7E) are shown as `.`
- Rows are aligned to 16-byte boundaries when possible
- Shorter rows are allowed at the end of data sections

### Symbols, Labels, and Annotations in Data

If a symbol, label, or annotation exists within a data section, the data stream is broken at that address. Labeled data is displayed as either `.BYTE` (1 byte) or `.WORD` (2 bytes) with a blank line before:

```
$0900    .BYTE $48,$45,$4C,$4C,$4F,$00                                          ; "HELLO."

DAT_0906: .BYTE $57                                                             ; Flag byte

WORD_0907: .WORD $1000                                                          ; Jump target

$0909    .BYTE $00,$00,$00,$00
```

- Blank line before any label or annotation
- Symbols/labels appear in the label column (e.g., `DAT_0906:`)
- Data type is `.BYTE` (1 byte) or `.WORD` (2 bytes) based on symbol type
- Annotations appear as comments
- Data before and after continues in standard 16-byte rows

## Usage Example

```go
state, _ := LoadState("game.orc")
disasm := NewDisassembler(state)

// Disassemble a range
output, err := disasm.Disassemble(0x0810, 0x0900)
if err != nil {
    return err
}

fmt.Print(output)
// Output:
//              LDA #$00          ; Initialize border
//              STA $D020
// L_0815:      JMP L_0815        ; Main loop
```
