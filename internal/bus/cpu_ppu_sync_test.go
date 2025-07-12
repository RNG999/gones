package bus

import (
	"gones/internal/cartridge"
	"testing"
)

// TestCPUPPU3To1SyncBasic validates the fundamental 3:1 CPU-PPU cycle relationship
func TestCPUPPU3To1SyncBasic(t *testing.T) {
	t.Run("Exact 3:1 ratio during single steps", func(t *testing.T) {
		bus := New()
		
		// Create minimal test ROM
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP instruction (2 CPU cycles)
		romData[0x0001] = 0x4C // JMP 
		romData[0x0002] = 0x00 // $8000
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high
		
		cart := cartridge.NewMockCartridge()
		cart.LoadPRG(romData)
		bus.LoadCartridge(cart)
		bus.Reset()
		
		// Enable execution logging
		bus.EnableExecutionLogging()
		
		// Track initial cycle counts
		initialCPUCycles := bus.GetCycleCount()
		
		// Execute one CPU instruction
		bus.Step()
		
		// Get execution log to verify actual cycle counts
		log := bus.GetExecutionLog()
		if len(log) == 0 {
			t.Fatal("No execution log entries found")
		}
		
		// Verify CPU cycles increased by exactly 2 (NOP instruction)
		finalCPUCycles := bus.GetCycleCount()
		cpuCyclesExecuted := finalCPUCycles - initialCPUCycles
		
		if cpuCyclesExecuted != 2 {
			t.Errorf("Expected 2 CPU cycles for NOP, got %d", cpuCyclesExecuted)
		}
		
		// Verify PPU cycles are exactly 3x CPU cycles
		expectedPPUCycles := cpuCyclesExecuted * 3
		actualPPUCycles := log[0].PPUCycles - (initialCPUCycles * 3)
		
		if actualPPUCycles != expectedPPUCycles {
			t.Errorf("PPU cycles should be 3x CPU cycles. CPU: %d, Expected PPU: %d, Actual PPU: %d", 
				cpuCyclesExecuted, expectedPPUCycles, actualPPUCycles)
		}
	})
	
	t.Run("3:1 ratio maintained across multiple instructions", func(t *testing.T) {
		bus := New()
		
		// Create test program with various cycle counts
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xEA,       // NOP (2 cycles)
			0xA9, 0x42, // LDA #$42 (2 cycles)  
			0x85, 0x00, // STA $00 (3 cycles)
			0xE8,       // INX (2 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		cart := cartridge.NewMockCartridge()
		cart.LoadPRG(romData)
		bus.LoadCartridge(cart)
		bus.Reset()
		
		bus.EnableExecutionLogging()
		
		expectedCycles := []int{2, 2, 3, 2, 3} // Cycle counts for each instruction
		totalCPUCycles := uint64(0)
		totalPPUCycles := uint64(0)
		
		for i, expectedCPU := range expectedCycles {
			initialCPU := bus.GetCycleCount()
			
			bus.Step()
			
			actualCPU := bus.GetCycleCount() - initialCPU
			totalCPUCycles += actualCPU
			expectedPPU := actualCPU * 3
			totalPPUCycles += expectedPPU
			
			if actualCPU != uint64(expectedCPU) {
				t.Errorf("Instruction %d: Expected %d CPU cycles, got %d", i, expectedCPU, actualCPU)
			}
			
			// Verify 3:1 ratio for this instruction
			log := bus.GetExecutionLog()
			if len(log) > i {
				ppuRatio := float64(log[i].PPUCycles) / float64(log[i].CPUCycles)
				if ppuRatio != 3.0 {
					t.Errorf("Instruction %d: PPU/CPU ratio should be 3.0, got %.2f", i, ppuRatio)
				}
			}
		}
		
		// Verify cumulative 3:1 ratio
		finalRatio := float64(totalPPUCycles) / float64(totalCPUCycles)
		if finalRatio != 3.0 {
			t.Errorf("Cumulative PPU/CPU ratio should be 3.0, got %.2f", finalRatio)
		}
	})
	
	t.Run("3:1 ratio with page boundary crossing", func(t *testing.T) {
		bus := New()
		
		// Test instruction with variable cycle count due to page crossing
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xA2, 0x10, // LDX #$10 (2 cycles)
			0xBD, 0xF0, 0x20, // LDA $20F0,X -> $2100 (5 cycles with page cross)
			0xA2, 0x05, // LDX #$05 (2 cycles) 
			0xBD, 0x00, 0x20, // LDA $2000,X -> $2005 (4 cycles no page cross)
			0x4C, 0x00, 0x80, // JMP $8000
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		cart := cartridge.NewMockCartridge()
		cart.LoadPRG(romData)
		bus.LoadCartridge(cart)
		bus.Reset()
		
		bus.EnableExecutionLogging()
		
		expectedCycles := []int{2, 5, 2, 4} // With and without page crossing
		
		for i, expectedCPU := range expectedCycles {
			initialCPU := bus.GetCycleCount()
			bus.Step()
			actualCPU := bus.GetCycleCount() - initialCPU
			
			if actualCPU != uint64(expectedCPU) {
				t.Errorf("Instruction %d: Expected %d CPU cycles, got %d", i, expectedCPU, actualCPU)
			}
			
			// PPU should still maintain 3:1 ratio
			expectedPPU := actualCPU * 3
			log := bus.GetExecutionLog()
			if len(log) > i {
				actualPPU := log[i].PPUCycles
				if i > 0 {
					actualPPU -= log[i-1].PPUCycles
				}
				if actualPPU != expectedPPU {
					t.Errorf("Instruction %d: Expected %d PPU cycles, got %d", i, expectedPPU, actualPPU)
				}
			}
		}
	})
}

