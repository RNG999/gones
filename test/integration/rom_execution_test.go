package integration

import (
	"bytes"
	"testing"

	"gones/internal/cartridge"
)

// TestROMExecution provides integration tests for ROM execution and compatibility
// This test suite validates end-to-end ROM loading, execution, and system integration

// ROMExecutionTestSuite represents a comprehensive ROM execution test
type ROMExecutionTestSuite struct {
	Name        string
	ROMData     []byte
	TestCases   []ROMExecutionTestCase
	Description string
}

// ROMExecutionTestCase represents a single test case within a ROM execution test
type ROMExecutionTestCase struct {
	Name           string
	MaxCycles      uint64
	ExpectedMemory map[uint16]uint8
	ExpectedCPU    CPUState
	ValidationFunc func(*testing.T, *IntegrationTestHelper) bool
	Description    string
}

// CPUState represents expected CPU state for validation
type CPUState struct {
	A, X, Y          uint8
	PC               uint16
	SP               uint8
	N, V, D, I, Z, C bool
}

// createTestROM creates a minimal test ROM with specific instruction sequence
func createTestROM(instructions []uint8) []byte {
	// Create iNES header
	header := make([]byte, 16)
	copy(header[0:4], "NES\x1A")
	header[4] = 1 // 16KB PRG ROM
	header[5] = 1 // 8KB CHR ROM
	// Other header bytes remain zero

	// Create PRG ROM (16KB)
	prgROM := make([]byte, 0x4000)

	// Copy instructions to ROM start
	copy(prgROM, instructions)

	// Set reset vector to point to ROM start (0x8000)
	prgROM[0x3FFC] = 0x00 // Reset vector low byte
	prgROM[0x3FFD] = 0x80 // Reset vector high byte

	// Create CHR ROM (8KB) - empty
	chrROM := make([]byte, 0x2000)

	// Combine all parts
	romData := append(header, prgROM...)
	romData = append(romData, chrROM...)

	return romData
}

