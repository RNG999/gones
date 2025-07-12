package integration

import (
	"runtime"
	"testing"
	"time"
)

// PerformanceBenchmarkHelper provides utilities for performance testing
type PerformanceBenchmarkHelper struct {
	*IntegrationTestHelper
	performanceMetrics *PerformanceMetrics
}

// PerformanceMetrics tracks performance characteristics
type PerformanceMetrics struct {
	InstructionsPerSecond   float64
	FramesPerSecond         float64
	CyclesPerSecond         float64
	MemoryAllocationsPerSec uint64
	GCPauseTimes            []time.Duration
	EmulationSpeedRatio     float64 // Speed relative to real hardware
}

// NewPerformanceBenchmarkHelper creates a performance benchmark helper
func NewPerformanceBenchmarkHelper() *PerformanceBenchmarkHelper {
	return &PerformanceBenchmarkHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		performanceMetrics: &PerformanceMetrics{
			GCPauseTimes: make([]time.Duration, 0),
		},
	}
}

// BenchmarkCPUInstructionThroughput benchmarks CPU instruction execution speed
func BenchmarkCPUInstructionThroughput(b *testing.B) {
	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Create program with various instruction types
	program := []uint8{
		// Fast instructions (2 cycles each)
		0xEA, // NOP
		0xE8, // INX
		0xC8, // INY
		0x18, // CLC
		0x38, // SEC
		0x58, // CLI
		0x78, // SEI
		0xAA, // TAX
		0x8A, // TXA
		0xA8, // TAY
		0x98, // TYA

		// Immediate loads (2 cycles each)
		0xA9, 0x42, // LDA #$42
		0xA2, 0x33, // LDX #$33
		0xA0, 0x55, // LDY #$55

		// Jump back
		0x4C, 0x00, 0x80, // JMP $8000
	}

	romData := make([]uint8, 0x8000)
	copy(romData, program)
	romData[0x7FFC] = 0x00
	romData[0x7FFD] = 0x80
	helper.GetMockCartridge().LoadPRG(romData)
	helper.Bus.Reset()

	b.ResetTimer()
	b.ReportAllocs()

	instructionCount := 0
	for i := 0; i < b.N; i++ {
		helper.Bus.Step()
		instructionCount++
	}

	// Calculate instructions per second
	duration := time.Duration(b.Elapsed().Nanoseconds())
	instructionsPerSecond := float64(instructionCount) / duration.Seconds()

	b.ReportMetric(instructionsPerSecond, "instructions/sec")
	b.ReportMetric(instructionsPerSecond/1000000, "Minstructions/sec")
}

// BenchmarkPPURenderingThroughput benchmarks PPU rendering performance
func BenchmarkPPURenderingThroughput(b *testing.B) {
	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Enable rendering for realistic PPU load
	helper.Memory.Write(0x2001, 0x1E) // Enable background and sprites
	helper.Memory.Write(0x2000, 0x80) // Enable NMI

	// Create pattern data for rendering
	for i := uint16(0); i < 0x1000; i++ {
		helper.Memory.Write(0x0000+i, uint8(i&0xFF)) // Pattern table 0
		helper.Memory.Write(0x1000+i, uint8(i&0xFF)) // Pattern table 1
	}

	// Set up nametables
	for i := uint16(0); i < 0x3C0; i++ {
		helper.Memory.Write(0x2000+i, uint8(i&0xFF)) // Nametable 0
	}

	// Set up sprites in OAM
	for i := 0; i < 64; i++ {
		helper.Memory.Write(0x2003, uint8(i*4))   // OAM address
		helper.Memory.Write(0x2004, uint8(i*4))   // Y position
		helper.Memory.Write(0x2004, uint8(i))     // Tile index
		helper.Memory.Write(0x2004, 0x00)         // Attributes
		helper.Memory.Write(0x2004, uint8(i*4+8)) // X position
	}

	b.ResetTimer()
	b.ReportAllocs()

	cycleCount := 0
	for i := 0; i < b.N; i++ {
		helper.PPU.Step()
		cycleCount++
	}

	// Calculate PPU cycles per second
	duration := time.Duration(b.Elapsed().Nanoseconds())
	cyclesPerSecond := float64(cycleCount) / duration.Seconds()

	b.ReportMetric(cyclesPerSecond, "ppu_cycles/sec")
	b.ReportMetric(cyclesPerSecond/5369319, "real_time_ratio")
}

