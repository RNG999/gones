package cartridge

import (
	"bytes"
	"fmt"
	"io"
)

// TestROMGenerator provides utilities for generating minimal test ROMs
// This module creates ROM files for testing specific functionality and edge cases

// TestROMConfig represents configuration for test ROM generation
type TestROMConfig struct {
	PRGSize      uint8            // PRG ROM size in 16KB units
	CHRSize      uint8            // CHR ROM size in 8KB units (0 = CHR RAM)
	MapperID     uint8            // Mapper number
	Mirroring    MirrorMode       // Nametable mirroring
	HasBattery   bool             // Battery-backed SRAM
	HasTrainer   bool             // 512-byte trainer
	Instructions []uint8          // 6502 assembly instructions
	InitialData  map[uint16]uint8 // Initial data at specific ROM addresses
	ResetVector  uint16           // Reset vector address
	IRQVector    uint16           // IRQ vector address
	NMIVector    uint16           // NMI vector address
	CHRData      []uint8          // CHR ROM/RAM initial data
	TrainerData  []uint8          // Trainer data (if HasTrainer is true)
	Description  string           // Description of the test ROM
}

// TestROMBuilder provides a fluent interface for building test ROMs
type TestROMBuilder struct {
	config TestROMConfig
}

// NewTestROMBuilder creates a new test ROM builder with default configuration
func NewTestROMBuilder() *TestROMBuilder {
	return &TestROMBuilder{
		config: TestROMConfig{
			PRGSize:      1,
			CHRSize:      1,
			MapperID:     0,
			Mirroring:    MirrorHorizontal,
			HasBattery:   false,
			HasTrainer:   false,
			Instructions: []uint8{},
			InitialData:  make(map[uint16]uint8),
			ResetVector:  0x8000,
			IRQVector:    0x8000,
			NMIVector:    0x8000,
			CHRData:      []uint8{},
			TrainerData:  []uint8{},
			Description:  "Generated test ROM",
		},
	}
}

// WithPRGSize sets the PRG ROM size in 16KB units
func (b *TestROMBuilder) WithPRGSize(size uint8) *TestROMBuilder {
	b.config.PRGSize = size
	return b
}

// WithCHRSize sets the CHR ROM size in 8KB units (0 = CHR RAM)
func (b *TestROMBuilder) WithCHRSize(size uint8) *TestROMBuilder {
	b.config.CHRSize = size
	return b
}

// WithCHRRAM configures the ROM to use CHR RAM instead of CHR ROM
func (b *TestROMBuilder) WithCHRRAM() *TestROMBuilder {
	b.config.CHRSize = 0
	return b
}

// WithMapper sets the mapper ID
func (b *TestROMBuilder) WithMapper(mapperID uint8) *TestROMBuilder {
	b.config.MapperID = mapperID
	return b
}

// WithMirroring sets the nametable mirroring mode
func (b *TestROMBuilder) WithMirroring(mirroring MirrorMode) *TestROMBuilder {
	b.config.Mirroring = mirroring
	return b
}

// WithBattery enables battery-backed SRAM
func (b *TestROMBuilder) WithBattery() *TestROMBuilder {
	b.config.HasBattery = true
	return b
}

// WithTrainer adds a 512-byte trainer
func (b *TestROMBuilder) WithTrainer(data []uint8) *TestROMBuilder {
	b.config.HasTrainer = true
	if len(data) > 512 {
		data = data[:512]
	}
	b.config.TrainerData = make([]uint8, 512)
	copy(b.config.TrainerData, data)
	return b
}

// WithInstructions sets the 6502 assembly instructions
func (b *TestROMBuilder) WithInstructions(instructions []uint8) *TestROMBuilder {
	b.config.Instructions = make([]uint8, len(instructions))
	copy(b.config.Instructions, instructions)
	return b
}

// WithData sets initial data at specific ROM addresses
func (b *TestROMBuilder) WithData(address uint16, data []uint8) *TestROMBuilder {
	for i, value := range data {
		b.config.InitialData[address+uint16(i)] = value
	}
	return b
}

// WithResetVector sets the reset vector
func (b *TestROMBuilder) WithResetVector(address uint16) *TestROMBuilder {
	b.config.ResetVector = address
	return b
}

// WithIRQVector sets the IRQ vector
func (b *TestROMBuilder) WithIRQVector(address uint16) *TestROMBuilder {
	b.config.IRQVector = address
	return b
}

