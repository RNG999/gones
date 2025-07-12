// Package ppu implements the Picture Processing Unit for the NES.
package ppu

import (
	"fmt"
	"gones/internal/memory"
)

// PPU represents the NES Picture Processing Unit (2C02)
type PPU struct {
	// PPU Registers (CPU-visible)
	ppuCtrl   uint8 // $2000 - PPUCTRL
	ppuMask   uint8 // $2001 - PPUMASK
	ppuStatus uint8 // $2002 - PPUSTATUS
	oamAddr   uint8 // $2003 - OAMADDR
	oamData   uint8 // $2004 - OAMDATA (read/write buffer)
	ppuScroll uint8 // $2005 - PPUSCROLL (write buffer)
	ppuAddr   uint8 // $2006 - PPUADDR (write buffer)
	ppuData   uint8 // $2007 - PPUDATA (read/write buffer)

	// Internal PPU State
	v uint16 // Current VRAM address (15 bits)
	t uint16 // Temporary VRAM address (15 bits) - address latch
	x uint8  // Fine X scroll (3 bits)
	w bool   // Write latch (toggles between first/second write)

	// PPU Memory
	memory *memory.PPUMemory

	// Rendering State
	scanline    int // Current scanline (-1 to 260)
	cycle       int // Current cycle (0 to 340)
	frameCount  uint64
	oddFrame    bool
	suppressVBL bool  // Suppress VBL flag setting
	readBuffer  uint8 // PPU read buffer for $2007

	// Sprite Data
	oam              [256]uint8 // Object Attribute Memory
	secondaryOAM     [32]uint8  // Secondary OAM for current scanline
	spriteCount      uint8      // Number of sprites on current scanline
	sprite0Hit       bool       // Sprite 0 hit flag
	spriteOverflow   bool       // Sprite overflow flag
	lastEvalScanline int        // Last scanline for which sprites were evaluated
	
	// Enhanced sprite 0 tracking (inspired by pretendo)
	spriteIndexes    [8]uint8   // Original sprite indices for secondary OAM entries
	sprite0OnScanline bool      // True if sprite 0 is present on current scanline

	// Frame Buffer
	frameBuffer [256 * 240]uint32 // RGB frame buffer

	// Callbacks
	nmiCallback           func()
	frameCompleteCallback func()

	// Rendering Control
	backgroundEnabled bool
	spritesEnabled    bool
	renderingEnabled  bool

	// Timing
	cycleCount uint64

	// Background pixel cache for sprite 0 hit optimization
	currentBackgroundPixel SpritePixel
	backgroundPixelCached  bool
}

// New creates a new PPU instance
func New() *PPU {
	return &PPU{
		scanline:   -1, // Start at pre-render scanline
		cycle:      0,
		frameCount: 0,
		oddFrame:   false,

		// Initialize frame buffer to black
		frameBuffer: [256 * 240]uint32{},
	}
}

// Reset resets the PPU to initial state
func (p *PPU) Reset() {
	p.ppuCtrl = 0
	p.ppuMask = 0
	p.ppuStatus = 0xA0 // VBL flag set, sprite overflow and sprite 0 hit clear
	p.oamAddr = 0
	p.oamData = 0
	p.ppuScroll = 0
	p.ppuAddr = 0
	p.ppuData = 0

	p.v = 0
	p.t = 0
	p.x = 0
	p.w = false

	p.scanline = -1
	p.cycle = 0
	p.frameCount = 0
	p.oddFrame = false
	p.suppressVBL = false
	p.readBuffer = 0

	p.spriteCount = 0
	p.sprite0Hit = false
	p.spriteOverflow = false

	p.backgroundEnabled = false
	p.spritesEnabled = false
	p.renderingEnabled = false

	p.cycleCount = 0
	p.lastEvalScanline = -999

	// Clear OAM
	for i := range p.oam {
		p.oam[i] = 0
	}

	// Clear frame buffer to black
	for i := range p.frameBuffer {
		p.frameBuffer[i] = 0x000000 // Black in RGB format
	}
}

// SetMemory sets the PPU memory interface
func (p *PPU) SetMemory(memory *memory.PPUMemory) {
	p.memory = memory
}

// SetNMICallback sets the NMI callback function
func (p *PPU) SetNMICallback(callback func()) {
	p.nmiCallback = callback
}

// SetFrameCompleteCallback sets the frame complete callback
func (p *PPU) SetFrameCompleteCallback(callback func()) {
	p.frameCompleteCallback = callback
}

// ReadRegister reads from a PPU register (CPU $2000-$2007)
func (p *PPU) ReadRegister(address uint16) uint8 {
	switch address {
	case 0x2000: // PPUCTRL - write only
		return p.ppuStatus & 0x1F // Return open bus with lower 5 bits
	case 0x2001: // PPUMASK - write only
		return p.ppuStatus & 0x1F // Return open bus with lower 5 bits
	case 0x2002: // PPUSTATUS
		status := p.ppuStatus
		// Debug: Log when PPUSTATUS is read and sprite 0 hit flag is cleared
		if status&0x40 != 0 {
			fmt.Printf("[PPUSTATUS_READ] Frame %d: Reading PPUSTATUS=0x%02X, clearing sprite 0 hit flag\n", 
				p.frameCount, status)
		}
		p.ppuStatus &= 0x3F // Clear VBL flag (bit 7) and sprite 0 hit flag (bit 6)
		p.sprite0Hit = false // Clear internal sprite 0 hit flag
		p.w = false         // Clear write latch
		return status
	case 0x2003: // OAMADDR - write only
		return p.ppuStatus & 0x1F // Return open bus with lower 5 bits
	case 0x2004: // OAMDATA
		return p.oam[p.oamAddr]
	case 0x2005: // PPUSCROLL - write only
		return p.ppuStatus & 0x1F // Return open bus with lower 5 bits
	case 0x2006: // PPUADDR - write only
		return p.ppuStatus & 0x1F // Return open bus with lower 5 bits
	case 0x2007: // PPUDATA
		return p.readPPUData()
	default:
		return 0
	}
}

