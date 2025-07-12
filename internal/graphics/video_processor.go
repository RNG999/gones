package graphics

import (
	"math"
)

// VideoProcessor applies video effects to frame buffer
type VideoProcessor struct {
	brightness float32
	contrast   float32
	saturation float32
}

// NewVideoProcessor creates a new video processor
func NewVideoProcessor(brightness, contrast, saturation float32) *VideoProcessor {
	return &VideoProcessor{
		brightness: brightness,
		contrast:   contrast,
		saturation: saturation,
	}
}

// ProcessFrame applies video effects to a frame buffer
func (vp *VideoProcessor) ProcessFrame(frameBuffer []uint32) []uint32 {
	// If all values are at default (1.0), no processing needed
	if vp.brightness == 1.0 && vp.contrast == 1.0 && vp.saturation == 1.0 {
		return frameBuffer
	}

	processed := make([]uint32, len(frameBuffer))
	
	for i, pixel := range frameBuffer {
		// Extract RGB components
		r := float32((pixel >> 16) & 0xFF)
		g := float32((pixel >> 8) & 0xFF)
		b := float32(pixel & 0xFF)
		
		// Apply brightness
		r *= vp.brightness
		g *= vp.brightness
		b *= vp.brightness
		
		// Apply contrast
		r = ((r/255.0 - 0.5) * vp.contrast + 0.5) * 255.0
		g = ((g/255.0 - 0.5) * vp.contrast + 0.5) * 255.0
		b = ((b/255.0 - 0.5) * vp.contrast + 0.5) * 255.0
		
		// Apply saturation by converting to HSL and back
		if vp.saturation != 1.0 {
			h, s, l := rgbToHSL(r/255.0, g/255.0, b/255.0)
			s *= vp.saturation
			if s > 1.0 {
				s = 1.0
			}
			r, g, b = hslToRGB(h, s, l)
			r *= 255.0
			g *= 255.0
			b *= 255.0
		}
		
		// Clamp values to 0-255 range
		r = clamp(r, 0, 255)
		g = clamp(g, 0, 255)
		b = clamp(b, 0, 255)
		
		// Reconstruct pixel
		processed[i] = (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
	}
	
	return processed
}

// clamp limits a value to a range
func clamp(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// rgbToHSL converts RGB to HSL color space
func rgbToHSL(r, g, b float32) (h, s, l float32) {
	max := math.Max(float64(r), math.Max(float64(g), float64(b)))
	min := math.Min(float64(r), math.Min(float64(g), float64(b)))
	
	l = float32((max + min) / 2.0)
	
	if max == min {
		h = 0
		s = 0
	} else {
		d := float32(max - min)
		if l > 0.5 {
			s = d / float32(2.0-max-min)
		} else {
			s = d / float32(max+min)
		}
		
		switch max {
		case float64(r):
			h = (g - b) / d
			if g < b {
				h += 6
			}
		case float64(g):
			h = (b-r)/d + 2
		case float64(b):
			h = (r-g)/d + 4
		}
		h /= 6
	}
	
	return h, s, l
}

// hslToRGB converts HSL to RGB color space
func hslToRGB(h, s, l float32) (r, g, b float32) {
	if s == 0 {
		r = l
		g = l
		b = l
	} else {
		var q float32
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q
		r = hueToRGB(p, q, h+1.0/3.0)
		g = hueToRGB(p, q, h)
		b = hueToRGB(p, q, h-1.0/3.0)
	}
	
	return r, g, b
}

// hueToRGB helper function for HSL to RGB conversion
func hueToRGB(p, q, t float32) float32 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

// SetBrightness updates the brightness value
func (vp *VideoProcessor) SetBrightness(brightness float32) {
	vp.brightness = brightness
}

// SetContrast updates the contrast value
func (vp *VideoProcessor) SetContrast(contrast float32) {
	vp.contrast = contrast
}

// SetSaturation updates the saturation value  
func (vp *VideoProcessor) SetSaturation(saturation float32) {
	vp.saturation = saturation
}