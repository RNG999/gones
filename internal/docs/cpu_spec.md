# NES CPU (6502) Specifications

## Overview
The NES uses a custom Ricoh 2A03 CPU, which is based on the MOS 6502 processor with the decimal mode disabled. The CPU runs at 1.789773 MHz (NTSC) or 1.662607 MHz (PAL).

## Registers
- **A** (Accumulator): 8-bit
- **X** (Index X): 8-bit
- **Y** (Index Y): 8-bit
- **PC** (Program Counter): 16-bit
- **SP** (Stack Pointer): 8-bit, hardwired to page $01 ($0100-$01FF)
- **P** (Processor Status): 8-bit
  - Bit 7: N (Negative)
  - Bit 6: V (Overflow)
  - Bit 5: - (Always 1)
  - Bit 4: B (Break Command)
  - Bit 3: D (Decimal Mode) - Disabled on NES
  - Bit 2: I (Interrupt Disable)
  - Bit 1: Z (Zero)
  - Bit 0: C (Carry)

## Memory Map
```
$0000-$07FF: 2KB internal RAM
$0800-$0FFF: Mirror of $0000-$07FF
$1000-$17FF: Mirror of $0000-$07FF
$1800-$1FFF: Mirror of $0000-$07FF
$2000-$2007: PPU registers
$2008-$3FFF: Mirrors of $2000-$2007 (every 8 bytes)
$4000-$4017: APU and I/O registers
$4018-$401F: APU and I/O functionality (normally disabled)
$4020-$FFFF: Cartridge space (PRG-ROM, PRG-RAM, mapper registers)
```

## Interrupts
### Reset
- Vector: $FFFC-$FFFD
- Triggered on power-on or reset button
- Clears decimal mode, sets interrupt disable flag

### NMI (Non-Maskable Interrupt)
- Vector: $FFFA-$FFFB
- Triggered by PPU at start of V-Blank (if enabled)
- Cannot be disabled
- 7 cycles to execute

### IRQ (Interrupt Request)
- Vector: $FFFE-$FFFF
- Can be triggered by APU DMC or cartridge
- Disabled when I flag is set
- 7 cycles to execute

## Addressing Modes

### Immediate
- Format: `OPC #$BB`
- Bytes: 2
- Cycles: 2

### Zero Page
- Format: `OPC $LL`
- Bytes: 2
- Cycles: 3

### Zero Page,X
- Format: `OPC $LL,X`
- Bytes: 2
- Cycles: 4

### Zero Page,Y
- Format: `OPC $LL,Y`
- Bytes: 2
- Cycles: 4

### Absolute
- Format: `OPC $LLHH`
- Bytes: 3
- Cycles: 4

### Absolute,X
- Format: `OPC $LLHH,X`
- Bytes: 3
- Cycles: 4 (+1 if page crossed)

### Absolute,Y
- Format: `OPC $LLHH,Y`
- Bytes: 3
- Cycles: 4 (+1 if page crossed)

### Indirect
- Format: `JMP ($LLHH)`
- Bytes: 3
- Cycles: 5
- Note: Has bug where ($xxFF) reads high byte from $xx00

### Indexed Indirect (Indirect,X)
- Format: `OPC ($LL,X)`
- Bytes: 2
- Cycles: 6

### Indirect Indexed (Indirect),Y
- Format: `OPC ($LL),Y`
- Bytes: 2
- Cycles: 5 (+1 if page crossed)

### Relative
- Format: `OPC $BB`
- Bytes: 2
- Cycles: 2 (+1 if branch taken, +1 if page crossed)

### Accumulator
- Format: `OPC A`
- Bytes: 1
- Cycles: 2

### Implied
- Format: `OPC`
- Bytes: 1
- Cycles: 2-7 (varies by instruction)

## Instruction Set

### Official Instructions

