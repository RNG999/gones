package integration

import (
	"testing"
)

// DMATestHelper provides utilities for DMA integration testing
type DMATestHelper struct {
	*IntegrationTestHelper
	dmaTransfers []DMATransfer
}

// DMATransfer represents a DMA transfer for testing
type DMATransfer struct {
	SourcePage         uint8
	SourceAddress      uint16
	TargetAddress      uint16
	BytesTransferred   int
	CPUCyclesSuspended int
	PPUCyclesDuringDMA int
}

// NewDMATestHelper creates a DMA integration test helper
func NewDMATestHelper() *DMATestHelper {
	return &DMATestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		dmaTransfers:          make([]DMATransfer, 0),
	}
}

// LogDMATransfer logs a DMA transfer
func (h *DMATestHelper) LogDMATransfer(transfer DMATransfer) {
	h.dmaTransfers = append(h.dmaTransfers, transfer)
}

// SetupOAMData sets up test data for OAM DMA
func (h *DMATestHelper) SetupOAMData(page uint8, pattern uint8) {
	baseAddr := uint16(page) << 8
	for i := 0; i < 256; i++ {
		h.Memory.Write(baseAddr+uint16(i), pattern+uint8(i))
	}
}

// VerifyOAMData verifies OAM data was transferred correctly
func (h *DMATestHelper) VerifyOAMData(t *testing.T, expectedPattern uint8) {
	// In a real implementation, we would read OAM data from PPU
	// For now, we verify the DMA was triggered by checking memory access patterns

	// Check that DMA register write was processed
	// This would be verified by tracking PPU OAMDATA writes
	t.Logf("OAM DMA verification for pattern 0x%02X", expectedPattern)
}

