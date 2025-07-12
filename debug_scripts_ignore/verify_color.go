package main
import (
    "fmt"
    "gones/internal/ppu"
)
func main() {
    p := ppu.New()
    
    // Test critical colors from bug reports
    testColors := []struct {
        index    uint8
        expected uint32
        name     string
    }{
        {0x22, 0x5C94FC, "Sky Blue"},
        {0x16, 0xB40000, "Mario Red"},
        {0x1A, 0x00A800, "Pipe Green"},
    }
    
    allCorrect := true
    for _, test := range testColors {
        actual := p.NESColorToRGB(test.index)
        fmt.Printf("Color 0x%02X (%s) = 0x%06X\n", test.index, test.name, actual)
        if actual == test.expected {
            fmt.Printf("✓ Color 0x%02X correctly mapped to 0x%06X (%s)\n", test.index, test.expected, test.name)
        } else {
            fmt.Printf("✗ Color 0x%02X expected 0x%06X but got 0x%06X (%s)\n", test.index, test.expected, actual, test.name)
            allCorrect = false
        }
        fmt.Println()
    }
    
    if allCorrect {
        fmt.Println("🎉 ALL CRITICAL COLORS RENDER CORRECTLY!")
    } else {
        fmt.Println("❌ Some colors are incorrect")
    }
}