// WriteRegister writes to a PPU register (CPU $2000-$2007)
func (p *PPU) WriteRegister(address uint16, value uint8) {
	switch address {
	case 0x2000: // PPUCTRL
		p.ppuCtrl = value
		p.t = (p.t & 0xF3FF) | ((uint16(value) & 0x03) << 10) // Nametable select
		p.updateRenderingFlags()
		p.checkNMI()
	case 0x2001: // PPUMASK
		p.ppuMask = value
		p.updateRenderingFlags()
	case 0x2002: // PPUSTATUS - read only
		// Writes are ignored
	case 0x2003: // OAMADDR
		p.oamAddr = value
	case 0x2004: // OAMDATA
		p.oam[p.oamAddr] = value
		p.oamAddr++ // Auto-increment
	case 0x2005: // PPUSCROLL
		p.writePPUScroll(value)
	case 0x2006: // PPUADDR
		p.writePPUAddr(value)
	case 0x2007: // PPUDATA
		p.writePPUData(value)
	}
}

// WriteOAM writes to OAM at the specified address (for DMA)
func (p *PPU) WriteOAM(address uint8, value uint8) {
	p.oam[address] = value
}

// Step advances the PPU by one cycle
func (p *PPU) Step() {
	p.cycleCount++

	// Advance cycle counter first
	p.cycle++
	if p.cycle > 340 {
		p.cycle = 0
		p.scanline++

		if p.scanline > 260 {
			p.scanline = -1
			p.frameCount++
			p.oddFrame = !p.oddFrame

			if p.frameCompleteCallback != nil {
				p.frameCompleteCallback()
			}
		}
	}

	// Handle VBlank start at scanline 241, cycle 1
	if p.scanline == 241 && p.cycle == 1 {
		// Set VBL flag
		p.ppuStatus |= 0x80
		// Clear sprite 0 hit and sprite overflow flags at VBlank START (critical timing fix)
		wasSprite0Hit := p.sprite0Hit
		p.ppuStatus &= 0x9F // Clear bits 6 (sprite 0 hit) and 5 (sprite overflow), keep VBL flag
		p.sprite0Hit = false    // Clear internal sprite 0 hit flag
		p.spriteOverflow = false // Clear internal sprite overflow flag
		
		// Log sprite 0 hit flag clearing for debugging
		if wasSprite0Hit {
			fmt.Printf("[SPRITE0_CLEAR] Frame %d: Sprite 0 hit flag cleared at VBlank start (scanline 241)\n", p.frameCount)
		}
		
		// Trigger NMI if enabled
		if p.ppuCtrl&0x80 != 0 && p.nmiCallback != nil {
			p.nmiCallback()
		}
	}

	// Handle VBlank end at scanline -1 (pre-render), cycle 1
	if p.scanline == -1 && p.cycle == 1 {
		// Clear VBL flag only (sprite flags already cleared at VBlank start)
		p.ppuStatus &= 0x7F // Clear bit 7 (VBL flag) only
	}
	
	// At start of visible frame, copy scroll position from t to v if rendering enabled
	if p.scanline == 0 && p.cycle == 0 && p.renderingEnabled {
		// This ensures the scroll position set during vblank takes effect
		p.v = p.t
	}

	// Handle rendering cycles
	if p.scanline >= -1 && p.scanline < 240 {
		p.renderCycle()
	}
}

// renderCycle handles rendering for a single PPU cycle
func (p *PPU) renderCycle() {
	// Handle pre-render scanline (-1) and visible scanlines (0-239)
	if p.scanline < -1 || p.scanline >= 240 {
		return
	}

	// Removed cycle-accurate scroll register updates as they were causing rendering corruption
	// The emulator will use simpler scroll implementation based on PPUSCROLL register writes

	// Sprite evaluation - do this once per scanline, only during visible scanlines
	if p.spritesEnabled && p.scanline >= 0 && p.scanline < 240 && p.cycle == 1 {
		// Only evaluate sprites at cycle 1 of each scanline to avoid redundant calls
		if p.lastEvalScanline != p.scanline {
			p.evaluateSprites()
		}
	}

	// Only render pixels during visible scanlines and cycles
	// TIMING FIX: Sprite 0 hit detection should start at cycle 2 according to NES spec
	if p.scanline < 0 || p.scanline >= 240 || p.cycle < 2 || p.cycle > 257 {
		return
	}

	// Skip if no memory interface
	if p.memory == nil {
		return
	}

	// Skip rendering entirely if both background and sprites are disabled
	if !p.backgroundEnabled && !p.spritesEnabled {
		return
	}

	// Calculate pixel position
	// TIMING FIX: Adjust for cycle 2 start (cycle 2 = pixel 0)
	pixelX := p.cycle - 2 // Convert to 0-based with correct timing
	pixelY := p.scanline

	// Initialize as transparent pixels
	var backgroundPixel SpritePixel = SpritePixel{transparent: true}
	var spritePixel SpritePixel = SpritePixel{transparent: true}

	// Render background pixel only if enabled via PPUMASK
	if p.backgroundEnabled {
		backgroundPixel = p.renderBackgroundPixel(pixelX, pixelY)
		// Cache background pixel for sprite 0 hit detection optimization
		p.currentBackgroundPixel = backgroundPixel
		p.backgroundPixelCached = true
	} else {
		// Clear background pixel cache when background rendering is disabled
		p.backgroundPixelCached = false
	}

	// Render sprite pixel if enabled
	if p.spritesEnabled {
		spritePixel = p.renderSpritePixel(pixelX, pixelY)
	} else {
		// Initialize as transparent sprite pixel when sprites disabled
		spritePixel = SpritePixel{
			transparent: true,
		}
	}

	// Combine background and sprite pixels
	finalColor := p.compositeFinalPixel(backgroundPixel, spritePixel)

	// Write to frame buffer
	frameBufferIndex := pixelY*256 + pixelX
	p.frameBuffer[frameBufferIndex] = finalColor
}

// SpritePixel represents a rendered pixel from background or sprite
type SpritePixel struct {
	colorIndex   uint8  // 0-3, where 0 is transparent
	paletteIndex uint8  // which palette (0-3 for sprites, 0-3 for background)
	rgbColor     uint32 // final RGB color
	spriteIndex  int8   // which sprite (0-63, or -1 for background)
	priority     bool   // sprite priority flag (false = in front, true = behind background)
	transparent  bool   // true if this pixel is transparent
}

