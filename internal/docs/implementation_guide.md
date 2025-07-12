# NES Emulator Implementation Guide

## Purpose
This guide bridges the detailed specifications with the existing GoNES codebase, providing specific implementation recommendations based on the current code structure.

## Current Code Analysis

### CPU Implementation Status
The current CPU structure is well-designed but incomplete:

**Strengths:**
- Proper register representation
- Memory interface abstraction
- Cycle counter for timing

**Missing Critical Features:**
- Instruction decode and execution
- Addressing mode implementations
- Interrupt handling (NMI/IRQ/Reset)
- Undocumented opcode support

### PPU Implementation Status
The PPU has a good foundation but needs significant expansion:

**Strengths:**
- Basic register structure
- Scanline/cycle tracking
- Memory layout planning

**Missing Critical Features:**
- Complete register behavior (especially $2007 buffering)
- Background rendering pipeline
- Sprite evaluation and rendering
- Scrolling logic implementation
- Nametable mirroring

### APU Implementation Status  
The APU structure exists but is largely unimplemented:

**Strengths:**
- Channel structure defined
- Frame counter concept

**Missing Critical Features:**
- All audio generation logic
- Channel mixing (non-linear)
- Length counters, envelopes, sweeps
- DMC sample playback

## Priority Implementation Order

### Phase 1: Core CPU (Week 1-2)
1. **Instruction Implementation**
   ```go
   // Add to CPU struct
   type CPU struct {
       // ... existing fields ...
       
       // Instruction cycle tracking
       remainingCycles uint8
       
       // Interrupt flags
       nmiRequested bool
       irqRequested bool
   }
   
   // Implement instruction lookup table
   var instructionTable = [256]Instruction{
       0x69: {opcode: 0x69, mnemonic: "ADC", mode: Immediate, cycles: 2},
       // ... all 256 opcodes
   }
   ```

2. **Critical Instructions First**
   - Load/Store: LDA, LDX, LDY, STA, STX, STY
   - Arithmetic: ADC, SBC
   - Logic: AND, ORA, EOR
   - Branches: BNE, BEQ, BPL, BMI
   - Jumps: JMP, JSR, RTS

3. **Addressing Modes**
   ```go
   type AddressingMode int
   const (
       Immediate AddressingMode = iota
       ZeroPage
       ZeroPageX
       ZeroPageY
       Absolute
       AbsoluteX
       AbsoluteY
       IndirectX
       IndirectY
       // ...
   )
   ```

### Phase 2: Basic PPU (Week 2-3)
1. **Register Implementation**
   ```go
   // Enhance PPU struct
   type PPU struct {
       // ... existing fields ...
       
       // Internal registers
       v uint16 // Current VRAM address
       t uint16 // Temporary VRAM address  
       x uint8  // Fine X scroll
       w bool   // Write toggle
       
       // Rendering state
       backgroundTileData [8]uint8
       spriteTileData [8][8]uint8
       
       // Flags
       nmiOccurred bool
       sprite0Hit bool
       spriteOverflow bool
   }
   ```

2. **Basic Rendering**
   - Background tile fetching
   - Simple sprite rendering (no priority)
   - Basic scrolling support

3. **VRAM Access**
   ```go
   func (ppu *PPU) readVRAM(address uint16) uint8 {
       address &= 0x3FFF
       switch {
       case address < 0x2000:
           // Pattern tables (CHR-ROM/RAM)
           return ppu.chr[address]
       case address < 0x3F00:
           // Nametables with mirroring
           return ppu.vram[ppu.mirrorNametableAddress(address)]
       case address < 0x4000:
           // Palette RAM with mirroring
           return ppu.paletteRAM[ppu.mirrorPaletteAddress(address)]
       }
       return 0
   }
   ```

### Phase 3: Memory System (Week 3-4)
1. **Bus Implementation**
   ```go
   type Bus struct {
       cpu    *cpu.CPU
       ppu    *ppu.PPU
       apu    *apu.APU
       ram    [0x800]uint8
       cartridge *cartridge.Cartridge
   }
   
   func (bus *Bus) Read(address uint16) uint8 {
       switch {
       case address < 0x2000:
           return bus.ram[address & 0x7FF] // RAM mirroring
       case address < 0x4000:
           return bus.ppu.ReadRegister(address) // PPU registers
       case address < 0x4020:
           return bus.readIORegister(address) // APU/Input
       default:
           return bus.cartridge.Read(address) // Cartridge
       }
   }
   ```

2. **Cartridge Loading**
   ```go
   type Cartridge struct {
       prgROM []uint8
       chrROM []uint8
       mapper Mapper
       mirroring MirroringType
   }
   
   type MirroringType int
   const (
       Horizontal MirroringType = iota
       Vertical
       SingleScreenA
       SingleScreenB
       FourScreen
   )
   ```

