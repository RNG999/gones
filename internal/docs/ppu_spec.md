# NES PPU (2C02) Specifications

## Overview
The NES Picture Processing Unit (PPU) is responsible for generating the video signal. It runs at 5.369318 MHz (NTSC) or 5.320342 MHz (PAL), which is 3x the CPU clock speed.

## Registers

### PPU Control Registers (CPU $2000-$2007)

#### $2000 - PPUCTRL (Write)
```
7  bit  0
---- ----
VPHB SINN
|||| ||||
|||| ||++- Base nametable address
|||| ||    (0 = $2000; 1 = $2400; 2 = $2800; 3 = $2C00)
|||| |+--- VRAM address increment per CPU read/write of PPUDATA
|||| |     (0: add 1, going across; 1: add 32, going down)
|||| +---- Sprite pattern table address for 8x8 sprites
||||       (0: $0000; 1: $1000; ignored in 8x16 mode)
|||+------ Background pattern table address (0: $0000; 1: $1000)
||+------- Sprite size (0: 8x8 pixels; 1: 8x16 pixels)
|+-------- PPU master/slave select
|          (0: read backdrop from EXT pins; 1: output color on EXT pins)
+--------- Generate an NMI at the start of the vertical blanking interval
           (0: off; 1: on)
```

#### $2001 - PPUMASK (Write)
```
7  bit  0
---- ----
BGRs bMmG
|||| ||||
|||| |||+- Greyscale (0: normal color, 1: produce a greyscale display)
|||| ||+-- 1: Show background in leftmost 8 pixels of screen, 0: Hide
|||| |+--- 1: Show sprites in leftmost 8 pixels of screen, 0: Hide
|||| +---- 1: Show background
|||+------ 1: Show sprites
||+------- Emphasize red (green on PAL/Dendy)
|+-------- Emphasize green (red on PAL/Dendy)
+--------- Emphasize blue
```

#### $2002 - PPUSTATUS (Read)
```
7  bit  0
---- ----
VSO. ....
|||| ||||
|||+-++++- Least significant bits previously written into a PPU register
|||        (due to register not being updated for this address)
||+------- Sprite overflow flag
|+-------- Sprite 0 Hit flag
+--------- Vertical blank flag

Reading $2002 clears bit 7 and the address latch for $2005/$2006
```

#### $2003 - OAMADDR (Write)
- Sets the OAM address for subsequent reads/writes to $2004
- Most games write $00 here and use DMA

#### $2004 - OAMDATA (Read/Write)
- Read/Write OAM data
- Reads during rendering return $FF
- Writes during rendering do not modify OAM

#### $2005 - PPUSCROLL (Write x2)
- First write: X scroll position
- Second write: Y scroll position
- Changes made during rendering have immediate effect

#### $2006 - PPUADDR (Write x2)
- First write: High byte of VRAM address
- Second write: Low byte of VRAM address
- After second write, the address is loaded into the PPU

#### $2007 - PPUDATA (Read/Write)
- Read/Write VRAM data
- Reads are buffered (except palette data)
- Auto-increments by 1 or 32 based on PPUCTRL bit 2

### Internal Registers

#### v - Current VRAM address (15 bits)
```
yyy NN YYYYY XXXXX
||| || ||||| +++++-- Coarse X scroll
||| || +++++-------- Coarse Y scroll
||| ++-------------- Nametable select
+++----------------- Fine Y scroll
```

#### t - Temporary VRAM address (15 bits)
- Same format as v register
- Also known as "address latch"

#### x - Fine X scroll (3 bits)
- Selects the pixel within a tile

#### w - First/second write toggle (1 bit)
- Shared by $2005 and $2006

## Memory Map

### PPU Memory Space ($0000-$3FFF)
```
$0000-$0FFF: Pattern table 0 (CHR ROM/RAM)
$1000-$1FFF: Pattern table 1 (CHR ROM/RAM)
$2000-$23FF: Nametable 0
$2400-$27FF: Nametable 1
$2800-$2BFF: Nametable 2
$2C00-$2FFF: Nametable 3
$3000-$3EFF: Mirrors of $2000-$2EFF
$3F00-$3F1F: Palette RAM
$3F20-$3FFF: Mirrors of $3F00-$3F1F
```

### Nametable Layout
Each nametable is 32x30 tiles (960 bytes) + 64 bytes of attributes:
```
$2000-$23BF: Tile indices (32x30 = 960 bytes)
$23C0-$23FF: Attribute table (8x8 = 64 bytes)
```

### Palette Memory ($3F00-$3F1F)
```
$3F00: Universal background color
$3F01-$3F03: Background palette 0
$3F04: Mirror of $3F00
$3F05-$3F07: Background palette 1
$3F08: Mirror of $3F00
$3F09-$3F0B: Background palette 2
$3F0C: Mirror of $3F00
$3F0D-$3F0F: Background palette 3
$3F10: Mirror of $3F00
$3F11-$3F13: Sprite palette 0
$3F14: Mirror of $3F00
$3F15-$3F17: Sprite palette 1
$3F18: Mirror of $3F00
$3F19-$3F1B: Sprite palette 2
$3F1C: Mirror of $3F00
$3F1D-$3F1F: Sprite palette 3
```

