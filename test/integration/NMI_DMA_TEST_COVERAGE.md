# NMI Generation and OAM/DMA Systems Test Coverage

## Overview

This document outlines the comprehensive test coverage for NMI (Non-Maskable Interrupt) generation and OAM (Object Attribute Memory) DMA transfer functionality in the NES emulator. These tests define the exact requirements for proper interrupt handling and sprite data transfer that the system integration must implement.

## Test Files Created

### 1. `nmi_timing_precision_test.go`
**Purpose**: Validates cycle-accurate NMI generation timing

**Test Coverage**:
- **VBlank NMI Exact Timing**: Verifies NMI occurs at precise VBlank start (scanline 241, cycle 1)
- **Scanline 241 Cycle 1 Precision**: Tests exact timing coordination with PPU rendering pipeline
- **Multiple Frame NMI Consistency**: Validates consistent timing across multiple frames (~29781 CPU cycles per frame for NTSC)

**Key Validation Points**:
- NMI generation at exact VBlank timing
- PPU status flag coordination with NMI triggering
- Frame count synchronization
- Cycle-accurate timing measurements

### 2. `oam_dma_comprehensive_test.go`
**Purpose**: Tests comprehensive OAM DMA transfer behavior

**Test Coverage**:
- **Basic OAM DMA Transfer**: 256-byte sprite data transfer from RAM to OAM
- **DMA CPU Suspension Timing**: Validates CPU halts for exactly 513/514 cycles
- **DMA Even/Odd Cycle Timing**: Tests timing difference based on CPU cycle alignment
- **DMA Source Page Validation**: Transfer from different memory regions (RAM, ROM, registers)
- **Multiple Sequential DMA Transfers**: Rapid consecutive DMA operations

**Key Validation Points**:
- 256-byte transfer completion
- CPU suspension duration (513 cycles even alignment, 514 cycles odd alignment)
- Source page accessibility and mirroring
- DMA completion signaling

### 3. `nmi_dma_coordination_test.go`
**Purpose**: Tests interaction between NMI generation and DMA transfers

**Test Coverage**:
- **NMI During DMA Transfer**: Interrupt handling while DMA is active
- **DMA Delay NMI Timing**: Effect of DMA on NMI timing precision
- **Rapid DMA Near VBlank**: Multiple DMAs around VBlank period
- **NMI Priority Over DMA**: Interrupt priority and execution order

**Key Validation Points**:
- NMI execution during DMA suspension
- Timing coordination between systems
- Interrupt priority enforcement
- System stability under load

### 4. `interrupt_edge_cases_test.go`
**Purpose**: Tests complex interrupt scenarios and edge cases

**Test Coverage**:
- **NMI Suppression Critical Timing**: PPUSTATUS read timing that can suppress NMI
- **Multiple NMI Edge Detection**: Rapid enable/disable cycles and edge detection
- **DMA Halt During Critical Instructions**: Instruction atomicity during DMA
- **NMI IRQ Priority Complex**: Complex interrupt priority scenarios

**Key Validation Points**:
- PPUSTATUS read race conditions
- Edge-triggered interrupt behavior
- Instruction completion atomicity
- Interrupt vector handling

### 5. `sprite_dma_gameplay_test.go`
**Purpose**: Tests NMI and DMA behavior in realistic gameplay scenarios

**Test Coverage**:
- **Typical VBlank Sprite Update**: Standard game loop with VBlank sprite updates
- **Double Buffered Sprite System**: Advanced double-buffering techniques
- **Sprite Animation System**: Frame-by-frame sprite animation
- **Performance Critical DMA Timing**: High-performance scenarios

**Key Validation Points**:
- Real-world game loop functionality
- Double-buffering coordinate
- Animation timing accuracy
- Performance optimization validation

### 6. `nmi_dma_system_validation_test.go`
**Purpose**: Comprehensive system validation and requirements coverage

**Test Coverage**:
- **System Requirements Validation**: End-to-end functionality verification
- **Requirements Coverage Report**: Documentation of all tested requirements

**Key Validation Points**:
- Complete system integration
- All requirements coverage
- Pass/fail summary reporting

## Technical Requirements Tested

### NMI Generation Requirements
1. **Exact VBlank Timing**: NMI triggers at scanline 241, cycle 1
2. **Enable/Disable Control**: PPUCTRL bit 7 controls NMI generation
3. **Edge Detection**: NMI only triggers on VBlank flag transition
4. **Suppression Behavior**: PPUSTATUS reads can suppress NMI under specific timing
5. **Frame Consistency**: NMI occurs every frame with consistent timing

### OAM DMA Requirements
1. **256-Byte Transfer**: Complete sprite data transfer (64 sprites × 4 bytes)
2. **CPU Suspension**: CPU halted for 513/514 cycles during transfer
3. **Cycle Alignment**: Even cycles = 513, odd cycles = 514
4. **Source Page Access**: DMA can read from any memory page
5. **Transfer Completion**: DMA completion properly signaled

### Interrupt Coordination Requirements
1. **NMI Priority**: NMI has priority over IRQ
2. **DMA Coordination**: NMI can execute during DMA suspension
3. **Instruction Atomicity**: Instructions complete before DMA suspension
4. **State Preservation**: Register and stack state properly maintained
5. **Timing Precision**: Frame-accurate timing maintained

### Edge Case Requirements
1. **Race Conditions**: Proper handling of critical timing scenarios
2. **Multiple Interrupts**: Correct handling of simultaneous interrupt conditions
3. **Rapid Operations**: System stability under rapid DMA/NMI sequences
4. **Error Recovery**: Graceful handling of invalid or edge-case operations

## Test Execution Strategy

### Unit-Level Testing
- Individual component behavior validation
- Isolated timing measurements
- State verification

### Integration Testing
- Component interaction validation
- End-to-end functionality testing
- Performance measurement

### Scenario Testing
- Real-world game loop simulation
- Complex interaction patterns
- Stress testing under load

## Success Criteria

### Functional Requirements
- ✅ NMI generates at exact VBlank timing
- ✅ OAM DMA transfers 256 bytes correctly
- ✅ CPU suspension lasts exactly 513/514 cycles
- ✅ Interrupts coordinate properly
- ✅ Edge cases handled gracefully

### Performance Requirements
- ✅ Frame timing consistent at ~29781 CPU cycles/frame (NTSC)
- ✅ DMA timing accurate to single cycle
- ✅ No timing drift over multiple frames
- ✅ Minimal performance impact from DMA operations

### Compatibility Requirements
- ✅ Matches original NES hardware behavior
- ✅ Supports all standard gameplay patterns
- ✅ Handles edge cases like original hardware
- ✅ Maintains timing accuracy under all conditions

## Implementation Validation

These tests serve as the definitive specification for NMI and DMA system implementation. Any implementation must pass all tests to be considered correct and complete. The tests are designed to be:

1. **Comprehensive**: Cover all required functionality and edge cases
2. **Precise**: Validate exact timing and behavior requirements
3. **Realistic**: Test actual usage patterns from real games
4. **Maintainable**: Clear test structure and documentation

The test suite ensures that the NMI and DMA systems will correctly support all NES software and provide accurate emulation of the original hardware behavior.