// TestROMExecution_BasicInstructionExecution tests basic instruction execution
func TestROMExecution_BasicInstructionExecution(t *testing.T) {
	testSuites := []ROMExecutionTestSuite{
		{
			Name: "Basic Load and Store Operations",
			ROMData: createTestROM([]uint8{
				0xA9, 0x42, // LDA #$42
				0x85, 0x00, // STA $00
				0xA9, 0x55, // LDA #$55
				0x85, 0x01, // STA $01
				0x4C, 0x08, 0x80, // JMP $8008 (infinite loop)
			}),
			TestCases: []ROMExecutionTestCase{
				{
					Name:      "Memory values after execution",
					MaxCycles: 1000,
					ExpectedMemory: map[uint16]uint8{
						0x0000: 0x42,
						0x0001: 0x55,
					},
					Description: "Verify LDA/STA operations write correct values to memory",
				},
			},
			Description: "Tests basic load and store instruction functionality",
		},
		{
			Name: "Arithmetic Operations",
			ROMData: createTestROM([]uint8{
				0x18,       // CLC
				0xA9, 0x10, // LDA #$10
				0x69, 0x05, // ADC #$05
				0x85, 0x10, // STA $10
				0x38,       // SEC
				0xE9, 0x03, // SBC #$03
				0x85, 0x11, // STA $11
				0x4C, 0x0C, 0x80, // JMP $800C (infinite loop)
			}),
			TestCases: []ROMExecutionTestCase{
				{
					Name:      "Addition result",
					MaxCycles: 1000,
					ExpectedMemory: map[uint16]uint8{
						0x0010: 0x15, // $10 + $05 = $15
						0x0011: 0x12, // $15 - $03 = $12
					},
					Description: "Verify ADC and SBC operations produce correct results",
				},
			},
			Description: "Tests arithmetic instruction functionality",
		},
		{
			Name: "Flag Operations",
			ROMData: createTestROM([]uint8{
				0x18,       // CLC
				0xA9, 0xFF, // LDA #$FF
				0x69, 0x01, // ADC #$01 (should set Z and C flags)
				0x85, 0x20, // STA $20
				0x38,       // SEC
				0xA9, 0x00, // LDA #$00
				0xE9, 0x01, // SBC #$01 (should set N flag, clear Z flag)
				0x85, 0x21, // STA $21
				0x4C, 0x10, 0x80, // JMP $8010 (infinite loop)
			}),
			TestCases: []ROMExecutionTestCase{
				{
					Name:      "Flag operations results",
					MaxCycles: 1000,
					ExpectedMemory: map[uint16]uint8{
						0x0020: 0x00, // $FF + $01 = $00 (with carry)
						0x0021: 0xFF, // $00 - $01 = $FF (with borrow)
					},
					Description: "Verify flag-affecting operations work correctly",
				},
			},
			Description: "Tests CPU flag manipulation and arithmetic with flags",
		},
		{
			Name: "Memory Addressing Modes",
			ROMData: createTestROM([]uint8{
				// Set up test data
				0xA9, 0x11, // LDA #$11
				0x85, 0x30, // STA $30
				0xA9, 0x22, // LDA #$22
				0x85, 0x31, // STA $31

				// Test zero page addressing
				0xA5, 0x30, // LDA $30
				0x85, 0x40, // STA $40

				// Test absolute addressing
				0xAD, 0x31, 0x00, // LDA $0031
				0x8D, 0x41, 0x00, // STA $0041

				0x4C, 0x16, 0x80, // JMP $8016 (infinite loop)
			}),
			TestCases: []ROMExecutionTestCase{
				{
					Name:      "Addressing mode results",
					MaxCycles: 1000,
					ExpectedMemory: map[uint16]uint8{
						0x0030: 0x11,
						0x0031: 0x22,
						0x0040: 0x11, // Copy of $30
						0x0041: 0x22, // Copy of $31
					},
					Description: "Verify different addressing modes work correctly",
				},
			},
			Description: "Tests various CPU addressing modes",
		},
	}

	for _, suite := range testSuites {
		t.Run(suite.Name, func(t *testing.T) {
			for _, testCase := range suite.TestCases {
				t.Run(testCase.Name, func(t *testing.T) {
					// Load ROM
					reader := bytes.NewReader(suite.ROMData)
					cart, err := cartridge.LoadFromReader(reader)
					if err != nil {
						t.Fatalf("Failed to load test ROM: %v", err)
					}

					// Create integration test helper
					helper := NewIntegrationTestHelper()
					helper.Cartridge = cart

					// Reset system and start execution
					helper.Bus.Reset()

					// Execute for specified number of cycles
					cycleCount := uint64(0)
					for cycleCount < testCase.MaxCycles {
						helper.Bus.Step()
						cycleCount++

						// Check for infinite loop (simple detection)
						if cycleCount > 100 {
							currentPC := helper.CPU.PC
							if currentPC >= 0x8000 {
								// Check if we're in a tight loop
								instruction := helper.Memory.Read(currentPC)
								if instruction == 0x4C { // JMP absolute
									lowByte := helper.Memory.Read(currentPC + 1)
									highByte := helper.Memory.Read(currentPC + 2)
									jumpAddr := uint16(lowByte) | (uint16(highByte) << 8)
									if jumpAddr == currentPC {
										// Infinite loop detected
										break
									}
								}
							}
						}
					}

					// Validate expected memory values
					for addr, expectedValue := range testCase.ExpectedMemory {
						actualValue := helper.Memory.Read(addr)
						if actualValue != expectedValue {
							t.Errorf("%s: Memory at 0x%04X expected 0x%02X, got 0x%02X",
								testCase.Description, addr, expectedValue, actualValue)
						}
					}

					// Run custom validation function if provided
					if testCase.ValidationFunc != nil {
						if !testCase.ValidationFunc(t, helper) {
							t.Errorf("%s: Custom validation failed", testCase.Description)
						}
					}

					t.Logf("%s completed in %d cycles", testCase.Description, cycleCount)
				})
			}
		})
	}
}

