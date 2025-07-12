package main

import (
	"fmt"
)

func main() {
	// Test the exact color conversion logic from UpdateRGBA8888
	pixels := []uint32{
		0x9290FF, // This should be purple-blue (index 0x22)
		0xFF0000, // Red
		0x00FF00, // Green  
		0x0000FF, // Blue
	}
	
	// Simulate the conversion logic from UpdateRGBA8888
	bytePixels := make([]byte, len(pixels)*4)
	
	fmt.Println("Testing UpdateRGBA8888 color conversion:")
	
	for i, pixel := range pixels {
		// Extract RGB components (from UpdateRGBA8888)
		r := byte(pixel >> 16) // Red
		g := byte(pixel >> 8)  // Green
		b := byte(pixel)       // Blue
		a := byte(0xFF)        // Alpha
		
		// Pack for RGBA8888 format  
		bytePixels[i*4+0] = r
		bytePixels[i*4+1] = g
		bytePixels[i*4+2] = b
		bytePixels[i*4+3] = a
		
		fmt.Printf("Pixel %d: Input=0x%06X -> R=%d G=%d B=%d A=%d\n", i, pixel, r, g, b, a)
		fmt.Printf("         Bytes: [%d, %d, %d, %d]\n", bytePixels[i*4+0], bytePixels[i*4+1], bytePixels[i*4+2], bytePixels[i*4+3])
		
		// Check if this produces the expected color
		reconstructed := (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
		fmt.Printf("         Reconstructed: 0x%06X (matches: %t)\n\n", reconstructed, reconstructed == (pixel&0xFFFFFF))
	}
}