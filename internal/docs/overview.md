# NES Emulator Specifications Overview

## Critical Specifications Summary

This document provides a high-level overview of the most critical specifications for NES emulator accuracy. These are the aspects that will make or break game compatibility and should be prioritized during implementation and testing.

## Most Critical for Game Compatibility

### 1. CPU Instruction Accuracy
**Priority: Critical**
- **Cycle-perfect instruction timing**: Even 1 cycle off can break timing-sensitive games
- **Undocumented opcodes**: Games like Battletoads use undocumented instructions
- **Interrupt timing**: NMI/IRQ must occur at exact cycle boundaries
- **DMA halt behavior**: OAM DMA completely halts CPU for 513-514 cycles

**Key Test Cases:**
- CPU timing test ROMs
- Games with precise timing (Battletoads, Silver Surfer)
- Games using undocumented opcodes (Bucky O'Hare)

### 2. PPU Rendering Pipeline
**Priority: Critical**
- **Exact scanline timing**: 341 cycles per scanline, 262 scanlines (NTSC)
- **Sprite 0 hit detection**: Used for precise timing in many games
- **Background/sprite priority**: Incorrect priority breaks visual effects
- **Mid-frame register changes**: Games change PPU registers during rendering

**Key Test Cases:**
- Sprite hit timing tests
- Scrolling games (Super Mario Bros 3)
- Status bar games (Metroid, Zelda)
- Raster effect games (Kirby's Adventure)

### 3. Memory Mapping and Mirroring
**Priority: Critical**
- **Nametable mirroring**: Horizontal vs vertical affects scrolling
- **Mapper implementations**: Each mapper has unique banking behavior
- **PPU address space**: Incorrect mirroring breaks graphics
- **Open bus behavior**: Some games rely on reading unmapped memory

**Key Test Cases:**
- Games with different mirroring (horizontal: Ice Climber, vertical: Super Mario Bros)
- Complex mappers (MMC3: Super Mario Bros 3, MMC5: Castlevania III)
- Mapper-specific test ROMs

## High Priority for Accuracy

### 4. APU Audio Generation
**Priority: High**
- **Channel mixing**: Non-linear mixing affects audio quality
- **Frame counter timing**: Controls envelope and sweep timing
- **DMC behavior**: Can steal CPU cycles and generate IRQs
- **Length counter**: Controls note duration

**Key Test Cases:**
- Audio test ROMs
- Music-heavy games (Mega Man series)
- Games using DMC samples (Zelda II)

### 5. Input System
**Priority: High**
- **Controller shift register**: 8-bit shift register per controller
- **Strobe timing**: Controls when to latch controller state
- **Multitap support**: Some games use 4 controllers
- **Light gun timing**: Requires precise PPU synchronization

**Key Test Cases:**
- Games requiring precise input (Track & Field)
- Light gun games (Duck Hunt)
- Four-player games (Gauntlet II)

### 6. Timing Synchronization
**Priority: High**
- **CPU/PPU clock ratio**: 3:1 for NTSC, 3.2:1 for PAL
- **Frame timing**: Affects game speed and audio pitch
- **NMI timing**: Race conditions can suppress interrupts
- **DMA conflicts**: DMC vs other operations

**Key Test Cases:**
- PAL vs NTSC timing
- Games sensitive to frame rate
- NMI suppression test ROMs

## Moderate Priority for Compatibility

### 7. Advanced PPU Features
**Priority: Moderate**
- **Sprite overflow**: Hardware bug affects flag setting
- **Palette corruption**: Reading $2007 during rendering
- **Color emphasis**: Affects entire screen brightness
- **Power-up state**: PPU state immediately after reset

**Key Test Cases:**
- Games checking sprite overflow
- Visual effect games using emphasis
- Power-on reliability

### 8. Edge Case Behaviors
**Priority: Moderate**
- **Bus conflicts**: Some mappers AND written data with ROM
- **OAM decay**: Sprite RAM decays when rendering disabled
- **Dummy reads**: Read-modify-write instructions read twice
- **Partial address decoding**: Hardware mirrors registers

**Key Test Cases:**
- Games with copy protection
- Long-running demos with rendering disabled
- Hardware-specific test ROMs

## Implementation Recommendations

### Development Phases

#### Phase 1: Core Functionality
1. Implement basic CPU with official instructions
2. Implement basic PPU with simple rendering
3. Implement NROM mapper (simplest cartridge type)
4. Basic input handling

#### Phase 2: Timing Accuracy
1. Cycle-accurate CPU instruction timing
2. Precise PPU scanline timing
3. Accurate interrupt handling
4. OAM DMA implementation

#### Phase 3: Advanced Features
1. Complex mappers (MMC1, MMC3)
2. APU with all 5 channels
3. Undocumented CPU instructions
4. Advanced PPU effects

#### Phase 4: Edge Cases
1. Bus conflicts and open bus
2. Hardware quirks and bugs
3. Power-on state emulation
4. Obscure mapper variants

### Testing Strategy

#### Continuous Testing
- Run basic games after each change
- Use automated test ROM suites
- Compare against reference implementations

#### Accuracy Validation
- Use hardware-specific test ROMs
- Test games known for timing sensitivity
- Validate audio output frequency
- Check visual effects accuracy

### Performance Considerations

#### Cycle-Accurate Emulation
- Necessary for highest accuracy
- CPU overhead: ~3x slower than instruction-level
- Required for timing-sensitive games

#### Optimizations
- JIT compilation for CPU
- Caching for PPU pattern/nametable access
- Audio buffering to reduce latency
- Frameskip for slow hardware

## Common Implementation Pitfalls

### 1. Timing Approximations
**Problem**: Using instruction-level timing instead of cycle-accurate
**Impact**: Breaks games with precise timing requirements
**Solution**: Implement cycle-perfect CPU and PPU timing

### 2. Incomplete Mapper Implementation
**Problem**: Only implementing common features of complex mappers
**Impact**: Games may work partially but have subtle bugs
**Solution**: Implement full mapper specifications including edge cases

### 3. PPU Register Timing
**Problem**: Applying register changes immediately instead of at correct cycles
**Impact**: Scrolling glitches, wrong graphics displayed
**Solution**: Track PPU internal state and apply changes at correct times

### 4. Missing Hardware Bugs
**Problem**: Implementing "corrected" behavior instead of actual hardware bugs
**Impact**: Games relying on bugs don't work correctly
**Solution**: Implement known hardware bugs as documented

### 5. Audio Mixing Errors
**Problem**: Using linear mixing instead of non-linear
**Impact**: Incorrect audio volume relationships
**Solution**: Use documented non-linear mixing formulas

## Validation and Testing

### Essential Test ROMs
- **blargg_nes_cpu_test5**: CPU instruction accuracy
- **sprite_hit_tests_2005.10.05**: Sprite 0 hit timing
- **ppu_vbl_nmi**: NMI timing
- **full_palette**: Color accuracy
- **dmc_dma_during_read4**: DMC vs controller timing

### Game Compatibility Targets
- **Tier 1**: Super Mario Bros, Tetris, Duck Hunt
- **Tier 2**: Mega Man 2, Metroid, Zelda
- **Tier 3**: Battletoads, Silver Surfer, Rad Racer
- **Tier 4**: Papillion, Bucky O'Hare (undocumented opcodes)

### Hardware Comparison
- Use Mesen, FCEUX, or real hardware as reference
- Compare CPU cycle counts for instruction sequences
- Verify PPU register behavior during rendering
- Validate audio output waveforms and frequencies

This specification overview should guide development priorities and testing strategies for achieving high NES emulator accuracy.