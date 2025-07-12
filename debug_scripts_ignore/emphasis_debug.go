package main

import "fmt"

func applyColorEmphasis(rgb uint32, emphasisBits uint8) uint32 {
	if emphasisBits == 0 {
		return rgb
	}
	
	redEmphasis := (emphasisBits & 0x01) != 0   
	greenEmphasis := (emphasisBits & 0x02) != 0  
	blueEmphasis := (emphasisBits & 0x04) != 0  
	
	// Extract RGB components
	r := float64((rgb >> 16) & 0xFF)
	g := float64((rgb >> 8) & 0xFF)
	b := float64(rgb & 0xFF)
	
	// Emphasis factor (darken non-emphasized channels)
	emphasisFactor := 0.75
	
	// Apply emphasis by darkening the non-emphasized color components
	if !redEmphasis {
		r *= emphasisFactor
	}
	if !greenEmphasis {
		g *= emphasisFactor
	}
	if !blueEmphasis {
		b *= emphasisFactor
	}
	
	// Clamp values to valid range
	if r > 255 { r = 255 }
	if g > 255 { g = 255 }
	if b > 255 { b = 255 }
	
	return (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
}

func main() {
	skyBlue := uint32(0x9290FF) // Color 0x22 - sky blue
	
	fmt.Printf("Testing color emphasis on sky blue (0x%06X):\n", skyBlue)
	fmt.Printf("Original: R=%d G=%d B=%d\n", (skyBlue>>16)&0xFF, (skyBlue>>8)&0xFF, skyBlue&0xFF)
	
	// Test different emphasis combinations
	emphasisTests := []struct{
		name string
		bits uint8
	}{
		{"No emphasis", 0x00},
		{"Red emphasis", 0x01},
		{"Green emphasis", 0x02}, 
		{"Blue emphasis", 0x04},
		{"Red+Green emphasis", 0x03},
		{"Red+Blue emphasis", 0x05},
		{"Green+Blue emphasis", 0x06},
		{"All emphasis", 0x07},
	}
	
	for _, test := range emphasisTests {
		result := applyColorEmphasis(skyBlue, test.bits)
		r, g, b := (result>>16)&0xFF, (result>>8)&0xFF, result&0xFF
		fmt.Printf("%s (0x%02X): 0x%06X -> R=%d G=%d B=%d", test.name, test.bits, result, r, g, b)
		if result != skyBlue {
			fmt.Printf(" (MODIFIED)")
		}
		fmt.Printf("\n")
	}
}