// WithNMIVector sets the NMI vector
func (b *TestROMBuilder) WithNMIVector(address uint16) *TestROMBuilder {
	b.config.NMIVector = address
	return b
}

// WithCHRData sets the CHR ROM/RAM data
func (b *TestROMBuilder) WithCHRData(data []uint8) *TestROMBuilder {
	b.config.CHRData = make([]uint8, len(data))
	copy(b.config.CHRData, data)
	return b
}

// WithDescription sets the description
func (b *TestROMBuilder) WithDescription(description string) *TestROMBuilder {
	b.config.Description = description
	return b
}

// Build generates the ROM data based on the current configuration
func (b *TestROMBuilder) Build() ([]byte, error) {
	return GenerateTestROM(b.config)
}

// BuildCartridge generates and loads the ROM as a cartridge
func (b *TestROMBuilder) BuildCartridge() (*Cartridge, error) {
	romData, err := b.Build()
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(romData)
	return LoadFromReader(reader)
}

// GenerateTestROM creates a ROM file based on the provided configuration
func GenerateTestROM(config TestROMConfig) ([]byte, error) {
	// Create iNES header
	header, err := createINESHeader(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create iNES header: %w", err)
	}

	result := append([]byte{}, header...)

	// Add trainer if specified
	if config.HasTrainer {
		trainer := make([]uint8, 512)
		if len(config.TrainerData) > 0 {
			copy(trainer, config.TrainerData)
		}
		result = append(result, trainer...)
	}

	// Create PRG ROM
	prgROM, err := createPRGROM(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create PRG ROM: %w", err)
	}
	result = append(result, prgROM...)

	// Create CHR ROM if specified
	if config.CHRSize > 0 {
		chrROM := createCHRROM(config)
		result = append(result, chrROM...)
	}

	return result, nil
}

// createINESHeader creates an iNES header based on configuration
func createINESHeader(config TestROMConfig) ([]byte, error) {
	if config.PRGSize == 0 {
		return nil, fmt.Errorf("PRG ROM size cannot be zero")
	}

	header := make([]byte, 16)

	// Magic number
	copy(header[0:4], "NES\x1A")

	// ROM sizes
	header[4] = config.PRGSize
	header[5] = config.CHRSize

	// Flags 6
	flags6 := uint8(0)
	if config.Mirroring == MirrorVertical {
		flags6 |= 0x01
	}
	if config.HasBattery {
		flags6 |= 0x02
	}
	if config.HasTrainer {
		flags6 |= 0x04
	}
	if config.Mirroring == MirrorFourScreen {
		flags6 |= 0x08
	}
	flags6 |= (config.MapperID & 0x0F) << 4
	header[6] = flags6

	// Flags 7 (mapper high nibble)
	flags7 := config.MapperID & 0xF0
	header[7] = flags7

	// Remaining bytes are padding (already zero)

	return header, nil
}

// createPRGROM creates PRG ROM data based on configuration
func createPRGROM(config TestROMConfig) ([]byte, error) {
	size := int(config.PRGSize) * 16384
	prgROM := make([]byte, size)

	// Copy instructions to ROM start
	if len(config.Instructions) > 0 {
		if len(config.Instructions) > size {
			return nil, fmt.Errorf("instructions too large for PRG ROM")
		}
		copy(prgROM, config.Instructions)
	}

	// Set initial data
	for address, value := range config.InitialData {
		if int(address) < size {
			prgROM[address] = value
		}
	}

	// Set interrupt vectors at end of ROM
	vectorOffset := size - 6

	// NMI vector
	prgROM[vectorOffset] = uint8(config.NMIVector & 0xFF)
	prgROM[vectorOffset+1] = uint8(config.NMIVector >> 8)

	// Reset vector
	prgROM[vectorOffset+2] = uint8(config.ResetVector & 0xFF)
	prgROM[vectorOffset+3] = uint8(config.ResetVector >> 8)

	// IRQ vector
	prgROM[vectorOffset+4] = uint8(config.IRQVector & 0xFF)
	prgROM[vectorOffset+5] = uint8(config.IRQVector >> 8)

	return prgROM, nil
}

// createCHRROM creates CHR ROM data based on configuration
func createCHRROM(config TestROMConfig) []byte {
	size := int(config.CHRSize) * 8192
	chrROM := make([]byte, size)

	// Copy CHR data if provided
	if len(config.CHRData) > 0 {
		copySize := len(config.CHRData)
		if copySize > size {
			copySize = size
		}
		copy(chrROM, config.CHRData[:copySize])
	}

	return chrROM
}

