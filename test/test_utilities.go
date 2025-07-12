package test

// MockCartridge provides a simple cartridge implementation for testing
type MockCartridge struct {
	chrData [0x2000]uint8
}

func (m *MockCartridge) ReadPRG(address uint16) uint8  { return 0 }
func (m *MockCartridge) WritePRG(address uint16, value uint8) {}
func (m *MockCartridge) ReadCHR(address uint16) uint8  { return m.chrData[address&0x1FFF] }
func (m *MockCartridge) WriteCHR(address uint16, value uint8) { m.chrData[address&0x1FFF] = value }