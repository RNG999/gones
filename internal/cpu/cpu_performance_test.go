package cpu

import (
	"runtime"
	"testing"
	"time"
)

// CPUPerformanceHelper provides CPU-specific performance testing utilities
type CPUPerformanceHelper struct {
	*CPUTestHelper
	cycleCounter uint64
	startTime    time.Time
}

// NewCPUPerformanceHelper creates a CPU performance test helper
func NewCPUPerformanceHelper() *CPUPerformanceHelper {
	return &CPUPerformanceHelper{
		CPUTestHelper: NewCPUTestHelper(),
		cycleCounter:  0,
		startTime:     time.Now(),
	}
}

// StepWithProfiling executes one CPU step while tracking performance metrics
func (h *CPUPerformanceHelper) StepWithProfiling() uint64 {
	cycles := h.CPU.Step()
	h.cycleCounter += cycles
	return cycles
}

// GetCyclesPerSecond calculates current cycle execution rate
func (h *CPUPerformanceHelper) GetCyclesPerSecond() float64 {
	elapsed := time.Since(h.startTime)
	if elapsed.Seconds() == 0 {
		return 0
	}
	return float64(h.cycleCounter) / elapsed.Seconds()
}

// BenchmarkBasicInstructions benchmarks fundamental CPU instruction performance
func BenchmarkBasicInstructions(b *testing.B) {
	b.Run("NOP", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)
		helper.LoadProgram(0x8000, 0xEA, 0x4C, 0x00, 0x80) // NOP; JMP $8000

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Register Transfers", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		program := []uint8{
			0xAA,             // TAX
			0x8A,             // TXA
			0xA8,             // TAY
			0x98,             // TYA
			0xBA,             // TSX
			0x9A,             // TXS
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Arithmetic Operations", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		program := []uint8{
			0xA9, 0x10, // LDA #$10
			0x69, 0x05, // ADC #$05
			0xE9, 0x03, // SBC #$03
			0x29, 0x0F, // AND #$0F
			0x09, 0xF0, // ORA #$F0
			0x49, 0xFF, // EOR #$FF
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Increment/Decrement", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		program := []uint8{
			0xE8,             // INX
			0xCA,             // DEX
			0xC8,             // INY
			0x88,             // DEY
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})
}

