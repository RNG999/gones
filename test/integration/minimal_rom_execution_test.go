package integration

import (
	"testing"
)

// MinimalROMTestHelper provides utilities for testing minimal ROM execution scenarios
type MinimalROMTestHelper struct {
	*IntegrationTestHelper
	executionTrace []ExecutionTraceEntry
	testResults    []TestResult
}

// ExecutionTraceEntry represents a single execution trace entry
type ExecutionTraceEntry struct {
	StepNumber  int
	PC          uint16
	Opcode      uint8
	Operands    []uint8
	Instruction string
	Cycles      uint64
	A, X, Y     uint8
	SP          uint8
	Flags       uint8
}

// TestResult represents the result of a minimal ROM test
type TestResult struct {
	TestName      string
	Passed        bool
	ExpectedValue uint8
	ActualValue   uint8
	CycleCount    uint64
	Description   string
}

// NewMinimalROMTestHelper creates a new minimal ROM test helper
func NewMinimalROMTestHelper() *MinimalROMTestHelper {
	return &MinimalROMTestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		executionTrace:        make([]ExecutionTraceEntry, 0),
		testResults:           make([]TestResult, 0),
	}
}

// StepWithTrace executes one step and records execution trace
func (h *MinimalROMTestHelper) StepWithTrace() {
	pc := h.CPU.PC
	opcode := h.Memory.Read(pc)

	// Decode instruction (simplified)
	instruction, operands := h.decodeInstruction(pc, opcode)

	preCycles := h.Bus.GetCycleCount()
	h.Bus.Step()
	postCycles := h.Bus.GetCycleCount()

	entry := ExecutionTraceEntry{
		StepNumber:  len(h.executionTrace) + 1,
		PC:          pc,
		Opcode:      opcode,
		Operands:    operands,
		Instruction: instruction,
		Cycles:      postCycles - preCycles,
		A:           h.CPU.A,
		X:           h.CPU.X,
		Y:           h.CPU.Y,
		SP:          h.CPU.SP,
		Flags:       h.getCPUFlags(),
	}

	h.executionTrace = append(h.executionTrace, entry)
}

// decodeInstruction provides basic instruction decoding for tracing
func (h *MinimalROMTestHelper) decodeInstruction(pc uint16, opcode uint8) (string, []uint8) {
	// Simplified instruction decoding for common opcodes
	operands := make([]uint8, 0)

	switch opcode {
	case 0x00:
		return "BRK", operands
	case 0x18:
		return "CLC", operands
	case 0x38:
		return "SEC", operands
	case 0x58:
		return "CLI", operands
	case 0x78:
		return "SEI", operands
	case 0x8A:
		return "TXA", operands
	case 0x98:
		return "TYA", operands
	case 0xA8:
		return "TAY", operands
	case 0xAA:
		return "TAX", operands
	case 0xCA:
		return "DEX", operands
	case 0xC8:
		return "INY", operands
	case 0xE8:
		return "INX", operands
	case 0xEA:
		return "NOP", operands
	case 0x40:
		return "RTI", operands
	case 0x60:
		return "RTS", operands
	case 0x48:
		return "PHA", operands
	case 0x68:
		return "PLA", operands
	case 0x08:
		return "PHP", operands
	case 0x28:
		return "PLP", operands

	// Two-byte instructions
	case 0xA9:
		operands = append(operands, h.Memory.Read(pc+1))
		return "LDA #$" + formatHex(operands[0]), operands
	case 0xA2:
		operands = append(operands, h.Memory.Read(pc+1))
		return "LDX #$" + formatHex(operands[0]), operands
	case 0xA0:
		operands = append(operands, h.Memory.Read(pc+1))
		return "LDY #$" + formatHex(operands[0]), operands
	case 0xC9:
		operands = append(operands, h.Memory.Read(pc+1))
		return "CMP #$" + formatHex(operands[0]), operands
	case 0xE0:
		operands = append(operands, h.Memory.Read(pc+1))
		return "CPX #$" + formatHex(operands[0]), operands
	case 0xC0:
		operands = append(operands, h.Memory.Read(pc+1))
		return "CPY #$" + formatHex(operands[0]), operands
	case 0x69:
		operands = append(operands, h.Memory.Read(pc+1))
		return "ADC #$" + formatHex(operands[0]), operands
	case 0xE9:
		operands = append(operands, h.Memory.Read(pc+1))
		return "SBC #$" + formatHex(operands[0]), operands
	case 0x85:
		operands = append(operands, h.Memory.Read(pc+1))
		return "STA $" + formatHex(operands[0]), operands
	case 0xA5:
		operands = append(operands, h.Memory.Read(pc+1))
		return "LDA $" + formatHex(operands[0]), operands

	// Three-byte instructions
	case 0x4C:
		operands = append(operands, h.Memory.Read(pc+1), h.Memory.Read(pc+2))
		return "JMP $" + formatHex(operands[1]) + formatHex(operands[0]), operands
	case 0x8D:
		operands = append(operands, h.Memory.Read(pc+1), h.Memory.Read(pc+2))
		return "STA $" + formatHex(operands[1]) + formatHex(operands[0]), operands
	case 0xAD:
		operands = append(operands, h.Memory.Read(pc+1), h.Memory.Read(pc+2))
		return "LDA $" + formatHex(operands[1]) + formatHex(operands[0]), operands

	default:
		return "UNK", operands
	}
}