#### Load/Store Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| A9 | LDA | Immediate | 2 | 2 | N,Z | Load Accumulator |
| A5 | LDA | Zero Page | 2 | 3 | N,Z | Load Accumulator |
| B5 | LDA | Zero Page,X | 2 | 4 | N,Z | Load Accumulator |
| AD | LDA | Absolute | 3 | 4 | N,Z | Load Accumulator |
| BD | LDA | Absolute,X | 3 | 4* | N,Z | Load Accumulator |
| B9 | LDA | Absolute,Y | 3 | 4* | N,Z | Load Accumulator |
| A1 | LDA | Indirect,X | 2 | 6 | N,Z | Load Accumulator |
| B1 | LDA | Indirect,Y | 2 | 5* | N,Z | Load Accumulator |
| A2 | LDX | Immediate | 2 | 2 | N,Z | Load X Register |
| A6 | LDX | Zero Page | 2 | 3 | N,Z | Load X Register |
| B6 | LDX | Zero Page,Y | 2 | 4 | N,Z | Load X Register |
| AE | LDX | Absolute | 3 | 4 | N,Z | Load X Register |
| BE | LDX | Absolute,Y | 3 | 4* | N,Z | Load X Register |
| A0 | LDY | Immediate | 2 | 2 | N,Z | Load Y Register |
| A4 | LDY | Zero Page | 2 | 3 | N,Z | Load Y Register |
| B4 | LDY | Zero Page,X | 2 | 4 | N,Z | Load Y Register |
| AC | LDY | Absolute | 3 | 4 | N,Z | Load Y Register |
| BC | LDY | Absolute,X | 3 | 4* | N,Z | Load Y Register |
| 85 | STA | Zero Page | 2 | 3 | - | Store Accumulator |
| 95 | STA | Zero Page,X | 2 | 4 | - | Store Accumulator |
| 8D | STA | Absolute | 3 | 4 | - | Store Accumulator |
| 9D | STA | Absolute,X | 3 | 5 | - | Store Accumulator |
| 99 | STA | Absolute,Y | 3 | 5 | - | Store Accumulator |
| 81 | STA | Indirect,X | 2 | 6 | - | Store Accumulator |
| 91 | STA | Indirect,Y | 2 | 6 | - | Store Accumulator |
| 86 | STX | Zero Page | 2 | 3 | - | Store X Register |
| 96 | STX | Zero Page,Y | 2 | 4 | - | Store X Register |
| 8E | STX | Absolute | 3 | 4 | - | Store X Register |
| 84 | STY | Zero Page | 2 | 3 | - | Store Y Register |
| 94 | STY | Zero Page,X | 2 | 4 | - | Store Y Register |
| 8C | STY | Absolute | 3 | 4 | - | Store Y Register |

#### Transfer Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| AA | TAX | Implied | 1 | 2 | N,Z | Transfer A to X |
| A8 | TAY | Implied | 1 | 2 | N,Z | Transfer A to Y |
| BA | TSX | Implied | 1 | 2 | N,Z | Transfer Stack Pointer to X |
| 8A | TXA | Implied | 1 | 2 | N,Z | Transfer X to A |
| 9A | TXS | Implied | 1 | 2 | - | Transfer X to Stack Pointer |
| 98 | TYA | Implied | 1 | 2 | N,Z | Transfer Y to A |

#### Stack Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| 48 | PHA | Implied | 1 | 3 | - | Push Accumulator |
| 08 | PHP | Implied | 1 | 3 | - | Push Processor Status |
| 68 | PLA | Implied | 1 | 4 | N,Z | Pull Accumulator |
| 28 | PLP | Implied | 1 | 4 | All | Pull Processor Status |

#### Arithmetic Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| 69 | ADC | Immediate | 2 | 2 | N,V,Z,C | Add with Carry |
| 65 | ADC | Zero Page | 2 | 3 | N,V,Z,C | Add with Carry |
| 75 | ADC | Zero Page,X | 2 | 4 | N,V,Z,C | Add with Carry |
| 6D | ADC | Absolute | 3 | 4 | N,V,Z,C | Add with Carry |
| 7D | ADC | Absolute,X | 3 | 4* | N,V,Z,C | Add with Carry |
| 79 | ADC | Absolute,Y | 3 | 4* | N,V,Z,C | Add with Carry |
| 61 | ADC | Indirect,X | 2 | 6 | N,V,Z,C | Add with Carry |
| 71 | ADC | Indirect,Y | 2 | 5* | N,V,Z,C | Add with Carry |
| E9 | SBC | Immediate | 2 | 2 | N,V,Z,C | Subtract with Carry |
| E5 | SBC | Zero Page | 2 | 3 | N,V,Z,C | Subtract with Carry |
| F5 | SBC | Zero Page,X | 2 | 4 | N,V,Z,C | Subtract with Carry |
| ED | SBC | Absolute | 3 | 4 | N,V,Z,C | Subtract with Carry |
| FD | SBC | Absolute,X | 3 | 4* | N,V,Z,C | Subtract with Carry |
| F9 | SBC | Absolute,Y | 3 | 4* | N,V,Z,C | Subtract with Carry |
| E1 | SBC | Indirect,X | 2 | 6 | N,V,Z,C | Subtract with Carry |
| F1 | SBC | Indirect,Y | 2 | 5* | N,V,Z,C | Subtract with Carry |

