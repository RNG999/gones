# NES Memory Architecture Specifications

## Overview
The NES has separate address spaces for the CPU and PPU, with various mapping and mirroring schemes. Understanding the memory architecture is crucial for accurate emulation.

## CPU Memory Map ($0000-$FFFF)

### RAM ($0000-$1FFF)
```
$0000-$07FF: 2KB internal RAM
$0800-$0FFF: Mirror of $0000-$07FF
$1000-$17FF: Mirror of $0000-$07FF
$1800-$1FFF: Mirror of $0000-$07FF
```

#### Zero Page ($0000-$00FF)
- Fast access area used for frequently accessed variables
- Special addressing modes available
- Stack hardwired to $0100-$01FF

#### Stack ($0100-$01FF)
- Stack pointer is 8-bit, indexes into this page
- Grows downward from $01FF
- Used for subroutine calls, interrupts, and temporary storage

### PPU Registers ($2000-$3FFF)
```
$2000-$2007: PPU registers
$2008-$3FFF: Mirrors of $2000-$2007 (repeats every 8 bytes)
```

### APU and I/O Registers ($4000-$401F)
```
$4000-$4013: APU registers
$4014: OAM DMA
$4015: APU status/control
$4016: Controller 1
$4017: Controller 2 / APU frame counter
$4018-$401F: CPU test registers (disabled on production units)
```

### Cartridge Space ($4020-$FFFF)
```
$4020-$5FFF: Expansion ROM/RAM (used by some mappers)
$6000-$7FFF: PRG-RAM (battery-backed save RAM)
$8000-$FFFF: PRG-ROM (program code and data)
```

#### PRG-ROM Banks
```
$8000-$BFFF: First 16KB PRG-ROM bank
$C000-$FFFF: Last 16KB PRG-ROM bank
```

For games with only 16KB PRG-ROM, $C000-$FFFF mirrors $8000-$BFFF.

### Interrupt Vectors
```
$FFFA-$FFFB: NMI vector
$FFFC-$FFFD: Reset vector
$FFFE-$FFFF: IRQ/BRK vector
```

## PPU Memory Map ($0000-$3FFF)

### Pattern Tables ($0000-$1FFF)
```
$0000-$0FFF: Pattern table 0 (256 tiles)
$1000-$1FFF: Pattern table 1 (256 tiles)
```

Each tile is 16 bytes:
- 8 bytes: Bit plane 0 (low bits)
- 8 bytes: Bit plane 1 (high bits)

### Name Tables ($2000-$2FFF)
```
$2000-$23FF: Nametable 0
$2400-$27FF: Nametable 1
$2800-$2BFF: Nametable 2
$2C00-$2FFF: Nametable 3
```

#### Nametable Structure (each is 1KB)
```
$2000-$23BF: Tile data (32x30 tiles = 960 bytes)
$23C0-$23FF: Attribute data (64 bytes)
```

#### Attribute Table Format
Each byte controls a 32x32 pixel area (4x4 tiles):
```
7  bit  0
---- ----
3322 1100
|||| ||++- Top-left 2x2 tile group palette
|||| ++--- Top-right 2x2 tile group palette
||++------ Bottom-left 2x2 tile group palette
++-------- Bottom-right 2x2 tile group palette
```

### Mirrors ($3000-$3EFF)
```
$3000-$3EFF: Mirror of $2000-$2EFF
```

### Palette RAM ($3F00-$3FFF)
```
$3F00: Universal background color
$3F01-$3F03: Background palette 0
$3F04: Mirror of $3F00
$3F05-$3F07: Background palette 1
$3F08: Mirror of $3F00
$3F09-$3F0B: Background palette 2
$3F0C: Mirror of $3F00
$3F0D-$3F0F: Background palette 3
$3F10: Mirror of $3F00 (usually transparent sprite color)
$3F11-$3F13: Sprite palette 0
$3F14: Mirror of $3F00
$3F15-$3F17: Sprite palette 1
$3F18: Mirror of $3F00
$3F19-$3F1B: Sprite palette 2
$3F1C: Mirror of $3F00
$3F1D-$3F1F: Sprite palette 3
$3F20-$3FFF: Mirrors of $3F00-$3F1F
```

## Mirroring Configurations