// evaluateSprites finds sprites visible on the current scanline (standard NES behavior)
func (p *PPU) evaluateSprites() {
	// Update last evaluation scanline
	p.lastEvalScanline = p.scanline

	p.spriteCount = 0
	p.spriteOverflow = false
	p.sprite0OnScanline = false

	// Clear secondary OAM and sprite indexes
	for i := range p.secondaryOAM {
		p.secondaryOAM[i] = 0xFF
	}
	for i := range p.spriteIndexes {
		p.spriteIndexes[i] = 0xFF
	}

	// Determine sprite height (8x8 or 8x16)
	spriteHeight := 8
	if p.ppuCtrl&0x20 != 0 { // PPUCTRL bit 5
		spriteHeight = 16
	}

	// Standard NES sprite evaluation: check sprites 0-63 in order
	spritesFound := 0
	for spriteIndex := 0; spriteIndex < 64; spriteIndex++ {
		oamIndex := spriteIndex * 4
		sY := int(p.oam[oamIndex])      // Y position
		tileIndex := p.oam[oamIndex+1]  // Tile index
		attributes := p.oam[oamIndex+2] // Attributes
		sX := int(p.oam[oamIndex+3])    // X position

		// Check if sprite is visible on current scanline
		if p.scanline >= sY+1 && p.scanline < sY+1+spriteHeight {
			if spritesFound < 8 {
				// Copy sprite to secondary OAM
				secondaryIndex := spritesFound * 4
				p.secondaryOAM[secondaryIndex] = uint8(sY)
				p.secondaryOAM[secondaryIndex+1] = tileIndex
				p.secondaryOAM[secondaryIndex+2] = attributes
				p.secondaryOAM[secondaryIndex+3] = uint8(sX)

				// Track original sprite index for sprite 0 detection
				p.spriteIndexes[spritesFound] = uint8(spriteIndex)
				
				// Mark if this is sprite 0
				if spriteIndex == 0 {
					p.sprite0OnScanline = true
					// Debug logging for Sprite 0 detection
					if p.frameCount%300 == 0 { // Log every 5 seconds
						fmt.Printf("[SPRITE0_DEBUG] Frame %d: Sprite 0 found at secondary index %d - Y:%d X:%d Tile:$%02X\n", 
							p.frameCount, spritesFound, sY, sX, tileIndex)
					}
				}

				spritesFound++
			} else {
				// More than 8 sprites on scanline - set overflow flag
				p.spriteOverflow = true
				p.ppuStatus |= 0x20 // Set sprite overflow flag in PPUSTATUS
				
				// CRITICAL DEBUG: Log if Sprite 0 would be dropped
				if spriteIndex == 0 {
					fmt.Printf("[SPRITE0_DROPPED] Frame %d: Sprite 0 dropped due to 8-sprite limit on scanline %d!\n", 
						p.frameCount, p.scanline)
				}
				
				// Debug logging for sprite overflow
				if p.frameCount%300 == 0 { // Log every 5 seconds
					fmt.Printf("[PPU_SPRITE] Sprite overflow detected on scanline %d (frame %d)\n", 
						p.scanline, p.frameCount)
				}
				break
			}
		}
	}

	p.spriteCount = uint8(spritesFound)
	
	// Comprehensive OAM debugging for freeze investigation
	if p.frameCount%300 == 0 { // Every 5 seconds
		p.debugOAMState()
	}
}

// debugOAMState logs detailed OAM information for debugging
func (p *PPU) debugOAMState() {
	// Only debug when Sprite 0 is on scanline to reduce spam
	if !p.sprite0OnScanline {
		return
	}
	
	fmt.Printf("\n=== OAM DEBUG Frame %d ===\n", p.frameCount)
	fmt.Printf("Sprite 0: Y=%d X=%d Tile=$%02X Attr=$%02X\n", 
		p.oam[0], p.oam[3], p.oam[1], p.oam[2])
	
	// Debug pattern table data for Sprite 0 tile
	p.debugTilePattern(p.oam[1])
	
	// Show all sprites on current scanline
	fmt.Printf("Scanline %d sprites:\n", p.scanline)
	for i := 0; i < int(p.spriteCount); i++ {
		idx := i * 4
		origIndex := p.spriteIndexes[i]
		if origIndex < 64 {
			fmt.Printf("  [%d] Orig:%d Y=%d X=%d Tile=$%02X\n", 
				i, origIndex, p.secondaryOAM[idx], p.secondaryOAM[idx+3], p.secondaryOAM[idx+1])
		}
	}
	
	fmt.Printf("Sprite 0 on scanline: %t\n", p.sprite0OnScanline)
	fmt.Printf("Sprite overflow: %t\n", p.spriteOverflow)
	fmt.Printf("========================\n\n")
}

// debugTilePattern logs pattern table data for a specific tile
func (p *PPU) debugTilePattern(tileIndex uint8) {
	if p.memory == nil {
		return
	}
	
	// Determine pattern table address (sprites use table 1 if PPUCTRL bit 3 is set)
	patternTableBase := uint16(0x0000)
	if p.ppuCtrl&0x08 != 0 {
		patternTableBase = 0x1000
	}
	
	tileAddr := patternTableBase + uint16(tileIndex)*16
	
	fmt.Printf("Pattern Table Debug - Tile $%02X at $%04X:\n", tileIndex, tileAddr)
	fmt.Printf("Low byte:  ")
	for i := 0; i < 8; i++ {
		fmt.Printf("%02X ", p.memory.Read(tileAddr+uint16(i)))
	}
	fmt.Printf("\nHigh byte: ")
	for i := 0; i < 8; i++ {
		fmt.Printf("%02X ", p.memory.Read(tileAddr+8+uint16(i)))
	}
	fmt.Printf("\nPattern visualization:\n")
	
	// Show tile pattern as ASCII art
	for row := 0; row < 8; row++ {
		lowByte := p.memory.Read(tileAddr + uint16(row))
		highByte := p.memory.Read(tileAddr + 8 + uint16(row))
		
		fmt.Printf("Row %d: ", row)
		for bit := 7; bit >= 0; bit-- {
			lowBit := (lowByte >> bit) & 1
			highBit := (highByte >> bit) & 1
			colorIndex := (highBit << 1) | lowBit
			
			switch colorIndex {
			case 0:
				fmt.Printf(".")  // Transparent
			case 1:
				fmt.Printf("1")  // Color 1
			case 2:
				fmt.Printf("2")  // Color 2
			case 3:
				fmt.Printf("3")  // Color 3
			}
		}
		fmt.Printf(" (L:%02X H:%02X)\n", lowByte, highByte)
	}
}