#### Increment/Decrement Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| E6 | INC | Zero Page | 2 | 5 | N,Z | Increment Memory |
| F6 | INC | Zero Page,X | 2 | 6 | N,Z | Increment Memory |
| EE | INC | Absolute | 3 | 6 | N,Z | Increment Memory |
| FE | INC | Absolute,X | 3 | 7 | N,Z | Increment Memory |
| E8 | INX | Implied | 1 | 2 | N,Z | Increment X Register |
| C8 | INY | Implied | 1 | 2 | N,Z | Increment Y Register |
| C6 | DEC | Zero Page | 2 | 5 | N,Z | Decrement Memory |
| D6 | DEC | Zero Page,X | 2 | 6 | N,Z | Decrement Memory |
| CE | DEC | Absolute | 3 | 6 | N,Z | Decrement Memory |
| DE | DEC | Absolute,X | 3 | 7 | N,Z | Decrement Memory |
| CA | DEX | Implied | 1 | 2 | N,Z | Decrement X Register |
| 88 | DEY | Implied | 1 | 2 | N,Z | Decrement Y Register |

#### Logical Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| 29 | AND | Immediate | 2 | 2 | N,Z | Logical AND |
| 25 | AND | Zero Page | 2 | 3 | N,Z | Logical AND |
| 35 | AND | Zero Page,X | 2 | 4 | N,Z | Logical AND |
| 2D | AND | Absolute | 3 | 4 | N,Z | Logical AND |
| 3D | AND | Absolute,X | 3 | 4* | N,Z | Logical AND |
| 39 | AND | Absolute,Y | 3 | 4* | N,Z | Logical AND |
| 21 | AND | Indirect,X | 2 | 6 | N,Z | Logical AND |
| 31 | AND | Indirect,Y | 2 | 5* | N,Z | Logical AND |
| 09 | ORA | Immediate | 2 | 2 | N,Z | Logical OR |
| 05 | ORA | Zero Page | 2 | 3 | N,Z | Logical OR |
| 15 | ORA | Zero Page,X | 2 | 4 | N,Z | Logical OR |
| 0D | ORA | Absolute | 3 | 4 | N,Z | Logical OR |
| 1D | ORA | Absolute,X | 3 | 4* | N,Z | Logical OR |
| 19 | ORA | Absolute,Y | 3 | 4* | N,Z | Logical OR |
| 01 | ORA | Indirect,X | 2 | 6 | N,Z | Logical OR |
| 11 | ORA | Indirect,Y | 2 | 5* | N,Z | Logical OR |
| 49 | EOR | Immediate | 2 | 2 | N,Z | Exclusive OR |
| 45 | EOR | Zero Page | 2 | 3 | N,Z | Exclusive OR |
| 55 | EOR | Zero Page,X | 2 | 4 | N,Z | Exclusive OR |
| 4D | EOR | Absolute | 3 | 4 | N,Z | Exclusive OR |
| 5D | EOR | Absolute,X | 3 | 4* | N,Z | Exclusive OR |
| 59 | EOR | Absolute,Y | 3 | 4* | N,Z | Exclusive OR |
| 41 | EOR | Indirect,X | 2 | 6 | N,Z | Exclusive OR |
| 51 | EOR | Indirect,Y | 2 | 5* | N,Z | Exclusive OR |

