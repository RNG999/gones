package cartridge

import (
	"fmt"
)

// LoadFromBytes creates a cartridge from raw ROM bytes for testing
func LoadFromBytes(data []byte) (*Cartridge, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("ROM data too small: need at least 16 bytes for header")
	}
	
	// Verify iNES header
	if string(data[0:4]) != "NES\x1a" {
		return nil, fmt.Errorf("invalid iNES header")
	}
	
	// Parse header
	prgROMSize := int(data[4]) * 16384  // 16KB units
	chrROMSize := int(data[5]) * 8192   // 8KB units
	flags6 := data[6]
	flags7 := data[7]
	
	// Calculate mapper number
	mapperNumber := (flags7 & 0xF0) | (flags6 >> 4)
	
	// Extract mirroring mode
	mirrorMode := uint8(0) // Horizontal by default
	if flags6&0x01 != 0 {
		mirrorMode = 1 // Vertical
	}
	
	// Calculate ROM data offsets
	headerSize := 16
	if flags6&0x04 != 0 { // Has trainer
		headerSize += 512
	}
	
	prgROMStart := headerSize
	chrROMStart := prgROMStart + prgROMSize
	
	// Validate data size
	expectedSize := chrROMStart + chrROMSize
	if len(data) < expectedSize {
		return nil, fmt.Errorf("ROM data too small: expected %d bytes, got %d", expectedSize, len(data))
	}
	
	// Extract ROM data
	var prgROM, chrROM []byte
	if prgROMSize > 0 {
		prgROM = make([]byte, prgROMSize)
		copy(prgROM, data[prgROMStart:prgROMStart+prgROMSize])
	}
	
	if chrROMSize > 0 {
		chrROM = make([]byte, chrROMSize)
		copy(chrROM, data[chrROMStart:chrROMStart+chrROMSize])
	} else {
		// CHR RAM if no CHR ROM
		chrROM = make([]byte, 8192)
	}
	
	// Create cartridge
	cart := &Cartridge{
		prgROM:     prgROM,
		chrROM:     chrROM,
		mapperID:   mapperNumber,
		mirror:     MirrorMode(mirrorMode),
		hasBattery: false,
		hasCHRRAM:  chrROMSize == 0,
	}
	
	// Initialize mapper based on mapper ID
	switch mapperNumber {
	case 0:
		cart.mapper = NewMapper000(cart)
	default:
		return nil, fmt.Errorf("unsupported mapper: %d", mapperNumber)
	}
	
	return cart, nil
}