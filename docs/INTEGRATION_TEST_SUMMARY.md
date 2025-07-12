# NES Emulator Integration Test Suite

## Overview

This comprehensive integration test suite validates the interaction between all major components of the NES emulator, ensuring proper timing, synchronization, and data flow between the CPU, PPU, memory, and I/O systems. The tests follow Test-Driven Development (TDD) principles and are designed to initially fail (red phase) until the full emulator implementation is complete.

## Test Files Created

### 1. `integration_test.go` - Main Integration Tests
**Scope**: System-level coordination and basic component interaction
- **TestSystemInitialization**: Verifies all components are created and initialized correctly
- **TestBasicExecution**: Tests CPU instruction execution through the full system
- **TestMemoryIntegration**: Validates memory access patterns across components
- **TestComponentCommunication**: Tests CPU-PPU, CPU-memory communication
- **TestErrorConditions**: Basic error handling and edge cases
- **TestSystemIntegrity**: Overall system stability and long-running tests

**Key Features**:
- `IntegrationTestHelper` with system bus, mock cartridge, and test utilities
- `MockCartridge` that tracks PRG/CHR access patterns
- Comprehensive ROM setup with proper interrupt vectors and basic programs

### 2. `system_timing_test.go` - CPU-PPU Timing Synchronization
**Scope**: Validates the critical 3:1 CPU-PPU timing relationship (NTSC)
- **TestCPUPPUTimingSynchronization**: Verifies PPU runs at exactly 3x CPU speed
- **TestFrameTimingSynchronization**: Tests frame-level timing coordination
- **TestInstructionLevelTiming**: Validates cycle-accurate instruction execution
- **TestTimingEdgeCases**: Tests DMA timing, interrupt coordination, reset timing

**Key Features**:
- `TimingTestHelper` with cycle counting and ratio verification
- Tests various instruction types and their timing impact on PPU
- Page boundary crossing timing validation
- Frame rate calculation and verification (60.1 Hz NTSC)

### 3. `memory_integration_test.go` - Cross-Component Memory Access
**Scope**: Memory system integration across all components
- **TestCrossComponentMemoryAccess**: CPU-PPU register communication, cartridge access
- **TestMemoryMirroring**: RAM mirroring, PPU register mirroring, nametable mirroring
- **TestConcurrentMemoryAccess**: CPU and PPU simultaneous memory access, DMA coordination

**Key Features**:
- `MemoryIntegrationHelper` with access logging
- Tests all memory regions: RAM, PPU registers, cartridge PRG/CHR
- Validates memory mirroring modes (horizontal, vertical, single-screen)
- Stress testing for memory system stability

### 4. `dma_integration_test.go` - DMA Coordination Tests
**Scope**: OAM DMA integration between CPU and PPU
- **TestOAMDMAIntegration**: Basic DMA transfer, timing accuracy, source page variations
- **TestDMABusArbitration**: CPU suspension, PPU continuation, multiple rapid transfers
- **TestDMAEdgeCases**: Unmapped memory sources, DMA during interrupts, cycle count accuracy

**Key Features**:
- `DMATestHelper` with DMA transfer logging
- Tests DMA from all memory regions (RAM, ROM, unmapped areas)
- Validates 513/514 cycle DMA timing based on alignment
- CPU-PPU coordination during DMA operations

### 5. `nmi_integration_test.go` - NMI Interrupt Handling
**Scope**: NMI generation from PPU to CPU interrupt handling
- **TestNMIGeneration**: VBlank NMI generation, enable/disable control, timing precision
- **TestNMIInterruptBehavior**: Interrupt sequence, register preservation, instruction interruption
- **TestNMITiming**: VBlank to NMI delay, PPUSTATUS read race conditions

**Key Features**:
- `NMITestHelper` with NMI event logging
- Comprehensive NMI handler setup and execution
- Tests critical timing windows and race conditions
- Edge detection and NMI suppression scenarios

### 6. `system_boot_test.go` - Boot and Initialization
**Scope**: Complete system boot sequence validation
- **TestSystemBoot**: Complete boot sequence, power-on state, reset behavior
- **TestBootComponentInitialization**: Individual component initialization sequences
- **TestBootTiming**: Boot duration, frame alignment during initialization

**Key Features**:
- `BootTestHelper` with comprehensive boot ROM
- Proper interrupt vector setup and CPU initialization
- PPU stabilization (2-frame wait) and memory clearing
- Boot sequence timing validation

### 7. `frame_timing_test.go` - Frame Timing Validation
**Scope**: NTSC frame timing accuracy and synchronization
- **TestNTSCFrameTiming**: Frame rate accuracy (60.098803 Hz), frame consistency
- **TestFrameTimingSynchronization**: CPU-PPU frame sync, frame drop detection
- **TestFrameTimingEdgeCases**: Rendering toggle, interrupt timing, odd frame cycle skip

**Key Features**:
- `FrameTimingHelper` with frame event tracking
- Statistical analysis of frame timing consistency
- VBlank timing precision validation
- Frame rate calculation and verification

### 8. `error_integration_test.go` - Error Conditions and Edge Cases
**Scope**: System robustness under error conditions
- **TestErrorConditions**: Invalid memory access, stack overflow/underflow, infinite loops
- **TestResourceExhaustion**: Memory stress, rapid PPU access, excessive DMA
- **TestTimingViolations**: PPU access during rendering, interrupt edge cases
- **TestSystemRecovery**: Recovery from stack corruption, timing violations