// BenchmarkAddressingModes benchmarks different addressing mode performance
func BenchmarkAddressingModes(b *testing.B) {
	b.Run("Immediate", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		program := []uint8{
			0xA9, 0x42, // LDA #$42
			0xA2, 0x33, // LDX #$33
			0xA0, 0x55, // LDY #$55
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Zero Page", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// Setup test data
		helper.Memory.SetByte(0x80, 0x42)
		helper.Memory.SetByte(0x81, 0x33)
		helper.Memory.SetByte(0x82, 0x55)

		program := []uint8{
			0xA5, 0x80, // LDA $80
			0xA6, 0x81, // LDX $81
			0xA4, 0x82, // LDY $82
			0x85, 0x83, // STA $83
			0x86, 0x84, // STX $84
			0x84, 0x85, // STY $85
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Zero Page Indexed", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// Setup test data and indices
		helper.CPU.X = 0x05
		helper.CPU.Y = 0x03
		for i := uint16(0x80); i < 0x90; i++ {
			helper.Memory.SetByte(i, uint8(i))
		}

		program := []uint8{
			0xB5, 0x80, // LDA $80,X
			0xB4, 0x81, // LDY $81,X
			0xB6, 0x82, // LDX $82,Y
			0x95, 0x83, // STA $83,X
			0x94, 0x84, // STY $84,X
			0x96, 0x85, // STX $85,Y
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Absolute", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// Setup test data
		helper.Memory.SetByte(0x3000, 0x42)
		helper.Memory.SetByte(0x3001, 0x33)
		helper.Memory.SetByte(0x3002, 0x55)

		program := []uint8{
			0xAD, 0x00, 0x30, // LDA $3000
			0xAE, 0x01, 0x30, // LDX $3001
			0xAC, 0x02, 0x30, // LDY $3002
			0x8D, 0x03, 0x30, // STA $3003
			0x8E, 0x04, 0x30, // STX $3004
			0x8C, 0x05, 0x30, // STY $3005
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Absolute Indexed", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// Setup test data and indices
		helper.CPU.X = 0x10
		helper.CPU.Y = 0x08
		for i := uint16(0x3000); i < 0x3100; i++ {
			helper.Memory.SetByte(i, uint8(i))
		}

		program := []uint8{
			0xBD, 0x00, 0x30, // LDA $3000,X
			0xB9, 0x00, 0x30, // LDA $3000,Y
			0xBE, 0x00, 0x30, // LDX $3000,Y
			0x9D, 0x40, 0x30, // STA $3040,X
			0x99, 0x50, 0x30, // STA $3050,Y
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})
}

// BenchmarkBranchInstructions benchmarks branch instruction performance
func BenchmarkBranchInstructions(b *testing.B) {
	b.Run("Branch Not Taken", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		program := []uint8{
			0xA9, 0x00, // LDA #$00 (sets Z flag)
			0xD0, 0x02, // BNE +2 (not taken, Z is set)
			0xEA,             // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Branch Taken No Page Cross", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		program := []uint8{
			0xA9, 0x01, // LDA #$01 (clears Z flag)
			0xD0, 0x01, // BNE +1 (taken)
			0xEA,             // NOP (skipped)
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Branch Taken Page Cross", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x80F0) // Near page boundary

		program := []uint8{
			0xA9, 0x01, // LDA #$01 (clears Z flag)
			0xD0, 0x20, // BNE +32 (crosses page boundary)
		}
		helper.LoadProgram(0x80F0, program...)

		// Add target instruction
		helper.LoadProgram(0x8112, 0x4C, 0xF0, 0x80) // JMP $80F0

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})
}

// BenchmarkStackOperations benchmarks stack instruction performance
func BenchmarkStackOperations(b *testing.B) {
	b.Run("Push/Pull Accumulator", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		program := []uint8{
			0xA9, 0x42, // LDA #$42
			0x48,             // PHA
			0x68,             // PLA
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Push/Pull Status", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		program := []uint8{
			0x38,             // SEC (set carry)
			0x08,             // PHP
			0x18,             // CLC (clear carry)
			0x28,             // PLP (restore carry)
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})
}

// BenchmarkReadModifyWrite benchmarks RMW instruction performance
func BenchmarkReadModifyWrite(b *testing.B) {
	b.Run("Zero Page RMW", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// Setup test data
		helper.Memory.SetByte(0x80, 0x40)
		helper.Memory.SetByte(0x81, 0x80)
		helper.Memory.SetByte(0x82, 0x01)

		program := []uint8{
			0xE6, 0x80, // INC $80
			0xC6, 0x80, // DEC $80
			0x06, 0x81, // ASL $81
			0x46, 0x81, // LSR $81
			0x26, 0x82, // ROL $82
			0x66, 0x82, // ROR $82
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Absolute RMW", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// Setup test data
		helper.Memory.SetByte(0x3000, 0x40)
		helper.Memory.SetByte(0x3001, 0x80)

		program := []uint8{
			0xEE, 0x00, 0x30, // INC $3000
			0xCE, 0x00, 0x30, // DEC $3000
			0x0E, 0x01, 0x30, // ASL $3001
			0x4E, 0x01, 0x30, // LSR $3001
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Indexed RMW", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// Setup test data and index
		helper.CPU.X = 0x05
		helper.Memory.SetByte(0x85, 0x40)
		helper.Memory.SetByte(0x3005, 0x80)

		program := []uint8{
			0xF6, 0x80, // INC $80,X
			0xD6, 0x80, // DEC $80,X
			0xFE, 0x00, 0x30, // INC $3000,X
			0xDE, 0x00, 0x30, // DEC $3000,X
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})
}

// BenchmarkComplexPrograms benchmarks realistic CPU instruction sequences
func BenchmarkComplexPrograms(b *testing.B) {
	b.Run("Multiplication 8x8", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// 8-bit x 8-bit multiplication using shift-and-add
		program := []uint8{
			// Initialize
			0xA9, 0x00, // LDA #$00 (result)
			0x85, 0x02, // STA $02
			0xA9, 0x0F, // LDA #$0F (multiplicand)
			0x85, 0x00, // STA $00
			0xA9, 0x0D, // LDA #$0D (multiplier)
			0x85, 0x01, // STA $01

			// Multiply loop
			0xA5, 0x01, // LDA $01
			0xF0, 0x0C, // BEQ +12 (done)
			0x4A,       // LSR A
			0x85, 0x01, // STA $01
			0x90, 0x06, // BCC +6
			0xA5, 0x02, // LDA $02
			0x18,       // CLC
			0x65, 0x00, // ADC $00
			0x85, 0x02, // STA $02
			0x06, 0x00, // ASL $00
			0x4C, 0x0E, 0x80, // JMP $800E (loop)

			// Done - restart
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Memory Copy Loop", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// Setup source data
		for i := uint16(0x3000); i < 0x3100; i++ {
			helper.Memory.SetByte(i, uint8(i&0xFF))
		}

		// Memory copy routine
		program := []uint8{
			// Initialize pointers
			0xA2, 0x00, // LDX #$00

			// Copy loop
			0xBD, 0x00, 0x30, // LDA $3000,X
			0x9D, 0x00, 0x31, // STA $3100,X
			0xE8,       // INX
			0xE0, 0x10, // CPX #$10
			0xD0, 0xF7, // BNE -9

			// Restart
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})

	b.Run("Sorting Algorithm", func(b *testing.B) {
		helper := NewCPUPerformanceHelper()
		helper.SetupResetVector(0x8000)

		// Setup unsorted data
		testData := []uint8{0x05, 0x02, 0x08, 0x01, 0x09, 0x03, 0x07, 0x04}
		for i, val := range testData {
			helper.Memory.SetByte(uint16(0x3000+i), val)
		}

		// Simple bubble sort
		program := []uint8{
			// Outer loop
			0xA2, 0x00, // LDX #$00 (i)

			// Inner loop
			0xA0, 0x00, // LDY #$00 (j)
			0xB9, 0x00, 0x30, // LDA $3000,Y
			0xC8,             // INY
			0xD9, 0x00, 0x30, // CMP $3000,Y
			0x90, 0x0F, // BCC +15 (no swap)

			// Swap elements
			0xAA,             // TAX (save A)
			0xB9, 0x00, 0x30, // LDA $3000,Y
			0x88,             // DEY
			0x99, 0x00, 0x30, // STA $3000,Y
			0x8A,             // TXA (restore A)
			0xC8,             // INY
			0x99, 0x00, 0x30, // STA $3000,Y

			// Continue inner loop
			0xC0, 0x07, // CPY #$07
			0xD0, 0xE7, // BNE -25 (inner loop)

			// Continue outer loop
			0xE8,       // INX
			0xE0, 0x07, // CPX #$07
			0xD0, 0xE1, // BNE -31 (outer loop)

			// Restart
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			helper.CPU.Step()
		}

		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "instructions/sec")
	})
}

// BenchmarkCPUEmulationSpeed measures CPU emulation speed vs real hardware
func BenchmarkCPUEmulationSpeed(b *testing.B) {
	helper := NewCPUPerformanceHelper()
	helper.SetupResetVector(0x8000)

	// Mixed instruction program
	program := []uint8{
		0xA9, 0x00, // LDA #$00
		0x85, 0x00, // STA $00
		0xA2, 0x10, // LDX #$10
		0xA5, 0x00, // LDA $00
		0x18,       // CLC
		0x69, 0x01, // ADC #$01
		0x85, 0x00, // STA $00
		0xCA,       // DEX
		0xD0, 0xF7, // BNE -9
		0x4C, 0x00, 0x80, // JMP $8000
	}
	helper.LoadProgram(0x8000, program...)

	// Real NES CPU runs at 1.789773 MHz
	realCPUFrequency := 1789773.0

	b.ResetTimer()

	start := time.Now()
	cycleCount := uint64(0)

	for i := 0; i < b.N; i++ {
		cycles := helper.CPU.Step()
		cycleCount += cycles
	}

	elapsed := time.Since(start)
	emulatedFrequency := float64(cycleCount) / elapsed.Seconds()
	speedRatio := emulatedFrequency / realCPUFrequency

	b.ReportMetric(emulatedFrequency, "cycles/sec")
	b.ReportMetric(speedRatio, "speed_ratio")
	b.ReportMetric(emulatedFrequency/1000000, "MHz")
}

// TestCPUPerformanceRegression validates CPU performance hasn't degraded
func TestCPUPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CPU performance regression test in short mode")
	}

	helper := NewCPUPerformanceHelper()
	helper.SetupResetVector(0x8000)

	// Performance thresholds (would be established empirically)
	thresholds := struct {
		MinInstructionsPerSec float64
		MinCyclesPerSec       float64
		MaxMemoryPerInst      uint64
	}{
		MinInstructionsPerSec: 100000, // 100K instructions/sec
		MinCyclesPerSec:       200000, // 200K cycles/sec
		MaxMemoryPerInst:      100,    // 100 bytes per instruction
	}

	t.Run("Instruction execution regression", func(t *testing.T) {
		program := []uint8{
			0xEA,             // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		start := time.Now()
		iterations := 10000

		for i := 0; i < iterations; i++ {
			helper.CPU.Step()
		}

		elapsed := time.Since(start)
		instructionsPerSec := float64(iterations) / elapsed.Seconds()

		t.Logf("CPU performance: %.0f instructions/sec", instructionsPerSec)

		if instructionsPerSec < thresholds.MinInstructionsPerSec {
			t.Errorf("CPU performance regression: %.0f < %.0f instructions/sec",
				instructionsPerSec, thresholds.MinInstructionsPerSec)
		}
	})

	t.Run("Cycle execution regression", func(t *testing.T) {
		program := []uint8{
			0xA9, 0x42, // LDA #$42 (2 cycles)
			0x85, 0x00, // STA $00 (3 cycles)
			0xA5, 0x00, // LDA $00 (3 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		helper.LoadProgram(0x8000, program...)

		start := time.Now()
		totalCycles := uint64(0)
		iterations := 1000

		for i := 0; i < iterations*4; i++ { // 4 instructions per iteration
			cycles := helper.CPU.Step()
			totalCycles += cycles
		}

		elapsed := time.Since(start)
		cyclesPerSec := float64(totalCycles) / elapsed.Seconds()

		t.Logf("CPU performance: %.0f cycles/sec", cyclesPerSec)

		if cyclesPerSec < thresholds.MinCyclesPerSec {
			t.Errorf("CPU cycle regression: %.0f < %.0f cycles/sec",
				cyclesPerSec, thresholds.MinCyclesPerSec)
		}
	})

	t.Run("Memory allocation regression", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Execute some instructions
		program := []uint8{
			0xEA,             // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}
		helper.LoadProgram(0x8000, program...)

		for i := 0; i < 1000; i++ {
			helper.CPU.Step()
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		allocatedBytes := m2.TotalAlloc - m1.TotalAlloc
		bytesPerInstruction := allocatedBytes / 1000

		t.Logf("Memory allocation: %d bytes for 1000 instructions (%.1f bytes/instruction)",
			allocatedBytes, float64(bytesPerInstruction))

		if bytesPerInstruction > thresholds.MaxMemoryPerInst {
			t.Errorf("Memory allocation regression: %d > %d bytes/instruction",
				bytesPerInstruction, thresholds.MaxMemoryPerInst)
		}
	})
}