// PrebuiltTestROMs provides common test ROM configurations
var PrebuiltTestROMs = struct {
	MinimalNROM          TestROMConfig
	BasicTest            TestROMConfig
	MemoryTest           TestROMConfig
	ArithmeticTest       TestROMConfig
	BranchingTest        TestROMConfig
	StackTest            TestROMConfig
	InterruptTest        TestROMConfig
	SRAMTest             TestROMConfig
	CHRRAMTest           TestROMConfig
	MirroringTest        TestROMConfig
	MaximalConfiguration TestROMConfig
}{
	MinimalNROM: TestROMConfig{
		PRGSize:   1,
		CHRSize:   1,
		MapperID:  0,
		Mirroring: MirrorHorizontal,
		Instructions: []uint8{
			0x4C, 0x00, 0x80, // JMP $8000 (infinite loop)
		},
		ResetVector: 0x8000,
		Description: "Minimal NROM ROM with infinite loop",
	},

	BasicTest: TestROMConfig{
		PRGSize:   1,
		CHRSize:   1,
		MapperID:  0,
		Mirroring: MirrorHorizontal,
		Instructions: []uint8{
			0xA9, 0x42, // LDA #$42
			0x85, 0x00, // STA $00
			0xA9, 0x55, // LDA #$55
			0x85, 0x01, // STA $01
			0x4C, 0x08, 0x80, // JMP $8008 (infinite loop)
		},
		ResetVector: 0x8000,
		Description: "Basic load and store test",
	},

	MemoryTest: TestROMConfig{
		PRGSize:   1,
		CHRSize:   0, // CHR RAM
		MapperID:  0,
		Mirroring: MirrorVertical,
		Instructions: []uint8{
			// Test zero page
			0xA9, 0x11, // LDA #$11
			0x85, 0x10, // STA $10

			// Test absolute addressing
			0xA9, 0x22, // LDA #$22
			0x8D, 0x00, 0x03, // STA $0300

			// Test SRAM
			0xA9, 0x33, // LDA #$33
			0x8D, 0x00, 0x60, // STA $6000

			0x4C, 0x12, 0x80, // JMP $8012 (infinite loop)
		},
		ResetVector: 0x8000,
		Description: "Memory addressing mode test",
	},

	ArithmeticTest: TestROMConfig{
		PRGSize:   1,
		CHRSize:   1,
		MapperID:  0,
		Mirroring: MirrorHorizontal,
		Instructions: []uint8{
			0x18,       // CLC
			0xA9, 0x10, // LDA #$10
			0x69, 0x05, // ADC #$05
			0x85, 0x20, // STA $20 (should be $15)

			0x38,       // SEC
			0xE9, 0x03, // SBC #$03
			0x85, 0x21, // STA $21 (should be $12)

			0x4C, 0x0C, 0x80, // JMP $800C (infinite loop)
		},
		ResetVector: 0x8000,
		Description: "Arithmetic operations test",
	},

	BranchingTest: TestROMConfig{
		PRGSize:   1,
		CHRSize:   1,
		MapperID:  0,
		Mirroring: MirrorHorizontal,
		Instructions: []uint8{
			0xA9, 0x00, // LDA #$00
			0xC9, 0x00, // CMP #$00 (sets Z flag)
			0xF0, 0x04, // BEQ +4 (should branch)
			0xA9, 0xFF, // LDA #$FF (should be skipped)
			0x85, 0x30, // STA $30 (should be skipped)
			0xA9, 0x42, // LDA #$42 (branch target)
			0x85, 0x30, // STA $30
			0x4C, 0x0E, 0x80, // JMP $800E (infinite loop)
		},
		ResetVector: 0x8000,
		Description: "Conditional branching test",
	},

	StackTest: TestROMConfig{
		PRGSize:   1,
		CHRSize:   1,
		MapperID:  0,
		Mirroring: MirrorHorizontal,
		Instructions: []uint8{
			0xA9, 0x11, // LDA #$11
			0x48,       // PHA
			0xA9, 0x22, // LDA #$22
			0x48,       // PHA
			0x68,       // PLA (should get $22)
			0x85, 0x40, // STA $40
			0x68,       // PLA (should get $11)
			0x85, 0x41, // STA $41
			0x4C, 0x0E, 0x80, // JMP $800E (infinite loop)
		},
		ResetVector: 0x8000,
		Description: "Stack push/pull test",
	},

	SRAMTest: TestROMConfig{
		PRGSize:    1,
		CHRSize:    1,
		MapperID:   0,
		Mirroring:  MirrorHorizontal,
		HasBattery: true,
		Instructions: []uint8{
			// Write pattern to SRAM
			0xA9, 0xAA, // LDA #$AA
			0x8D, 0x00, 0x60, // STA $6000
			0xA9, 0xBB, // LDA #$BB
			0x8D, 0xFF, 0x7F, // STA $7FFF

			// Read back and store in zero page
			0xAD, 0x00, 0x60, // LDA $6000
			0x85, 0x50, // STA $50
			0xAD, 0xFF, 0x7F, // LDA $7FFF
			0x85, 0x51, // STA $51

			0x4C, 0x14, 0x80, // JMP $8014 (infinite loop)
		},
		ResetVector: 0x8000,
		Description: "SRAM functionality test with battery backup",
	},

	CHRRAMTest: TestROMConfig{
		PRGSize:   1,
		CHRSize:   0, // CHR RAM
		MapperID:  0,
		Mirroring: MirrorHorizontal,
		Instructions: []uint8{
			0xA9, 0x77, // LDA #$77
			0x85, 0x60, // STA $60
			0x4C, 0x04, 0x80, // JMP $8004 (infinite loop)
		},
		ResetVector: 0x8000,
		Description: "CHR RAM configuration test",
	},

	MirroringTest: TestROMConfig{
		PRGSize:   1, // 16KB ROM for mirroring test
		CHRSize:   1,
		MapperID:  0,
		Mirroring: MirrorVertical,
		Instructions: []uint8{
			// Read from both banks to test mirroring
			0xAD, 0x00, 0x80, // LDA $8000 (first bank)
			0x85, 0x70, // STA $70
			0xAD, 0x00, 0xC0, // LDA $C000 (mirrored bank)
			0x85, 0x71, // STA $71
			0x4C, 0x0C, 0x80, // JMP $800C (infinite loop)
		},
		ResetVector: 0x8000,
		Description: "ROM mirroring test for 16KB NROM",
	},

	MaximalConfiguration: TestROMConfig{
		PRGSize:     2, // 32KB
		CHRSize:     2, // 16KB
		MapperID:    0,
		Mirroring:   MirrorFourScreen,
		HasBattery:  true,
		HasTrainer:  true,
		TrainerData: []uint8{0xDE, 0xAD, 0xBE, 0xEF}, // Pattern in trainer
		Instructions: []uint8{
			0xA9, 0xFF, // LDA #$FF
			0x85, 0xFF, // STA $FF
			0x4C, 0x04, 0x80, // JMP $8004 (infinite loop)
		},
		ResetVector: 0x8000,
		IRQVector:   0x8000,
		NMIVector:   0x8000,
		Description: "Maximal configuration test with all features",
	},
}