#### Shift/Rotate Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| 0A | ASL | Accumulator | 1 | 2 | N,Z,C | Arithmetic Shift Left |
| 06 | ASL | Zero Page | 2 | 5 | N,Z,C | Arithmetic Shift Left |
| 16 | ASL | Zero Page,X | 2 | 6 | N,Z,C | Arithmetic Shift Left |
| 0E | ASL | Absolute | 3 | 6 | N,Z,C | Arithmetic Shift Left |
| 1E | ASL | Absolute,X | 3 | 7 | N,Z,C | Arithmetic Shift Left |
| 4A | LSR | Accumulator | 1 | 2 | N,Z,C | Logical Shift Right |
| 46 | LSR | Zero Page | 2 | 5 | N,Z,C | Logical Shift Right |
| 56 | LSR | Zero Page,X | 2 | 6 | N,Z,C | Logical Shift Right |
| 4E | LSR | Absolute | 3 | 6 | N,Z,C | Logical Shift Right |
| 5E | LSR | Absolute,X | 3 | 7 | N,Z,C | Logical Shift Right |
| 2A | ROL | Accumulator | 1 | 2 | N,Z,C | Rotate Left |
| 26 | ROL | Zero Page | 2 | 5 | N,Z,C | Rotate Left |
| 36 | ROL | Zero Page,X | 2 | 6 | N,Z,C | Rotate Left |
| 2E | ROL | Absolute | 3 | 6 | N,Z,C | Rotate Left |
| 3E | ROL | Absolute,X | 3 | 7 | N,Z,C | Rotate Left |
| 6A | ROR | Accumulator | 1 | 2 | N,Z,C | Rotate Right |
| 66 | ROR | Zero Page | 2 | 5 | N,Z,C | Rotate Right |
| 76 | ROR | Zero Page,X | 2 | 6 | N,Z,C | Rotate Right |
| 6E | ROR | Absolute | 3 | 6 | N,Z,C | Rotate Right |
| 7E | ROR | Absolute,X | 3 | 7 | N,Z,C | Rotate Right |

#### Compare Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| C9 | CMP | Immediate | 2 | 2 | N,Z,C | Compare Accumulator |
| C5 | CMP | Zero Page | 2 | 3 | N,Z,C | Compare Accumulator |
| D5 | CMP | Zero Page,X | 2 | 4 | N,Z,C | Compare Accumulator |
| CD | CMP | Absolute | 3 | 4 | N,Z,C | Compare Accumulator |
| DD | CMP | Absolute,X | 3 | 4* | N,Z,C | Compare Accumulator |
| D9 | CMP | Absolute,Y | 3 | 4* | N,Z,C | Compare Accumulator |
| C1 | CMP | Indirect,X | 2 | 6 | N,Z,C | Compare Accumulator |
| D1 | CMP | Indirect,Y | 2 | 5* | N,Z,C | Compare Accumulator |
| E0 | CPX | Immediate | 2 | 2 | N,Z,C | Compare X Register |
| E4 | CPX | Zero Page | 2 | 3 | N,Z,C | Compare X Register |
| EC | CPX | Absolute | 3 | 4 | N,Z,C | Compare X Register |
| C0 | CPY | Immediate | 2 | 2 | N,Z,C | Compare Y Register |
| C4 | CPY | Zero Page | 2 | 3 | N,Z,C | Compare Y Register |
| CC | CPY | Absolute | 3 | 4 | N,Z,C | Compare Y Register |

#### Bit Test Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| 24 | BIT | Zero Page | 2 | 3 | N,V,Z | Test Bits |
| 2C | BIT | Absolute | 3 | 4 | N,V,Z | Test Bits |

#### Branch Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| 10 | BPL | Relative | 2 | 2** | - | Branch if Plus |
| 30 | BMI | Relative | 2 | 2** | - | Branch if Minus |
| 50 | BVC | Relative | 2 | 2** | - | Branch if Overflow Clear |
| 70 | BVS | Relative | 2 | 2** | - | Branch if Overflow Set |
| 90 | BCC | Relative | 2 | 2** | - | Branch if Carry Clear |
| B0 | BCS | Relative | 2 | 2** | - | Branch if Carry Set |
| D0 | BNE | Relative | 2 | 2** | - | Branch if Not Equal |
| F0 | BEQ | Relative | 2 | 2** | - | Branch if Equal |

