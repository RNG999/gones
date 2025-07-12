# NES Development Specification

## Service Overview

**NESdev Wiki** is the comprehensive knowledge hub for Nintendo Entertainment System (NES) and Famicom development. Established as "the source for all your NES programming needs," it serves as the central repository for technical documentation, programming guides, and community resources for developers interested in creating software for the NES platform.

### Target Audience
- Homebrew game developers
- Emulator authors
- Hardware enthusiasts
- Retro gaming programmers
- Technical researchers studying 8-bit systems

### Community Resources
- **NESdev BBS**: Active community forums for technical discussions
- **IRC Channel**: #NESdev on EFnet for real-time developer chat
- **Discord Server**: Modern communication platform for the NESdev community
- **Sister Project**: SNESdev Wiki (launched May 2022)

## Technical Architecture

### CPU Specifications (2A03/2A07)
The NES CPU is a Ricoh-manufactured variant of the MOS Technology 6502 processor with the following characteristics:

- **Architecture**: Modified 6502 (lacks decimal mode)
- **Clock Frequencies**:
  - NTSC (RP2A03): 1.789773 MHz (~559 ns per cycle)
  - PAL (RP2A07): 1.662607 MHz (~601 ns per cycle)
  - Dendy: 1.773448 MHz (~564 ns per cycle)
- **Integration**: CPU and APU combined on single chip
- **Instruction Set**: Full 6502 opcodes including unofficial opcodes
- **Emulation Recommendation**: 21441960 Hz for cycle-accurate NTSC emulation

### PPU Specifications (Picture Processing Unit)
The NES PPU was considered an advanced 2D graphics processor in the early 1980s:

- **Video Output**: Composite video signal, 240 visible lines with 256x240 pixel picture region
- **Memory Architecture**:
  - 10 KB total addressable space
  - 8 KB ROM/RAM for tile pattern graphics (Pattern Tables)
  - 2 KB console RAM for nametables (background maps)
  - 32 bytes palette RAM at $3F00-$3F1F
  - 256 bytes Object Attribute Memory (OAM) for sprite data
- **Timing Relationships**:
  - NTSC: Exactly 3 PPU cycles per CPU cycle
  - PAL: 3.2 PPU cycles per CPU cycle
  - 262 total scanlines per frame (NTSC)
  - 341 PPU cycles per scanline
  - ~2270 PPU cycles available during vblank for updates
- **Rendering Pipeline**:
  - Background rendering: 4 memory fetches per 8-pixel tile
  - Sprite evaluation: Up to 8 sprites per scanline
  - Palette system: 4 background + 4 sprite palettes, 4 colors each
  - Color depth: 6-bit values representing 64 possible colors

#### PPU Color System
- **Palette RAM Structure**: 4 palettes for backgrounds, 4 for sprites
- **Color Encoding**: 6-bit values (2 bits brightness, 4 bits hue)
- **Backdrop Color**: Palette entry 0 serves as transparent/backdrop color
- **Color Variants**: 
  - 2C02 (NTSC): Standard home console palette
  - 2C07 (PAL): Different color phase timing
  - 2C03/2C04/2C05: Special arcade variants

#### PPU Rendering Quirks
- OAM uses dynamic memory requiring refresh during rendering
- Odd frames skip one cycle for improved image quality
- Sprite 0 hit detection starts at cycle 2 of scanline
- Mid-frame rendering toggles can cause visual artifacts
- Pattern table data uses 2bpp (bits per pixel) encoding

### Memory Architecture
- **CPU Address Space**: 64KB addressable memory
- **PPU Address Space**: Separate 16KB address space
- **Cartridge Mappers**: Hardware extensions for expanded memory access
- **Dynamic Memory**: OAM requires refresh during active rendering

### Audio Processing Unit (APU)
- Integrated with CPU chip
- Multiple sound channels for music and effects
- Programmable sound generation

## Development Environment

### Programming Languages
1. **6502 Assembly Language** (Primary)
   - Direct hardware control
   - Cycle-accurate programming
   - Maximum performance optimization

2. **C Language** (via CC65 compiler)
   - Higher-level abstraction
   - Easier development workflow
   - Some performance trade-offs

### Essential Development Tools
- **Assemblers**: For 6502 assembly code compilation
- **Disassemblers**: For analyzing existing ROMs
- **Graphics Utilities**: Tile and sprite editors
- **Emulators**: Testing and debugging platforms
- **CC65**: C compiler and toolchain for 6502 targets

### File Formats
- **iNES**: Original ROM format standard
- **NES 2.0**: Extended ROM format with additional metadata
- **UNIF**: Universal NES Image Format
- **NSF**: NES Sound Format for music
- **FDS**: Famicom Disk System format
- **IPS**: Patch format for ROM modifications

## Core Programming Concepts

### Fundamental Skills
1. **Cycle Counting**: Precise timing control for raster effects
2. **Interrupt Handling**: NMI for vblank, IRQ for mappers
3. **Memory Management**: Working within limited RAM constraints
4. **Bank Switching**: Using mappers for extended ROM/RAM

### Graphics Programming
- **Sprite Management**: 64 sprite limit, 8 per scanline
- **Background Scrolling**: Smooth scrolling techniques
- **Pattern Tables**: Tile-based graphics system
- **Palette Management**: Working with limited colors
- **Raster Effects**: Mid-frame graphics modifications

### Audio Programming
- **Channel Management**: Pulse, triangle, noise, and DMC channels
- **Music Engines**: Composing and playing background music
- **Sound Effects**: Integrating gameplay audio