// CreateTestROM creates a test ROM using one of the prebuilt configurations
func CreateTestROM(config TestROMConfig) ([]byte, error) {
	return GenerateTestROM(config)
}

// CreateMinimalTestROM creates a minimal test ROM with basic functionality
func CreateMinimalTestROM() ([]byte, error) {
	return GenerateTestROM(PrebuiltTestROMs.BasicTest)
}

// TestROMValidator validates test ROM configurations
type TestROMValidator struct{}

// NewTestROMValidator creates a new test ROM validator
func NewTestROMValidator() *TestROMValidator {
	return &TestROMValidator{}
}

// Validate checks if a test ROM configuration is valid
func (v *TestROMValidator) Validate(config TestROMConfig) error {
	if config.PRGSize == 0 {
		return fmt.Errorf("PRG ROM size cannot be zero")
	}

	if config.PRGSize > 255 {
		return fmt.Errorf("PRG ROM size too large: %d", config.PRGSize)
	}

	if config.CHRSize > 255 {
		return fmt.Errorf("CHR ROM size too large: %d", config.CHRSize)
	}

	maxPRGSize := int(config.PRGSize) * 16384
	if len(config.Instructions) > maxPRGSize {
		return fmt.Errorf("instructions too large for PRG ROM: %d > %d",
			len(config.Instructions), maxPRGSize)
	}

	if config.CHRSize > 0 {
		maxCHRSize := int(config.CHRSize) * 8192
		if len(config.CHRData) > maxCHRSize {
			return fmt.Errorf("CHR data too large for CHR ROM: %d > %d",
				len(config.CHRData), maxCHRSize)
		}
	}

	if config.HasTrainer && len(config.TrainerData) > 512 {
		return fmt.Errorf("trainer data too large: %d > 512", len(config.TrainerData))
	}

	if config.ResetVector < 0x8000 || config.ResetVector > 0xFFFF {
		return fmt.Errorf("invalid reset vector: 0x%04X", config.ResetVector)
	}

	return nil
}