// renderBackgroundPixel renders a single background pixel
func (p *PPU) renderBackgroundPixel(pixelX, pixelY int) SpritePixel {
	// Direct computation (simpler and faster than caching)
	var scrollX, scrollY int
	var effectiveNametable int
	
	if p.t != 0 || p.x != 0 {
		// Extract scroll values directly from registers using bit operations
		scrollX = int(p.t&0x001F)<<3 + int(p.x)  // coarse X * 8 + fine X
		scrollY = int((p.t>>5)&0x001F)<<3 + int((p.t>>12)&0x0007)  // coarse Y * 8 + fine Y
		effectiveNametable = int((p.t >> 10) & 0x0003)  // nametable select
	} else {
		// No scroll applied
		scrollX = 0
		scrollY = 0
		effectiveNametable = 0
	}
	
	// Apply scroll to get world coordinates
	worldX := pixelX + scrollX
	worldY := pixelY + scrollY
	
	// Conservative bounds checking to prevent extreme values while allowing normal scroll
	// Allow reasonable negative and positive scroll values that games might use
	if worldX < -256 || worldX >= 768 {
		// Clamp to safe range for extreme values
		if worldX < -256 {
			worldX = -256
		} else {
			worldX = 767
		}
	}
	
	if worldY < -240 || worldY >= 720 {
		// Clamp to safe range for extreme values
		if worldY < -240 {
			worldY = -240
		} else {
			worldY = 719
		}
	}
	
	// Restore original nametable wrapping logic (proven to work)
	// Handle both positive and negative scroll values
	finalNametable := effectiveNametable
	
	// Handle negative X coordinates
	if worldX < 0 {
		finalNametable ^= 1 // Toggle horizontal nametable for negative X
		worldX += 256
	}
	// Handle positive X coordinates  
	if worldX >= 256 {
		finalNametable ^= 1 // Toggle horizontal nametable
		worldX -= 256
	}
	
	// Handle negative Y coordinates
	if worldY < 0 {
		finalNametable ^= 2 // Toggle vertical nametable for negative Y
		worldY += 240
	}
	// Handle positive Y coordinates
	if worldY >= 240 {
		finalNametable ^= 2 // Toggle vertical nametable
		worldY -= 240
	}
	
	// Calculate tile coordinates using bit shifts (faster than division)
	tileX := worldX >> 3  // worldX / 8
	tileY := worldY >> 3  // worldY / 8
	pixelInTileX := worldX & 7  // worldX % 8
	pixelInTileY := worldY & 7  // worldY % 8
	
	// Additional bounds validation for tile coordinates
	if tileX < 0 || tileX >= 32 || tileY < 0 || tileY >= 30 {
		// Return transparent pixel for out-of-bounds tiles
		return SpritePixel{transparent: true}
	}

	// Fetch nametable byte - determines which tile to use
	nametableAddr := 0x2000 | (uint16(finalNametable&3) << 10) | uint16(tileY*32+tileX)
	tileID := p.memory.Read(nametableAddr)

	// Fetch attribute table byte - determines palette selection
	attributeAddr := 0x23C0 | (uint16(finalNametable&3) << 10) | uint16((tileY>>2)*8+(tileX>>2))
	attributeByte := p.memory.Read(attributeAddr)

	// Extract 2-bit palette index from attribute byte 
	// Each attribute byte controls a 4x4 tile area (32x32 pixels)
	// Divided into 4 quadrants of 2x2 tiles each
	// blockID: 0=top-left, 1=top-right, 2=bottom-left, 3=bottom-right
	// Use bit operations for better performance
	blockID := ((tileX & 3) >> 1) + ((tileY & 3) >> 1) * 2
	paletteIndex := (attributeByte >> (blockID << 1)) & 0x03

	// Determine pattern table base address from PPUCTRL bit 4
	var patternTableBase uint16
	if p.ppuCtrl&0x10 != 0 {
		patternTableBase = 0x1000 // Pattern table 1
	} else {
		patternTableBase = 0x0000 // Pattern table 0
	}

	// Fetch pattern table data
	patternAddr := patternTableBase + uint16(tileID)*16 + uint16(pixelInTileY)

	// Read pattern data using standard NES format
	patternLow := p.memory.Read(patternAddr)
	patternHigh := p.memory.Read(patternAddr + 0x08)

	// Extract the specific pixel bits
	bitShift := 7 - pixelInTileX
	bit0 := (patternLow >> bitShift) & 1
	bit1 := (patternHigh >> bitShift) & 1
	colorIndex := (bit1 << 1) | bit0

	// Calculate palette address
	var paletteAddr uint16
	if colorIndex == 0 {
		paletteAddr = 0x3F00 // Universal background color
	} else {
		paletteAddr = 0x3F00 + uint16(paletteIndex)*4 + uint16(colorIndex)
	}

	// Read color and convert to RGB
	nesColorIndex := p.memory.Read(paletteAddr)
	rgbColor := p.NESColorToRGB(nesColorIndex)

	// Color debugging can be enabled here if needed

	return SpritePixel{
		colorIndex:   colorIndex,
		paletteIndex: paletteIndex,
		rgbColor:     rgbColor,
		spriteIndex:  -1, // Background
		priority:     false,
		transparent:  colorIndex == 0,
	}
}