// formatHex formats a byte as a two-digit hex string
func formatHex(b uint8) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[b>>4], hex[b&0xF]})
}

// getCPUFlags returns the CPU flags as a single byte
func (h *MinimalROMTestHelper) getCPUFlags() uint8 {
	flags := uint8(0x20) // Unused flag always set
	if h.CPU.N {
		flags |= 0x80
	}
	if h.CPU.V {
		flags |= 0x40
	}
	if h.CPU.B {
		flags |= 0x10
	}
	if h.CPU.D {
		flags |= 0x08
	}
	if h.CPU.I {
		flags |= 0x04
	}
	if h.CPU.Z {
		flags |= 0x02
	}
	if h.CPU.C {
		flags |= 0x01
	}
	return flags
}

// RunTestProgram runs a test program and returns the result
func (h *MinimalROMTestHelper) RunTestProgram(program []uint8, maxSteps int, testAddr uint16, expectedValue uint8, testName string) TestResult {
	// Load program
	romData := make([]uint8, 0x8000)
	copy(romData, program)
	romData[0x7FFC] = 0x00
	romData[0x7FFD] = 0x80
	h.GetMockCartridge().LoadPRG(romData)
	h.Bus.Reset()

	// Run program
	for i := 0; i < maxSteps; i++ {
		h.StepWithTrace()

		// Check for infinite loop or halt condition
		if len(h.executionTrace) >= 2 {
			current := h.executionTrace[len(h.executionTrace)-1]
			previous := h.executionTrace[len(h.executionTrace)-2]

			// Check for simple infinite loop (JMP to same address)
			if current.PC == previous.PC && current.Opcode == 0x4C {
				break
			}
		}
	}

	// Check result
	actualValue := h.Memory.Read(testAddr)
	passed := actualValue == expectedValue

	result := TestResult{
		TestName:      testName,
		Passed:        passed,
		ExpectedValue: expectedValue,
		ActualValue:   actualValue,
		CycleCount:    h.Bus.GetCycleCount(),
		Description:   "Test completed",
	}

	h.testResults = append(h.testResults, result)
	return result
}

// GetExecutionTrace returns the execution trace
func (h *MinimalROMTestHelper) GetExecutionTrace() []ExecutionTraceEntry {
	return h.executionTrace
}

// GetTestResults returns all test results
func (h *MinimalROMTestHelper) GetTestResults() []TestResult {
	return h.testResults
}

// ClearTrace clears the execution trace
func (h *MinimalROMTestHelper) ClearTrace() {
	h.executionTrace = h.executionTrace[:0]
}