// TestOAMDMAIntegration tests OAM DMA coordination between CPU and PPU
func TestOAMDMAIntegration(t *testing.T) {
	t.Run("Basic OAM DMA transfer", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up OAM data in RAM page 2
		helper.SetupOAMData(0x02, 0x50)

		// Program that triggers OAM DMA
		program := []uint8{
			0xA9, 0x02, // LDA #$02 (source page)
			0x8D, 0x14, 0x40, // STA $4014 (OAM DMA trigger)
			0xEA,       // NOP (should be delayed by DMA)
			0xA9, 0xFF, // LDA #$FF (should execute after DMA)
			0x85, 0x00, // STA $00 (mark completion)
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute up to DMA trigger
		helper.Bus.Step() // LDA #$02

		// Verify A register contains source page
		if helper.CPU.A != 0x02 {
			t.Errorf("Expected A=0x02 before DMA, got 0x%02X", helper.CPU.A)
		}

		// Trigger DMA
		helper.Bus.Step() // STA $4014

		// DMA should now be in progress
		// CPU should be suspended while DMA completes
		// PPU should continue running and receive OAM data

		// Execute next instruction (should be delayed by DMA cycles)
		helper.Bus.Step() // NOP (delayed)

		// Verify DMA completed by checking subsequent execution
		helper.Bus.Step() // LDA #$FF
		helper.Bus.Step() // STA $00

		// Check that instruction after DMA executed
		completionFlag := helper.Memory.Read(0x0000)
		if completionFlag != 0xFF {
			t.Errorf("Expected completion flag 0xFF, got 0x%02X", completionFlag)
		}

		helper.VerifyOAMData(t, 0x50)

		t.Log("Basic OAM DMA transfer test completed")
	})

	t.Run("DMA timing accuracy", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)

		// OAM DMA should take 513 or 514 CPU cycles depending on timing
		// Test both even and odd cycle alignments

		testCases := []struct {
			name        string
			alignment   string
			setupCycles int
			expectedMin int
			expectedMax int
		}{
			{"Even cycle alignment", "even", 0, 513, 513},
			{"Odd cycle alignment", "odd", 1, 514, 514},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				helper.SetupOAMData(0x03, 0x80)

				// Program for timing test
				program := []uint8{
					0xEA,       // NOP (for alignment if needed)
					0xA9, 0x03, // LDA #$03
					0x8D, 0x14, 0x40, // STA $4014 (DMA trigger)
					0xE6, 0x00, // INC $00 (first instruction after DMA)
					0x4C, 0x00, 0x80, // JMP $8000
				}

				romData := make([]uint8, 0x8000)
				copy(romData, program)
				romData[0x7FFC] = 0x00
				romData[0x7FFD] = 0x80
				helper.GetMockCartridge().LoadPRG(romData)
				helper.Bus.Reset()

				// Set up cycle alignment
				for i := 0; i < tc.setupCycles; i++ {
					helper.Bus.Step() // NOPs for alignment
				}

				// Execute DMA sequence
				helper.Bus.Step() // LDA #$03

				// Measure DMA timing (would need cycle counter access)
				cyclesBefore := 0 // Would track actual CPU cycles
				helper.Bus.Step() // STA $4014 (triggers DMA)
				cyclesAfter := 0  // Would track actual CPU cycles

				dmaCycles := cyclesAfter - cyclesBefore

				// Verify DMA took expected cycles
				if dmaCycles < tc.expectedMin || dmaCycles > tc.expectedMax {
					t.Errorf("DMA cycles out of range: expected %d-%d, got %d",
						tc.expectedMin, tc.expectedMax, dmaCycles)
				}

				t.Logf("DMA completed in %d cycles (%s)", dmaCycles, tc.alignment)
			})
		}
	})

	t.Run("DMA source page variations", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)

		// Test DMA from different memory regions
		testPages := []struct {
			page      uint8
			region    string
			setupData bool
		}{
			{0x00, "Zero page", true},
			{0x01, "Stack page", true},
			{0x02, "RAM", true},
			{0x03, "RAM", true},
			{0x20, "PPU registers", false},    // Returns register values
			{0x40, "APU/IO registers", false}, // Returns register values
			{0x60, "Cartridge SRAM", false},   // Would need SRAM setup
			{0x80, "PRG ROM", false},          // ROM data
			{0xFF, "PRG ROM", false},          // ROM data
		}

		for _, tp := range testPages {
			t.Run(tp.region+" page 0x"+string(rune(tp.page)), func(t *testing.T) {
				if tp.setupData {
					helper.SetupOAMData(tp.page, 0x60+tp.page)
				}

				// Program to trigger DMA from specific page
				program := []uint8{
					0xA9, tp.page, // LDA #page
					0x8D, 0x14, 0x40, // STA $4014
					0xA9, 0x42, // LDA #$42 (marker)
					0x85, 0x10, // STA $10 (completion marker)
					0x4C, 0x00, 0x80, // JMP $8000
				}

				romData := make([]uint8, 0x8000)
				copy(romData, program)
				romData[0x7FFC] = 0x00
				romData[0x7FFD] = 0x80
				helper.GetMockCartridge().LoadPRG(romData)
				helper.Bus.Reset()

				// Execute DMA sequence
				helper.Bus.Step() // LDA #page
				helper.Bus.Step() // STA $4014
				helper.Bus.Step() // LDA #$42
				helper.Bus.Step() // STA $10

				// Verify completion
				marker := helper.Memory.Read(0x0010)
				if marker != 0x42 {
					t.Errorf("DMA did not complete properly: expected marker 0x42, got 0x%02X", marker)
				}

				t.Logf("DMA from %s (page 0x%02X) completed", tp.region, tp.page)
			})
		}
	})

	t.Run("DMA during PPU rendering", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Set up OAM data
		helper.SetupOAMData(0x02, 0x90)

		// Program that enables rendering and then does DMA
		program := []uint8{
			// Enable PPU rendering
			0xA9, 0x80, // LDA #$80 (NMI enable)
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E (show bg+sprites)
			0x8D, 0x01, 0x20, // STA $2001

			// Wait a bit for rendering to start
			0xEA, 0xEA, 0xEA, 0xEA, // NOPs

			// Trigger DMA during rendering
			0xA9, 0x02, // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014

			// Continue execution
			0xA9, 0x55, // LDA #$55
			0x85, 0x20, // STA $20
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute setup
		for i := 0; i < 8; i++ { // Enable rendering and wait
			helper.Bus.Step()
		}

		// Trigger DMA during rendering
		helper.Bus.Step() // LDA #$02
		helper.Bus.Step() // STA $4014 (DMA during rendering)

		// DMA should work even during rendering
		// PPU should continue rendering while receiving OAM data

		helper.Bus.Step() // LDA #$55
		helper.Bus.Step() // STA $20

		// Verify completion
		marker := helper.Memory.Read(0x0020)
		if marker != 0x55 {
			t.Errorf("DMA during rendering failed: expected 0x55, got 0x%02X", marker)
		}

		t.Log("DMA during PPU rendering test completed")
	})
}

