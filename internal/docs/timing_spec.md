# NES System Timing Specifications

## Overview
Precise timing is critical for NES emulation accuracy. The CPU, PPU, and APU all run at different frequencies and must be synchronized correctly to achieve accurate behavior.

## Clock Frequencies

### Master Clock (NTSC)
- Crystal oscillator: 21.477272 MHz
- Divided by 12 for CPU: 1.789773 MHz
- Divided by 4 for PPU: 5.369318 MHz

### Master Clock (PAL)
- Crystal oscillator: 26.601712 MHz  
- Divided by 16 for CPU: 1.662607 MHz
- Divided by 5 for PPU: 5.320342 MHz

### Clock Ratios
- **NTSC**: PPU runs 3x faster than CPU (5.369318 / 1.789773 = 3.0)
- **PAL**: PPU runs 3.2x faster than CPU (5.320342 / 1.662607 = 3.2)

## Frame Timing

### NTSC Timing
- 262 scanlines per frame
- 341 PPU cycles per scanline
- Total: 89,342 PPU cycles per frame
- CPU cycles per frame: 29,780.67 (89,342 / 3)
- Frame rate: ~60.0988 Hz

### PAL Timing
- 312 scanlines per frame
- 341 PPU cycles per scanline
- Total: 106,392 PPU cycles per frame
- CPU cycles per frame: 33,247.5 (106,392 / 3.2)
- Frame rate: ~50.0070 Hz

## Scanline Breakdown

### NTSC Scanlines
```
Scanline -1 (261): Pre-render scanline
Scanlines 0-239: Visible scanlines (240 total)
Scanline 240: Post-render scanline
Scanlines 241-260: Vertical blanking (20 scanlines)
```

### PAL Scanlines
```
Scanline -1 (311): Pre-render scanline
Scanlines 0-239: Visible scanlines (240 total)
Scanline 240: Post-render scanline
Scanlines 241-310: Vertical blanking (70 scanlines)
```

## CPU Timing

### Instruction Timing
CPU instructions take 2-7 cycles depending on the instruction and addressing mode:

#### Base Instruction Cycles
- Immediate: 2 cycles
- Zero Page: 3 cycles
- Zero Page,X/Y: 4 cycles
- Absolute: 4 cycles
- Absolute,X/Y: 4-5 cycles (5 if page crossed)
- Indirect,X: 6 cycles
- Indirect,Y: 5-6 cycles (6 if page crossed)

#### Page Crossing Penalty
When an indexed addressing mode crosses a page boundary:
- Read instructions: +1 cycle
- Write instructions: Always use the maximum cycles

#### Branch Instructions
- 2 cycles if branch not taken
- 3 cycles if branch taken (same page)
- 4 cycles if branch taken (page crossed)

### Interrupt Timing
- NMI/IRQ: 7 cycles from interrupt detection to first instruction of handler
- Reset: Variable, depends on state when reset occurs

### DMA Timing
#### OAM DMA ($4014)
- 513 cycles if started on odd CPU cycle
- 514 cycles if started on even CPU cycle
- CPU is completely halted during DMA

#### DMC DMA
- Steals 1-4 CPU cycles per sample byte
- Number of cycles depends on when the DMA occurs relative to CPU instruction

## PPU Timing

### Rendering Cycle Breakdown
Each scanline consists of 341 PPU cycles:

```
Cycles 0: Idle cycle
Cycles 1-256: Background and sprite rendering
Cycles 257-320: Sprite evaluation for next scanline
Cycles 321-336: Background fetching for next scanline
Cycles 337-340: Background fetch (unused)
```

### Background Rendering (Cycles 1-256)
Every 8 cycles, the PPU fetches background data:
1. Nametable byte (2 cycles)
2. Attribute table byte (2 cycles)
3. Pattern low byte (2 cycles)
4. Pattern high byte (2 cycles)

### Sprite Evaluation (Cycles 257-320)
- 64 cycles to evaluate all 64 sprites
- Determines which 8 sprites are on the next scanline
- Fetches pattern data for those sprites

### Critical PPU Timing Events

#### VBlank Start
- Occurs at cycle 1 of scanline 241
- NMI triggered here if enabled ($2000 bit 7)
- VBlank flag set in $2002

#### VBlank End
- Occurs at cycle 1-3 of pre-render scanline
- VBlank flag cleared in $2002
- Sprite 0 hit flag cleared
- Sprite overflow flag cleared

