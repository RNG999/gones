package graphics

import (
	"fmt"
	"os"
)

// HeadlessBackend implements the Backend interface for headless operation
type HeadlessBackend struct {
	initialized bool
	config      Config
}

// HeadlessWindow implements the Window interface for headless operation
type HeadlessWindow struct {
	title       string
	width       int
	height      int
	running     bool
	frameCount  int
	outputPath  string
}

// NewHeadlessBackend creates a new headless graphics backend
func NewHeadlessBackend() Backend {
	return &HeadlessBackend{}
}

// Initialize initializes the headless backend
func (b *HeadlessBackend) Initialize(config Config) error {
	if b.initialized {
		return fmt.Errorf("headless backend already initialized")
	}

	b.config = config
	b.initialized = true

	return nil
}

// CreateWindow creates a headless "window" (no actual window)
func (b *HeadlessBackend) CreateWindow(title string, width, height int) (Window, error) {
	if !b.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	return &HeadlessWindow{
		title:      title,
		width:      width,
		height:     height,
		running:    true,
		frameCount: 0,
		outputPath: "frame_output",
	}, nil
}

// Cleanup releases all headless resources
func (b *HeadlessBackend) Cleanup() error {
	b.initialized = false
	return nil
}

// IsHeadless returns true (this is a headless backend)
func (b *HeadlessBackend) IsHeadless() bool {
	return true
}

// GetName returns the backend name
func (b *HeadlessBackend) GetName() string {
	return "Headless"
}

// HeadlessWindow implementation

// SetTitle sets the window title (for logging purposes)
func (w *HeadlessWindow) SetTitle(title string) {
	w.title = title
}

// GetSize returns window dimensions
func (w *HeadlessWindow) GetSize() (width, height int) {
	return w.width, w.height
}

// ShouldClose returns true if window should close
func (w *HeadlessWindow) ShouldClose() bool {
	return !w.running
}

// SwapBuffers does nothing in headless mode
func (w *HeadlessWindow) SwapBuffers() {
	// No-op for headless
}

// PollEvents returns empty events list (no input in headless mode)
func (w *HeadlessWindow) PollEvents() []InputEvent {
	return nil
}

// RenderFrame optionally saves the frame to disk
func (w *HeadlessWindow) RenderFrame(frameBuffer [256 * 240]uint32) error {
	w.frameCount++

	// Save specific frames for debugging
	if w.frameCount == 31 || w.frameCount == 61 || w.frameCount == 120 {
		filename := fmt.Sprintf("frame_%03d.ppm", w.frameCount)
		return w.saveFrameAsPPM(frameBuffer, filename)
	}

	return nil
}

// saveFrameAsPPM saves the frame buffer as a PPM image file
func (w *HeadlessWindow) saveFrameAsPPM(frameBuffer [256 * 240]uint32, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filename, err)
	}
	defer file.Close()

	// PPM header
	fmt.Fprintf(file, "P3\n256 240\n255\n")

	// RGB data
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			pixel := frameBuffer[y*256+x]
			r := (pixel >> 16) & 0xFF
			g := (pixel >> 8) & 0xFF
			b := pixel & 0xFF
			fmt.Fprintf(file, "%d %d %d ", r, g, b)
		}
		fmt.Fprintf(file, "\n")
	}

	return nil
}

// Cleanup releases window resources
func (w *HeadlessWindow) Cleanup() error {
	w.running = false
	return nil
}

// SetOutputPath sets the output path for frame dumps
func (w *HeadlessWindow) SetOutputPath(path string) {
	w.outputPath = path
}

// GetFrameCount returns the current frame count
func (w *HeadlessWindow) GetFrameCount() int {
	return w.frameCount
}