// BenchmarkSystemThroughput benchmarks full system execution
func BenchmarkSystemThroughput(b *testing.B) {
	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Create realistic NES program
	program := []uint8{
		// Initialize
		0xA9, 0x00, // LDA #$00
		0x85, 0x00, // STA $00 (counter)

		// Main loop
		0xA5, 0x00, // LDA $00
		0x18,       // CLC
		0x69, 0x01, // ADC #$01
		0x85, 0x00, // STA $00

		// Some calculations
		0xA5, 0x00, // LDA $00
		0x0A,       // ASL A
		0x85, 0x01, // STA $01

		// Memory operations
		0xA2, 0x00, // LDX #$00
		0xBD, 0x00, 0x30, // LDA $3000,X
		0x9D, 0x00, 0x31, // STA $3100,X
		0xE8,       // INX
		0xE0, 0x10, // CPX #$10
		0xD0, 0xF7, // BNE -9

		// PPU operations
		0xA9, 0x3F, // LDA #$3F
		0x8D, 0x06, 0x20, // STA $2006
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006
		0xA9, 0x0F, // LDA #$0F
		0x8D, 0x07, 0x20, // STA $2007

		// Jump back
		0x4C, 0x04, 0x80, // JMP $8004 (main loop)
	}

	romData := make([]uint8, 0x8000)
	copy(romData, program)
	romData[0x7FFC] = 0x00
	romData[0x7FFD] = 0x80
	helper.GetMockCartridge().LoadPRG(romData)
	helper.Bus.Reset()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		helper.Bus.Step()
	}

	// Calculate system throughput
	duration := time.Duration(b.Elapsed().Nanoseconds())
	stepsPerSecond := float64(b.N) / duration.Seconds()

	b.ReportMetric(stepsPerSecond, "system_steps/sec")
}

// BenchmarkFrameGeneration benchmarks complete frame generation
func BenchmarkFrameGeneration(b *testing.B) {
	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Enable full rendering
	helper.Memory.Write(0x2001, 0x1E) // Show background and sprites
	helper.Memory.Write(0x2000, 0x90) // Enable NMI, use pattern table 1

	// Create test pattern
	program := []uint8{
		// VBlank handler
		0x48, // PHA
		0x8A, // TXA
		0x48, // PHA
		0x98, // TYA
		0x48, // PHA

		// Update graphics during VBlank
		0xA9, 0x20, // LDA #$20
		0x8D, 0x06, 0x20, // STA $2006
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006

		0xA2, 0x00, // LDX #$00
		0xA9, 0x01, // LDA #$01
		0x8D, 0x07, 0x20, // STA $2007
		0xE8,       // INX
		0xE0, 0x20, // CPX #$20
		0xD0, 0xF8, // BNE -8

		// Restore registers
		0x68, // PLA
		0xA8, // TAY
		0x68, // PLA
		0xAA, // TAX
		0x68, // PLA
		0x40, // RTI

		// Main program
		0xEA,             // NOP
		0x4C, 0x20, 0x80, // JMP $8020 (main loop)
	}

	romData := make([]uint8, 0x8000)
	copy(romData, program)
	// Set NMI vector to VBlank handler
	romData[0x7FFA] = 0x00
	romData[0x7FFB] = 0x80
	// Set reset vector to main program
	romData[0x7FFC] = 0x20
	romData[0x7FFD] = 0x80
	helper.GetMockCartridge().LoadPRG(romData)
	helper.Bus.Reset()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		helper.RunFrame()
	}

	// Calculate frame rate
	duration := time.Duration(b.Elapsed().Nanoseconds())
	framesPerSecond := float64(b.N) / duration.Seconds()

	b.ReportMetric(framesPerSecond, "fps")
	b.ReportMetric(framesPerSecond/60.0988, "real_time_ratio")
}

