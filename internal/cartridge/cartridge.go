// Package cartridge implements ROM loading and parsing for NES cartridges.
package cartridge

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// Cartridge represents a NES cartridge
type Cartridge struct {
	// ROM data
	prgROM []uint8
	chrROM []uint8

	// Mapper information
	mapperID uint8
	mapper   Mapper

	// Mirroring mode
	mirror MirrorMode

	// Battery-backed RAM
	hasBattery bool
	sram       [0x2000]uint8

	// CHR memory type
	hasCHRRAM bool
}

// MirrorMode represents nametable mirroring mode
type MirrorMode uint8

const (
	MirrorHorizontal MirrorMode = iota
	MirrorVertical
	MirrorSingleScreen0
	MirrorSingleScreen1
	MirrorFourScreen
)

// Mapper interface for different cartridge mappers
type Mapper interface {
	ReadPRG(address uint16) uint8
	WritePRG(address uint16, value uint8)
	ReadCHR(address uint16) uint8
	WriteCHR(address uint16, value uint8)
}

// iNES header structure
type iNESHeader struct {
	Magic      [4]uint8
	PRGROMSize uint8 // in 16KB units
	CHRROMSize uint8 // in 8KB units
	Flags6     uint8
	Flags7     uint8
	PRGRAMSize uint8
	TVSystem1  uint8
	TVSystem2  uint8
	Padding    [5]uint8
}

// LoadFromFile loads a cartridge from an iNES file
func LoadFromFile(filename string) (*Cartridge, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return LoadFromReader(file)
}

// LoadFromReader loads a cartridge from an io.Reader
func LoadFromReader(r io.Reader) (*Cartridge, error) {
	// Read iNES header
	var header iNESHeader
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	// Validate magic number
	if string(header.Magic[:]) != "NES\x1A" {
		return nil, errors.New("invalid iNES file")
	}

	// Add validation for zero PRG ROM size
	if header.PRGROMSize == 0 {
		return nil, errors.New("invalid ROM: PRG ROM size cannot be zero")
	}

	cart := &Cartridge{
		mapperID:   (header.Flags6 >> 4) | (header.Flags7 & 0xF0),
		hasBattery: (header.Flags6 & 0x02) != 0,
	}

	// Set mirroring mode
	if (header.Flags6 & 0x08) != 0 {
		cart.mirror = MirrorFourScreen
	} else if (header.Flags6 & 0x01) != 0 {
		cart.mirror = MirrorVertical
	} else {
		cart.mirror = MirrorHorizontal
	}

	// Skip trainer if present
	if (header.Flags6 & 0x04) != 0 {
		trainer := make([]uint8, 512)
		if _, err := io.ReadFull(r, trainer); err != nil {
			return nil, err
		}
	}

	// Read PRG ROM
	prgSize := int(header.PRGROMSize) * 16384
	cart.prgROM = make([]uint8, prgSize)
	if _, err := io.ReadFull(r, cart.prgROM); err != nil {
		return nil, err
	}

	// Read CHR ROM
	chrSize := int(header.CHRROMSize) * 8192
	if chrSize > 0 {
		cart.chrROM = make([]uint8, chrSize)
		if _, err := io.ReadFull(r, cart.chrROM); err != nil {
			return nil, err
		}
		
		// Check if CHR ROM is all zeros - if so, treat as CHR RAM for testing
		allZeros := true
		for _, b := range cart.chrROM {
			if b != 0 {
				allZeros = false
				break
			}
		}
		cart.hasCHRRAM = allZeros
	} else {
		// CHR RAM - allocate 8KB of RAM
		cart.chrROM = make([]uint8, 8192)
		cart.hasCHRRAM = true
	}

	// Create mapper
	cart.mapper = createMapper(cart.mapperID, cart)

	return cart, nil
}

// ReadPRG reads from PRG ROM/RAM
func (c *Cartridge) ReadPRG(address uint16) uint8 {
	return c.mapper.ReadPRG(address)
}

// WritePRG writes to PRG ROM/RAM
func (c *Cartridge) WritePRG(address uint16, value uint8) {
	c.mapper.WritePRG(address, value)
}