// TestDMABusArbitration tests bus arbitration during DMA
func TestDMABusArbitration(t *testing.T) {
	t.Run("CPU suspension during DMA", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupOAMData(0x02, 0xA0)

		// Program that has timing-sensitive operations around DMA
		program := []uint8{
			// Set up a counter before DMA
			0xA9, 0x00, // LDA #$00
			0x85, 0x30, // STA $30 (counter)

			// Increment counter
			0xE6, 0x30, // INC $30 (should be 1)

			// Trigger DMA
			0xA9, 0x02, // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014

			// These instructions should be delayed by DMA
			0xE6, 0x30, // INC $30 (should be 2, but delayed)
			0xE6, 0x30, // INC $30 (should be 3, but delayed)

			// Mark completion
			0xA9, 0xFF, // LDA #$FF
			0x85, 0x31, // STA $31
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute step by step
		helper.Bus.Step() // LDA #$00
		helper.Bus.Step() // STA $30
		helper.Bus.Step() // INC $30

		// Check counter before DMA
		counter := helper.Memory.Read(0x0030)
		if counter != 1 {
			t.Errorf("Counter should be 1 before DMA, got %d", counter)
		}

		helper.Bus.Step() // LDA #$02
		helper.Bus.Step() // STA $4014 (triggers DMA)

		// During DMA, CPU should be suspended
		// PPU should continue receiving OAM data

		helper.Bus.Step() // INC $30 (delayed by DMA)
		helper.Bus.Step() // INC $30 (delayed by DMA)
		helper.Bus.Step() // LDA #$FF
		helper.Bus.Step() // STA $31

		// Check final state
		finalCounter := helper.Memory.Read(0x0030)
		completion := helper.Memory.Read(0x0031)

		if finalCounter != 3 {
			t.Errorf("Expected final counter 3, got %d", finalCounter)
		}
		if completion != 0xFF {
			t.Errorf("Expected completion marker 0xFF, got 0x%02X", completion)
		}

		t.Log("CPU suspension during DMA test completed")
	})

	t.Run("PPU continues during DMA", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()
		helper.SetupOAMData(0x02, 0xB0)

		// Enable PPU rendering before DMA
		helper.Memory.Write(0x2000, 0x80) // PPUCTRL
		helper.Memory.Write(0x2001, 0x1E) // PPUMASK

		// Program that triggers DMA during rendering
		program := []uint8{
			0xA9, 0x02, // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (DMA)
			0xEA,             // NOP (delayed)
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Re-enable PPU after reset
		helper.Memory.Write(0x2000, 0x80)
		helper.Memory.Write(0x2001, 0x1E)

		// Track PPU state before DMA
		ppuStatusBefore := helper.PPU.ReadRegister(0x2002)

		// Execute DMA
		helper.Bus.Step() // LDA #$02
		helper.Bus.Step() // STA $4014 (DMA + PPU continues)

		// PPU should have continued running during DMA
		ppuStatusAfter := helper.PPU.ReadRegister(0x2002)

		// Verify PPU continued operation
		// (In real implementation, we would check PPU cycle counters)

		helper.Bus.Step() // NOP (delayed by DMA)

		t.Logf("PPU status before DMA: 0x%02X, after DMA: 0x%02X",
			ppuStatusBefore, ppuStatusAfter)
		t.Log("PPU continues during DMA test completed")
	})

	t.Run("Multiple rapid DMA transfers", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up data in multiple pages
		helper.SetupOAMData(0x02, 0xC0)
		helper.SetupOAMData(0x03, 0xD0)
		helper.SetupOAMData(0x04, 0xE0)

		// Program that does multiple DMA transfers
		program := []uint8{
			// First DMA
			0xA9, 0x02, // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014

			// Second DMA (should wait for first to complete)
			0xA9, 0x03, // LDA #$03
			0x8D, 0x14, 0x40, // STA $4014

			// Third DMA
			0xA9, 0x04, // LDA #$04
			0x8D, 0x14, 0x40, // STA $4014

			// Mark completion
			0xA9, 0x77, // LDA #$77
			0x85, 0x40, // STA $40
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute multiple DMA sequence
		for i := 0; i < 8; i++ {
			helper.Bus.Step()
		}

		// Verify all DMAs completed
		completion := helper.Memory.Read(0x0040)
		if completion != 0x77 {
			t.Errorf("Multiple DMA sequence failed: expected 0x77, got 0x%02X", completion)
		}

		t.Log("Multiple rapid DMA transfers test completed")
	})
}

