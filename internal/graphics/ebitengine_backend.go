//go:build !headless
// +build !headless

package graphics

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// EbitengineBackend implements the Backend interface using Ebitengine
type EbitengineBackend struct {
	initialized bool
	config      Config
	game        *EbitengineGame
}

// EbitengineWindow implements the Window interface for Ebitengine
type EbitengineWindow struct {
	backend            *EbitengineBackend
	title              string
	width              int
	height             int
	game               *EbitengineGame
	running            bool
	events             []InputEvent
	emulatorUpdateFunc func() error
}

// EbitengineGame implements ebiten.Game for the NES emulator
type EbitengineGame struct {
	window       *EbitengineWindow
	frameBuffer  [256 * 240]uint32
	frameImage   *ebiten.Image
	nesWidth     int
	nesHeight    int
	windowWidth  int
	windowHeight int

	// Key state tracking for continuous input detection
	previousKeyStates map[ebiten.Key]bool
	scale             int
	drawCount         int // For limiting debug logs
	
	// Reusable image buffer to prevent memory leaks
	imageBuffer *image.RGBA
}

// NewEbitengineBackend creates a new Ebitengine graphics backend
func NewEbitengineBackend() Backend {
	return &EbitengineBackend{}
}

// Initialize initializes the Ebitengine backend
func (b *EbitengineBackend) Initialize(config Config) error {
	if b.initialized {
		return fmt.Errorf("Ebitengine backend already initialized")
	}

	b.config = config
	b.initialized = true

	return nil
}

// CreateWindow creates an Ebitengine window
func (b *EbitengineBackend) CreateWindow(title string, width, height int) (Window, error) {
	if !b.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	if b.config.Headless {
		return nil, fmt.Errorf("cannot create window in headless mode")
	}

	// Calculate appropriate scale for NES resolution (256x240)
	scale := 1
	if width >= 512 && height >= 480 {
		scale = 2
	}
	if width >= 1024 && height >= 960 {
		scale = 4
	}

	game := &EbitengineGame{
		nesWidth:          256,
		nesHeight:         240,
		windowWidth:       width,
		windowHeight:      height,
		scale:             scale,
		frameImage:        ebiten.NewImage(256, 240),
		previousKeyStates: make(map[ebiten.Key]bool),
		imageBuffer:       image.NewRGBA(image.Rect(0, 0, 256, 240)), // Pre-allocate reusable buffer
	}

	window := &EbitengineWindow{
		backend: b,
		title:   title,
		width:   width,
		height:  height,
		game:    game,
		running: true,
	}

	game.window = window
	b.game = game

	// Configure Ebitengine
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Configure VSync for optimal 60FPS performance
	if b.config.VSync {
		ebiten.SetVsyncEnabled(true)
	} else {
		// Even without VSync, we want to target 60FPS
		ebiten.SetVsyncEnabled(false)
	}

	// Note: Ebitengine automatically targets 60FPS when VSync is enabled
	// For non-VSync mode, the game loop handles frame limiting

	if b.config.Fullscreen {
		ebiten.SetFullscreen(true)
	}

	// Set the filter mode (disable for better performance if not needed)
	if b.config.Filter == "linear" {
		ebiten.SetScreenFilterEnabled(true)
	} else {
		ebiten.SetScreenFilterEnabled(false) // Better performance
	}

	return window, nil
}

// Cleanup releases all Ebitengine resources
func (b *EbitengineBackend) Cleanup() error {
	b.initialized = false
	return nil
}

// IsHeadless returns true if running in headless mode
func (b *EbitengineBackend) IsHeadless() bool {
	return b.config.Headless
}

// GetName returns the backend name
func (b *EbitengineBackend) GetName() string {
	return "Ebitengine"
}

// EbitengineWindow implementation

// SetTitle sets the window title
func (w *EbitengineWindow) SetTitle(title string) {
	w.title = title
	ebiten.SetWindowTitle(title)
}

// GetSize returns window dimensions
func (w *EbitengineWindow) GetSize() (width, height int) {
	return w.width, w.height
}

// ShouldClose returns true if window should close
func (w *EbitengineWindow) ShouldClose() bool {
	return !w.running
}

// SwapBuffers is handled automatically by Ebitengine
func (w *EbitengineWindow) SwapBuffers() {
	// Ebitengine handles buffer swapping automatically
}