// TestMinimalROMBasicExecution tests basic ROM execution scenarios
func TestMinimalROMBasicExecution(t *testing.T) {
	t.Run("Simple register load test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Load value into A register and store to memory
		program := []uint8{
			0xA9, 0x42, // LDA #$42
			0x85, 0x00, // STA $00
			0x4C, 0x04, 0x80, // JMP $8004 (infinite loop)
		}

		result := helper.RunTestProgram(program, 100, 0x0000, 0x42, "Simple LDA/STA test")

		if !result.Passed {
			t.Errorf("Test failed: expected 0x%02X at $0000, got 0x%02X",
				result.ExpectedValue, result.ActualValue)
		}

		// Verify execution trace
		trace := helper.GetExecutionTrace()
		if len(trace) < 3 {
			t.Fatalf("Expected at least 3 trace entries, got %d", len(trace))
		}

		// Check individual instructions
		if trace[0].Instruction != "LDA #$42" {
			t.Errorf("Expected first instruction to be 'LDA #$42', got '%s'", trace[0].Instruction)
		}

		if trace[1].Instruction != "STA $00" {
			t.Errorf("Expected second instruction to be 'STA $00', got '%s'", trace[1].Instruction)
		}

		// Verify A register contains expected value
		if trace[0].A != 0x42 {
			t.Errorf("A register should be 0x42 after LDA, got 0x%02X", trace[0].A)
		}

		t.Logf("Test completed in %d cycles", result.CycleCount)
	})

	t.Run("Arithmetic operations test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Basic arithmetic
		program := []uint8{
			0x18,       // CLC
			0xA9, 0x10, // LDA #$10
			0x69, 0x05, // ADC #$05
			0x85, 0x01, // STA $01    ; Result should be $15
			0x38,       // SEC
			0xE9, 0x03, // SBC #$03
			0x85, 0x02, // STA $02    ; Result should be $12
			0x4C, 0x0C, 0x80, // JMP $800C (infinite loop)
		}

		result1 := helper.RunTestProgram(program, 100, 0x0001, 0x15, "Addition test")
		helper.ClearTrace()

		// Check second result (need to run again for SBC result)
		helper.RunTestProgram(program, 100, 0x0002, 0x12, "Subtraction test")

		if !result1.Passed {
			t.Errorf("Addition test failed: expected 0x15, got 0x%02X", result1.ActualValue)
		}

		// Verify subtraction result
		sbcResult := helper.Memory.Read(0x0002)
		if sbcResult != 0x12 {
			t.Errorf("Subtraction test failed: expected 0x12, got 0x%02X", sbcResult)
		}

		// Verify execution trace includes arithmetic operations
		trace := helper.GetExecutionTrace()
		found_adc := false
		found_sbc := false

		for _, entry := range trace {
			if entry.Instruction == "ADC #$05" {
				found_adc = true
			}
			if entry.Instruction == "SBC #$03" {
				found_sbc = true
			}
		}

		if !found_adc {
			t.Error("ADC instruction not found in trace")
		}
		if !found_sbc {
			t.Error("SBC instruction not found in trace")
		}
	})

	t.Run("Flag operations test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Flag manipulation
		program := []uint8{
			0x18,       // CLC
			0x85, 0x10, // STA $10 (save flags state)
			0x38,       // SEC
			0x85, 0x11, // STA $11 (save flags state)
			0x78,       // SEI
			0x85, 0x12, // STA $12 (save flags state)
			0x58,       // CLI
			0x85, 0x13, // STA $13 (save flags state)
			0x4C, 0x0E, 0x80, // JMP $800E (infinite loop)
		}

		helper.RunTestProgram(program, 100, 0x0010, 0x00, "Flag operations test")

		// Verify flag changes in trace
		trace := helper.GetExecutionTrace()

		flagInstructions := []string{"CLC", "SEC", "SEI", "CLI"}
		foundCount := 0

		for _, entry := range trace {
			for _, flagInst := range flagInstructions {
				if entry.Instruction == flagInst {
					foundCount++
					break
				}
			}
		}

		if foundCount != len(flagInstructions) {
			t.Errorf("Expected %d flag instructions, found %d", len(flagInstructions), foundCount)
		}

		// Check flag states
		clcEntry := -1
		secEntry := -1

		for i, entry := range trace {
			if entry.Instruction == "CLC" {
				clcEntry = i
			}
			if entry.Instruction == "SEC" {
				secEntry = i
			}
		}

		if clcEntry >= 0 && clcEntry+1 < len(trace) {
			// Check that carry is clear after CLC
			if trace[clcEntry+1].Flags&0x01 != 0 {
				t.Error("Carry flag should be clear after CLC")
			}
		}

		if secEntry >= 0 && secEntry+1 < len(trace) {
			// Check that carry is set after SEC
			if trace[secEntry+1].Flags&0x01 == 0 {
				t.Error("Carry flag should be set after SEC")
			}
		}
	})
}