// BenchmarkMemoryAccess benchmarks memory subsystem performance
func BenchmarkMemoryAccess(b *testing.B) {
	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Test different memory access patterns
	b.Run("Sequential CPU RAM", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			address := uint16(i % 0x800) // CPU RAM range
			helper.Memory.Write(address, uint8(i))
			_ = helper.Memory.Read(address)
		}
	})

	b.Run("Random CPU RAM", func(b *testing.B) {
		addresses := make([]uint16, 1000)
		for i := range addresses {
			addresses[i] = uint16(i*37) % 0x800 // Pseudo-random pattern
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			address := addresses[i%len(addresses)]
			helper.Memory.Write(address, uint8(i))
			_ = helper.Memory.Read(address)
		}
	})

	b.Run("PPU Register Access", func(b *testing.B) {
		ppuAddresses := []uint16{0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2007}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			address := ppuAddresses[i%len(ppuAddresses)]
			if address == 0x2002 {
				_ = helper.Memory.Read(address) // PPUSTATUS read-only
			} else {
				helper.Memory.Write(address, uint8(i))
			}
		}
	})

	b.Run("Cartridge ROM Access", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			address := uint16(0x8000 + (i % 0x8000)) // PRG ROM range
			_ = helper.Memory.Read(address)
		}
	})
}

// BenchmarkDMATransfer benchmarks OAM DMA performance
func BenchmarkDMATransfer(b *testing.B) {
	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Fill page $02 with sprite data
	for i := uint16(0x0200); i < 0x0300; i++ {
		helper.Memory.Write(i, uint8(i&0xFF))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Trigger OAM DMA
		helper.Memory.Write(0x4014, 0x02) // DMA from page $02
	}

	// Calculate DMA transfers per second
	duration := time.Duration(b.Elapsed().Nanoseconds())
	transfersPerSecond := float64(b.N) / duration.Seconds()

	b.ReportMetric(transfersPerSecond, "dma_transfers/sec")
}

// BenchmarkInterruptHandling benchmarks interrupt processing performance
func BenchmarkInterruptHandling(b *testing.B) {
	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Set up interrupt vectors
	romData := make([]uint8, 0x8000)

	// NMI handler (fast)
	romData[0x0000] = 0x40 // RTI

	// IRQ handler (fast)
	romData[0x0010] = 0x40 // RTI

	// Main program
	romData[0x0100] = 0xEA // NOP
	romData[0x0101] = 0x4C // JMP
	romData[0x0102] = 0x00 // $8100
	romData[0x0103] = 0x81

	// Set vectors
	romData[0x7FFA] = 0x00 // NMI vector
	romData[0x7FFB] = 0x80
	romData[0x7FFC] = 0x00 // Reset vector
	romData[0x7FFD] = 0x81
	romData[0x7FFE] = 0x10 // IRQ vector
	romData[0x7FFF] = 0x80

	helper.GetMockCartridge().LoadPRG(romData)
	helper.Bus.Reset()

	b.Run("NMI Processing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate NMI processing by executing RTI instruction
			helper.Memory.Write(helper.CPU.PC, 0x40) // RTI at current PC
			helper.Bus.Step()                        // Execute RTI
		}
	})

	b.Run("IRQ Processing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate IRQ processing by executing RTI instruction
			helper.Memory.Write(helper.CPU.PC, 0x40) // RTI at current PC
			helper.Bus.Step()                        // Execute RTI
		}
	})
}

// BenchmarkEmulationSpeed measures emulation speed vs real hardware
func BenchmarkEmulationSpeed(b *testing.B) {
	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Create CPU-intensive program
	program := []uint8{
		// Multiplication routine (8-bit x 8-bit)
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

		// Done - repeat
		0x4C, 0x00, 0x80, // JMP $8000
	}

	romData := make([]uint8, 0x8000)
	copy(romData, program)
	romData[0x7FFC] = 0x00
	romData[0x7FFD] = 0x80
	helper.GetMockCartridge().LoadPRG(romData)
	helper.Bus.Reset()

	// Measure how many instructions we can execute in 1ms
	// Real NES executes ~1790 cycles per millisecond
	realCyclesPerMs := 1790.0

	_ = time.Now()
	cycleCount := 0

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testStart := time.Now()
		stepCount := 0

		// Execute for 1ms worth of cycles
		for time.Since(testStart) < time.Millisecond {
			helper.Bus.Step()
			stepCount++
			cycleCount += 2 // Assume average 2 cycles per step
		}

		elapsed := time.Since(testStart)
		achievedCyclesPerMs := float64(stepCount*2) / elapsed.Seconds() * 1000
		speedRatio := achievedCyclesPerMs / realCyclesPerMs

		b.ReportMetric(speedRatio, "emulation_speed_ratio")
	}
}