// renderSpritePixel renders a single sprite pixel
func (p *PPU) renderSpritePixel(pixelX, pixelY int) SpritePixel {

	// Check each sprite in secondary OAM (in forward order for correct priority)
	// Lower OAM index = higher priority, so first non-transparent sprite wins
	for i := 0; i < int(p.spriteCount); i++ {
		secondaryIndex := i * 4

		sY := int(p.secondaryOAM[secondaryIndex])
		tileIndex := p.secondaryOAM[secondaryIndex+1]
		attributes := p.secondaryOAM[secondaryIndex+2]
		sX := int(p.secondaryOAM[secondaryIndex+3])

		// Determine sprite height and handle 8x16 mode
		spriteHeight := 8
		if p.ppuCtrl&0x20 != 0 { // 8x16 sprites
			spriteHeight = 16
		}

		// Check if current pixel is within this sprite (X and Y bounds)
		if pixelX >= sX && pixelX < sX+8 &&
			pixelY >= sY+1 && pixelY < sY+1+spriteHeight {
			spritePixelX := pixelX - sX
			spritePixelY := pixelY - (sY + 1) // Y+1 because sprites are delayed by 1 scanline

			// Critical: Validate sprite pixel coordinates before processing
			if spritePixelX < 0 || spritePixelX >= 8 || 
			   spritePixelY < 0 || spritePixelY >= spriteHeight {
				continue // Skip this sprite if coordinates are invalid
			}

			// Handle sprite flipping
			if attributes&0x40 != 0 { // Horizontal flip
				spritePixelX = 7 - spritePixelX
			}
			if attributes&0x80 != 0 { // Vertical flip
				spritePixelY = spriteHeight - 1 - spritePixelY
			}
			
			// Validate coordinates after flipping to prevent collision freeze
			if spritePixelX < 0 || spritePixelX >= 8 || 
			   spritePixelY < 0 || spritePixelY >= spriteHeight {
				continue // Skip if flipping created invalid coordinates
			}

			// Get sprite pixel data
			colorIndex := p.getSpritePixelColor(tileIndex, spritePixelX, spritePixelY, attributes)

			// Reduced debug: Only log when sprite 0 has non-transparent pixels
			if p.isOriginalSprite0(i) && colorIndex != 0 && pixelX >= 89 && pixelX <= 95 && pixelY >= 28 && pixelY <= 32 {
				fmt.Printf("[SPRITE0_PIXEL] Frame %d: Sprite 0 at (%d,%d) -> sprite pixel (%d,%d), colorIndex=%d\n", 
					p.frameCount, pixelX, pixelY, spritePixelX, spritePixelY, colorIndex)
			}

			if colorIndex != 0 { // Non-transparent pixel
				
				// Check for sprite 0 hit FIRST, before any other processing
				// This ensures sprite 0 hit is never overridden by subsequent sprites
				if p.isOriginalSprite0(i) && !p.sprite0Hit {
					// Reduced debug: Only log when actually attempting sprite 0 hit check
					if pixelX >= 90 && pixelX <= 95 && pixelY >= 28 && pixelY <= 32 {
						fmt.Printf("[SPRITE0_CHECK] Frame %d: Checking hit at (%d,%d), colorIdx %d\n", 
							p.frameCount, pixelX, pixelY, colorIndex)
					}
					p.checkSprite0Hit(pixelX, pixelY, colorIndex)
				}

				// Extract palette index from attributes (bits 1-0)
				paletteIndex := attributes & 0x03

				// Calculate sprite palette address
				paletteAddr := 0x3F10 + uint16(paletteIndex)*4 + uint16(colorIndex)
				nesColorIndex := p.memory.Read(paletteAddr)
				rgbColor := p.NESColorToRGB(nesColorIndex)

				spritePixel := SpritePixel{
					colorIndex:   colorIndex,
					paletteIndex: paletteIndex,
					rgbColor:     rgbColor,
					spriteIndex:  int8(i),
					priority:     (attributes & 0x20) != 0, // Background priority flag
					transparent:  false,
				}

				return spritePixel
			}
		}
	}

	// No sprite pixel found - return transparent
	return SpritePixel{
		colorIndex:  0,
		rgbColor:    0,
		spriteIndex: -1,
		transparent: true,
	}
}

// getSpritePixelColor gets the color index for a sprite pixel
func (p *PPU) getSpritePixelColor(tileIndex uint8, pixelX, pixelY int, attributes uint8) uint8 {
	// Critical bounds checking to prevent freeze during sprite collisions
	if pixelX < 0 || pixelX >= 8 || pixelY < 0 || pixelY >= 16 {
		return 0 // Return transparent for invalid coordinates
	}
	
	var patternTableBase uint16

	// For 8x8 sprites, use PPUCTRL bit 3 to select pattern table
	if p.ppuCtrl&0x20 == 0 { // 8x8 sprites
		if p.ppuCtrl&0x08 != 0 {
			patternTableBase = 0x1000 // Pattern table 1
		} else {
			patternTableBase = 0x0000 // Pattern table 0
		}
	} else { // 8x16 sprites
		// For 8x16 sprites, tile index bit 0 selects pattern table
		if tileIndex&0x01 != 0 {
			patternTableBase = 0x1000
		} else {
			patternTableBase = 0x0000
		}

		// Clear bit 0 for 8x16 tile addressing
		tileIndex &= 0xFE

		// Handle top vs bottom tile in 8x16 mode
		if pixelY >= 8 {
			tileIndex += 1 // Bottom tile
			pixelY -= 8
		}
	}

	// Calculate pattern address with validation
	patternAddr := patternTableBase + uint16(tileIndex)*16 + uint16(pixelY)
	
	// Additional safety: Ensure pattern address is within valid range
	if patternAddr >= 0x2000 || patternAddr+0x08 >= 0x2000 {
		return 0 // Invalid pattern table access
	}

	// Read pattern data
	patternLow := p.memory.Read(patternAddr)
	patternHigh := p.memory.Read(patternAddr + 0x08)

	// Extract pixel color
	bitShift := 7 - pixelX
	bit0 := (patternLow >> bitShift) & 1
	bit1 := (patternHigh >> bitShift) & 1
	colorIndex := (bit1 << 1) | bit0

	return colorIndex
}

// isOriginalSprite0 checks if the sprite at index i in secondary OAM is original sprite 0
func (p *PPU) isOriginalSprite0(secondaryOAMIndex int) bool {
	if secondaryOAMIndex >= int(p.spriteCount) {
		return false
	}

	// OPTIMIZED FIX: Use sprite index tracking (inspired by pretendo)
	// This is much cleaner and more reliable than attribute comparison
	return p.spriteIndexes[secondaryOAMIndex] == 0
}