// TestMinimalROMControlFlow tests control flow instructions
func TestMinimalROMControlFlow(t *testing.T) {
	t.Run("Unconditional jump test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Unconditional jump
		program := []uint8{
			0xA9, 0xAA, // LDA #$AA
			0x4C, 0x08, 0x80, // JMP $8008
			0xA9, 0xBB, // LDA #$BB (should be skipped)
			0x85, 0x00, // STA $00 (should be skipped)
			0xA9, 0xCC, // LDA #$CC (jump target)
			0x85, 0x00, // STA $00
			0x4C, 0x0C, 0x80, // JMP $800C (infinite loop)
		}

		result := helper.RunTestProgram(program, 100, 0x0000, 0xCC, "Unconditional jump test")

		if !result.Passed {
			t.Errorf("Jump test failed: expected 0xCC, got 0x%02X", result.ActualValue)
		}

		// Verify that skipped instructions didn't execute
		trace := helper.GetExecutionTrace()

		for _, entry := range trace {
			if entry.Instruction == "LDA #$BB" {
				t.Error("Skipped instruction 'LDA #$BB' was executed")
			}
		}

		// Verify jump instruction was executed
		foundJump := false
		for _, entry := range trace {
			if entry.Instruction == "JMP $8008" {
				foundJump = true
				break
			}
		}

		if !foundJump {
			t.Error("Jump instruction not found in trace")
		}
	})

	t.Run("Compare and branch test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Compare and conditional execution
		program := []uint8{
			0xA9, 0x05, // LDA #$05
			0xC9, 0x05, // CMP #$05 (should set Z flag)
			0xD0, 0x04, // BNE +4 (should not branch)
			0xA9, 0xFF, // LDA #$FF (should execute)
			0x85, 0x00, // STA $00 (should execute)
			0xA9, 0x03, // LDA #$03
			0xC9, 0x05, // CMP #$05 (should clear Z flag)
			0xF0, 0x04, // BEQ +4 (should not branch)
			0xA9, 0xEE, // LDA #$EE (should execute)
			0x85, 0x01, // STA $01 (should execute)
			0x4C, 0x14, 0x80, // JMP $8014 (infinite loop)
		}

		result1 := helper.RunTestProgram(program, 100, 0x0000, 0xFF, "Compare equal test")
		if !result1.Passed {
			t.Errorf("Compare equal test failed: expected 0xFF, got 0x%02X", result1.ActualValue)
		}

		result2Value := helper.Memory.Read(0x0001)
		if result2Value != 0xEE {
			t.Errorf("Compare not equal test failed: expected 0xEE, got 0x%02X", result2Value)
		}

		// Verify CMP instructions in trace
		trace := helper.GetExecutionTrace()
		cmpCount := 0

		for _, entry := range trace {
			if entry.Instruction == "CMP #$05" {
				cmpCount++
			}
		}

		if cmpCount != 2 {
			t.Errorf("Expected 2 CMP instructions, found %d", cmpCount)
		}
	})
}