### Input Handling
- **Controller Reading**: Standard and special controllers
- **Input Buffering**: Smooth control response
- **Multiple Players**: Supporting various input configurations

### Advanced Techniques
- **Compression**: Fitting more content in limited ROM space
- **Unofficial Opcodes**: Using undocumented CPU features
- **Math Routines**: Fixed-point arithmetic, lookup tables
- **Random Number Generation**: Pseudo-random algorithms

## Learning Path

### Getting Started
1. **"Before the Basics"**: Understanding the NES architecture
2. **Installing CC65**: Setting up the development environment
3. **First Program**: Simple "Hello World" examples

### Progressive Learning
1. **CPU Basics**: Understanding 6502 assembly
2. **PPU Programming**: Drawing graphics on screen
3. **APU Introduction**: Creating simple sounds
4. **Input Handling**: Reading controller input
5. **Complete Game**: Combining all concepts

### Resources
- **Wiki Documentation**: Comprehensive technical references
- **Tutorial Collection**: Step-by-step programming guides
- **Open Source Projects**: Learning from existing code
- **Community Support**: Forums and chat for questions

## Development Workflow

### Modern Development
1. **Cross-Development**: Using modern PCs for NES development
2. **Version Control**: Git integration for source management
3. **Build Automation**: Makefiles and build scripts
4. **Testing Strategy**: Emulator testing before hardware

### Deployment Options
- **Emulators**: Primary testing platform
- **Flash Cartridges**: PowerPak, Everdrive for hardware testing
- **ROM Distribution**: Sharing homebrew games online
- **Physical Releases**: Modern cartridge production options

## Best Practices

### Code Organization
- Modular assembly files
- Clear commenting for timing-critical sections
- Consistent naming conventions
- Separation of engine and game logic

### Performance Optimization
- Minimize cycles in critical loops
- Use lookup tables over calculations
- Unroll loops where beneficial
- Profile and optimize bottlenecks

### Debugging Strategies
- Emulator debugging features
- Visual debugging with color codes
- Serial output for hardware debugging
- Systematic testing approach

## Platform Specifications Summary

The NES represents a carefully balanced architecture where every component works in harmony:
- **Limited Resources**: 2KB RAM, specific CPU/PPU timing
- **Deterministic Behavior**: Predictable hardware timing
- **Expandability**: Mapper chips extend capabilities
- **Proven Design**: Successful platform for decades of development

This specification serves as a foundation for understanding NES development through the lens of the comprehensive NESdev Wiki resources. The platform's constraints inspire creative solutions and efficient programming techniques that remain relevant for embedded and retro development today.

## Mapper 0 (NROM) Specifications

### Memory Mapping
- **NROM-128**: 16 KiB total PRG ROM
  - CPU $8000-$BFFF: First 16 KB of ROM
  - CPU $C000-$FFFF: Mirror of $8000-$BFFF
- **NROM-256**: 32 KiB total PRG ROM  
  - CPU $8000-$BFFF: First 16 KB of ROM
  - CPU $C000-$FFFF: Last 16 KB of ROM

### CHR Configuration
- **CHR ROM**: 8 KiB fixed (no bank switching)
- **CHR RAM**: Supported by most modern emulators
- **Pattern Table Access**: Via PPUCTRL for tile animation

### Key Limitations
- No bank switching capabilities
- Subject to bus conflicts
- No IRQ support
- No additional audio capabilities
- Horizontal or Vertical mirroring only (solder pad controlled)

## PPU Detailed Specifications

### VRAM Address Format
```
yyy NN YYYYY XXXXX
||| || ||||| +++++-- coarse X scroll (5 bits)
||| || +++++-------- coarse Y scroll (5 bits)
||| ++-------------- nametable select (2 bits)
+++----------------- fine Y scroll (3 bits)
```

### Sprite Evaluation Cycle-by-Cycle
- **Cycles 1-64**: Secondary OAM initialization to $FF
- **Cycles 65-256**: Sprite evaluation
  - Find sprites on current scanline
  - Copy up to 8 in-range sprites to secondary OAM
  - Check for sprite overflow condition
- **Cycles 257-320**: Sprite fetches
  - Read sprite details for 8 selected sprites
  - Fetch tile data, attributes, coordinates

### Rendering Pipeline Details
- **Background Rendering**: 4 memory fetches per 8-pixel tile
  1. Nametable byte fetch
  2. Attribute table byte fetch  
  3. Pattern table low byte fetch
  4. Pattern table high byte fetch

## Emulator Development Guidelines

### Critical Requirements
1. **No Forced Modifications**: Never modify game data or behavior
2. **Specification Compliance**: Follow NESdev Wiki specifications exactly
3. **Debugging Separation**: Debug code must not affect game logic
4. **Mapper 0 Priority**: Ensure all NROM games work correctly
5. **Timing Accuracy**: Implement cycle-accurate PPU behavior

### Implementation Standards
- **VRAM Address Handling**: Strict adherence to PPU address format
- **Sprite Evaluation**: Must follow exact cycle-by-cycle behavior
- **No Bus Conflicts**: Proper handling for Mapper 0
- **Palette Processing**: Accurate 6-bit color encoding
- **OAM Refresh**: Required during active rendering

### Testing Requirements
- **Super Mario Bros**: Must display correctly (Mapper 0 reference)
- **All NROM Games**: Complete compatibility required
- **Visual Accuracy**: Correct colors and sprite rendering
- **Timing Accuracy**: No glitches or artifacts