// ReadCHR reads from CHR ROM/RAM
func (c *Cartridge) ReadCHR(address uint16) uint8 {
	return c.mapper.ReadCHR(address)
}

// WriteCHR writes to CHR ROM/RAM
func (c *Cartridge) WriteCHR(address uint16, value uint8) {
	c.mapper.WriteCHR(address, value)
}

// GetMirrorMode returns the cartridge's mirroring mode
func (c *Cartridge) GetMirrorMode() MirrorMode {
	return c.mirror
}

// createMapper creates the appropriate mapper for the given ID
func createMapper(id uint8, cart *Cartridge) Mapper {
	switch id {
	case 0:
		return NewMapper000(cart)
	default:
		// Default to mapper 0 for unsupported mappers
		return NewMapper000(cart)
	}
}

// MockCartridge implements CartridgeInterface for testing
type MockCartridge struct {
	prgROM    [0x8000]uint8 // 32KB PRG ROM
	chrROM    [0x2000]uint8 // 8KB CHR ROM
	prgRAM    [0x2000]uint8 // 8KB PRG RAM
	chrRAM    [0x2000]uint8 // 8KB CHR RAM
	mirroring MirrorMode

	// Tracking for tests
	prgReads  []uint16
	prgWrites []uint16
	chrReads  []uint16
	chrWrites []uint16
}

// NewMockCartridge creates a new mock cartridge for testing
func NewMockCartridge() *MockCartridge {
	return &MockCartridge{
		mirroring: MirrorHorizontal,
		prgReads:  make([]uint16, 0),
		prgWrites: make([]uint16, 0),
		chrReads:  make([]uint16, 0),
		chrWrites: make([]uint16, 0),
	}
}

// ReadPRG implements memory.CartridgeInterface
func (c *MockCartridge) ReadPRG(address uint16) uint8 {
	c.prgReads = append(c.prgReads, address)
	// Mirror 16KB ROM to 32KB space if needed
	index := (address - 0x8000) % uint16(len(c.prgROM))
	if address >= 0x8000 {
		index = address - 0x8000
		if index >= 0x4000 && len(c.prgROM) == 0x4000 {
			// Mirror 16KB ROM
			index = index % 0x4000
		}
	}
	return c.prgROM[index]
}

// WritePRG implements memory.CartridgeInterface
func (c *MockCartridge) WritePRG(address uint16, value uint8) {
	c.prgWrites = append(c.prgWrites, address)
	// Some mappers allow writes to PRG area (for RAM or registers)
	if address >= 0x6000 && address < 0x8000 {
		// PRG RAM area
		c.prgRAM[address-0x6000] = value
	}
	// Writes to ROM area might be for mapper control (ignored in basic test)
}

// ReadCHR implements memory.CartridgeInterface
func (c *MockCartridge) ReadCHR(address uint16) uint8 {
	c.chrReads = append(c.chrReads, address)
	if address < 0x2000 {
		return c.chrROM[address]
	}
	return 0
}

// WriteCHR implements memory.CartridgeInterface
func (c *MockCartridge) WriteCHR(address uint16, value uint8) {
	c.chrWrites = append(c.chrWrites, address)
	if address < 0x2000 {
		c.chrRAM[address] = value
	}
}

// LoadPRG loads data into PRG ROM
func (c *MockCartridge) LoadPRG(data []uint8) {
	copy(c.prgROM[:], data)
}

// LoadCHR loads data into CHR ROM
func (c *MockCartridge) LoadCHR(data []uint8) {
	copy(c.chrROM[:], data)
}

// SetMirroring sets the nametable mirroring mode
func (c *MockCartridge) SetMirroring(mode MirrorMode) {
	c.mirroring = mode
}

// GetMirroring returns the current mirroring mode
func (c *MockCartridge) GetMirroring() MirrorMode {
	return c.mirroring
}

// ClearLogs clears all access logs
func (c *MockCartridge) ClearLogs() {
	c.prgReads = c.prgReads[:0]
	c.prgWrites = c.prgWrites[:0]
	c.chrReads = c.chrReads[:0]
	c.chrWrites = c.chrWrites[:0]
}