// TestMinimalROMMemoryOperations tests memory operations
func TestMinimalROMMemoryOperations(t *testing.T) {
	t.Run("Zero page operations test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Zero page memory operations
		program := []uint8{
			0xA9, 0x11, // LDA #$11
			0x85, 0x10, // STA $10
			0xA9, 0x22, // LDA #$22
			0x85, 0x11, // STA $11
			0xA5, 0x10, // LDA $10
			0x18,       // CLC
			0x65, 0x11, // ADC $11
			0x85, 0x12, // STA $12 (should be $33)
			0x4C, 0x10, 0x80, // JMP $8010 (infinite loop)
		}

		result := helper.RunTestProgram(program, 100, 0x0012, 0x33, "Zero page operations test")

		if !result.Passed {
			t.Errorf("Zero page test failed: expected 0x33, got 0x%02X", result.ActualValue)
		}

		// Verify intermediate values
		value1 := helper.Memory.Read(0x0010)
		value2 := helper.Memory.Read(0x0011)

		if value1 != 0x11 {
			t.Errorf("Expected 0x11 at $10, got 0x%02X", value1)
		}

		if value2 != 0x22 {
			t.Errorf("Expected 0x22 at $11, got 0x%02X", value2)
		}

		// Verify zero page instructions in trace
		trace := helper.GetExecutionTrace()
		zpInstructions := 0

		for _, entry := range trace {
			if len(entry.Instruction) > 4 && entry.Instruction[len(entry.Instruction)-3:] != "$10" &&
				len(entry.Operands) == 1 && entry.Operands[0] < 0xFF {
				zpInstructions++
			}
		}

		t.Logf("Zero page operations completed in %d cycles", result.CycleCount)
	})

	t.Run("Absolute addressing test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Absolute addressing
		program := []uint8{
			0xA9, 0x55, // LDA #$55
			0x8D, 0x00, 0x03, // STA $0300
			0xA9, 0xAA, // LDA #$AA
			0x8D, 0x01, 0x03, // STA $0301
			0xAD, 0x00, 0x03, // LDA $0300
			0x18,             // CLC
			0x6D, 0x01, 0x03, // ADC $0301
			0x8D, 0x02, 0x03, // STA $0302 (should be $FF)
			0x4C, 0x14, 0x80, // JMP $8014 (infinite loop)
		}

		result := helper.RunTestProgram(program, 100, 0x0302, 0xFF, "Absolute addressing test")

		if !result.Passed {
			t.Errorf("Absolute addressing test failed: expected 0xFF, got 0x%02X", result.ActualValue)
		}

		// Verify absolute addressing instructions in trace
		trace := helper.GetExecutionTrace()
		absInstructions := 0

		for _, entry := range trace {
			if len(entry.Operands) == 2 {
				absInstructions++
			}
		}

		if absInstructions < 4 {
			t.Errorf("Expected at least 4 absolute instructions, found %d", absInstructions)
		}

		t.Logf("Absolute addressing test completed in %d cycles", result.CycleCount)
	})
}

// TestMinimalROMStackOperations tests stack operations
func TestMinimalROMStackOperations(t *testing.T) {
	t.Run("Push and pull test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Stack operations
		program := []uint8{
			0xA9, 0x77, // LDA #$77
			0x48,       // PHA
			0xA9, 0x88, // LDA #$88
			0x48,       // PHA
			0x68,       // PLA (should get $88)
			0x85, 0x00, // STA $00
			0x68,       // PLA (should get $77)
			0x85, 0x01, // STA $01
			0x4C, 0x0E, 0x80, // JMP $800E (infinite loop)
		}

		result1 := helper.RunTestProgram(program, 100, 0x0000, 0x88, "Stack LIFO test 1")
		if !result1.Passed {
			t.Errorf("Stack LIFO test 1 failed: expected 0x88, got 0x%02X", result1.ActualValue)
		}

		result2Value := helper.Memory.Read(0x0001)
		if result2Value != 0x77 {
			t.Errorf("Stack LIFO test 2 failed: expected 0x77, got 0x%02X", result2Value)
		}

		// Verify stack pointer changes
		trace := helper.GetExecutionTrace()
		initialSP := uint8(0xFD) // Default stack pointer after reset

		spChanges := 0
		currentSP := initialSP

		for _, entry := range trace {
			if entry.SP != currentSP {
				spChanges++
				currentSP = entry.SP
			}
		}

		if spChanges < 4 { // 2 pushes + 2 pulls should change SP at least 4 times
			t.Errorf("Expected at least 4 stack pointer changes, got %d", spChanges)
		}

		// Final stack pointer should be back to initial value
		finalSP := trace[len(trace)-1].SP
		if finalSP != initialSP {
			t.Errorf("Stack pointer should return to initial value: expected 0x%02X, got 0x%02X",
				initialSP, finalSP)
		}

		t.Logf("Stack operations test completed in %d cycles", result1.CycleCount)
	})
}