// BenchmarkGarbageCollectionImpact measures GC impact on timing
func BenchmarkGarbageCollectionImpact(b *testing.B) {
	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Simple loop program
	program := []uint8{
		0xEA,             // NOP
		0x4C, 0x00, 0x80, // JMP $8000
	}

	romData := make([]uint8, 0x8000)
	copy(romData, program)
	romData[0x7FFC] = 0x00
	romData[0x7FFD] = 0x80
	helper.GetMockCartridge().LoadPRG(romData)
	helper.Bus.Reset()

	b.Run("Without GC Pressure", func(b *testing.B) {
		runtime.GC() // Start clean
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			helper.Bus.Step()
		}
	})

	b.Run("With GC Pressure", func(b *testing.B) {
		// Create allocation pressure
		var allocs [][]byte

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Allocate some memory to trigger GC
			if i%100 == 0 {
				allocs = append(allocs, make([]byte, 1024))
				if len(allocs) > 1000 {
					allocs = allocs[:0] // Clear to trigger GC
				}
			}

			helper.Bus.Step()
		}

		// Prevent compiler optimization
		_ = allocs
	})
}

// TestPerformanceRegression validates performance hasn't degraded
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	helper := NewPerformanceBenchmarkHelper()
	helper.SetupBasicROM(0x8000)

	// Performance baseline thresholds (these would be established empirically)
	thresholds := struct {
		MinInstructionsPerSec float64
		MinFramesPerSecond    float64
		MaxMemoryPerFrame     uint64
	}{
		MinInstructionsPerSec: 100000, // 100K instructions/sec minimum
		MinFramesPerSecond:    30,     // Should achieve at least 30 fps
		MaxMemoryPerFrame:     1024,   // Max 1KB allocation per frame
	}

	t.Run("Instruction execution speed", func(t *testing.T) {
		program := []uint8{
			0xEA,             // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		start := time.Now()
		iterations := 10000

		for i := 0; i < iterations; i++ {
			helper.Bus.Step()
		}

		elapsed := time.Since(start)
		instructionsPerSec := float64(iterations) / elapsed.Seconds()

		t.Logf("Achieved %.0f instructions/sec", instructionsPerSec)

		if instructionsPerSec < thresholds.MinInstructionsPerSec {
			t.Errorf("Performance regression: %.0f instructions/sec < %.0f threshold",
				instructionsPerSec, thresholds.MinInstructionsPerSec)
		}
	})

	t.Run("Frame generation speed", func(t *testing.T) {
		start := time.Now()
		frameCount := 100

		for i := 0; i < frameCount; i++ {
			helper.RunFrame()
		}

		elapsed := time.Since(start)
		framesPerSec := float64(frameCount) / elapsed.Seconds()

		t.Logf("Achieved %.1f fps", framesPerSec)

		if framesPerSec < thresholds.MinFramesPerSecond {
			t.Errorf("Performance regression: %.1f fps < %.1f threshold",
				framesPerSec, thresholds.MinFramesPerSecond)
		}
	})

	t.Run("Memory allocation efficiency", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Run one frame
		helper.RunFrame()

		runtime.GC()
		runtime.ReadMemStats(&m2)

		allocatedBytes := m2.TotalAlloc - m1.TotalAlloc

		t.Logf("Allocated %d bytes per frame", allocatedBytes)

		if allocatedBytes > thresholds.MaxMemoryPerFrame {
			t.Errorf("Memory regression: %d bytes/frame > %d threshold",
				allocatedBytes, thresholds.MaxMemoryPerFrame)
		}
	})
}