### Phase 4: Advanced Features (Week 4-6)
1. **Precise Timing**
   ```go
   func (nes *NES) Step() {
       // CPU runs 1 cycle
       if nes.cpu.remainingCycles == 0 {
           cycles := nes.cpu.Step()
           nes.cpu.remainingCycles = cycles - 1
       } else {
           nes.cpu.remainingCycles--
       }
       
       // PPU runs 3 cycles per CPU cycle (NTSC)
       for i := 0; i < 3; i++ {
           nes.ppu.Step()
       }
       
       // APU runs at CPU speed
       nes.apu.Step()
   }
   ```

2. **Interrupt Handling**
   ```go
   func (cpu *CPU) handleInterrupts() {
       if cpu.nmiRequested {
           cpu.nmi()
           cpu.nmiRequested = false
       } else if cpu.irqRequested && !cpu.I {
           cpu.irq()
           cpu.irqRequested = false
       }
   }
   ```

## Critical Implementation Details

### CPU Instruction Execution
```go
func (cpu *CPU) Step() uint8 {
    if cpu.remainingCycles > 0 {
        cpu.remainingCycles--
        return 1
    }
    
    // Handle interrupts before instruction fetch
    cpu.handleInterrupts()
    
    // Fetch instruction
    opcode := cpu.memory.Read(cpu.PC)
    instruction := instructionTable[opcode]
    
    // Execute instruction
    cpu.PC++
    cycles := cpu.executeInstruction(instruction)
    cpu.remainingCycles = cycles - 1
    
    return cycles
}
```

### PPU Register Access
```go
func (ppu *PPU) ReadRegister(address uint16) uint8 {
    switch address & 0x7 {
    case 0x2: // PPUSTATUS
        result := ppu.status
        ppu.status &= 0x7F // Clear VBlank flag
        ppu.w = false      // Reset write toggle
        return result
        
    case 0x7: // PPUDATA
        value := ppu.readVRAM(ppu.v)
        
        // Buffered reads (except palette)
        if (ppu.v & 0x3F00) == 0x3F00 {
            return value // Palette reads are immediate
        } else {
            result := ppu.readBuffer
            ppu.readBuffer = value
            return result
        }
    }
    return 0
}
```

### Timing Synchronization
```go
type NES struct {
    cpu *cpu.CPU
    ppu *ppu.PPU
    apu *apu.APU
    bus *bus.Bus
    
    // Timing
    cpuCycles uint64
    ppuCycles uint64
}

const (
    CPU_FREQUENCY_NTSC = 1789773  // Hz
    PPU_FREQUENCY_NTSC = 5369318  // Hz
    CYCLES_PER_FRAME   = 29781    // CPU cycles per frame
)
```

## Testing Strategy

### Unit Tests
```go
func TestCPUInstructions(t *testing.T) {
    cpu := cpu.New(NewMockMemory())
    
    // Test LDA immediate
    cpu.memory.Write(0x8000, 0xA9) // LDA #$42
    cpu.memory.Write(0x8001, 0x42)
    cpu.PC = 0x8000
    
    cycles := cpu.Step()
    assert.Equal(t, 2, cycles)
    assert.Equal(t, 0x42, cpu.A)
    assert.Equal(t, false, cpu.Z)
    assert.Equal(t, false, cpu.N)
}
```

### Integration Tests
```go
func TestNESLoadROM(t *testing.T) {
    nes := NewNES()
    
    // Load test ROM
    rom, err := LoadROM("test_roms/nestest.nes")
    require.NoError(t, err)
    
    nes.LoadCartridge(rom)
    nes.Reset()
    
    // Run for specific number of cycles
    for i := 0; i < 1000; i++ {
        nes.Step()
    }
    
    // Verify expected state
    assert.Equal(t, expectedPC, nes.cpu.PC)
}
```

## Performance Considerations

### Optimization Targets
1. **CPU Instruction Dispatch**
   - Use lookup table instead of switch statement
   - Inline simple addressing mode calculations

2. **PPU Rendering**
   - Batch pixel operations
   - Use bitwise operations for tile decoding
   - Cache pattern table data

3. **Memory Access**
   - Minimize bounds checking
   - Use memory mapping where possible

### Profiling Points
```go
// Add build tags for profiling
//go:build profile
func init() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}
```

## Next Steps

1. **Implement Core CPU Instructions** (Priority 1)
   - Focus on most common opcodes first
   - Add cycle-accurate timing
   - Include interrupt handling

2. **Complete PPU Register Behavior** (Priority 1)
   - Implement $2007 buffering
   - Add proper scroll register handling
   - Implement basic background rendering

3. **Add Cartridge Loading** (Priority 2)
   - Support iNES format
   - Implement NROM mapper
   - Add proper mirroring

4. **Testing Infrastructure** (Priority 2)
   - Add test ROM runner
   - Create automated test suite
   - Add timing validation

This implementation guide provides a concrete roadmap for building accurate NES emulation based on the detailed specifications provided in the other documents.