## Rendering

### Frame Timing
- NTSC: 341 PPU cycles per scanline, 262 scanlines per frame
- PAL: 341 PPU cycles per scanline, 312 scanlines per frame

### Scanline Types
- -1 (261): Pre-render scanline
- 0-239: Visible scanlines
- 240: Post-render scanline (idle)
- 241-260: Vertical blanking

### Rendering Pipeline

#### Background Rendering (per scanline)
For each of the 341 cycles:
- Cycles 0: Idle
- Cycles 1-256: Fetch background data
  - Every 8 cycles:
    1. Nametable byte
    2. Attribute table byte
    3. Pattern table bitmap 0
    4. Pattern table bitmap 1
- Cycles 257-320: Fetch sprite data for next scanline
- Cycles 321-336: Fetch first two tiles of next scanline
- Cycles 337-340: Fetch nametable bytes (unused)

#### Sprite Rendering
1. **Sprite Evaluation (cycles 1-64)**:
   - Clear secondary OAM
   
2. **Sprite Evaluation (cycles 65-256)**:
   - Evaluate which sprites are on the next scanline
   - Copy up to 8 sprites to secondary OAM
   - Set sprite overflow flag if >8 sprites found

3. **Sprite Fetching (cycles 257-320)**:
   - Fetch sprite patterns for sprites in secondary OAM

### Sprite Attributes (OAM)
Each sprite uses 4 bytes:
```
Byte 0: Y position
Byte 1: Tile index
Byte 2: Attributes
  76543210
  |||   ||
  |||   ++- Palette (4 to 7) for this sprite
  ||+------ Priority (0: in front of background; 1: behind background)
  |+------- Flip sprite horizontally
  +-------- Flip sprite vertically
Byte 3: X position
```

### Sprite 0 Hit
Triggered when:
- An opaque background pixel overlaps an opaque sprite 0 pixel
- Both background and sprite rendering are enabled
- Not on pixel 255
- Not on scanline 0

### Rendering Disable Mid-Frame
When rendering is disabled mid-frame:
- The current VRAM address is maintained
- No more increments occur
- When re-enabled, rendering continues from the current address

## Scrolling

### Coarse X/Y Increment
During rendering:
- Coarse X increments every 8 pixels
- At pixel 256, coarse Y increments
- Coarse X wraps from 31 to 0 and toggles horizontal nametable
- Coarse Y wraps from 29 to 0 and toggles vertical nametable
- Y=30,31 are attribute table rows (invalid for scrolling)

### Mid-Frame Scroll Changes
Writes to $2000, $2005, and $2006 during rendering:
- $2000: Changes nametable bits immediately
- $2005: First write changes fine X immediately
- $2005: Second write has complex effects
- $2006: Corrupts rendering for that frame

## Color Generation

### NES Color Palette
The NES can display 64 colors (though some are duplicates):
- Hue: 0-15 (0 and 13-15 are grays)
- Brightness: 0-3 (0 is black for most hues)
- Color $0D is "blacker than black" and should not be used

### Color Emphasis
The emphasis bits affect the entire screen:
- Multiple bits can be set simultaneously
- On NTSC: Dims colors and emphasizes the selected channel
- On PAL: Dims colors differently due to different video encoding

## Mirroring

### Nametable Mirroring
The NES has 2KB of VRAM for nametables, but addresses 4KB:
- **Horizontal**: $2000=$2400, $2800=$2C00
- **Vertical**: $2000=$2800, $2400=$2C00
- **Single-screen**: All nametables show the same 1KB
- **Four-screen**: Cartridge provides additional 2KB

### Palette Mirroring
- $3F10, $3F14, $3F18, $3F1C mirror $3F00, $3F04, $3F08, $3F0C
- Addresses $3F20-$3FFF mirror $3F00-$3F1F

## PPU-CPU Synchronization

### Power-Up State
- PPU registers are in an undefined state
- $2002 returns $+0++++00 (+ = unknown)
- First frame is variable length
- Best practice: Wait for 2 vblanks before using PPU

### Register Behavior During Rendering
- $2003/4: Writes ignored, reads return $FF
- $2005/6: Immediate effect, can corrupt rendering
- $2007: Increments address, can corrupt rendering

### NMI Timing
- NMI occurs at cycle 1 of scanline 241
- Can be suppressed if $2002 is read near vblank start
- Race condition window: ~2 CPU cycles

## Common PPU Bugs and Quirks

### Sprite Overflow Bug
The sprite overflow flag has a hardware bug:
- Doesn't work correctly with sprites at Y=255
- Evaluation can "fail" and not set the flag
- Games rarely rely on this flag

### Palette Corruption
Reading $2007 during rendering can corrupt palette data due to the way the PPU multiplexes the address bus.

### OAM Decay
OAM is dynamic RAM and decays if not refreshed:
- Occurs when rendering is disabled for extended periods
- Bytes gradually decay to $FF
- Prevention: Write $00 to $2003 periodically

### First Visible Scanline
On the first visible scanline (scanline 0):
- The PPU skips the first idle cycle
- This can affect cycle-timed code