// TestMinimalROMComplexScenarios tests more complex execution scenarios
func TestMinimalROMComplexScenarios(t *testing.T) {
	t.Run("Fibonacci sequence test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Calculate Fibonacci sequence (F(6) = 8)
		program := []uint8{
			0xA9, 0x01, // LDA #$01    ; F(1) = 1
			0x85, 0x10, // STA $10     ; Store F(n-1)
			0x85, 0x11, // STA $11     ; Store F(n)
			0xA2, 0x05, // LDX #$05    ; Counter (calculate F(6))

			// Loop start at $8008
			0xA5, 0x10, // LDA $10     ; Load F(n-1)
			0x18,       // CLC
			0x65, 0x11, // ADC $11     ; Add F(n)
			0x85, 0x12, // STA $12     ; Store F(n+1)
			0xA5, 0x11, // LDA $11     ; Load F(n)
			0x85, 0x10, // STA $10     ; Store as new F(n-1)
			0xA5, 0x12, // LDA $12     ; Load F(n+1)
			0x85, 0x11, // STA $11     ; Store as new F(n)
			0xCA,       // DEX         ; Decrement counter
			0xD0, 0xF0, // BNE $8008   ; Branch if not zero

			// Result is in $11
			0x4C, 0x1A, 0x80, // JMP $801A (infinite loop)
		}

		result := helper.RunTestProgram(program, 1000, 0x0011, 0x08, "Fibonacci F(6) test")

		if !result.Passed {
			t.Errorf("Fibonacci test failed: expected F(6)=8, got %d", result.ActualValue)
		}

		// Verify intermediate Fibonacci values
		fib5 := helper.Memory.Read(0x0010) // F(5) should be 5
		if fib5 != 0x05 {
			t.Errorf("F(5) should be 5, got %d", fib5)
		}

		trace := helper.GetExecutionTrace()
		t.Logf("Fibonacci calculation completed in %d steps, %d cycles",
			len(trace), result.CycleCount)

		// Verify loop execution
		loopIterations := 0
		for _, entry := range trace {
			if entry.Instruction == "BNE $8008" {
				loopIterations++
			}
		}

		if loopIterations != 5 { // Should loop 5 times
			t.Errorf("Expected 5 loop iterations, got %d", loopIterations)
		}
	})

	t.Run("Memory copy test", func(t *testing.T) {
		helper := NewMinimalROMTestHelper()

		// Test program: Copy 4 bytes from $20-$23 to $30-$33
		program := []uint8{
			// Initialize source data
			0xA9, 0xAA, // LDA #$AA
			0x85, 0x20, // STA $20
			0xA9, 0xBB, // LDA #$BB
			0x85, 0x21, // STA $21
			0xA9, 0xCC, // LDA #$CC
			0x85, 0x22, // STA $22
			0xA9, 0xDD, // LDA #$DD
			0x85, 0x23, // STA $23

			// Copy loop
			0xA2, 0x00, // LDX #$00    ; Index

			// Loop start at $8012
			0xB5, 0x20, // LDA $20,X   ; Load from source
			0x95, 0x30, // STA $30,X   ; Store to destination
			0xE8,       // INX         ; Increment index
			0xE0, 0x04, // CPX #$04    ; Compare with 4
			0xD0, 0xF8, // BNE $8012   ; Branch if not equal

			0x4C, 0x1C, 0x80, // JMP $801C (infinite loop)
		}

		result := helper.RunTestProgram(program, 1000, 0x0030, 0xAA, "Memory copy test")

		if !result.Passed {
			t.Errorf("Memory copy test failed: expected 0xAA at $30, got 0x%02X", result.ActualValue)
		}

		// Verify all copied bytes
		expectedValues := []uint8{0xAA, 0xBB, 0xCC, 0xDD}
		for i, expected := range expectedValues {
			actual := helper.Memory.Read(uint16(0x0030 + i))
			if actual != expected {
				t.Errorf("Memory copy failed at $%02X: expected 0x%02X, got 0x%02X",
					0x30+i, expected, actual)
			}
		}

		trace := helper.GetExecutionTrace()
		t.Logf("Memory copy completed in %d steps, %d cycles", len(trace), result.CycleCount)

		// Verify indexed addressing was used
		indexedOps := 0
		for _, entry := range trace {
			if entry.Instruction == "LDA $20,X" || entry.Instruction == "STA $30,X" {
				indexedOps++
			}
		}

		if indexedOps != 8 { // 4 loads + 4 stores
			t.Errorf("Expected 8 indexed operations, found %d", indexedOps)
		}
	})
}