// TestROMExecution_CartridgeMemoryAccess tests cartridge-specific memory access
func TestROMExecution_CartridgeMemoryAccess(t *testing.T) {
	t.Run("SRAM access during execution", func(t *testing.T) {
		romData := createTestROM([]uint8{
			// Write to SRAM
			0xA9, 0xAA, // LDA #$AA
			0x8D, 0x00, 0x60, // STA $6000
			0xA9, 0xBB, // LDA #$BB
			0x8D, 0xFF, 0x7F, // STA $7FFF

			// Read from SRAM
			0xAD, 0x00, 0x60, // LDA $6000
			0x85, 0x50, // STA $50
			0xAD, 0xFF, 0x7F, // LDA $7FFF
			0x85, 0x51, // STA $51

			0x4C, 0x14, 0x80, // JMP $8014 (infinite loop)
		})

		reader := bytes.NewReader(romData)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load SRAM test ROM: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.Reset()

		// Execute program
		for i := 0; i < 1000; i++ {
			helper.Bus.Step()

			// Check for completion (infinite loop)
			if helper.CPU.PC == 0x8014 {
				break
			}
		}

		// Verify SRAM was accessed correctly
		if helper.Memory.Read(0x50) != 0xAA {
			t.Errorf("SRAM read failed: expected 0xAA at $50, got 0x%02X", helper.Memory.Read(0x50))
		}

		if helper.Memory.Read(0x51) != 0xBB {
			t.Errorf("SRAM read failed: expected 0xBB at $51, got 0x%02X", helper.Memory.Read(0x51))
		}

		// Verify SRAM retention by reading directly from cartridge
		if cart.ReadPRG(0x6000) != 0xAA {
			t.Errorf("SRAM not retained: expected 0xAA at $6000, got 0x%02X", cart.ReadPRG(0x6000))
		}

		if cart.ReadPRG(0x7FFF) != 0xBB {
			t.Errorf("SRAM not retained: expected 0xBB at $7FFF, got 0x%02X", cart.ReadPRG(0x7FFF))
		}

		t.Logf("SRAM access test completed successfully")
	})

	t.Run("ROM mirroring verification", func(t *testing.T) {
		// Create ROM with 16KB PRG (should be mirrored)
		instructions := []uint8{
			0xAD, 0x00, 0x80, // LDA $8000 (read from first bank)
			0x85, 0x60, // STA $60
			0xAD, 0x00, 0xC0, // LDA $C000 (read from mirrored bank)
			0x85, 0x61, // STA $61
			0x4C, 0x0C, 0x80, // JMP $800C (infinite loop)
		}

		romData := createTestROM(instructions)

		reader := bytes.NewReader(romData)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load mirroring test ROM: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.Reset()

		// Execute program
		for i := 0; i < 1000; i++ {
			helper.Bus.Step()

			if helper.CPU.PC == 0x800C {
				break
			}
		}

		// Both reads should return the same value (LDA opcode = 0xAD)
		value1 := helper.Memory.Read(0x60)
		value2 := helper.Memory.Read(0x61)

		if value1 != value2 {
			t.Errorf("ROM mirroring failed: $8000 read 0x%02X, $C000 read 0x%02X", value1, value2)
		}

		if value1 != 0xAD {
			t.Errorf("ROM read incorrect: expected 0xAD (LDA opcode), got 0x%02X", value1)
		}

		t.Logf("ROM mirroring verification completed successfully")
	})

	t.Run("CHR RAM functionality", func(t *testing.T) {
		// Create ROM with CHR RAM (CHR size = 0)
		header := make([]byte, 16)
		copy(header[0:4], "NES\x1A")
		header[4] = 1 // 16KB PRG ROM
		header[5] = 0 // CHR RAM (no CHR ROM)

		// Simple program that doesn't need CHR data
		prgROM := make([]byte, 0x4000)
		instructions := []uint8{
			0xA9, 0x77, // LDA #$77
			0x85, 0x70, // STA $70
			0x4C, 0x04, 0x80, // JMP $8004 (infinite loop)
		}
		copy(prgROM, instructions)

		// Set reset vector
		prgROM[0x3FFC] = 0x00
		prgROM[0x3FFD] = 0x80

		romData := append(header, prgROM...)

		reader := bytes.NewReader(romData)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load CHR RAM test ROM: %v", err)
		}

		// Verify CHR RAM is available and writable
		cart.WriteCHR(0x0000, 0x12)
		cart.WriteCHR(0x1FFF, 0x34)

		if cart.ReadCHR(0x0000) != 0x12 {
			t.Errorf("CHR RAM write/read failed at 0x0000: expected 0x12, got 0x%02X", cart.ReadCHR(0x0000))
		}

		if cart.ReadCHR(0x1FFF) != 0x34 {
			t.Errorf("CHR RAM write/read failed at 0x1FFF: expected 0x34, got 0x%02X", cart.ReadCHR(0x1FFF))
		}

		// Execute simple program to ensure basic functionality
		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.Reset()

		for i := 0; i < 100; i++ {
			helper.Bus.Step()
			if helper.CPU.PC == 0x8004 {
				break
			}
		}

		if helper.Memory.Read(0x70) != 0x77 {
			t.Errorf("Program execution failed with CHR RAM")
		}

		t.Logf("CHR RAM functionality test completed successfully")
	})
}