### Nametable Mirroring
The NES has only 2KB of VRAM for nametables, but addresses 4KB. Mirroring is controlled by the cartridge:

#### Horizontal Mirroring (Vertical Arrangement)
```
$2000 = $2400 (Nametable A)
$2800 = $2C00 (Nametable B)
```
Used for vertical scrolling games.

#### Vertical Mirroring (Horizontal Arrangement)
```
$2000 = $2800 (Nametable A)
$2400 = $2C00 (Nametable B)
```
Used for horizontal scrolling games.

#### Single-Screen Mirroring
All four nametables map to the same 1KB:
```
$2000 = $2400 = $2800 = $2C00
```
Can map to either nametable A or B.

#### Four-Screen Mirroring
Cartridge provides additional 2KB of VRAM:
```
$2000: Nametable 0 (NES VRAM)
$2400: Nametable 1 (NES VRAM)
$2800: Nametable 2 (Cart VRAM)
$2C00: Nametable 3 (Cart VRAM)
```

## Cartridge Memory Mapping

### Common Configurations

#### NROM (Mapper 0)
- 16KB or 32KB PRG-ROM
- 8KB CHR-ROM (or CHR-RAM)
- No bankswitching

#### MMC1 (Mapper 1)
- Up to 512KB PRG-ROM (bankswitched)
- Up to 128KB CHR-ROM (bankswitched)
- 8KB PRG-RAM (optional, battery-backed)
- Mirroring control

#### MMC3 (Mapper 4)
- Up to 512KB PRG-ROM
- Up to 256KB CHR-ROM
- 8KB PRG-RAM (optional)
- IRQ counter for raster effects

### Bank Switching
Mappers can dynamically change which ROM banks are visible in the CPU/PPU address space:

#### PRG Banking
- Fixed bank: Usually the last 16KB at $C000-$FFFF
- Switchable banks: At $8000-$BFFF or finer granularity

#### CHR Banking
- Can switch pattern tables independently
- Some mappers allow 1KB or 2KB CHR banks

## Memory Access Timing

### CPU Memory Access
- Read cycle: 2 CPU cycles minimum
- Write cycle: 2 CPU cycles minimum
- Page crossing adds 1 cycle for some instructions

### PPU Memory Access
- VRAM access during rendering: Automatic, follows rendering pattern
- VRAM access during vblank: Via $2006/$2007
- Palette access: Immediate, no buffering

### DMA (Direct Memory Access)
- OAM DMA ($4014): 513-514 CPU cycles
- DMC DMA: 1-4 CPU cycles per sample byte

## Memory Conflicts and Edge Cases

### Bus Conflicts
Some cartridges don't disable PRG-ROM during writes, causing bus conflicts:
- The written value should be ANDed with the ROM value
- Only affects certain mappers (e.g., CNROM)

### Open Bus
Reading from unmapped addresses returns "open bus":
- Usually the last value on the data bus
- Decays over time on real hardware
- PPU registers $2000-$2007 affect bits 0-4 of open bus

### PPU/CPU Synchronization
- PPU runs at 3x CPU speed
- Certain operations must respect this timing
- Race conditions possible with $2002, $2005, $2006

## PRG-RAM (Save RAM)

### Battery-Backed RAM
- Usually 8KB at $6000-$7FFF
- Some games use only 2KB or 4KB
- Must be preserved between sessions

### Work RAM
- Non-battery backed RAM for temporary use
- Same address range as save RAM
- Lost on power-off

## Important Implementation Notes

1. **Mirroring Detection**: Games may rely on specific mirroring behavior. Incorrect mirroring can cause graphical glitches or crashes.

2. **Bus Conflicts**: The AND behavior for bus conflicts is important for some games that intentionally use this for copy protection.

3. **Open Bus Behavior**: While many emulators return 0 for unmapped reads, accurate open bus emulation is needed for some test ROMs and edge cases.

4. **RAM Power-On State**: RAM should be initialized to a pattern (typically $00 or $FF with some random bits) rather than all zeros.

5. **Partial Decoding**: Some hardware uses partial address decoding, causing registers to be mirrored at unexpected addresses.

6. **Memory Access Timing**: Accurate timing is crucial for games that use cycle-timed code or race the PPU.

7. **Cartridge Variations**: Different cartridge revisions may have different RAM sizes or mirroring behavior even with the same mapper number.