// checkSprite0Hit checks for sprite 0 hit detection
func (p *PPU) checkSprite0Hit(pixelX, pixelY int, spriteColorIndex uint8) {

	if p.sprite0Hit {
		return // Already set - never clear this flag once set
	}

	// Only check if both background and sprite rendering are enabled
	if !p.backgroundEnabled || !p.spritesEnabled {
		return
	}

	// Add bounds checking to prevent invalid coordinate access
	if pixelX < 0 || pixelX >= 256 || pixelY < 0 || pixelY >= 240 {
		return
	}

	// PRETENDO-INSPIRED FIX: Exclude rightmost pixel (x == 255) from sprite 0 hit detection
	// This matches the behavior of real NES hardware
	if pixelX >= 255 {
		return
	}

	// Skip checking sprite 0 hit in leftmost 8 pixels if clipping is enabled
	// PPUMASK bit 1 = show background in leftmost 8 pixels (0 = clip, 1 = show)
	// PPUMASK bit 2 = show sprites in leftmost 8 pixels (0 = clip, 1 = show)
	if pixelX < 8 && (p.ppuMask&0x02 == 0 || p.ppuMask&0x04 == 0) {
		return
	}

	// Additional safety: only check for valid sprite color index
	if spriteColorIndex == 0 || spriteColorIndex > 3 {
		return
	}

	// CRITICAL FIX: Always render fresh background pixel instead of using cache
	// The cached pixel might be stale or from wrong coordinates, causing missed detection
	backgroundPixel := p.renderBackgroundPixel(pixelX, pixelY)

	// Note: Removed artificial sprite 0 hit forcing - let natural background pixels determine hits

	// Debug: Only log when background is non-transparent (potential hit condition)
	if pixelX >= 90 && pixelX <= 95 && pixelY >= 28 && pixelY <= 32 && !backgroundPixel.transparent {
		fmt.Printf("[SPRITE0_BG] Frame %d: BG at (%d,%d) colorIndex=%d, sprite=%d\n", 
			p.frameCount, pixelX, pixelY, backgroundPixel.colorIndex, spriteColorIndex)
	}

	// Hit occurs when both background and sprite 0 have non-transparent pixels
	if !backgroundPixel.transparent && backgroundPixel.colorIndex != 0 && spriteColorIndex != 0 {
		p.sprite0Hit = true
		p.ppuStatus |= 0x40 // Set sprite 0 hit flag in PPUSTATUS
		
		// Log when sprite 0 hit is detected (state change only)
		fmt.Printf("[SPRITE0_HIT] Frame %d: Sprite 0 hit detected at pixel (%d,%d) - BG color: %d, Sprite color: %d\n", 
			p.frameCount, pixelX, pixelY, backgroundPixel.colorIndex, spriteColorIndex)
		
		// Additional detailed analysis for freeze investigation
		p.debugSprite0Hit(pixelX, pixelY, backgroundPixel, spriteColorIndex)
	}
}

// debugSprite0Hit provides detailed analysis of sprite 0 hit occurrence
func (p *PPU) debugSprite0Hit(pixelX, pixelY int, backgroundPixel SpritePixel, spriteColorIndex uint8) {
	// Only debug every 300 frames to avoid spam (5 seconds at 60 FPS)
	if p.frameCount%300 != 0 {
		return
	}
	
	fmt.Printf("\n=== SPRITE 0 HIT ANALYSIS Frame %d ===\n", p.frameCount)
	fmt.Printf("Hit Location: (%d,%d) Scanline: %d Cycle: %d\n", pixelX, pixelY, p.scanline, p.cycle)
	fmt.Printf("Background: colorIdx=%d transparent=%t rgbColor=0x%06X\n", 
		backgroundPixel.colorIndex, backgroundPixel.transparent, backgroundPixel.rgbColor)
	fmt.Printf("Sprite: colorIdx=%d\n", spriteColorIndex)
	
	// Analyze surrounding background pixels
	fmt.Printf("Surrounding background pixels:\n")
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			testX := pixelX + dx
			testY := pixelY + dy
			if testX >= 0 && testX < 256 && testY >= 0 && testY < 240 {
				testBG := p.renderBackgroundPixel(testX, testY)
				if dx == 0 && dy == 0 {
					fmt.Printf("[%d,%d]=*%d* ", testX, testY, testBG.colorIndex)
				} else {
					fmt.Printf("[%d,%d]=%d ", testX, testY, testBG.colorIndex)
				}
			}
		}
		fmt.Printf("\n")
	}
	
	// Check PPU control registers
	fmt.Printf("PPU State: CTRL=$%02X MASK=$%02X STATUS=$%02X\n", p.ppuCtrl, p.ppuMask, p.ppuStatus)
	fmt.Printf("Background enabled: %t, Sprites enabled: %t\n", p.backgroundEnabled, p.spritesEnabled)
	fmt.Printf("Scroll: v=$%04X t=$%04X x=%d\n", p.v, p.t, p.x)
	
	// Get nametable data at hit location
	p.debugBackgroundTileAtLocation(pixelX, pixelY)
	fmt.Printf("=====================================\n\n")
}

// debugBackgroundTileAtLocation shows background tile info at specific coordinates  
func (p *PPU) debugBackgroundTileAtLocation(pixelX, pixelY int) {
	if p.memory == nil {
		return
	}
	
	// Calculate tile coordinates
	tileX := pixelX / 8
	tileY := pixelY / 8
	
	// Calculate nametable address
	nametableBase := uint16(0x2000)
	nametableAddr := nametableBase + uint16(tileY*32+tileX)
	
	// Get tile index
	tileIndex := p.memory.Read(nametableAddr)
	
	// Get attribute
	attrX := tileX / 4
	attrY := tileY / 4
	attrAddr := nametableBase + 0x3C0 + uint16(attrY*8+attrX)
	attrByte := p.memory.Read(attrAddr)
	
	// Calculate which quadrant of the attribute byte
	quadrantX := (tileX % 4) / 2
	quadrantY := (tileY % 4) / 2
	quadrant := quadrantY*2 + quadrantX
	paletteIndex := (attrByte >> (quadrant * 2)) & 0x03
	
	fmt.Printf("Background Tile at (%d,%d):\n", pixelX, pixelY)
	fmt.Printf("Tile coord: (%d,%d) Index: $%02X Palette: %d\n", tileX, tileY, tileIndex, paletteIndex)
	fmt.Printf("Nametable addr: $%04X Attr addr: $%04X (byte=$%02X)\n", nametableAddr, attrAddr, attrByte)
	
	// Show pattern data for this background tile
	p.debugBackgroundTilePattern(tileIndex, pixelX%8, pixelY%8)
}

// debugBackgroundTilePattern shows pattern data for background tile
func (p *PPU) debugBackgroundTilePattern(tileIndex uint8, pixelInTileX, pixelInTileY int) {
	if p.memory == nil {
		return
	}
	
	// Background tiles use pattern table 0 or 1 based on PPUCTRL bit 4
	patternTableBase := uint16(0x0000)
	if p.ppuCtrl&0x10 != 0 {
		patternTableBase = 0x1000
	}
	
	tileAddr := patternTableBase + uint16(tileIndex)*16
	
	fmt.Printf("BG Pattern Tile $%02X at $%04X:\n", tileIndex, tileAddr)
	
	// Show just the specific pixel we're interested in
	if pixelInTileY >= 0 && pixelInTileY < 8 {
		lowByte := p.memory.Read(tileAddr + uint16(pixelInTileY))
		highByte := p.memory.Read(tileAddr + 8 + uint16(pixelInTileY))
		
		bit := 7 - pixelInTileX
		lowBit := (lowByte >> bit) & 1
		highBit := (highByte >> bit) & 1
		colorIndex := (highBit << 1) | lowBit
		
		fmt.Printf("Pixel (%d,%d) in tile: colorIndex=%d (L:%02X H:%02X bit %d)\n", 
			pixelInTileX, pixelInTileY, colorIndex, lowByte, highByte, bit)
	}
}

