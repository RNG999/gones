package bus

// Test helper methods for bus testing

// SetFrameBufferForTesting sets a frame buffer for testing purposes
func (b *Bus) SetFrameBufferForTesting(frameBuffer [256 * 240]uint32) {
	if b.PPU != nil {
		b.PPU.SetFrameBufferForTesting(frameBuffer)
	}
}

// StepWithError executes one emulation step and returns any error (exposed for testing)
func (b *Bus) StepWithError() error {
	// Handle DMA suspension
	if b.dmaInProgress && b.dmaSuspendCycles > 0 {
		b.dmaSuspendCycles--
		b.totalCycles++
		return nil
	}
	
	if b.dmaInProgress && b.dmaSuspendCycles == 0 {
		b.dmaInProgress = false
	}
	
	// Handle NMI
	if b.nmiPending {
		b.nmiPending = false
		if b.CPU != nil {
			b.CPU.TriggerNMI()
		}
	}
	
	// Execute one CPU instruction
	if b.CPU != nil {
		cycles := b.CPU.Step()
		
		b.cpuCycles += uint64(cycles)
		
		// Run PPU for 3x CPU cycles (NTSC timing)
		if b.PPU != nil {
			for i := uint64(0); i < cycles*3; i++ {
				b.PPU.Step()
				b.ppuCycles++
			}
		}
		
		b.totalCycles += uint64(cycles)
		
		// Check for frame completion
		if b.ppuCycles >= b.cyclesPerFrame {
			b.ppuCycles -= b.cyclesPerFrame
			b.frameCount++
			b.oddFrame = !b.oddFrame
		}
	}
	
	return nil
}