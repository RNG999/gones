package cpu

import (
	"testing"
)

// MockMemory implements MemoryInterface for testing
type MockMemory struct {
	data       [0x10000]uint8 // 64KB address space
	readCount  map[uint16]int
	writeCount map[uint16]int
}

// NewMockMemory creates a new mock memory instance
func NewMockMemory() *MockMemory {
	return &MockMemory{
		readCount:  make(map[uint16]int),
		writeCount: make(map[uint16]int),
	}
}

// Read implements the MemoryInterface Read method
func (m *MockMemory) Read(address uint16) uint8 {
	m.readCount[address]++
	return m.data[address]
}

// Write implements the MemoryInterface Write method
func (m *MockMemory) Write(address uint16, value uint8) {
	m.writeCount[address]++
	m.data[address] = value
}

// SetByte sets a byte at the given address
func (m *MockMemory) SetByte(address uint16, value uint8) {
	m.data[address] = value
}

// SetBytes sets multiple bytes starting at the given address
func (m *MockMemory) SetBytes(address uint16, values ...uint8) {
	for i, value := range values {
		m.data[address+uint16(i)] = value
	}
}

// GetReadCount returns the number of times an address was read
func (m *MockMemory) GetReadCount(address uint16) int {
	return m.readCount[address]
}

// GetWriteCount returns the number of times an address was written
func (m *MockMemory) GetWriteCount(address uint16) int {
	return m.writeCount[address]
}

// ClearCounts resets all read/write counts
func (m *MockMemory) ClearCounts() {
	m.readCount = make(map[uint16]int)
	m.writeCount = make(map[uint16]int)
}

// CPUTestHelper provides common test utilities
type CPUTestHelper struct {
	CPU    *CPU
	Memory *MockMemory
}

// NewCPUTestHelper creates a new test helper
func NewCPUTestHelper() *CPUTestHelper {
	memory := NewMockMemory()
	cpu := New(memory)
	return &CPUTestHelper{
		CPU:    cpu,
		Memory: memory,
	}
}

// SetupResetVector sets the reset vector and performs reset
func (h *CPUTestHelper) SetupResetVector(address uint16) {
	h.Memory.SetBytes(0xFFFC, uint8(address&0xFF), uint8(address>>8))
	h.CPU.Reset()
}

// LoadProgram loads a program starting at the given address
func (h *CPUTestHelper) LoadProgram(address uint16, program ...uint8) {
	h.Memory.SetBytes(address, program...)
}

// AssertRegisters checks if CPU registers match expected values
func (h *CPUTestHelper) AssertRegisters(t *testing.T, testName string, expectedA, expectedX, expectedY, expectedSP uint8, expectedPC uint16) {
	t.Helper()

	if h.CPU.A != expectedA {
		t.Errorf("%s: Expected A=0x%02X, got 0x%02X", testName, expectedA, h.CPU.A)
	}
	if h.CPU.X != expectedX {
		t.Errorf("%s: Expected X=0x%02X, got 0x%02X", testName, expectedX, h.CPU.X)
	}
	if h.CPU.Y != expectedY {
		t.Errorf("%s: Expected Y=0x%02X, got 0x%02X", testName, expectedY, h.CPU.Y)
	}
	if h.CPU.SP != expectedSP {
		t.Errorf("%s: Expected SP=0x%02X, got 0x%02X", testName, expectedSP, h.CPU.SP)
	}
	if h.CPU.PC != expectedPC {
		t.Errorf("%s: Expected PC=0x%04X, got 0x%04X", testName, expectedPC, h.CPU.PC)
	}
}

// AssertFlags checks if CPU flags match expected values
func (h *CPUTestHelper) AssertFlags(t *testing.T, testName string, expectedN, expectedV, expectedB, expectedD, expectedI, expectedZ, expectedC bool) {
	t.Helper()

	flags := []struct {
		name     string
		actual   bool
		expected bool
	}{
		{"N", h.CPU.N, expectedN},
		{"V", h.CPU.V, expectedV},
		{"B", h.CPU.B, expectedB},
		{"D", h.CPU.D, expectedD},
		{"I", h.CPU.I, expectedI},
		{"Z", h.CPU.Z, expectedZ},
		{"C", h.CPU.C, expectedC},
	}

	for _, flag := range flags {
		if flag.actual != flag.expected {
			t.Errorf("%s: Expected %s=%v, got %v", testName, flag.name, flag.expected, flag.actual)
		}
	}
}

// AssertMemory checks if memory at address matches expected value
func (h *CPUTestHelper) AssertMemory(t *testing.T, testName string, address uint16, expected uint8) {
	t.Helper()
	actual := h.Memory.Read(address)
	if actual != expected {
		t.Errorf("%s: Expected memory[0x%04X]=0x%02X, got 0x%02X", testName, address, expected, actual)
	}
}

// AssertCycles checks if the cycle count matches expected value
func (h *CPUTestHelper) AssertCycles(t *testing.T, testName string, expected uint64) {
	t.Helper()
	if h.CPU.cycles != expected {
		t.Errorf("%s: Expected %d cycles, got %d", testName, expected, h.CPU.cycles)
	}
}

// Status register methods are now implemented in cpu.go