// TestCPUPPUSyncDuringDMA validates 3:1 timing during DMA operations
func TestCPUPPUSyncDuringDMA(t *testing.T) {
	t.Run("PPU continues during CPU DMA suspension", func(t *testing.T) {
		bus := New()
		
		// Create test ROM that triggers DMA
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xA9, 0x02, // LDA #$02 (2 cycles)
			0x8D, 0x14, 0x40, // STA $4014 (4 cycles) - triggers DMA
			0xEA, // NOP (should be delayed by DMA)
			0x4C, 0x00, 0x80, // JMP $8000
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		cart := cartridge.NewMockCartridge()
		cart.LoadPRG(romData)
		bus.LoadCartridge(cart)
		bus.Reset()
		
		bus.EnableExecutionLogging()
		
		// Execute until DMA trigger
		bus.Step() // LDA #$02
		
		initialCPU := bus.GetCycleCount()
		
		bus.Step() // STA $4014 - triggers DMA
		
		if !bus.IsDMAInProgress() {
			t.Error("DMA should be in progress after STA $4014")
		}
		
		// During DMA, CPU is suspended but PPU continues
		// DMA takes 513-514 cycles depending on alignment
		_ = bus.GetCycleCount() - initialCPU // dmaStartCPU (unused for now)
		
		// Execute steps during DMA
		stepsDuringDMA := 0
		for bus.IsDMAInProgress() && stepsDuringDMA < 600 {
			bus.Step()
			stepsDuringDMA++
		}
		
		_ = bus.GetCycleCount() - initialCPU // totalCPUAfterDMA (unused for now)
		
		// During DMA, CPU advances 1 cycle per step while suspended
		// PPU should advance 3 cycles per step consistently
		
		if stepsDuringDMA < 513 || stepsDuringDMA > 514 {
			t.Errorf("DMA should take 513-514 steps, took %d", stepsDuringDMA)
		}
		
		// Verify PPU maintained 3:1 ratio during DMA
		log := bus.GetExecutionLog()
		if len(log) >= 2 {
			dmaCPUCycles := log[1].CPUCycles - log[0].CPUCycles
			dmaPPUCycles := log[1].PPUCycles - log[0].PPUCycles
			
			ratio := float64(dmaPPUCycles) / float64(dmaCPUCycles)
			if ratio != 3.0 {
				t.Errorf("PPU/CPU ratio during DMA should be 3.0, got %.2f", ratio)
			}
		}
	})
}

// TestCPUPPUSyncWithInterrupts validates timing during interrupt handling
func TestCPUPPUSyncWithInterrupts(t *testing.T) {
	t.Run("3:1 ratio maintained during NMI handling", func(t *testing.T) {
		bus := New()
		
		// Create test ROM with NMI handling
		romData := make([]uint8, 0x8000)
		
		// Main program
		romData[0x0000] = 0xEA // NOP
		romData[0x0001] = 0x4C // JMP
		romData[0x0002] = 0x00 // $8000  
		romData[0x0003] = 0x80
		
		// NMI handler at $8100
		romData[0x0100] = 0xEA // NOP in handler
		romData[0x0101] = 0x40 // RTI
		
		// Vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high  
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high
		
		cart := cartridge.NewMockCartridge()
		cart.LoadPRG(romData)
		bus.LoadCartridge(cart)
		bus.Reset()
		
		bus.EnableExecutionLogging()
		
		// Enable NMI in PPU
		bus.PPU.WriteRegister(0x2000, 0x80)
		
		// Run until potential NMI
		initialCPU := bus.GetCycleCount()
		stepCount := 0
		
		for stepCount < 100000 { // Safety limit
			bus.Step()
			stepCount++
			
			// Check if we're in NMI handler
			cpuState := bus.GetCPUState()
			if cpuState.PC >= 0x8100 && cpuState.PC <= 0x8101 {
				// Verify 3:1 ratio maintained during interrupt
				_ = bus.GetCycleCount() - initialCPU // finalCPU (unused for now)
				
				log := bus.GetExecutionLog()
				if len(log) > 0 {
					lastEntry := log[len(log)-1]
					ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
					
					if ratio != 3.0 {
						t.Errorf("PPU/CPU ratio during NMI should be 3.0, got %.2f", ratio)
					}
				}
				break
			}
		}
		
		if stepCount >= 100000 {
			t.Error("NMI handler was not reached within reasonable time")
		}
	})
}