// compositeFinalPixel combines background and sprite pixels according to priority
func (p *PPU) compositeFinalPixel(background, sprite SpritePixel) uint32 {
	// If no sprite pixel, use background
	if sprite.transparent {
		if background.transparent {
			// Both transparent - use backdrop color
			backdropColor := p.memory.Read(0x3F00)
			rgbColor := p.NESColorToRGB(backdropColor)

			// Backdrop color debugging can be enabled here if needed

			return rgbColor
		}
		return background.rgbColor
	}

	// If no background pixel or background is transparent, use sprite
	if background.transparent {
		return sprite.rgbColor
	}

	// Both pixels are opaque - check sprite priority
	// But if background rendering is disabled, ignore background priority
	if sprite.priority && p.backgroundEnabled {
		// Sprite has background priority - background wins
		return background.rgbColor
	} else {
		// Sprite has foreground priority - sprite wins, or background rendering is disabled
		return sprite.rgbColor
	}
}

// updateRenderingFlags updates internal rendering state based on PPUMASK
func (p *PPU) updateRenderingFlags() {
	p.backgroundEnabled = (p.ppuMask & 0x08) != 0
	p.spritesEnabled = (p.ppuMask & 0x10) != 0
	p.renderingEnabled = p.backgroundEnabled || p.spritesEnabled
}

// checkNMI checks if an NMI should be triggered
func (p *PPU) checkNMI() {
	if (p.ppuCtrl&0x80 != 0) && (p.ppuStatus&0x80 != 0) && p.nmiCallback != nil {
		p.nmiCallback()
	}
}

// writePPUScroll handles writes to PPUSCROLL ($2005)
func (p *PPU) writePPUScroll(value uint8) {
	if !p.w {
		// First write: X scroll
		p.t = (p.t & 0xFFE0) | (uint16(value) >> 3) // Coarse X
		p.x = value & 0x07                          // Fine X
		p.w = true
	} else {
		// Second write: Y scroll
		p.t = (p.t & 0x8FFF) | ((uint16(value) & 0x07) << 12) // Fine Y
		p.t = (p.t & 0xFC1F) | ((uint16(value) & 0xF8) << 2)  // Coarse Y
		p.w = false
	}
}


// writePPUAddr handles writes to PPUADDR ($2006)
func (p *PPU) writePPUAddr(value uint8) {
	if !p.w {
		// First write: high byte
		p.t = (p.t & 0x80FF) | ((uint16(value) & 0x3F) << 8)
		p.w = true
	} else {
		// Second write: low byte
		p.t = (p.t & 0xFF00) | uint16(value)
		p.v = p.t
		p.w = false
	}
}

// readPPUData handles reads from PPUDATA ($2007)
func (p *PPU) readPPUData() uint8 {
	var data uint8

	if p.memory == nil {
		// No memory - return 0 but still increment address
		data = 0
	} else {
		if p.v >= 0x3F00 {
			// Palette data is not buffered
			data = p.memory.Read(p.v)
			p.readBuffer = p.memory.Read(p.v & 0x2FFF) // Update buffer with underlying nametable
		} else {
			// Other data is buffered
			data = p.readBuffer
			p.readBuffer = p.memory.Read(p.v)
		}
	}

	// Auto-increment address (this must happen regardless of memory availability)
	if p.ppuCtrl&0x04 != 0 {
		p.v += 32 // Increment by 32 (down)
	} else {
		p.v += 1 // Increment by 1 (across)
	}
	p.v &= 0x3FFF // Wrap to 14-bit address space

	return data
}

// writePPUData handles writes to PPUDATA ($2007)
func (p *PPU) writePPUData(value uint8) {
	if p.memory != nil {
		p.memory.Write(p.v, value)
	}

	// Auto-increment address (this must happen regardless of memory availability)
	if p.ppuCtrl&0x04 != 0 {
		p.v += 32 // Increment by 32 (down)
	} else {
		p.v += 1 // Increment by 1 (across)
	}
	p.v &= 0x3FFF // Wrap to 14-bit address space
}

// GetFrameBuffer returns the current frame buffer
func (p *PPU) GetFrameBuffer() [256 * 240]uint32 {
	return p.frameBuffer
}

// GetFrameCount returns the current frame count
func (p *PPU) GetFrameCount() uint64 {
	return p.frameCount
}

// SetFrameCount sets the frame count (for synchronization)
func (p *PPU) SetFrameCount(count uint64) {
	p.frameCount = count
}

// GetScanline returns the current scanline
func (p *PPU) GetScanline() int {
	return p.scanline
}

// GetCycle returns the current cycle
func (p *PPU) GetCycle() int {
	return p.cycle
}

// IsRenderingEnabled returns true if rendering is enabled
func (p *PPU) IsRenderingEnabled() bool {
	return p.renderingEnabled
}

// IsVBlank returns true if currently in vertical blank
func (p *PPU) IsVBlank() bool {
	return (p.ppuStatus & 0x80) != 0
}

// GetCycleCount returns the total PPU cycle count
func (p *PPU) GetCycleCount() uint64 {
	return p.cycleCount
}

// EnableBackgroundDebugLogging enables background debug logging
func (p *PPU) EnableBackgroundDebugLogging(enabled bool) {
	// Debug logging placeholder - can be extended for actual logging
}

// SetBackgroundDebugVerbosity sets the verbosity level for background debug logging
func (p *PPU) SetBackgroundDebugVerbosity(level int) {
	// Debug verbosity placeholder - can be extended for actual logging
}