// PollEvents processes input events and returns them
func (w *EbitengineWindow) PollEvents() []InputEvent {
	events := w.events
	w.events = nil // Clear events after returning them

	// Debug log for polled events (disabled for performance - uncomment if needed for debugging)
	// if len(events) > 0 && len(events) <= 2 { // Only log when there are 1-2 events to avoid spam
	//	log.Printf("[INPUT_DEBUG] PollEvents returning %d events", len(events))
	//	for i, event := range events {
	//		if event.Type == InputEventTypeButton {
	//			log.Printf("[INPUT_DEBUG] Event %d: Button %v = %v", i, event.Button, event.Pressed)
	//		}
	//	}
	// }

	return events
}

// RenderFrame renders a NES frame buffer to the window
func (w *EbitengineWindow) RenderFrame(frameBuffer [256 * 240]uint32) error {
	if w.game == nil {
		return fmt.Errorf("game not initialized")
	}

	// Copy frame buffer to game
	w.game.frameBuffer = frameBuffer

	// Convert frame buffer to Ebitengine image using reusable buffer
	img := w.game.imageBuffer // Reuse pre-allocated buffer
	nonBlackCount := 0
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			pixel := frameBuffer[y*256+x]
			r := uint8((pixel >> 16) & 0xFF)
			g := uint8((pixel >> 8) & 0xFF)
			b := uint8(pixel & 0xFF)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
			if pixel != 0x000000 {
				nonBlackCount++
			}
		}
	}

	// Debug log frame content very rarely to avoid performance impact
	// Only log every 1800 frames (30 seconds at 60fps) and only if substantial content
	frameNum := w.game.drawCount
	if nonBlackCount > 1000 && frameNum%1800 == 0 {
		log.Printf("[Ebitengine] RenderFrame: %d non-black pixels (frame %d)", nonBlackCount, frameNum)
	}

	w.game.frameImage.ReplacePixels(img.Pix)
	return nil
}

// Cleanup releases window resources
func (w *EbitengineWindow) Cleanup() error {
	w.running = false
	return nil
}

// Run starts the Ebitengine game loop
func (w *EbitengineWindow) Run() error {
	if w.game == nil {
		return fmt.Errorf("game not initialized")
	}

	// Start the Ebitengine game loop
	return ebiten.RunGame(w.game)
}

// SetEmulatorUpdateFunc sets the emulator update function
func (w *EbitengineWindow) SetEmulatorUpdateFunc(updateFunc func() error) {
	w.emulatorUpdateFunc = updateFunc
}

// EbitengineGame implementation

// Update implements ebiten.Game.Update
func (g *EbitengineGame) Update() error {
	if g.window == nil {
		return nil
	}

	// Process keyboard input
	g.processInput()

	// Update the emulator if function is provided
	if g.window.emulatorUpdateFunc != nil {
		if err := g.window.emulatorUpdateFunc(); err != nil {
			// Log error but don't stop the game
			log.Printf("[Ebitengine] Emulator update error: %v", err)
		}
	}

	return nil
}

// Draw implements ebiten.Game.Draw
func (g *EbitengineGame) Draw(screen *ebiten.Image) {
	if g.frameImage == nil {
		// Clear screen to black if no frame available
		screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 255})
		return
	}

	// Clear the screen first
	screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 255})

	// Calculate drawing options for proper scaling and centering
	op := &ebiten.DrawImageOptions{}

	// Calculate scale to fit the window while maintaining aspect ratio
	scaleX := float64(g.windowWidth) / float64(g.nesWidth)
	scaleY := float64(g.windowHeight) / float64(g.nesHeight)

	// Use the smaller scale to maintain aspect ratio
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Center the image
	offsetX := (float64(g.windowWidth) - float64(g.nesWidth)*scale) / 2
	offsetY := (float64(g.windowHeight) - float64(g.nesHeight)*scale) / 2

	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)

	// Draw the NES frame
	screen.DrawImage(g.frameImage, op)

	// Debug: Log very rarely to avoid performance impact
	g.drawCount++
	if g.drawCount%1800 == 0 { // Log every 1800 frames (about once per 30 seconds)
		log.Printf("[Ebitengine] Drawing frame %d - %dx%d scaled %.2fx at offset (%.1f,%.1f)",
			g.drawCount, g.nesWidth, g.nesHeight, scale, offsetX, offsetY)
	}
}

// Layout implements ebiten.Game.Layout
func (g *EbitengineGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// Update window dimensions
	g.windowWidth = outsideWidth
	g.windowHeight = outsideHeight

	// Return the screen size - we'll handle scaling in Draw()
	return outsideWidth, outsideHeight
}

