package main

import (
	"fmt"
	"gones/internal/ppu"
)

func main() {
	fmt.Println("Testing PPU color generation directly...")
	
	// Create a PPU instance 
	ppuInstance := ppu.New()
	
	// Test what color this produces
	colorResult := ppuInstance.NESColorToRGB(0x22)
	
	fmt.Printf("Direct palette test:\n")
	fmt.Printf("Color index 0x22 -> RGB: 0x%06X\n", colorResult)
	fmt.Printf("RGB values: R=%d G=%d B=%d\n", (colorResult>>16)&0xFF, (colorResult>>8)&0xFF, colorResult&0xFF)
	
	// Check a few other critical colors
	testColors := []uint8{0x00, 0x0F, 0x20, 0x21, 0x22, 0x30}
	fmt.Printf("\nPalette color test:\n")
	for _, colorIndex := range testColors {
		rgb := ppuInstance.NESColorToRGB(colorIndex)
		fmt.Printf("Index 0x%02X -> RGB: 0x%06X (R=%d G=%d B=%d)\n", 
			colorIndex, rgb, (rgb>>16)&0xFF, (rgb>>8)&0xFF, rgb&0xFF)
	}
}