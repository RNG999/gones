package ppu

// Test helper methods for PPU testing

// SetFrameBufferForTesting sets a frame buffer for testing purposes
func (p *PPU) SetFrameBufferForTesting(frameBuffer [256 * 240]uint32) {
	p.frameBuffer = frameBuffer
}