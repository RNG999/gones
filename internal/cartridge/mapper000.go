// Package cartridge implements ROM loading and parsing for NES cartridges.
package cartridge

// Mapper000 implements NROM (mapper 0)
// NROM is the simplest mapper with no bank switching capabilities.
// It supports:
// - 16KB or 32KB PRG ROM (16KB is mirrored to fill 32KB address space)
// - 8KB CHR ROM or CHR RAM
// - 8KB PRG RAM (SRAM) at 0x6000-0x7FFF (optionally battery-backed)
type Mapper000 struct {
	cart     *Cartridge
	prgBanks uint8 // Number of 16KB PRG banks (1 or 2)
}

// NewMapper000 creates a new NROM mapper
func NewMapper000(cart *Cartridge) *Mapper000 {
	return &Mapper000{
		cart:     cart,
		prgBanks: uint8(len(cart.prgROM) / 0x4000),
	}
}

// ReadPRG reads from PRG ROM/RAM
// Memory map:
// 0x6000-0x7FFF: 8KB PRG RAM (SRAM)
// 0x8000-0xFFFF: 32KB PRG ROM space
//   - For 16KB ROMs: mirrored (0x8000-0xBFFF mirrors to 0xC000-0xFFFF)
//   - For 32KB ROMs: direct mapped
func (m *Mapper000) ReadPRG(address uint16) uint8 {
	if address >= 0x8000 {
		// Map to PRG ROM
		if len(m.cart.prgROM) == 0 {
			return 0 // Handle zero-length ROM gracefully
		}
		// Calculate offset from base of ROM
		offset := address - 0x8000
		if m.prgBanks == 1 {
			// 16KB ROM, mirrored to fill 32KB space
			// Mask to 16KB boundary
			offset &= 0x3FFF
		}
		// For 32KB ROM, offset can be up to 0x7FFF
		// Ensure we don't exceed ROM bounds
		if int(offset) < len(m.cart.prgROM) {
			return m.cart.prgROM[offset]
		}
		return 0
	} else if address >= 0x6000 && address < 0x8000 {
		// PRG RAM (SRAM) - 8KB range from 0x6000-0x7FFF
		return m.cart.sram[address-0x6000]
	}
	return 0
}

// WritePRG writes to PRG RAM
func (m *Mapper000) WritePRG(address uint16, value uint8) {
	if address >= 0x6000 && address < 0x8000 {
		// PRG RAM (SRAM) - 8KB range from 0x6000-0x7FFF
		m.cart.sram[address-0x6000] = value
	}
	// Writes to ROM area are ignored
}

// ReadCHR reads from CHR ROM/RAM
func (m *Mapper000) ReadCHR(address uint16) uint8 {
	if address < 0x2000 {
		// CHR memory is at PPU addresses 0x0000-0x1FFF (8KB)
		if len(m.cart.chrROM) > 0 && int(address) < len(m.cart.chrROM) {
			return m.cart.chrROM[address]
		}
	}
	return 0
}

// WriteCHR writes to CHR RAM
func (m *Mapper000) WriteCHR(address uint16, value uint8) {
	if address < 0x2000 {
		// CHR memory is at PPU addresses 0x0000-0x1FFF (8KB)
		// Only allow writes to CHR RAM (when CHR ROM size was 0 in header)
		if m.cart.hasCHRRAM && len(m.cart.chrROM) > 0 && int(address) < len(m.cart.chrROM) {
			m.cart.chrROM[address] = value
		}
		// Writes to CHR ROM are ignored
	}
}