// Test basic CPU initialization
func TestCPUInitialization(t *testing.T) {
	helper := NewCPUTestHelper()

	// Test initial state
	if helper.CPU.A != 0 {
		t.Errorf("Expected A=0, got %d", helper.CPU.A)
	}
	if helper.CPU.X != 0 {
		t.Errorf("Expected X=0, got %d", helper.CPU.X)
	}
	if helper.CPU.Y != 0 {
		t.Errorf("Expected Y=0, got %d", helper.CPU.Y)
	}
	if helper.CPU.SP != 0xFD {
		t.Errorf("Expected SP=0xFD, got 0x%02X", helper.CPU.SP)
	}
	if helper.CPU.PC != 0 {
		t.Errorf("Expected PC=0, got 0x%04X", helper.CPU.PC)
	}
}

// Test CPU reset functionality
func TestCPUReset(t *testing.T) {
	helper := NewCPUTestHelper()

	// Set reset vector to 0x8000
	helper.Memory.SetBytes(0xFFFC, 0x00, 0x80)

	// Modify CPU state
	helper.CPU.A = 0x55
	helper.CPU.X = 0xAA
	helper.CPU.Y = 0xFF
	helper.CPU.SP = 0x00
	helper.CPU.PC = 0x1234
	helper.CPU.I = false

	// Perform reset
	helper.CPU.Reset()

	// Check reset behavior
	// A, X, Y should be initialized to 0x00 after reset (following rgnes implementation)
	if helper.CPU.A != 0x00 {
		t.Errorf("Expected A=0x00 after reset, got 0x%02X", helper.CPU.A)
	}
	if helper.CPU.X != 0x00 {
		t.Errorf("Expected X=0x00 after reset, got 0x%02X", helper.CPU.X)
	}
	if helper.CPU.Y != 0x00 {
		t.Errorf("Expected Y=0x00 after reset, got 0x%02X", helper.CPU.Y)
	}

	// SP should be set to 0xFD
	if helper.CPU.SP != 0xFD {
		t.Errorf("Expected SP=0xFD after reset, got 0x%02X", helper.CPU.SP)
	}

	// PC should be loaded from reset vector
	if helper.CPU.PC != 0x8000 {
		t.Errorf("Expected PC=0x8000 after reset, got 0x%04X", helper.CPU.PC)
	}

	// Interrupt disable flag should be set
	if !helper.CPU.I {
		t.Errorf("Expected I flag to be set after reset")
	}
}

// Test mock memory functionality
func TestMockMemory(t *testing.T) {
	memory := NewMockMemory()

	// Test write and read
	memory.Write(0x1234, 0xAB)
	value := memory.Read(0x1234)
	if value != 0xAB {
		t.Errorf("Expected 0xAB, got 0x%02X", value)
	}

	// Test read/write counting
	if memory.GetReadCount(0x1234) != 1 {
		t.Errorf("Expected read count 1, got %d", memory.GetReadCount(0x1234))
	}
	if memory.GetWriteCount(0x1234) != 1 {
		t.Errorf("Expected write count 1, got %d", memory.GetWriteCount(0x1234))
	}

	// Test SetBytes
	memory.SetBytes(0x2000, 0x12, 0x34, 0x56)
	if memory.Read(0x2000) != 0x12 {
		t.Errorf("Expected 0x12 at 0x2000")
	}
	if memory.Read(0x2001) != 0x34 {
		t.Errorf("Expected 0x34 at 0x2001")
	}
	if memory.Read(0x2002) != 0x56 {
		t.Errorf("Expected 0x56 at 0x2002")
	}
}

// Test status register byte operations
func TestStatusRegister(t *testing.T) {
	helper := NewCPUTestHelper()

	// Test setting individual flags
	helper.CPU.N = true
	helper.CPU.V = false
	helper.CPU.B = true
	helper.CPU.D = false
	helper.CPU.I = true
	helper.CPU.Z = false
	helper.CPU.C = true

	// Expected: N=1, V=0, U=1, B=1, D=0, I=1, Z=0, C=1 = 0xB5
	expected := uint8(0xB5)
	actual := helper.CPU.GetStatusByte()
	if actual != expected {
		t.Errorf("Expected status byte 0x%02X, got 0x%02X", expected, actual)
	}

	// Test setting from byte
	helper.CPU.SetStatusByte(0x42) // 01000010 = V=1, Z=1
	if !helper.CPU.V {
		t.Errorf("Expected V flag to be set")
	}
	if !helper.CPU.Z {
		t.Errorf("Expected Z flag to be set")
	}
	if helper.CPU.N || helper.CPU.B || helper.CPU.D || helper.CPU.I || helper.CPU.C {
		t.Errorf("Expected other flags to be clear")
	}
}

// Placeholder test for Step function - will fail until implemented
func TestCPUStep(t *testing.T) {
	helper := NewCPUTestHelper()
	helper.SetupResetVector(0x8000)

	// Load a simple NOP instruction at 0x8000
	helper.LoadProgram(0x8000, 0xEA) // NOP

	// Step should execute one instruction and return cycle count
	cycles := helper.CPU.Step()

	// NOP should take 2 cycles and advance PC by 1
	if cycles != 2 {
		t.Errorf("Expected NOP to take 2 cycles, got %d", cycles)
	}

	if helper.CPU.PC != 0x8001 {
		t.Errorf("Expected PC to advance to 0x8001, got 0x%04X", helper.CPU.PC)
	}
}