**Key Features**:
- `ErrorTestHelper` with error event logging
- Comprehensive error condition simulation
- System stability verification under stress
- Recovery mechanism validation

## Test Design Principles

### Test-Driven Development (TDD)
- **Red Phase**: All tests are designed to fail initially until proper implementation
- **Immutable Test Specifications**: Tests define the required behavior, implementation must conform
- **Data-Agnostic Implementation**: No hardcoded test data in implementation logic

### Integration Test Philosophy
- **System-Level Validation**: Tests entire component interaction chains
- **Timing-Critical Testing**: Validates cycle-accurate emulation requirements
- **Edge Case Coverage**: Tests boundary conditions and error scenarios
- **Realistic Workloads**: Uses NES-like programs and access patterns

### Component Coordination Testing
- **CPU-PPU Synchronization**: 3:1 timing ratio validation
- **Memory Bus Arbitration**: Concurrent access handling
- **Interrupt Coordination**: NMI generation and handling
- **DMA Integration**: CPU suspension with PPU continuation

## Critical Integration Scenarios Tested

### 1. CPU-PPU Synchronization (3:1 Ratio NTSC)
- PPU executes exactly 3 cycles for every CPU cycle
- Frame timing: 341 PPU cycles/scanline × 262 scanlines = 89,342 PPU cycles/frame
- CPU timing: 29,780.67 CPU cycles/frame
- Page boundary crossing effects on timing

### 2. Memory Integration
- CPU access to PPU registers ($2000-$3FFF with mirroring)
- PPU access to VRAM through memory interface
- Cartridge integration (PRG ROM/RAM, CHR ROM/RAM)
- Memory mirroring validation (RAM, PPU registers, nametables)

### 3. OAM DMA Coordination
- CPU suspension for 513/514 cycles (even/odd alignment)
- PPU continues operation during DMA
- 256-byte transfer from any memory source to OAM
- Bus arbitration between CPU and DMA

### 4. NMI Integration
- VBlank flag generation at scanline 241, cycle 1
- NMI edge detection and timing precision
- PPUSTATUS read race conditions
- Interrupt handling with register preservation

### 5. System Boot
- CPU reset sequence and initialization
- PPU stabilization (2-frame wait)
- Memory clearing (RAM, OAM, VRAM)
- Interrupt vector setup and validation

### 6. Frame Timing
- NTSC frame rate: 60.098803 Hz
- Frame consistency and jitter analysis
- Odd frame cycle skip with rendering enabled
- VBlank synchronization

### 7. Error Handling
- Invalid memory access patterns
- Stack overflow/underflow conditions
- PPU timing violations
- System recovery mechanisms

## Expected Test Outcomes (TDD Red Phase)

Currently, all tests should **FAIL** because:
1. Component interfaces may not be fully implemented
2. Timing synchronization is not yet cycle-accurate
3. DMA implementation may be incomplete
4. NMI generation and handling may not be implemented
5. Memory mirroring may not be fully functional
6. Frame timing may not be precise

## Implementation Guidance

### To Move to TDD Green Phase:
1. **Implement Missing Interfaces**: Ensure all component interfaces are fully implemented
2. **Add Cycle Tracking**: Implement precise cycle counting in CPU and PPU
3. **Implement DMA**: Add OAM DMA with proper CPU suspension and timing
4. **Add NMI Generation**: Implement VBlank NMI generation in PPU
5. **Complete Memory Mirroring**: Implement all memory mirroring modes
6. **Add Frame Timing**: Implement precise NTSC timing with odd frame skip

### Key Integration Points:
- **Bus Coordination**: Ensure proper component step coordination (CPU:PPU 1:3 ratio)
- **Memory Interface**: Complete memory map with proper mirroring
- **Interrupt System**: Link PPU VBlank to CPU NMI handling
- **DMA System**: Implement cycle-accurate OAM DMA
- **Timing Precision**: Ensure all operations are cycle-accurate

## Test Execution

To run the integration tests (once basic implementation is complete):

```bash
# Run all integration tests
go test -v integration_test.go system_timing_test.go memory_integration_test.go dma_integration_test.go nmi_integration_test.go system_boot_test.go frame_timing_test.go error_integration_test.go

# Run specific test categories
go test -v integration_test.go -run TestSystemInitialization
go test -v system_timing_test.go -run TestCPUPPUTimingSynchronization
go test -v memory_integration_test.go -run TestCrossComponentMemoryAccess
go test -v dma_integration_test.go -run TestOAMDMAIntegration
go test -v nmi_integration_test.go -run TestNMIGeneration
```

## Framework for Full System Validation

This test suite provides a comprehensive framework for validating:
- **Functional Correctness**: All components work together correctly
- **Timing Accuracy**: Cycle-accurate emulation requirements are met
- **System Stability**: Robust operation under various conditions
- **Edge Case Handling**: Proper behavior in boundary conditions
- **Integration Quality**: Seamless component interaction

The tests serve as both validation tools and specification documents, ensuring the emulator meets NES hardware requirements for timing, synchronization, and functionality.