// processInput processes keyboard and controller input
func (g *EbitengineGame) processInput() {
	if g.window == nil {
		return
	}

	var events []InputEvent

	// Check for quit events
	if ebiten.IsKeyPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		events = append(events, InputEvent{
			Type:    InputEventTypeQuit,
			Pressed: true,
		})
	}

	// Process keyboard input
	keyMappings := map[ebiten.Key]Key{
		ebiten.KeyEscape:     KeyEscape,
		ebiten.KeyEnter:      KeyEnter,
		ebiten.KeySpace:      KeySpace,
		ebiten.KeyArrowUp:    KeyUp,
		ebiten.KeyArrowDown:  KeyDown,
		ebiten.KeyArrowLeft:  KeyLeft,
		ebiten.KeyArrowRight: KeyRight,
		ebiten.KeyW:          KeyW,
		ebiten.KeyA:          KeyA,
		ebiten.KeyS:          KeyS,
		ebiten.KeyD:          KeyD,
		ebiten.KeyJ:          KeyJ,
		ebiten.KeyK:          KeyK,
		ebiten.KeyX:          KeyX,
		ebiten.KeyZ:          KeyZ,
		// Number keys for Player 2 controller
		ebiten.Key1:          Key1,
		ebiten.Key2:          Key2,
		ebiten.Key3:          Key3,
		ebiten.Key4:          Key4,
		ebiten.Key5:          Key5,
		ebiten.Key6:          Key6,
		ebiten.Key7:          Key7,
		ebiten.Key8:          Key8,
		ebiten.KeyF1:         KeyF1,
		ebiten.KeyF2:         KeyF2,
		ebiten.KeyF3:         KeyF3,
		ebiten.KeyF4:         KeyF4,
		ebiten.KeyF5:         KeyF5,
		ebiten.KeyF6:         KeyF6,
		ebiten.KeyF7:         KeyF7,
		ebiten.KeyF8:         KeyF8,
		ebiten.KeyF9:         KeyF9,
		ebiten.KeyF10:        KeyF10,
		ebiten.KeyF11:        KeyF11,
		ebiten.KeyF12:        KeyF12,
	}

	// Optimized key change detection - only check keys that actually changed
	var rawKeyEvents []InputEvent
	for ebitenKey, key := range keyMappings {
		// Use Ebitengine's efficient key change detection
		if inpututil.IsKeyJustPressed(ebitenKey) {
			rawKeyEvents = append(rawKeyEvents, InputEvent{
				Type:    InputEventTypeKey,
				Key:     key,
				Pressed: true,
			})
			g.previousKeyStates[ebitenKey] = true
		} else if inpututil.IsKeyJustReleased(ebitenKey) {
			rawKeyEvents = append(rawKeyEvents, InputEvent{
				Type:    InputEventTypeKey,
				Key:     key,
				Pressed: false,
			})
			g.previousKeyStates[ebitenKey] = false
		}
	}

	// Map keys to NES controller buttons
	var finalEvents []InputEvent
	buttonMappings := map[Key]Button{
		// Player 1 controller (existing mappings)
		KeyUp:    ButtonUp,
		KeyDown:  ButtonDown,
		KeyLeft:  ButtonLeft,
		KeyRight: ButtonRight,
		KeyW:     ButtonUp,
		KeyS:     ButtonDown,
		KeyA:     ButtonLeft,
		KeyD:     ButtonRight,
		KeyJ:     ButtonA,
		KeyK:     ButtonB,
		KeyEnter: ButtonStart,
		KeySpace: ButtonSelect,
		// Player 2 controller (number keys 1-8)
		Key1:     Button2Up,
		Key2:     Button2Down,
		Key3:     Button2Left,
		Key4:     Button2Right,
		Key5:     Button2A,
		Key6:     Button2B,
		Key7:     Button2Start,
		Key8:     Button2Select,
	}

	// Convert key events to button events
	for _, event := range rawKeyEvents {
		// ボタンにマッピングできるキーなら、ボタンイベントに変換して追加
		if button, exists := buttonMappings[event.Key]; exists {
			finalEvents = append(finalEvents, InputEvent{
				Type:    InputEventTypeButton,
				Button:  button,
				Pressed: event.Pressed,
			})
			// Debug logging disabled for performance - uncomment if needed for debugging
			// log.Printf("[INPUT_DEBUG] Mapped Key %v to Button %v (Pressed: %t)", event.Key, button, event.Pressed)

		} else {
			// マッピングできないイベント（Quitなど）は、そのまま追加
			finalEvents = append(finalEvents, event)
		}
	}

	// Store events for retrieval by PollEvents
	g.window.events = append(g.window.events, finalEvents...)
}

// Debug logging for development
func (g *EbitengineGame) logDebug(msg string) {
	log.Printf("[Ebitengine] %s", msg)
}
