package integration

import (
	"testing"
)

// TestSpriteDMAGameplayScenarios tests NMI and DMA behavior in realistic gameplay scenarios
func TestSpriteDMAGameplayScenarios(t *testing.T) {
	t.Run("Typical_VBlank_Sprite_Update", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that performs typical sprite DMA
		nmiHandler := []uint8{
			// Save registers
			0x48,             // PHA
			0x8A,             // TXA
			0x48,             // PHA
			0x98,             // TYA
			0x48,             // PHA
			
			// Perform OAM DMA
			0xA9, 0x02,       // LDA #$02 (sprite data page)
			0x8D, 0x14, 0x40, // STA $4014 (trigger DMA)
			
			// Mark NMI completion
			0xE6, 0x50,       // INC $50 (NMI counter)
			
			// Restore registers
			0x68,             // PLA
			0xA8,             // TAY
			0x68,             // PLA
			0xAA,             // TAX
			0x68,             // PLA
			
			0x40,             // RTI
		}

		// Main game loop simulation
		program := []uint8{
			// Initialize graphics
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)
			
			// Set up initial sprite data
			0xA2, 0x00,       // LDX #$00
			0xA9, 0x80,       // LDA #$80 (Y position)
			0x9D, 0x00, 0x02, // STA $0200,X (sprite Y)
			0xA9, 0x01,       // LDA #$01 (tile index)
			0x9D, 0x01, 0x02, // STA $0201,X (sprite tile)
			0xA9, 0x00,       // LDA #$00 (attributes)
			0x9D, 0x02, 0x02, // STA $0202,X (sprite attr)
			0xA9, 0x80,       // LDA #$80 (X position)
			0x9D, 0x03, 0x02, // STA $0203,X (sprite X)
			0x8A,             // TXA
			0x18,             // CLC
			0x69, 0x04,       // ADC #$04
			0xAA,             // TAX
			0xE0, 0x20,       // CPX #$20 (8 sprites)
			0x90, 0xE5,       // BCC (sprite setup loop)
			
			// Main game loop
			0xE6, 0x51,       // INC $51 (game loop counter)
			
			// Update sprite positions (simple animation)
			0xA2, 0x00,       // LDX #$00
			0xBD, 0x03, 0x02, // LDA $0203,X (sprite X)
			0x18,             // CLC
			0x69, 0x01,       // ADC #$01 (move right)
			0x9D, 0x03, 0x02, // STA $0203,X (update X)
			0x8A,             // TXA
			0x18,             // CLC
			0x69, 0x04,       // ADC #$04
			0xAA,             // TAX
			0xE0, 0x20,       // CPX #$20
			0x90, 0xF0,       // BCC (update loop)
			
			// Wait for VBlank (game loop continues)
			0x4C, 0x24, 0x80, // JMP to game loop
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize test variables
		helper.Memory.Write(0x0050, 0x00) // NMI counter
		helper.Memory.Write(0x0051, 0x00) // Game loop counter

		// Execute several frames to test typical behavior
		framesProcessed := 0
		lastNMICount := uint8(0)
		
		for i := 0; i < 150000 && framesProcessed < 5; i++ {
			helper.Bus.Step()
			
			currentNMICount := helper.Memory.Read(0x0050)
			if currentNMICount > lastNMICount {
				framesProcessed++
				gameLoopCount := helper.Memory.Read(0x0051)
				
				// Check sprite positions after DMA
				sprite0X := helper.Memory.Read(0x0203)
				sprite1X := helper.Memory.Read(0x0207)
				
				t.Logf("Frame %d completed (step %d):", framesProcessed, i)
				t.Logf("  NMI count: %d", currentNMICount)
				t.Logf("  Game loops: %d", gameLoopCount)
				t.Logf("  Sprite 0 X: %d", sprite0X)
				t.Logf("  Sprite 1 X: %d", sprite1X)
				
				lastNMICount = currentNMICount
			}
		}

		if framesProcessed < 3 {
			t.Errorf("Expected at least 3 frames, got %d", framesProcessed)
		}

		t.Logf("Typical VBlank sprite update test completed: %d frames processed", framesProcessed)
	})

	t.Run("Double_Buffered_Sprite_System", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler with double buffering
		nmiHandler := []uint8{
			0x48,             // PHA
			
			// Check which buffer to use
			0xA5, 0x60,       // LDA $60 (buffer selector)
			0xF0, 0x06,       // BEQ (use buffer 0)
			
			// Use buffer 1 (page $03)
			0xA9, 0x03,       // LDA #$03
			0x8D, 0x14, 0x40, // STA $4014
			0x4C, 0x0B, 0x81, // JMP to end
			
			// Use buffer 0 (page $02)
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014
			
			// Toggle buffer
			0xA5, 0x60,       // LDA $60
			0x49, 0x01,       // EOR #$01 (toggle bit 0)
			0x85, 0x60,       // STA $60
			
			// Mark frame
			0xE6, 0x61,       // INC $61
			
			0x68,             // PLA
			0x40,             // RTI
		}

		// Game loop with double buffering
		program := []uint8{
			// Setup
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001
			
			// Initialize both buffers
			0xA9, 0x00,       // LDA #$00
			0x85, 0x60,       // STA $60 (buffer selector)
			
			// Set up buffer 0 (page $02)
			0xA2, 0x00,       // LDX #$00
			0xA9, 0x88,       // LDA #$88
			0x9D, 0x00, 0x02, // STA $0200,X
			0xA9, 0x10,       // LDA #$10
			0x9D, 0x01, 0x02, // STA $0201,X
			0xA9, 0x00,       // LDA #$00
			0x9D, 0x02, 0x02, // STA $0202,X
			0x8A,             // TXA
			0x9D, 0x03, 0x02, // STA $0203,X
			0x18,             // CLC
			0x69, 0x04,       // ADC #$04
			0xAA,             // TAX
			0xE0, 0x10,       // CPX #$10
			0x90, 0xEB,       // BCC
			
			// Set up buffer 1 (page $03)
			0xA2, 0x00,       // LDX #$00
			0xA9, 0x90,       // LDA #$90
			0x9D, 0x00, 0x03, // STA $0300,X
			0xA9, 0x20,       // LDA #$20
			0x9D, 0x01, 0x03, // STA $0301,X
			0xA9, 0x01,       // LDA #$01
			0x9D, 0x02, 0x03, // STA $0302,X
			0x8A,             // TXA
			0x69, 0x08,       // ADC #$08
			0x9D, 0x03, 0x03, // STA $0303,X
			0x18,             // CLC
			0x69, 0x04,       // ADC #$04
			0xAA,             // TAX
			0xE0, 0x10,       // CPX #$10
			0x90, 0xE8,       // BCC
			
			// Main loop
			0xE6, 0x62,       // INC $62 (main loop counter)
			0x4C, 0x3A, 0x80, // JMP to main loop
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize
		helper.Memory.Write(0x0060, 0x00) // Buffer selector
		helper.Memory.Write(0x0061, 0x00) // Frame counter
		helper.Memory.Write(0x0062, 0x00) // Main loop counter

		// Execute and verify double buffering
		frames := 0
		lastFrameCount := uint8(0)
		bufferStates := []uint8{}
		
		for i := 0; i < 100000 && frames < 6; i++ {
			helper.Bus.Step()
			
			currentFrameCount := helper.Memory.Read(0x0061)
			if currentFrameCount > lastFrameCount {
				frames++
				bufferSelector := helper.Memory.Read(0x0060)
				bufferStates = append(bufferStates, bufferSelector)
				
				t.Logf("Frame %d: Buffer selector = %d", frames, bufferSelector)
				lastFrameCount = currentFrameCount
			}
		}

		// Verify buffer alternation
		if len(bufferStates) >= 4 {
			alternating := true
			for i := 1; i < len(bufferStates); i++ {
				if bufferStates[i] == bufferStates[i-1] {
					alternating = false
					break
				}
			}
			
			if alternating {
				t.Log("Double buffering working correctly: buffers alternate")
			} else {
				t.Error("Double buffering failed: buffers not alternating")
			}
		}

		t.Logf("Double buffered system test completed: %d frames", frames)
	})

	t.Run("Sprite_Animation_System", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler for sprite animation
		nmiHandler := []uint8{
			0x48,             // PHA
			0x8A,             // TXA
			0x48,             // PHA
			
			// Animate sprites (cycle through tile patterns)
			0xA2, 0x01,       // LDX #$01 (tile offset in sprite)
			0xBD, 0x00, 0x02, // LDA $0200,X (current tile)
			0x18,             // CLC
			0x69, 0x01,       // ADC #$01 (next frame)
			0x29, 0x0F,       // AND #$0F (wrap at 16)
			0x09, 0x10,       // ORA #$10 (base tile)
			0x9D, 0x00, 0x02, // STA $0200,X (update tile)
			0x8A,             // TXA
			0x18,             // CLC
			0x69, 0x04,       // ADC #$04 (next sprite)
			0xAA,             // TAX
			0xE0, 0x11,       // CPX #$11 (check all 4 sprites)
			0x90, 0xEA,       // BCC (continue animation)
			
			// Perform DMA
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014
			
			// Mark frame
			0xE6, 0x70,       // INC $70
			
			0x68,             // PLA
			0xAA,             // TAX
			0x68,             // PLA
			0x40,             // RTI
		}

		// Game setup
		program := []uint8{
			// Graphics setup
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001
			
			// Initialize animated sprites
			0xA2, 0x00,       // LDX #$00
			0xA9, 0x80,       // LDA #$80 (Y position)
			0x9D, 0x00, 0x02, // STA $0200,X
			0xA9, 0x10,       // LDA #$10 (base tile)
			0x9D, 0x01, 0x02, // STA $0201,X
			0xA9, 0x00,       // LDA #$00 (attributes)
			0x9D, 0x02, 0x02, // STA $0202,X
			0x8A,             // TXA
			0x0A,             // ASL
			0x0A,             // ASL
			0x0A,             // ASL (multiply by 8)
			0x69, 0x40,       // ADC #$40 (base X + spacing)
			0x9D, 0x03, 0x02, // STA $0203,X
			0x8A,             // TXA
			0x18,             // CLC
			0x69, 0x04,       // ADC #$04
			0xAA,             // TAX
			0xE0, 0x10,       // CPX #$10 (4 sprites)
			0x90, 0xE1,       // BCC
			
			// Game loop
			0xE6, 0x71,       // INC $71 (game counter)
			0x4C, 0x25, 0x80, // JMP to loop
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize
		helper.Memory.Write(0x0070, 0x00) // Frame counter
		helper.Memory.Write(0x0071, 0x00) // Game counter

		// Execute and verify animation
		frames := 0
		lastFrameCount := uint8(0)
		spriteAnimationStates := [][]uint8{}
		
		for i := 0; i < 120000 && frames < 8; i++ {
			helper.Bus.Step()
			
			currentFrameCount := helper.Memory.Read(0x0070)
			if currentFrameCount > lastFrameCount {
				frames++
				
				// Capture sprite animation state
				animState := make([]uint8, 4)
				for j := 0; j < 4; j++ {
					animState[j] = helper.Memory.Read(uint16(0x0201 + j*4)) // Tile indices
				}
				spriteAnimationStates = append(spriteAnimationStates, animState)
				
				t.Logf("Frame %d animation state: [%02X %02X %02X %02X]",
					frames, animState[0], animState[1], animState[2], animState[3])
				
				lastFrameCount = currentFrameCount
			}
		}

		// Verify animation progression
		if len(spriteAnimationStates) >= 3 {
			animating := false
			firstState := spriteAnimationStates[0]
			for i := 1; i < len(spriteAnimationStates); i++ {
				currentState := spriteAnimationStates[i]
				for j := 0; j < 4; j++ {
					if currentState[j] != firstState[j] {
						animating = true
						break
					}
				}
				if animating {
					break
				}
			}
			
			if animating {
				t.Log("Sprite animation working: tiles changing between frames")
			} else {
				t.Error("Sprite animation failed: tiles not changing")
			}
		}

		t.Logf("Sprite animation test completed: %d frames animated", frames)
	})

	t.Run("Performance_Critical_DMA_Timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Optimized NMI handler for performance testing
		nmiHandler := []uint8{
			// Minimal NMI handler
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (DMA only)
			0xE6, 0x80,       // INC $80 (frame counter)
			0x40,             // RTI
		}

		// Performance test program
		program := []uint8{
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001
			
			// Critical timing loop
			0xE6, 0x81,       // INC $81 (tight loop counter)
			0xE6, 0x81,       // INC $81
			0xE6, 0x81,       // INC $81
			0xE6, 0x81,       // INC $81
			0x4C, 0x08, 0x80, // JMP (continue tight loop)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize
		helper.Memory.Write(0x0080, 0x00) // Frame counter
		helper.Memory.Write(0x0081, 0x00) // Loop counter

		// Measure performance over multiple frames
		startCycles := helper.Bus.GetCycleCount()
		
		// Run for specific number of frames
		targetFrames := uint8(5)
		for i := 0; i < 200000; i++ {
			helper.Bus.Step()
			
			currentFrames := helper.Memory.Read(0x0080)
			if currentFrames >= targetFrames {
				break
			}
		}

		endCycles := helper.Bus.GetCycleCount()
		finalFrames := helper.Memory.Read(0x0080)
		finalLoopCount := helper.Memory.Read(0x0081)
		
		totalCycles := endCycles - startCycles
		avgCyclesPerFrame := totalCycles / uint64(finalFrames)
		
		t.Logf("Performance critical DMA timing results:")
		t.Logf("  Frames processed: %d", finalFrames)
		t.Logf("  Total cycles: %d", totalCycles)
		t.Logf("  Average cycles per frame: %d", avgCyclesPerFrame)
		t.Logf("  Loop executions: %d", finalLoopCount)
		
		// Expected NTSC timing is ~29781 cycles per frame
		expectedCycles := uint64(29781)
		tolerance := uint64(1000) // Allow some tolerance
		
		if avgCyclesPerFrame < expectedCycles-tolerance || avgCyclesPerFrame > expectedCycles+tolerance {
			t.Errorf("Frame timing outside expected range: %d cycles (expected ~%d)",
				avgCyclesPerFrame, expectedCycles)
		} else {
			t.Log("Frame timing within expected NTSC range")
		}
	})
}