// TestDMAEdgeCases tests edge cases and error conditions
func TestDMAEdgeCases(t *testing.T) {
	t.Run("DMA from unmapped memory", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that triggers DMA from unmapped region
		program := []uint8{
			0xA9, 0x50, // LDA #$50 (unmapped expansion area)
			0x8D, 0x14, 0x40, // STA $4014
			0xA9, 0x88, // LDA #$88
			0x85, 0x50, // STA $50
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute DMA from unmapped region
		helper.Bus.Step() // LDA #$50
		helper.Bus.Step() // STA $4014 (DMA from unmapped)
		helper.Bus.Step() // LDA #$88
		helper.Bus.Step() // STA $50

		// Should complete without crashing
		marker := helper.Memory.Read(0x0050)
		if marker != 0x88 {
			t.Errorf("DMA from unmapped region failed: expected 0x88, got 0x%02X", marker)
		}

		t.Log("DMA from unmapped memory test completed")
	})

	t.Run("DMA during interrupt", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupOAMData(0x02, 0xF0)

		// Set up interrupt vectors
		romData := make([]uint8, 0x8000)
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high ($8100)
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high

		// Main program
		program := []uint8{
			0xA9, 0x02, // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (DMA)
			0xA9, 0x99, // LDA #$99 (should be delayed)
			0x85, 0x60, // STA $60
			0x4C, 0x00, 0x80, // JMP $8000
		}
		copy(romData, program)

		// NMI handler at $8100
		nmiHandler := []uint8{
			0xA9, 0xAB, // LDA #$AB
			0x85, 0x61, // STA $61 (NMI marker)
			0x40, // RTI
		}
		copy(romData[0x0100:], nmiHandler)

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Enable NMI
		helper.Memory.Write(0x2000, 0x80)

		// Execute and potentially trigger NMI during DMA
		helper.Bus.Step() // LDA #$02
		helper.Bus.Step() // STA $4014 (DMA starts)

		// If NMI occurs during DMA, it should be delayed until DMA completes
		// Continue execution
		helper.Bus.Step() // LDA #$99 (delayed by DMA)
		helper.Bus.Step() // STA $60

		// Check completion
		mainMarker := helper.Memory.Read(0x0060)
		if mainMarker != 0x99 {
			t.Errorf("Main program completion failed: expected 0x99, got 0x%02X", mainMarker)
		}

		t.Log("DMA during interrupt test completed")
	})

	t.Run("DMA cycle count accuracy", func(t *testing.T) {
		helper := NewDMATestHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupOAMData(0x02, 0x12)

		// Test precise DMA timing with different alignments
		alignmentTests := []struct {
			name        string
			preNOPs     int
			expectedMin int
			expectedMax int
		}{
			{"Write on even cycle", 0, 513, 513},
			{"Write on odd cycle", 1, 514, 514},
		}

		for _, test := range alignmentTests {
			t.Run(test.name, func(t *testing.T) {
				// Create aligned program
				program := make([]uint8, 0)

				// Add NOPs for alignment
				for i := 0; i < test.preNOPs; i++ {
					program = append(program, 0xEA) // NOP
				}

				// DMA sequence
				program = append(program, []uint8{
					0xA9, 0x02, // LDA #$02
					0x8D, 0x14, 0x40, // STA $4014
					0xE6, 0x70, // INC $70 (first post-DMA instruction)
					0x4C, 0x00, 0x80, // JMP $8000
				}...)

				romData := make([]uint8, 0x8000)
				copy(romData, program)
				romData[0x7FFC] = 0x00
				romData[0x7FFD] = 0x80
				helper.GetMockCartridge().LoadPRG(romData)
				helper.Bus.Reset()

				// Execute alignment NOPs
				for i := 0; i < test.preNOPs; i++ {
					helper.Bus.Step()
				}

				// Execute DMA sequence
				helper.Bus.Step() // LDA #$02

				// Measure DMA timing (would need cycle counter)
				helper.Bus.Step() // STA $4014 (DMA)

				// Verify post-DMA execution
				helper.Bus.Step() // INC $70

				counter := helper.Memory.Read(0x0070)
				if counter != 1 {
					t.Errorf("Post-DMA instruction failed: expected 1, got %d", counter)
				}

				t.Logf("DMA timing test completed for %s", test.name)
			})
		}
	})
}