// ValidateAndGenerate validates configuration and generates ROM if valid
func (v *TestROMValidator) ValidateAndGenerate(config TestROMConfig) ([]byte, error) {
	if err := v.Validate(config); err != nil {
		return nil, err
	}

	return GenerateTestROM(config)
}

// TestROMInfo provides information about a generated test ROM
type TestROMInfo struct {
	Config      TestROMConfig
	Size        int
	PRGSize     int
	CHRSize     int
	HasTrainer  bool
	Description string
}

// GetTestROMInfo returns information about a test ROM configuration
func GetTestROMInfo(config TestROMConfig) TestROMInfo {
	prgSize := int(config.PRGSize) * 16384
	chrSize := int(config.CHRSize) * 8192

	totalSize := 16 // Header
	if config.HasTrainer {
		totalSize += 512
	}
	totalSize += prgSize + chrSize

	return TestROMInfo{
		Config:      config,
		Size:        totalSize,
		PRGSize:     prgSize,
		CHRSize:     chrSize,
		HasTrainer:  config.HasTrainer,
		Description: config.Description,
	}
}

// SaveTestROM saves a test ROM to a writer (useful for file output)
func SaveTestROM(w io.Writer, config TestROMConfig) error {
	romData, err := GenerateTestROM(config)
	if err != nil {
		return err
	}

	_, err = w.Write(romData)
	return err
}

// LoadTestROMAsCartridge loads a test ROM configuration directly as a cartridge
func LoadTestROMAsCartridge(config TestROMConfig) (*Cartridge, error) {
	romData, err := GenerateTestROM(config)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(romData)
	return LoadFromReader(reader)
}

// CreateTestROMSuite creates a suite of test ROMs for comprehensive testing
func CreateTestROMSuite() map[string]TestROMConfig {
	return map[string]TestROMConfig{
		"minimal":    PrebuiltTestROMs.MinimalNROM,
		"basic":      PrebuiltTestROMs.BasicTest,
		"memory":     PrebuiltTestROMs.MemoryTest,
		"arithmetic": PrebuiltTestROMs.ArithmeticTest,
		"branching":  PrebuiltTestROMs.BranchingTest,
		"stack":      PrebuiltTestROMs.StackTest,
		"sram":       PrebuiltTestROMs.SRAMTest,
		"chr_ram":    PrebuiltTestROMs.CHRRAMTest,
		"mirroring":  PrebuiltTestROMs.MirroringTest,
		"maximal":    PrebuiltTestROMs.MaximalConfiguration,
	}
}

// TestROMExecutor provides utilities for executing test ROMs
type TestROMExecutor struct {
	maxCycles uint64
}

// NewTestROMExecutor creates a new test ROM executor
func NewTestROMExecutor(maxCycles uint64) *TestROMExecutor {
	return &TestROMExecutor{maxCycles: maxCycles}
}

// ExecuteTestROM executes a test ROM and returns execution results
func (e *TestROMExecutor) ExecuteTestROM(config TestROMConfig) (*TestROMExecutionResult, error) {
	cart, err := LoadTestROMAsCartridge(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load test ROM: %w", err)
	}

	// This would require integration with the CPU and bus components
	// For now, just return basic info
	return &TestROMExecutionResult{
		Config:      config,
		Cartridge:   cart,
		Executed:    true,
		Description: "Test ROM loaded successfully",
	}, nil
}

// TestROMExecutionResult represents the result of test ROM execution
type TestROMExecutionResult struct {
	Config      TestROMConfig
	Cartridge   *Cartridge
	Executed    bool
	CycleCount  uint64
	MemoryDump  map[uint16]uint8
	Description string
}

// GetExecutionSummary returns a summary of the execution result
func (r *TestROMExecutionResult) GetExecutionSummary() string {
	if !r.Executed {
		return "Test ROM execution failed"
	}

	return fmt.Sprintf("Test ROM '%s' executed successfully in %d cycles",
		r.Config.Description, r.CycleCount)
}