#### Pre-render Scanline Quirks
- Cycle 0 is skipped on odd frames when rendering is enabled
- This creates the frame-to-frame timing variation

## APU Timing

### Frame Counter Timing

#### 4-Step Mode (Default)
```
Step    CPU cycles    Action
0       7457          Clock envelopes & linear counter
1       14913         Clock envelopes & linear counter, length counters & sweep
2       22371         Clock envelopes & linear counter
3       29829         Clock envelopes & linear counter, length counters & sweep
IRQ     29830         Frame IRQ (if enabled)
```

#### 5-Step Mode
```
Step    CPU cycles    Action
0       7457          Clock envelopes & linear counter
1       14913         Clock envelopes & linear counter, length counters & sweep
2       22371         Clock envelopes & linear counter
3       29829         (No action)
4       37281         Clock envelopes & linear counter, length counters & sweep
```

### Channel Timer Updates
- Pulse/Noise/Triangle: Clocked every CPU cycle
- DMC: Clocked at rates from DMC rate table

## Synchronization Points

### CPU-PPU Synchronization
Critical points where CPU and PPU timing interact:

#### $2002 Status Register Reads
- Reading during VBlank start can suppress NMI
- Race condition window: ~2 CPU cycles

#### $2005/$2006 Writes During Rendering
- Immediate effect on PPU address registers
- Can cause rendering artifacts if timed incorrectly

#### OAM DMA
- Triggered by CPU write to $4014
- PPU continues running while CPU is halted
- DMA occurs over 256 consecutive PPU cycles

### CPU-APU Synchronization

#### Frame Counter Reset
- Writing to $4017 resets frame counter after 3-4 CPU cycles
- Delay depends on even/odd CPU cycle timing

#### DMC Sample Fetches
- DMC channel can steal CPU cycles
- Affects timing of CPU instructions

## Emulation Timing Strategies

### Cycle-Accurate Approach
Run CPU and PPU in lockstep:
1. Execute 1 CPU instruction
2. Execute 3x PPU cycles (NTSC) or 3.2x (PAL)
3. Handle any synchronization events
4. Repeat

### Scanline-Accurate Approach
Run components per scanline:
1. Execute ~113 CPU cycles (1 scanline worth)
2. Execute full PPU scanline (341 cycles)
3. Handle end-of-scanline events
4. Repeat for 262/312 scanlines

### Frame-Accurate Approach
Run components per frame:
1. Execute full CPU frame (~29,781 cycles)
2. Execute full PPU frame (89,342 cycles)
3. Handle frame events
4. Less accurate but faster

## Critical Timing Edge Cases

### Sprite 0 Hit Timing
- Occurs during pixel rendering
- Exact cycle depends on sprite position
- Must account for sprite-background priority

### NMI Suppression
Reading $2002 at exactly the wrong time can suppress NMI:
- If read during cycles 0-1 of scanline 241
- Creates timing-dependent bugs in some games

### Mid-Scanline Register Changes
Changes to certain PPU registers mid-scanline:
- $2000: Affects current scanline rendering
- $2005: Complex interactions with rendering
- $2006: Can corrupt rendering

### DMC Conflicts
DMC samples can conflict with other operations:
- Controller reads can be corrupted
- Sprite DMA can be delayed
- IRQ timing can be affected

## Testing Timing Accuracy

### Test ROMs
- cpu_timing_test6: Tests CPU instruction timing
- ppu_vbl_nmi: Tests NMI timing
- sprite_hit_tests: Tests sprite 0 hit timing
- dmc_dma_during_read4: Tests DMC vs controller timing

### Timing Validation
Compare against known-good hardware:
- Frame rate should match expected values
- Audio pitch should be correct
- Scrolling should be smooth
- No timing-dependent glitches

## Implementation Notes

1. **Fractional Cycles**: NTSC PPU runs at exactly 3x CPU speed, but PAL has fractional relationship. Handle carefully.

2. **Catch-up Timing**: When emulator falls behind, ensure synchronization events still occur at correct relative times.

3. **Variable Length Frames**: First frame after power-on has different timing. Pre-render scanline cycle 0 skip only occurs on odd frames.

4. **Register Access Timing**: Some register effects are immediate, others are delayed by 1-2 cycles.

5. **Audio Buffer Management**: APU timing affects audio sample generation rate. Buffer appropriately for smooth audio.

6. **Save State Timing**: When implementing save states, preserve cycle counters and timing state for accuracy.