package graphics

import "fmt"

// TerminalBackend implements the Backend interface for terminal-based rendering
type TerminalBackend struct {
	initialized bool
	config      Config
}

// TerminalWindow implements the Window interface for terminal rendering
type TerminalWindow struct {
	title       string
	width       int
	height      int
	running     bool
}

// NewTerminalBackend creates a new terminal graphics backend
func NewTerminalBackend() Backend {
	return &TerminalBackend{}
}

// Initialize initializes the terminal backend
func (b *TerminalBackend) Initialize(config Config) error {
	if b.initialized {
		return fmt.Errorf("terminal backend already initialized")
	}

	b.config = config
	b.initialized = true

	return nil
}

// CreateWindow creates a terminal "window"
func (b *TerminalBackend) CreateWindow(title string, width, height int) (Window, error) {
	if !b.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	return &TerminalWindow{
		title:   title,
		width:   width,
		height:  height,
		running: true,
	}, nil
}

// Cleanup releases all terminal resources
func (b *TerminalBackend) Cleanup() error {
	b.initialized = false
	return nil
}

// IsHeadless returns false (terminal has basic output)
func (b *TerminalBackend) IsHeadless() bool {
	return false
}

// GetName returns the backend name
func (b *TerminalBackend) GetName() string {
	return "Terminal"
}

// TerminalWindow implementation

// SetTitle sets the window title (for terminal title)
func (w *TerminalWindow) SetTitle(title string) {
	w.title = title
	fmt.Printf("\033]0;%s\007", title) // Set terminal title
}

// GetSize returns window dimensions
func (w *TerminalWindow) GetSize() (width, height int) {
	return w.width, w.height
}

// ShouldClose returns true if window should close
func (w *TerminalWindow) ShouldClose() bool {
	return !w.running
}

// SwapBuffers does nothing for terminal
func (w *TerminalWindow) SwapBuffers() {
	// No-op for terminal
}

// PollEvents returns empty events list (no input handling for now)
func (w *TerminalWindow) PollEvents() []InputEvent {
	return nil
}

// RenderFrame renders the frame as ASCII art to terminal
func (w *TerminalWindow) RenderFrame(frameBuffer [256 * 240]uint32) error {
	// Simple ASCII art rendering (very basic)
	// This is just a placeholder - real implementation would be more sophisticated
	
	// Clear screen
	fmt.Print("\033[2J\033[H")
	
	// Render every 8th pixel as a character
	for y := 0; y < 240; y += 8 {
		for x := 0; x < 256; x += 4 {
			pixel := frameBuffer[y*256+x]
			if pixel == 0x000000 {
				fmt.Print(" ")
			} else {
				fmt.Print("█")
			}
		}
		fmt.Println()
	}
	
	return nil
}

// Cleanup releases window resources
func (w *TerminalWindow) Cleanup() error {
	w.running = false
	return nil
}