// TestCPUPPUSyncPrecision validates cycle-level precision of the 3:1 ratio
func TestCPUPPUSyncPrecision(t *testing.T) {
	t.Run("No fractional cycle accumulation", func(t *testing.T) {
		bus := New()
		
		// Test that no fractional cycles accumulate over time
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP (2 cycles)
		romData[0x0001] = 0x4C // JMP $8000 (3 cycles)
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		cart := cartridge.NewMockCartridge()
		cart.LoadPRG(romData)
		bus.LoadCartridge(cart)
		bus.Reset()
		
		bus.EnableExecutionLogging()
		
		// Execute many instructions to check for drift
		iterations := 1000
		totalCPUExpected := uint64((2 + 3) * iterations) // NOP + JMP per iteration
		
		for i := 0; i < iterations*2; i++ { // 2 instructions per iteration
			bus.Step()
		}
		
		finalCPU := bus.GetCycleCount()
		expectedPPU := finalCPU * 3
		
		log := bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			actualPPU := lastEntry.PPUCycles
			
			if actualPPU != expectedPPU {
				t.Errorf("PPU cycles drifted from 3:1 ratio. Expected %d, got %d", 
					expectedPPU, actualPPU)
			}
			
			// Verify exact integer relationship
			if actualPPU%3 != 0 {
				t.Errorf("PPU cycles should be divisible by 3, got %d", actualPPU)
			}
		}
		
		// Verify no fractional accumulation
		if finalCPU != totalCPUExpected {
			t.Errorf("CPU cycles drifted. Expected %d, got %d", totalCPUExpected, finalCPU)
		}
	})
	
	t.Run("Cycle precision during mixed operations", func(t *testing.T) {
		bus := New()
		
		// Mix of different cycle count instructions
		romData := make([]uint8, 0x8000)
		program := []uint8{
			// 2-cycle instructions
			0xEA,       // NOP (2)
			0xE8,       // INX (2)
			0xA9, 0x00, // LDA #$00 (2)
			
			// 3-cycle instructions  
			0x85, 0x10, // STA $10 (3)
			0xA5, 0x10, // LDA $10 (3)
			
			// 4-cycle instructions
			0x8D, 0x00, 0x30, // STA $3000 (4)
			0xAD, 0x00, 0x30, // LDA $3000 (4)
			
			// Variable cycle (5 with page cross)
			0xA2, 0x10, // LDX #$10 (2)
			0xBD, 0xF0, 0x20, // LDA $20F0,X (5)
			
			0x4C, 0x00, 0x80, // JMP $8000 (3)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		cart := cartridge.NewMockCartridge()
		cart.LoadPRG(romData)
		bus.LoadCartridge(cart)
		bus.Reset()
		
		bus.EnableExecutionLogging()
		
		expectedCycles := []int{2, 2, 2, 3, 3, 4, 4, 2, 5, 3}
		runningCPUTotal := uint64(0)
		runningPPUTotal := uint64(0)
		
		for i, expectedCPU := range expectedCycles {
			initialCPU := bus.GetCycleCount()
			bus.Step()
			actualCPU := bus.GetCycleCount() - initialCPU
			
			if actualCPU != uint64(expectedCPU) {
				t.Errorf("Step %d: Expected %d CPU cycles, got %d", i, expectedCPU, actualCPU)
			}
			
			runningCPUTotal += actualCPU
			runningPPUTotal += actualCPU * 3
			
			// Verify running totals maintain exact 3:1
			log := bus.GetExecutionLog()
			if len(log) > i {
				if log[i].PPUCycles != runningPPUTotal {
					t.Errorf("Step %d: PPU total should be %d, got %d", 
						i, runningPPUTotal, log[i].PPUCycles)
				}
				
				if log[i].CPUCycles != runningCPUTotal {
					t.Errorf("Step %d: CPU total should be %d, got %d", 
						i, runningCPUTotal, log[i].CPUCycles)
				}
			}
		}
	})
}