// TestROMExecution_EdgeCases tests edge cases in ROM execution
func TestROMExecution_EdgeCases(t *testing.T) {
	t.Run("Cross-bank execution", func(t *testing.T) {
		// Create 32KB ROM with code that spans banks
		header := make([]byte, 16)
		copy(header[0:4], "NES\x1A")
		header[4] = 2 // 32KB PRG ROM
		header[5] = 1 // 8KB CHR ROM

		prgROM := make([]byte, 0x8000)

		// Code in first bank
		prgROM[0x0000] = 0xA9 // LDA #$55
		prgROM[0x0001] = 0x55
		prgROM[0x0002] = 0x85 // STA $80
		prgROM[0x0003] = 0x80
		prgROM[0x0004] = 0x4C // JMP $C000 (second bank)
		prgROM[0x0005] = 0x00
		prgROM[0x0006] = 0xC0

		// Code in second bank
		prgROM[0x4000] = 0xA9 // LDA #$AA
		prgROM[0x4001] = 0xAA
		prgROM[0x4002] = 0x85 // STA $81
		prgROM[0x4003] = 0x81
		prgROM[0x4004] = 0x4C // JMP $C004 (infinite loop)
		prgROM[0x4005] = 0x04
		prgROM[0x4006] = 0xC0

		// Set reset vector to first bank
		prgROM[0x7FFC] = 0x00
		prgROM[0x7FFD] = 0x80

		chrROM := make([]byte, 0x2000)

		romData := append(header, prgROM...)
		romData = append(romData, chrROM...)

		reader := bytes.NewReader(romData)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load cross-bank test ROM: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.Reset()

		// Execute program
		for i := 0; i < 1000; i++ {
			helper.Bus.Step()
			if helper.CPU.PC == 0xC004 {
				break
			}
		}

		// Verify both banks executed
		if helper.Memory.Read(0x80) != 0x55 {
			t.Errorf("First bank didn't execute: expected 0x55 at $80, got 0x%02X", helper.Memory.Read(0x80))
		}

		if helper.Memory.Read(0x81) != 0xAA {
			t.Errorf("Second bank didn't execute: expected 0xAA at $81, got 0x%02X", helper.Memory.Read(0x81))
		}

		t.Logf("Cross-bank execution test completed successfully")
	})

	t.Run("Stack operations with ROM", func(t *testing.T) {
		romData := createTestROM([]uint8{
			0xA9, 0x12, // LDA #$12
			0x48,       // PHA
			0xA9, 0x34, // LDA #$34
			0x48,       // PHA
			0x68,       // PLA
			0x85, 0x90, // STA $90
			0x68,       // PLA
			0x85, 0x91, // STA $91
			0x4C, 0x0E, 0x80, // JMP $800E (infinite loop)
		})

		reader := bytes.NewReader(romData)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load stack test ROM: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.Reset()

		// Execute program
		for i := 0; i < 1000; i++ {
			helper.Bus.Step()
			if helper.CPU.PC == 0x800E {
				break
			}
		}

		// Verify LIFO behavior
		if helper.Memory.Read(0x90) != 0x34 { // Last pushed, first popped
			t.Errorf("Stack LIFO failed: expected 0x34 at $90, got 0x%02X", helper.Memory.Read(0x90))
		}

		if helper.Memory.Read(0x91) != 0x12 { // First pushed, last popped
			t.Errorf("Stack LIFO failed: expected 0x12 at $91, got 0x%02X", helper.Memory.Read(0x91))
		}

		t.Logf("Stack operations test completed successfully")
	})

	t.Run("Reset vector validation", func(t *testing.T) {
		// Create ROM with custom reset vector
		header := make([]byte, 16)
		copy(header[0:4], "NES\x1A")
		header[4] = 1 // 16KB PRG ROM
		header[5] = 1 // 8KB CHR ROM

		prgROM := make([]byte, 0x4000)

		// Code at custom location
		prgROM[0x1000] = 0xA9 // LDA #$CC
		prgROM[0x1001] = 0xCC
		prgROM[0x1002] = 0x85 // STA $A0
		prgROM[0x1003] = 0xA0
		prgROM[0x1004] = 0x4C // JMP $9004 (infinite loop)
		prgROM[0x1005] = 0x04
		prgROM[0x1006] = 0x90

		// Set custom reset vector to 0x9000 (ROM offset 0x1000)
		prgROM[0x3FFC] = 0x00 // Reset vector low
		prgROM[0x3FFD] = 0x90 // Reset vector high

		chrROM := make([]byte, 0x2000)

		romData := append(header, prgROM...)
		romData = append(romData, chrROM...)

		reader := bytes.NewReader(romData)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load reset vector test ROM: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.Reset()

		// Verify reset vector was read correctly
		expectedPC := uint16(0x9000)
		if helper.CPU.PC != expectedPC {
			t.Errorf("Reset vector not honored: expected PC=0x%04X, got PC=0x%04X", expectedPC, helper.CPU.PC)
		}

		// Execute program
		for i := 0; i < 1000; i++ {
			helper.Bus.Step()
			if helper.CPU.PC == 0x9004 {
				break
			}
		}

		// Verify program executed
		if helper.Memory.Read(0xA0) != 0xCC {
			t.Errorf("Custom reset vector program didn't execute: expected 0xCC at $A0, got 0x%02X", helper.Memory.Read(0xA0))
		}

		t.Logf("Reset vector validation completed successfully")
	})
}