#### Jump/Call Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| 4C | JMP | Absolute | 3 | 3 | - | Jump |
| 6C | JMP | Indirect | 3 | 5 | - | Jump (with page boundary bug) |
| 20 | JSR | Absolute | 3 | 6 | - | Jump to Subroutine |
| 60 | RTS | Implied | 1 | 6 | - | Return from Subroutine |
| 40 | RTI | Implied | 1 | 6 | All | Return from Interrupt |

#### Flag Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| 18 | CLC | Implied | 1 | 2 | C | Clear Carry Flag |
| 38 | SEC | Implied | 1 | 2 | C | Set Carry Flag |
| 58 | CLI | Implied | 1 | 2 | I | Clear Interrupt Disable |
| 78 | SEI | Implied | 1 | 2 | I | Set Interrupt Disable |
| B8 | CLV | Implied | 1 | 2 | V | Clear Overflow Flag |
| D8 | CLD | Implied | 1 | 2 | D | Clear Decimal Mode |
| F8 | SED | Implied | 1 | 2 | D | Set Decimal Mode |

#### Other Operations
| Opcode | Mnemonic | Addressing Mode | Bytes | Cycles | Flags | Description |
|--------|----------|-----------------|-------|--------|-------|-------------|
| 00 | BRK | Implied | 1 | 7 | B,I | Force Interrupt |
| EA | NOP | Implied | 1 | 2 | - | No Operation |

Notes:
- \* Add 1 cycle if page boundary is crossed
- \*\* Add 1 cycle if branch is taken, add 2 cycles if page boundary is crossed

### Undocumented Instructions (Used by Some Games)

These instructions are not officially documented but are stable and used by some NES games:

| Opcode | Mnemonic | Description | Cycles |
|--------|----------|-------------|--------|
| 0B/2B | ANC | AND immediate with accumulator, set carry from bit 7 | 2 |
| 4B | ALR | AND immediate with accumulator, then LSR | 2 |
| 6B | ARR | AND immediate with accumulator, then ROR | 2 |
| CB | AXS | AND X with accumulator, subtract immediate, store in X | 2 |
| 87/97/83/8F | SAX | Store A AND X | 3-4 |
| A7/B7/A3/AF/B3 | LAX | Load A and X with same value | 3-5 |
| C7/D7/CF/DF/DB/C3/D3 | DCP | Decrement memory then compare with A | 5-8 |
| E7/F7/EF/FF/FB/E3/F3 | ISC | Increment memory then SBC | 5-8 |
| 27/37/2F/3F/3B/23/33 | RLA | ROL memory then AND with A | 5-8 |
| 67/77/6F/7F/7B/63/73 | RRA | ROR memory then ADC | 5-8 |
| 07/17/0F/1F/1B/03/13 | SLO | ASL memory then ORA with A | 5-8 |
| 47/57/4F/5F/5B/43/53 | SRE | LSR memory then EOR with A | 5-8 |

## Critical Timing Notes

1. **DMA Timing**: When OAM DMA ($4014) is written, the CPU is suspended for 513 or 514 cycles (513 on odd CPU cycles, 514 on even).

2. **Interrupt Timing**: 
   - Interrupts are polled before the last cycle of each instruction
   - If an interrupt is detected, it takes 7 cycles to execute
   - The interrupt sequence: push PC high, push PC low, push P with B flag clear, load vector

3. **Page Crossing**: Instructions that can cross page boundaries take an extra cycle when they do. The extra cycle occurs even if the instruction doesn't need to read from the new page (e.g., STA absolute,X always takes 5 cycles).

4. **Dummy Reads**: Many instructions perform dummy reads that can trigger side effects on hardware registers. For example:
   - Read-modify-write instructions read twice and write twice
   - Indexed addressing modes may read from the wrong address before correction

5. **Pipeline Behavior**: The 6502 has a simple pipeline that can lead to some quirks:
   - The processor always reads the byte after an instruction
   - Branch instructions fetch the opcode of the next instruction regardless of whether the branch is taken