// NES 2C02 Color Palette (NTSC) - Based on Dendy emulator palette
var nesColorPalette = [64]uint32{
	// Row 0 (0x00-0x0F)
	0xFF666666, 0xFF002A88, 0xFF1412A7, 0xFF3B00A4, 0xFF5C007E, 0xFF6E0040, 0xFF6C0600, 0xFF561D00,
	0xFF333500, 0xFF0B4800, 0xFF005200, 0xFF004F08, 0xFF00404D, 0xFF000000, 0xFF000000, 0xFF000000,
	// Row 1 (0x10-0x1F)
	0xFFADADAD, 0xFF155FD9, 0xFF4240FF, 0xFF7527FE, 0xFFA01ACC, 0xFFB71E7B, 0xFFB53120, 0xFF994E00,
	0xFF6B6D00, 0xFF388700, 0xFF0C9300, 0xFF008F32, 0xFF007C8D, 0xFF000000, 0xFF000000, 0xFF000000,
	// Row 2 (0x20-0x2F)
	0xFFFFFEFF, 0xFF64B0FF, 0xFF9290FF, 0xFFC676FF, 0xFFF36AFF, 0xFFFE6ECC, 0xFFFE8170, 0xFFEA9E22,
	0xFFBCBE00, 0xFF88D800, 0xFF5CE430, 0xFF45E082, 0xFF48CDDE, 0xFF4F4F4F, 0xFF000000, 0xFF000000,
	// Row 3 (0x30-0x3F)
	0xFFFFFEFF, 0xFFC0DFFF, 0xFFD3D2FF, 0xFFE8C8FF, 0xFFFBC2FF, 0xFFFEC4EA, 0xFFFECCC5, 0xFFF7D8A5,
	0xFFE4E594, 0xFFCFF29B, 0xFFBEFBB3, 0xFFB8F8D8, 0xFFB8F8F8, 0xFF000000, 0xFF000000, 0xFF000000,
}

// NESColorToRGB converts a NES color index to RGB value
func NESColorToRGB(colorIndex uint8) uint32 {
	if colorIndex >= 64 {
		return 0x000000 // Return black for invalid indices
	}
	// Remove alpha channel to return RGB format (0x00RRGGBB)
	return nesColorPalette[colorIndex] & 0x00FFFFFF
}

// NESColorToRGB converts a NES color index to RGB value (PPU method)
func (p *PPU) NESColorToRGB(colorIndex uint8) uint32 {
	return NESColorToRGB(colorIndex)
}

// ClearFrameBuffer clears the frame buffer to a specific color
func (p *PPU) ClearFrameBuffer(color uint32) {
	for i := range p.frameBuffer {
		p.frameBuffer[i] = color
	}
}

// Scroll helper methods for VRAM address manipulation

// getCoarseX extracts the coarse X scroll from v register (bits 0-4)
func (p *PPU) getCoarseX() int {
	return int(p.v & 0x001F)
}

// getCoarseY extracts the coarse Y scroll from v register (bits 5-9)
func (p *PPU) getCoarseY() int {
	return int((p.v >> 5) & 0x001F)
}

// getFineY extracts the fine Y scroll from v register (bits 12-14)
func (p *PPU) getFineY() int {
	return int((p.v >> 12) & 0x0007)
}

// getNametable extracts the nametable select from v register (bits 10-11)
func (p *PPU) getNametable() int {
	return int((p.v >> 10) & 0x0003)
}

// incrementX increments the coarse X and wraps to next nametable if needed
func (p *PPU) incrementX() {
	// If coarse X == 31
	if (p.v & 0x001F) == 31 {
		p.v &= ^uint16(0x001F) // Clear coarse X
		p.v ^= 0x0400         // Switch horizontal nametable
	} else {
		p.v++ // Increment coarse X
	}
}

// incrementY increments fine Y, and if it overflows, increments coarse Y
func (p *PPU) incrementY() {
	// If fine Y < 7
	if (p.v & 0x7000) != 0x7000 {
		p.v += 0x1000 // Increment fine Y
	} else {
		p.v &= ^uint16(0x7000) // Clear fine Y
		y := (p.v & 0x03E0) >> 5 // Coarse Y
		if y == 29 {
			y = 0
			p.v ^= 0x0800 // Switch vertical nametable
		} else if y == 31 {
			y = 0 // Wrap around without switching nametable
		} else {
			y++ // Increment coarse Y
		}
		p.v = (p.v & ^uint16(0x03E0)) | (y << 5) // Put coarse Y back into v
	}
}

// copyX copies all X-related bits from t to v (bits 10, 4-0)
func (p *PPU) copyX() {
	p.v = (p.v & 0xFBE0) | (p.t & 0x041F)
}

// copyY copies all Y-related bits from t to v (bits 11, 14-5)
func (p *PPU) copyY() {
	p.v = (p.v & 0x841F) | (p.t & 0x7BE0)
}

// Debug types for integration testing
type PerformanceAlert struct {
	AlertType   string
	Message     string
	Severity    int
	Timestamp   int64
	FrameNumber uint64
}

type FrameAnalysisData struct {
	FrameNumber     uint64
	RenderTime      int64
	ScanlineCount   int
	TileCount       int
	SpriteCount     int
	MemoryAccesses  int
	BackgroundTiles int
	SpriteTiles     int
}

type ScanlineAnalysis struct {
	ScanlineNumber int
	CycleCount     int
	TileFetches    int
	SpriteFetches  int
	MemoryAccesses []MemoryAccessEvent
	RenderingTime  int64
}

type MemoryAccessEvent struct {
	Address    uint16
	Value      uint8
	AccessType string // "read" or "write"
	Cycle      int
	Scanline   int
	Timestamp  int64
}

type PixelTraceResult struct {
	X             int
	Y             int
	ColorIndex    uint8
	RGBValue      uint32
	Source        string // "background" or "sprite"
	PatternAddr   uint16
	AttributeData uint8
}

type ShiftRegisterState struct {
	PatternLow    uint16
	PatternHigh   uint16
	AttributeLow  uint16
	AttributeHigh uint16
	NextTileID    uint8
	NextAttribute uint8
}

type ScrollDebugInfo struct {
	ScrollX     int
	ScrollY     int
	FineX       uint8
	VramAddress uint16
	TempAddress uint16
	WriteLatch  bool
	Nametable   int
}

type BackgroundRenderingMetrics struct {
	TilesRendered    int
	PatternFetches   int
	AttributeFetches int
	NameTableFetches int
	ScrollUpdates    int
	VramWrites       int
	VramReads        int
}

type DebugFilter struct {
	FilterType string
	Parameters map[string]interface{}
	Enabled    bool
}

type PixelRegion struct {
	StartX int
	StartY int
	Width  int
	Height int
	Name   string
}