// TestROMExecution_ErrorConditions tests error handling during ROM execution
func TestROMExecution_ErrorConditions(t *testing.T) {
	t.Run("Invalid ROM execution", func(t *testing.T) {
		// Create ROM with invalid reset vector
		header := make([]byte, 16)
		copy(header[0:4], "NES\x1A")
		header[4] = 1 // 16KB PRG ROM
		header[5] = 1 // 8KB CHR ROM

		prgROM := make([]byte, 0x4000)
		chrROM := make([]byte, 0x2000)

		// Set reset vector to invalid address (outside ROM)
		prgROM[0x3FFC] = 0x00 // Reset vector low
		prgROM[0x3FFD] = 0x00 // Reset vector high (points to $0000)

		romData := append(header, prgROM...)
		romData = append(romData, chrROM...)

		reader := bytes.NewReader(romData)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load invalid ROM: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart

		// System should handle this gracefully
		helper.Bus.Reset()

		// PC should be set to reset vector value (even if invalid)
		if helper.CPU.PC != 0x0000 {
			t.Errorf("PC not set to reset vector: expected 0x0000, got 0x%04X", helper.CPU.PC)
		}

		// System should not crash when executing from invalid address
		for i := 0; i < 10; i++ {
			helper.Bus.Step() // Should not crash
		}

		t.Logf("Invalid ROM execution handled gracefully")
	})

	t.Run("Execution without cartridge", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		// No cartridge loaded

		helper.Bus.Reset()

		// Should handle gracefully
		for i := 0; i < 10; i++ {
			helper.Bus.Step() // Should not crash
		}

		t.Logf("Execution without cartridge handled gracefully")
	})
}

// BenchmarkROMExecution_Performance benchmarks ROM execution performance
func BenchmarkROMExecution_Performance(b *testing.B) {
	// Create a simple loop program for benchmarking
	romData := createTestROM([]uint8{
		0xA2, 0x00, // LDX #$00     ; Initialize counter
		0xE8,       // INX          ; Increment X
		0xE0, 0xFF, // CPX #$FF     ; Compare with 255
		0xD0, 0xFB, // BNE -5       ; Branch if not equal (loop)
		0x4C, 0x06, 0x80, // JMP $8006    ; Infinite loop when done
	})

	reader := bytes.NewReader(romData)
	cart, err := cartridge.LoadFromReader(reader)
	if err != nil {
		b.Fatalf("Failed to load benchmark ROM: %v", err)
	}

	helper := NewIntegrationTestHelper()
	helper.Cartridge = cart

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		helper.Bus.Reset()

		// Execute the loop (should take ~255 iterations)
		for j := 0; j < 1000; j++ {
			helper.Bus.Step()

			// Stop when we reach the final infinite loop
			if helper.CPU.PC == 0x8006 {
				break
			}
		}
	}
}
