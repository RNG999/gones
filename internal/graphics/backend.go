// Package graphics provides an abstraction layer for different rendering backends
package graphics

// Backend represents a graphics rendering backend (SDL2, Ebitengine, etc.)
type Backend interface {
	// Initialize initializes the graphics backend
	Initialize(config Config) error

	// CreateWindow creates a window for rendering (returns nil for headless)
	CreateWindow(title string, width, height int) (Window, error)

	// Cleanup releases all resources
	Cleanup() error

	// IsHeadless returns true if running in headless mode
	IsHeadless() bool

	// GetName returns the backend name for identification
	GetName() string
}

// Window represents a rendering window
type Window interface {
	// SetTitle sets the window title
	SetTitle(title string)

	// GetSize returns window dimensions
	GetSize() (width, height int)

	// ShouldClose returns true if window should close
	ShouldClose() bool

	// SwapBuffers presents the rendered frame
	SwapBuffers()

	// PollEvents processes input events
	PollEvents() []InputEvent

	// RenderFrame renders a NES frame buffer to the window
	RenderFrame(frameBuffer [256 * 240]uint32) error

	// Cleanup releases window resources
	Cleanup() error
}

// Config contains configuration for graphics backends
type Config struct {
	// Window configuration
	WindowTitle  string
	WindowWidth  int
	WindowHeight int
	Fullscreen   bool
	VSync        bool

	// Rendering configuration
	Filter       string // "nearest", "linear"
	AspectRatio  string // "4:3", "stretch"
	
	// Backend-specific options
	Headless     bool
	Debug        bool
}

// InputEvent represents an input event from the window
type InputEvent struct {
	Type      InputEventType
	Key       Key
	Button    Button
	Pressed   bool
	Modifiers ModifierKey
}

// InputEventType represents the type of input event
type InputEventType int

const (
	InputEventTypeKey InputEventType = iota
	InputEventTypeButton
	InputEventTypeQuit
)

// Key represents keyboard keys
type Key int

const (
	KeyUnknown Key = iota
	KeyEscape
	KeyEnter
	KeySpace
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyW
	KeyA
	KeyS
	KeyD
	KeyJ
	KeyK
	KeyX
	KeyZ
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// Button represents controller buttons
type Button int

const (
	ButtonUnknown Button = iota
	ButtonA
	ButtonB
	ButtonSelect
	ButtonStart
	ButtonUp
	ButtonDown
	ButtonLeft
	ButtonRight
	// Player 2 controller buttons
	Button2A
	Button2B
	Button2Select
	Button2Start
	Button2Up
	Button2Down
	Button2Left
	Button2Right
)

// ModifierKey represents modifier keys
type ModifierKey int

const (
	ModifierNone  ModifierKey = 0
	ModifierShift ModifierKey = 1 << iota
	ModifierCtrl
	ModifierAlt
	ModifierSuper
)

// BackendType represents different graphics backend types
type BackendType string

const (
	BackendEbitengine BackendType = "ebitengine"
	BackendHeadless  BackendType = "headless"
	BackendTerminal  BackendType = "terminal"
)

// CreateBackend creates a graphics backend of the specified type
func CreateBackend(backendType BackendType) (Backend, error) {
	switch backendType {
	case BackendEbitengine:
		return NewEbitengineBackend(), nil
	case BackendHeadless:
		return NewHeadlessBackend(), nil
	case BackendTerminal:
		return NewTerminalBackend(), nil
	default:
		// Default to Ebitengine for GUI mode
		return NewEbitengineBackend(), nil
	}
}

// Helper type assertion functions

// AsEbitengineWindow tries to cast a Window to EbitengineWindow
func AsEbitengineWindow(window Window) (*EbitengineWindow, bool) {
	if ebitengineWindow, ok := window.(*EbitengineWindow); ok {
		return ebitengineWindow, true
	}
	return nil, false
}