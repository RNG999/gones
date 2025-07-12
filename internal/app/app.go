// Package app implements the main NES emulator application with GUI support.
package app

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gones/internal/bus"
	"gones/internal/cartridge"
	"gones/internal/graphics"
	"gones/internal/input"
)

// Application represents the main NES emulator application
type Application struct {
	// Core emulation components
	bus *bus.Bus

	// Graphics backend
	graphicsBackend graphics.Backend
	window         graphics.Window
	videoProcessor *graphics.VideoProcessor

	// Application state
	config   *Config
	emulator *Emulator
	states   *StateManager

	// Control flags
	running     bool
	paused      bool
	showMenu    bool
	initialized bool
	headless    bool

	// Performance tracking
	frameCount  uint64
	startTime   time.Time
	lastFPSTime time.Time
	currentFPS  float64
	
	// Enhanced FPS monitoring
	lastFrameTime    time.Time
	frameCountAtLastFPS uint64
	averageFPS       float64
	maxFrameTime     time.Duration
	minFrameTime     time.Duration
	lastFPSLog       time.Time
	
	// Performance timing hooks
	inputTime     time.Duration
	emulatorTime  time.Duration
	renderTime    time.Duration
	totalInputTime   time.Duration
	totalEmulatorTime time.Duration
	totalRenderTime   time.Duration
	
	// Frame consistency monitoring
	recentFrameTimes [10]time.Duration // Rolling buffer of last 10 frame times
	frameTimeIndex   int               // Current index in the rolling buffer
	frameTimeSum     time.Duration     // Sum of times in rolling buffer
	frameVariance    float64           // Frame time variance for consistency
	
	// Memory monitoring and periodic cleanup
	lastMemoryCheck    time.Time
	lastCleanup        time.Time
	initialMemoryUsage uint64
	lastMemoryUsage    uint64
	memoryGrowthRate   float64

	// ROM management
	romPath   string
	cartridge *cartridge.Cartridge
	
	// ESC key confirmation tracking
	lastESCTime time.Time

	// Input state caching to prevent redundant updates
	lastController1State [8]bool
	lastController2State [8]bool
	inputStateInitialized bool
	
	// Debug logging frequency control
	debugFrameCounter uint64
}

// ApplicationError represents application-specific errors
type ApplicationError struct {
	Component string
	Operation string
	Err       error
}

func (e *ApplicationError) Error() string {
	return fmt.Sprintf("Application %s error during %s: %v", e.Component, e.Operation, e.Err)
}

// NewApplication creates a new NES emulator application
func NewApplication(configPath string) (*Application, error) {
	return NewApplicationWithMode(configPath, false)
}

// NewApplicationWithMode creates a new NES emulator application with optional headless mode
func NewApplicationWithMode(configPath string, headless bool) (*Application, error) {
	app := &Application{
		config:      NewConfig(),
		running:     false,
		paused:      false,
		showMenu:    false,
		initialized: false,
		headless:    headless,
		startTime:   time.Now(),
		lastFPSTime: time.Now(),
	}

	// Load configuration
	if configPath != "" {
		if err := app.config.LoadFromFile(configPath); err != nil {
			// Log warning but continue with defaults
			fmt.Printf("[APP_WARNING] Could not load config from %s, using defaults: %v\n", configPath, err)
		}
	}

	// Initialize components
	if err := app.initializeComponents(headless); err != nil {
		return nil, &ApplicationError{
			Component: "initialization",
			Operation: "component setup",
			Err:       err,
		}
	}

	return app, nil
}

// initializeComponents initializes all application components
func (app *Application) initializeComponents(headless bool) error {
	// Create system bus
	app.bus = bus.New()

	// Initialize graphics backend
	if err := app.initializeGraphicsBackend(headless); err != nil {
		return fmt.Errorf("failed to initialize graphics backend: %v", err)
	}

	// Note: Audio system will be implemented with the graphics backend in the future
	// For now, emulator runs without audio to avoid SDL2 dependencies

	// Note: UI system removed to eliminate SDL2 dependency
	// UI will be reimplemented using the graphics backend's UI capabilities

	// Create emulator
	app.emulator = NewEmulator(app.bus, app.config)

	// Create state manager
	app.states = NewStateManager(app.config.Paths.SaveStates)

	app.initialized = true
	return nil
}

// initializeGraphicsBackend initializes the graphics backend based on configuration
func (app *Application) initializeGraphicsBackend(headless bool) error {
	// Determine backend type
	var backendType graphics.BackendType
	if headless {
		backendType = graphics.BackendHeadless
	} else {
		switch app.config.Video.Backend {
		case "ebitengine":
			backendType = graphics.BackendEbitengine
		case "headless":
			backendType = graphics.BackendHeadless
		case "terminal":
			backendType = graphics.BackendTerminal
		default:
			// Default to Ebitengine for best compatibility
			backendType = graphics.BackendEbitengine
		}
	}

	// Create graphics backend
	var err error
	app.graphicsBackend, err = graphics.CreateBackend(backendType)
	if err != nil {
		return fmt.Errorf("failed to create graphics backend: %v", err)
	}

	// Initialize backend
	graphicsConfig := graphics.Config{
		WindowTitle:  "gones - Go NES Emulator",
		WindowWidth:  app.config.Window.Width,
		WindowHeight: app.config.Window.Height,
		Fullscreen:   app.config.Window.Fullscreen,
		VSync:        app.config.Video.VSync,
		Filter:       app.config.Video.Filter,
		AspectRatio:  app.config.Video.AspectRatio,
		Headless:     headless,
		Debug:        app.config.Debug.EnableLogging,
	}

	if err := app.graphicsBackend.Initialize(graphicsConfig); err != nil {
		// If Ebitengine fails (e.g., no DISPLAY), fallback to headless mode
		if backendType == graphics.BackendEbitengine {
			fmt.Printf("[APP_WARNING] Ebitengine backend failed (%v), falling back to headless mode\n", err)
			app.graphicsBackend, err = graphics.CreateBackend(graphics.BackendHeadless)
			if err != nil {
				return fmt.Errorf("failed to create fallback headless backend: %v", err)
			}
			graphicsConfig.Headless = true
			if err := app.graphicsBackend.Initialize(graphicsConfig); err != nil {
				return fmt.Errorf("failed to initialize fallback headless backend: %v", err)
			}
		} else {
			return fmt.Errorf("failed to initialize graphics backend: %v", err)
		}
	}

	// Create window (only if not headless)
	if !headless && !app.graphicsBackend.IsHeadless() {
		app.window, err = app.graphicsBackend.CreateWindow(
			graphicsConfig.WindowTitle,
			graphicsConfig.WindowWidth,
			graphicsConfig.WindowHeight,
		)
		if err != nil {
			return fmt.Errorf("failed to create window: %v", err)
		}
	}

	// Initialize video processor
	app.videoProcessor = graphics.NewVideoProcessor(
		app.config.Video.Brightness,
		app.config.Video.Contrast,
		app.config.Video.Saturation,
	)

	return nil
}

// Note: Audio functions removed to eliminate SDL2 dependency
// Audio will be reimplemented using the graphics backend's audio capabilities

// LoadROM loads a ROM file into the emulator
func (app *Application) LoadROM(romPath string) error {
	if !app.initialized {
		return errors.New("application not initialized")
	}

	// Load cartridge
	cart, err := cartridge.LoadFromFile(romPath)
	if err != nil {
		return &ApplicationError{
			Component: "cartridge",
			Operation: "load ROM",
			Err:       err,
		}
	}

	// Store cartridge and path
	app.cartridge = cart
	app.romPath = romPath

	// Load cartridge into bus
	app.bus.LoadCartridge(cart)

	// Reset system
	app.bus.Reset()

	// Note: Audio sample rate configuration will be restored when audio backend is added

	// Update window title (if window exists)
	if app.window != nil {
		romName := filepath.Base(romPath)
		title := fmt.Sprintf("gones - %s", romName)
		app.window.SetTitle(title)
	}

	// Start the emulator
	app.emulator.Start()

	return nil
}

// Run starts the main application loop
func (app *Application) Run() error {
	if !app.initialized {
		return errors.New("application not initialized")
	}

	app.running = true
	app.startTime = time.Now()
	app.lastFPSTime = time.Now()

	if app.config.Debug.EnableLogging {
		fmt.Printf("[APP_DEBUG] Starting emulator with %s backend...\n", app.graphicsBackend.GetName())
	}

	// Check if we're using Ebitengine backend
	if app.graphicsBackend.GetName() == "Ebitengine" && app.window != nil {
		// For Ebitengine, we need to start the game loop differently
		if ebitengineWindow, ok := graphics.AsEbitengineWindow(app.window); ok {
			// Set up the emulator update function for Ebitengine
			// Simplified for better timing consistency
			ebitengineWindow.SetEmulatorUpdateFunc(func() error {
				frameStartTime := time.Now()
				
				// Process input events (no individual timing to reduce overhead)
				if err := app.processInput(); err != nil {
					if app.config.Debug.EnableLogging {
						fmt.Printf("[APP_ERROR] Input processing error: %v\n", err)
					}
				}
				
				// Update emulator state - this now runs exactly one frame
				emulatorStart := time.Now()
				if err := app.updateEmulator(); err != nil {
					return err
				}
				app.emulatorTime = time.Since(emulatorStart)
				
				// Render the frame
				renderStart := time.Now()
				if err := app.render(); err != nil {
					return err
				}
				app.renderTime = time.Since(renderStart)
				
				// Simplified performance metrics update
				app.updatePerformanceMetricsMinimal(frameStartTime)
				
				// Check if window should close
				if app.window != nil && app.window.ShouldClose() {
					app.Stop()
				}
				
				return nil
			})
			return ebitengineWindow.Run()
		}
	}

	// Standard main application loop for other backends
	for app.running {
		frameStartTime := time.Now()

		// Process input events with timing
		inputStart := time.Now()
		if err := app.processInput(); err != nil {
			if app.config.Debug.EnableLogging {
				fmt.Printf("[APP_ERROR] Input processing error: %v\n", err)
			}
		}
		app.inputTime = time.Since(inputStart)
		app.totalInputTime += app.inputTime

		// Update emulator (if not paused and ROM loaded) with timing
		emulatorStart := time.Now()
		if err := app.updateEmulator(); err != nil {
			if app.config.Debug.EnableLogging {
				fmt.Printf("[APP_DEBUG] Emulator update error: %v\n", err)
			}
		}
		app.emulatorTime = time.Since(emulatorStart)
		app.totalEmulatorTime += app.emulatorTime

		// Render frame with timing
		renderStart := time.Now()
		if err := app.render(); err != nil {
			if app.config.Debug.EnableLogging {
				fmt.Printf("[APP_ERROR] Render error: %v\n", err)
			}
		}
		app.renderTime = time.Since(renderStart)
		app.totalRenderTime += app.renderTime

		// Update performance metrics
		app.updatePerformanceMetrics(frameStartTime)

		// Check if window should close
		if app.window != nil && app.window.ShouldClose() {
			app.Stop()
		}

		// Simple frame rate limiting for non-Ebitengine backends
		time.Sleep(16 * time.Millisecond) // ~60 FPS
	}

	if app.config.Debug.EnableLogging {
		fmt.Println("[APP_DEBUG] Emulator main loop ended")
	}
	return nil
}

// updateEmulator updates the emulator state
func (app *Application) updateEmulator() error {
	if !app.paused && app.cartridge != nil {
		if err := app.emulator.Update(); err != nil {
			return err
		}

		// Note: Audio processing will be added back when audio backend is implemented
	}
	return nil
}

// processInput processes input events from graphics backend
func (app *Application) processInput() error {
	if app.window == nil {
		return nil
	}

	events := app.window.PollEvents()

	// Early return if no events to process - major performance optimization
	if len(events) == 0 {
		return nil
	}

	// Track controller button states for atomic update
	var controller1Changed bool
	var controller2Changed bool
	controller1Buttons := app.lastController1State // Start with cached state
	controller2Buttons := app.lastController2State // Start with cached state
	
	// Initialize input state cache on first run
	if !app.inputStateInitialized && app.bus != nil && app.cartridge != nil {
		inputState := app.bus.GetInputState()
		if inputState != nil {
			// Controller 1
			if inputState.Controller1 != nil {
				app.lastController1State[0] = inputState.Controller1.IsPressed(input.A)
				app.lastController1State[1] = inputState.Controller1.IsPressed(input.B)
				app.lastController1State[2] = inputState.Controller1.IsPressed(input.Select)
				app.lastController1State[3] = inputState.Controller1.IsPressed(input.Start)
				app.lastController1State[4] = inputState.Controller1.IsPressed(input.Up)
				app.lastController1State[5] = inputState.Controller1.IsPressed(input.Down)
				app.lastController1State[6] = inputState.Controller1.IsPressed(input.Left)
				app.lastController1State[7] = inputState.Controller1.IsPressed(input.Right)
				controller1Buttons = app.lastController1State
			}
			// Controller 2
			if inputState.Controller2 != nil {
				app.lastController2State[0] = inputState.Controller2.IsPressed(input.A)
				app.lastController2State[1] = inputState.Controller2.IsPressed(input.B)
				app.lastController2State[2] = inputState.Controller2.IsPressed(input.Select)
				app.lastController2State[3] = inputState.Controller2.IsPressed(input.Start)
				app.lastController2State[4] = inputState.Controller2.IsPressed(input.Up)
				app.lastController2State[5] = inputState.Controller2.IsPressed(input.Down)
				app.lastController2State[6] = inputState.Controller2.IsPressed(input.Left)
				app.lastController2State[7] = inputState.Controller2.IsPressed(input.Right)
				controller2Buttons = app.lastController2State
			}
		}
		app.inputStateInitialized = true
	}

	// Process input events and update button array
	for _, event := range events {
		switch event.Type {
		case graphics.InputEventTypeQuit:
			app.Stop()
			return nil

		case graphics.InputEventTypeButton:
			// Check for special key combinations first
			if app.handleSpecialInput(event) {
				continue
			}

			// Update controller button array for atomic setting
			if app.cartridge != nil {
				// Check if this is a 2P controller button
				if is2PButton(event.Button) {
					buttonIndex := get2PButtonIndex(event.Button)
					if buttonIndex >= 0 {
						controller2Buttons[buttonIndex] = event.Pressed
						controller2Changed = true
						// Disabled for performance - very verbose logging
						// if app.config.Debug.EnableLogging {
						//	log.Printf("[APP_DEBUG] 2P Button: %v -> Index: %d = %v", event.Button, buttonIndex, event.Pressed)
						// }
					}
				} else {
					// 1P controller buttons
					button := graphicsButtonToInputButton(event.Button)
					// Disabled for performance - very verbose logging
					// if app.config.Debug.EnableLogging {
					//	log.Printf("[APP_DEBUG] 1P Button: %v -> Input Button: %v (%d) = %v", event.Button, button, uint8(button), event.Pressed)
					// }
					
					// Map to array index (NES button order: A, B, Select, Start, Up, Down, Left, Right)
					var buttonIndex int
					switch button {
					case input.A:      buttonIndex = 0
					case input.B:      buttonIndex = 1
					case input.Select: buttonIndex = 2
					case input.Start:  buttonIndex = 3
					case input.Up:     buttonIndex = 4
					case input.Down:   buttonIndex = 5
					case input.Left:   buttonIndex = 6
					case input.Right:  buttonIndex = 7
					default: continue // Skip unknown buttons
					}
					
					controller1Buttons[buttonIndex] = event.Pressed
					controller1Changed = true
				}
			}

		case graphics.InputEventTypeKey:
			// Handle key events (function keys, etc.)
			if app.handleKeyInput(event) {
				continue
			}
		}
	}

	// Apply controller button state atomically ONLY if any buttons actually changed
	if controller1Changed && app.bus != nil && app.cartridge != nil {
		// Double-check that state actually changed to prevent redundant updates
		if app.inputStateChanged(app.lastController1State, controller1Buttons) {
			// Reduced frequency debug logging - only log occasionally to avoid performance impact
			app.debugFrameCounter++
			if app.config.Debug.EnableLogging && app.debugFrameCounter%300 == 0 {
				log.Printf("[APP_DEBUG] 1P Controller update: [A:%t B:%t Sel:%t Start:%t U:%t D:%t L:%t R:%t]", 
					controller1Buttons[0], controller1Buttons[1], controller1Buttons[2], controller1Buttons[3],
					controller1Buttons[4], controller1Buttons[5], controller1Buttons[6], controller1Buttons[7])
			}
			app.bus.SetControllerButtons(0, controller1Buttons)
			app.lastController1State = controller1Buttons // Cache new state
		}
	}
	
	if controller2Changed && app.bus != nil && app.cartridge != nil {
		// Double-check that state actually changed to prevent redundant updates
		if app.inputStateChanged(app.lastController2State, controller2Buttons) {
			// Reduced frequency debug logging - only log occasionally to avoid performance impact
			if app.config.Debug.EnableLogging && app.debugFrameCounter%300 == 0 {
				log.Printf("[APP_DEBUG] 2P Controller update: [A:%t B:%t Sel:%t Start:%t U:%t D:%t L:%t R:%t]", 
					controller2Buttons[0], controller2Buttons[1], controller2Buttons[2], controller2Buttons[3],
					controller2Buttons[4], controller2Buttons[5], controller2Buttons[6], controller2Buttons[7])
			}
			app.bus.SetControllerButtons(2, controller2Buttons)
			app.lastController2State = controller2Buttons // Cache new state
		}
	}

	return nil
}

// inputStateChanged compares two controller button states to detect changes
func (app *Application) inputStateChanged(oldState, newState [8]bool) bool {
	for i := 0; i < 8; i++ {
		if oldState[i] != newState[i] {
			return true
		}
	}
	return false
}

// handleSpecialInput handles special input combinations (menu, pause, etc.)
func (app *Application) handleSpecialInput(event graphics.InputEvent) bool {
	// Only handle key press events for special combinations
	if !event.Pressed {
		return false
	}

	// Handle escape key for quitting - require double-tap within 3 seconds
	if event.Type == graphics.InputEventTypeKey && event.Key == graphics.KeyEscape {
		now := time.Now()
		if !app.lastESCTime.IsZero() && now.Sub(app.lastESCTime) < 3*time.Second {
			// Second ESC within 3 seconds - confirm quit
			fmt.Println("👋 ESC double-tap confirmed - Shutting down emulator...")
			app.Stop()
			return true
		} else {
			// First ESC or too much time passed - warn user
			fmt.Println("⚠️  ESC pressed - Press ESC again within 3 seconds to quit, or continue playing...")
			app.lastESCTime = now
			return true
		}
	}

	// Reset ESC timer if any other key is pressed
	if event.Type == graphics.InputEventTypeKey && event.Key != graphics.KeyEscape {
		app.lastESCTime = time.Time{} // Reset ESC timer
	}

	// Handle function keys for save states
	if event.Type == graphics.InputEventTypeKey {
		switch event.Key {
		case graphics.KeyF1, graphics.KeyF2, graphics.KeyF3, graphics.KeyF4, graphics.KeyF5,
			 graphics.KeyF6, graphics.KeyF7, graphics.KeyF8, graphics.KeyF9, graphics.KeyF10:
			slot := int(event.Key - graphics.KeyF1)
			if event.Modifiers&graphics.ModifierShift != 0 {
				// Load state
				if err := app.LoadState(slot); err != nil {
					fmt.Printf("Failed to load state %d: %v\n", slot, err)
				}
			} else {
				// Save state
				if err := app.SaveState(slot); err != nil {
					fmt.Printf("Failed to save state %d: %v\n", slot, err)
				}
			}
			return true
		}
	}

	// DISABLED: Removed Select button pause functionality to allow Select to reach the game
	// The Select button should be available for NES games, not consumed by pause functionality
	
	// Example: Start + Select = Show menu (disabled due to isButtonPressed always returning false)
	// if event.Button == graphics.ButtonStart && app.isButtonPressed(graphics.ButtonSelect) {
	//	app.ToggleMenu()
	//	return true
	// }

	// DISABLED: Select button alone for pause - this was intercepting Select button from games
	// if event.Button == graphics.ButtonSelect && !app.showMenu {
	//	app.TogglePause()
	//	return true
	// }

	return false
}

// handleKeyInput handles key input events
func (app *Application) handleKeyInput(event graphics.InputEvent) bool {
	// Handle other key events here
	return false
}

// isButtonPressed checks if a button is currently pressed
func (app *Application) isButtonPressed(button graphics.Button) bool {
	// This is a simplified check - in a real implementation,
	// you might want to check the actual input state
	return false
}

// graphicsButtonToInputButton converts graphics.Button to input.Button
func graphicsButtonToInputButton(gButton graphics.Button) input.Button {
	switch gButton {
	case graphics.ButtonA:
		return input.A
	case graphics.ButtonB:
		return input.B
	case graphics.ButtonSelect:
		return input.Select
	case graphics.ButtonStart:
		return input.Start
	case graphics.ButtonUp:
		return input.Up
	case graphics.ButtonDown:
		return input.Down
	case graphics.ButtonLeft:
		return input.Left
	case graphics.ButtonRight:
		return input.Right
	default:
		return input.A // default fallback
	}
}

// is2PButton checks if the button belongs to 2P controller
func is2PButton(gButton graphics.Button) bool {
	switch gButton {
	case graphics.Button2A, graphics.Button2B, graphics.Button2Select, graphics.Button2Start,
		 graphics.Button2Up, graphics.Button2Down, graphics.Button2Left, graphics.Button2Right:
		return true
	default:
		return false
	}
}

// get2PButtonIndex returns the array index for 2P controller buttons
func get2PButtonIndex(gButton graphics.Button) int {
	switch gButton {
	case graphics.Button2A:      return 0
	case graphics.Button2B:      return 1
	case graphics.Button2Select: return 2
	case graphics.Button2Start:  return 3
	case graphics.Button2Up:     return 4
	case graphics.Button2Down:   return 5
	case graphics.Button2Left:   return 6
	case graphics.Button2Right:  return 7
	default:                     return -1
	}
}

// SetControllerButtons sets all button states at once (array approach like ChibiNES/Fogleman)
func (app *Application) SetControllerButtons(controller int, buttons [8]bool) {
	if app.bus != nil {
		app.bus.SetControllerButtons(controller, buttons)
	}
}

// GetBus returns the bus for direct access (useful for testing and advanced control)
func (app *Application) GetBus() *bus.Bus {
	return app.bus
}

// render renders the current frame
func (app *Application) render() error {
	// Skip rendering if no window available (headless mode)
	if app.window == nil {
		return nil
	}

	// Render emulator output (if ROM loaded)
	if app.cartridge != nil {
		frameBufferSlice := app.bus.GetFrameBuffer()
		
		// Apply video processing if configured
		if app.videoProcessor != nil {
			frameBufferSlice = app.videoProcessor.ProcessFrame(frameBufferSlice)
		}
		
		// Convert slice to array
		var frameBuffer [256 * 240]uint32
		copy(frameBuffer[:], frameBufferSlice)
		if err := app.window.RenderFrame(frameBuffer); err != nil {
			return fmt.Errorf("failed to render NES frame: %v", err)
		}
	}

	// Render UI overlays (TODO: Update UI system for new graphics backend)
	// if app.showMenu && app.ui != nil {
	//     if err := app.ui.RenderMenu(); err != nil {
	//         return fmt.Errorf("failed to render menu: %v", err)
	//     }
	// }

	// Present frame
	app.window.SwapBuffers()

	return nil
}

// updatePerformanceMetrics updates performance tracking with high-precision timing
func (app *Application) updatePerformanceMetrics(frameStartTime time.Time) {
	now := time.Now()
	app.frameCount++

	// Calculate frame time
	frameTime := now.Sub(frameStartTime)
	
	// Initialize timing on first frame
	if app.lastFrameTime.IsZero() {
		app.lastFrameTime = frameStartTime
		app.lastFPSTime = now
		app.frameCountAtLastFPS = app.frameCount
		app.minFrameTime = frameTime
		app.maxFrameTime = frameTime
		app.lastFPSLog = now
		app.lastMemoryCheck = now
		app.lastCleanup = now
		
		// Initialize memory baseline
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		app.initialMemoryUsage = memStats.Alloc
		app.lastMemoryUsage = memStats.Alloc
		return
	}

	// Track min/max frame times
	if frameTime < app.minFrameTime {
		app.minFrameTime = frameTime
	}
	if frameTime > app.maxFrameTime {
		app.maxFrameTime = frameTime
	}

	// Frame consistency monitoring - rolling buffer with O(1) variance calculation
	oldFrameTime := app.recentFrameTimes[app.frameTimeIndex]
	app.frameTimeSum -= oldFrameTime // Remove old value from sum
	app.recentFrameTimes[app.frameTimeIndex] = frameTime // Add new value
	app.frameTimeSum += frameTime // Add new value to sum
	app.frameTimeIndex = (app.frameTimeIndex + 1) % 10 // Advance index

	// O(1) variance calculation using rolling statistics
	if app.frameCount >= 10 {
		// Use Welford's online algorithm for rolling variance
		// This maintains running mean and variance in O(1) time
		avgFrameTime := app.frameTimeSum / 10
		
		// For simplicity, we'll use a simplified rolling variance
		// that's accurate enough for performance monitoring
		if app.frameCount == 10 {
			// Initialize variance on first complete buffer
			variance := 0.0
			for _, ft := range app.recentFrameTimes {
				diff := float64(ft - avgFrameTime)
				variance += diff * diff
			}
			app.frameVariance = variance / 10.0
		} else {
			// Rolling update: use exponential moving average for variance estimation
			// This gives recent frames more weight and is O(1)
			newDiff := float64(frameTime - avgFrameTime)
			oldDiff := float64(oldFrameTime - avgFrameTime)
			
			// Exponential smoothing factor (0.1 = 10% weight to new value)
			alpha := 0.1
			newVarianceContrib := newDiff * newDiff
			oldVarianceContrib := oldDiff * oldDiff
			
			app.frameVariance = app.frameVariance*(1-alpha) + (newVarianceContrib-oldVarianceContrib)*alpha
			
			// Ensure variance is never negative (can happen due to floating point errors)
			if app.frameVariance < 0 {
				app.frameVariance = 0
			}
		}
	}

	// Update FPS calculation every second for accuracy
	if now.Sub(app.lastFPSTime) >= time.Second {
		elapsed := now.Sub(app.lastFPSTime).Seconds()
		framesInPeriod := app.frameCount - app.frameCountAtLastFPS
		app.currentFPS = float64(framesInPeriod) / elapsed
		
		// Calculate average FPS since start
		totalElapsed := now.Sub(app.startTime).Seconds()
		if totalElapsed > 0 {
			app.averageFPS = float64(app.frameCount) / totalElapsed
		}
		
		// Update tracking variables
		app.lastFPSTime = now
		app.frameCountAtLastFPS = app.frameCount

		// Log FPS every 5 seconds if debug logging is enabled
		if app.config.Debug.EnableLogging && now.Sub(app.lastFPSLog) >= 5*time.Second {
			targetFrameTime := time.Duration(16670000) // 16.67ms in nanoseconds for 60 FPS target
			app.logFPSMetrics(now, frameTime, targetFrameTime)
			app.lastFPSLog = now
		}
	}

	// Memory monitoring every 30 seconds
	if now.Sub(app.lastMemoryCheck) >= 30*time.Second {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		
		currentMemory := memStats.Alloc
		memoryIncrease := float64(currentMemory) - float64(app.lastMemoryUsage)
		timeDiff := now.Sub(app.lastMemoryCheck).Seconds()
		app.memoryGrowthRate = memoryIncrease / timeDiff / (1024 * 1024) // MB per second
		
		if app.config.Debug.EnableLogging {
			log.Printf("[MEMORY] Current: %.2f MB | Growth: %.3f MB/s | Since start: +%.2f MB", 
				float64(currentMemory)/(1024*1024),
				app.memoryGrowthRate,
				float64(currentMemory-app.initialMemoryUsage)/(1024*1024))
		}
		
		app.lastMemoryUsage = currentMemory
		app.lastMemoryCheck = now
		
		// Warn about high memory growth
		if app.memoryGrowthRate > 0.1 { // More than 0.1 MB/s growth
			log.Printf("[MEMORY_WARNING] High memory growth rate: %.3f MB/s", app.memoryGrowthRate)
		}
	}

	// Periodic resource cleanup every 5 minutes to prevent progressive slowdown
	if now.Sub(app.lastCleanup) >= 5*time.Minute {
		app.performPeriodicCleanup()
		app.lastCleanup = now
	}

	// Warn about dropped frames (frames taking longer than 16.67ms for 60fps)
	if frameTime > 20*time.Millisecond && app.config.Debug.EnableLogging {
		if app.frameCount%300 == 0 { // Only warn occasionally to avoid spam
			log.Printf("[FPS_WARNING] Slow frame detected: %.2fms (target: 16.67ms)", 
				float64(frameTime.Nanoseconds())/1000000.0)
		}
	}

	app.lastFrameTime = now
}

// updatePerformanceMetricsMinimal provides basic performance tracking with minimal overhead
func (app *Application) updatePerformanceMetricsMinimal(frameStartTime time.Time) {
	now := time.Now()
	app.frameCount++
	
	// Calculate frame time
	frameTime := now.Sub(frameStartTime)
	
	// Initialize timing on first frame
	if app.lastFrameTime.IsZero() {
		app.lastFrameTime = frameStartTime
		app.lastFPSTime = now
		app.frameCountAtLastFPS = app.frameCount
		app.minFrameTime = frameTime
		app.maxFrameTime = frameTime
		app.lastFPSLog = now
		return
	}
	
	// Track min/max frame times
	if frameTime < app.minFrameTime {
		app.minFrameTime = frameTime
	}
	if frameTime > app.maxFrameTime {
		app.maxFrameTime = frameTime
	}
	
	// Update FPS calculation every second
	if now.Sub(app.lastFPSTime) >= time.Second {
		elapsed := now.Sub(app.lastFPSTime).Seconds()
		framesInPeriod := app.frameCount - app.frameCountAtLastFPS
		app.currentFPS = float64(framesInPeriod) / elapsed
		
		// Calculate average FPS since start
		totalElapsed := now.Sub(app.startTime).Seconds()
		if totalElapsed > 0 {
			app.averageFPS = float64(app.frameCount) / totalElapsed
		}
		
		// Update tracking variables
		app.lastFPSTime = now
		app.frameCountAtLastFPS = app.frameCount
		
		// Log FPS less frequently to reduce overhead
		if app.config.Debug.EnableLogging && now.Sub(app.lastFPSLog) >= 10*time.Second {
			log.Printf("[FPS] Current: %.1f FPS | Average: %.1f FPS | Frame: %d | Emulator: %.2fms | Render: %.2fms", 
				app.currentFPS, app.averageFPS, app.frameCount,
				float64(app.emulatorTime.Nanoseconds())/1000000.0,
				float64(app.renderTime.Nanoseconds())/1000000.0)
			app.lastFPSLog = now
		}
	}
	
	app.lastFrameTime = now
}

// logFPSMetrics logs detailed FPS and performance information
func (app *Application) logFPSMetrics(now time.Time, lastFrameTime, targetFrameTime time.Duration) {
	log.Printf("[FPS] Current: %.1f FPS | Average: %.1f FPS | Frame: %d | Runtime: %.1fs", 
		app.currentFPS, app.averageFPS, app.frameCount, now.Sub(app.startTime).Seconds())
	
	log.Printf("[TIMING] Frame: %.2fms | Min: %.2fms | Max: %.2fms | Target: %.2fms",
		float64(lastFrameTime.Nanoseconds())/1000000.0,
		float64(app.minFrameTime.Nanoseconds())/1000000.0,
		float64(app.maxFrameTime.Nanoseconds())/1000000.0,
		float64(targetFrameTime.Nanoseconds())/1000000.0)
	
	// Component timing breakdown (current frame)
	log.Printf("[COMPONENTS] Input: %.2fms | Emulator: %.2fms | Render: %.2fms",
		float64(app.inputTime.Nanoseconds())/1000000.0,
		float64(app.emulatorTime.Nanoseconds())/1000000.0,
		float64(app.renderTime.Nanoseconds())/1000000.0)
	
	// Average component timing (since start)
	if app.frameCount > 0 {
		avgInput := float64(app.totalInputTime.Nanoseconds()) / float64(app.frameCount) / 1000000.0
		avgEmulator := float64(app.totalEmulatorTime.Nanoseconds()) / float64(app.frameCount) / 1000000.0
		avgRender := float64(app.totalRenderTime.Nanoseconds()) / float64(app.frameCount) / 1000000.0
		
		log.Printf("[AVERAGES] Input: %.2fms | Emulator: %.2fms | Render: %.2fms",
			avgInput, avgEmulator, avgRender)
	}
	
	// Frame consistency analysis
	if app.frameCount >= 10 {
		avgRecentFrameTime := float64(app.frameTimeSum.Nanoseconds()) / 10.0 / 1000000.0
		// Ensure we don't take sqrt of negative number
		var frameStdDev float64
		if app.frameVariance >= 0 {
			frameStdDev = math.Sqrt(app.frameVariance) / 1000000.0 // Convert to milliseconds
		} else {
			frameStdDev = 0.0
		}
		
		log.Printf("[CONSISTENCY] Recent avg: %.2fms | Std dev: %.2fms | Variance: %.2f",
			avgRecentFrameTime, frameStdDev, app.frameVariance/1000000000000.0)
		
		// Frame pacing assessment
		if frameStdDev < 2.0 {
			log.Printf("[PACING] ✅ Excellent frame pacing (±%.2fms)", frameStdDev)
		} else if frameStdDev < 5.0 {
			log.Printf("[PACING] ⚠️  Moderate frame pacing (±%.2fms)", frameStdDev)
		} else {
			log.Printf("[PACING] ❌ Poor frame pacing (±%.2fms)", frameStdDev)
		}
	}
	
	// Overall performance assessment
	if app.currentFPS >= 58.0 {
		log.Printf("[PERFORMANCE] ✅ Excellent performance (%.1f FPS)", app.currentFPS)
	} else if app.currentFPS >= 45.0 {
		log.Printf("[PERFORMANCE] ⚠️  Moderate performance (%.1f FPS)", app.currentFPS)
	} else {
		log.Printf("[PERFORMANCE] ❌ Poor performance (%.1f FPS)", app.currentFPS)
	}
}

// performPeriodicCleanup performs periodic resource cleanup to prevent progressive slowdown
func (app *Application) performPeriodicCleanup() {
	log.Printf("[CLEANUP] Starting periodic resource cleanup (frame %d)", app.frameCount)
	
	// Reset accumulated performance data to prevent memory growth
	app.totalInputTime = 0
	app.totalEmulatorTime = 0
	app.totalRenderTime = 0
	
	// Reset min/max frame times for fresh measurements
	app.minFrameTime = time.Duration(16670000) // Reset to 16.67ms target
	app.maxFrameTime = time.Duration(16670000)
	
	// Clear frame consistency buffer
	for i := range app.recentFrameTimes {
		app.recentFrameTimes[i] = 0
	}
	app.frameTimeSum = 0
	app.frameTimeIndex = 0
	app.frameVariance = 0
	
	// Force garbage collection to reclaim memory
	runtime.GC()
	runtime.GC() // Run twice for better cleanup
	
	// Log memory status after cleanup
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	log.Printf("[CLEANUP] Memory after GC: %.2f MB | Heap objects: %d", 
		float64(memStats.Alloc)/(1024*1024), memStats.HeapObjects)
	
	log.Printf("[CLEANUP] Cleanup completed - performance data reset")
}

// Stop stops the application
func (app *Application) Stop() {
	app.running = false
}

// Pause pauses the emulator
func (app *Application) Pause() {
	app.paused = true
}

// Resume resumes the emulator
func (app *Application) Resume() {
	app.paused = false
}

// TogglePause toggles pause state
func (app *Application) TogglePause() {
	app.paused = !app.paused
}

// ShowMenu shows the menu
func (app *Application) ShowMenu() {
	app.showMenu = true
	app.paused = true
}

// HideMenu hides the menu
func (app *Application) HideMenu() {
	app.showMenu = false
	app.paused = false
}

// ToggleMenu toggles menu visibility
func (app *Application) ToggleMenu() {
	if app.showMenu {
		app.HideMenu()
	} else {
		app.ShowMenu()
	}
}

// SaveState saves the current emulator state
func (app *Application) SaveState(slot int) error {
	if app.cartridge == nil {
		return errors.New("no ROM loaded")
	}

	return app.states.SaveState(app.bus, slot, app.romPath)
}

// LoadState loads a saved emulator state
func (app *Application) LoadState(slot int) error {
	if app.cartridge == nil {
		return errors.New("no ROM loaded")
	}

	return app.states.LoadState(app.bus, slot, app.romPath)
}

// Reset resets the emulator
func (app *Application) Reset() {
	if app.bus != nil {
		app.bus.Reset()
	}
}

// IsRunning returns whether the application is running
func (app *Application) IsRunning() bool {
	return app.running
}

// IsPaused returns whether the emulator is paused
func (app *Application) IsPaused() bool {
	return app.paused
}

// IsMenuVisible returns whether the menu is visible
func (app *Application) IsMenuVisible() bool {
	return app.showMenu
}

// GetFPS returns the current FPS
func (app *Application) GetFPS() float64 {
	return app.currentFPS
}

// GetFrameCount returns the total frame count
func (app *Application) GetFrameCount() uint64 {
	return app.frameCount
}

// GetUptime returns the application uptime
func (app *Application) GetUptime() time.Duration {
	return time.Since(app.startTime)
}

// GetROMPath returns the currently loaded ROM path
func (app *Application) GetROMPath() string {
	return app.romPath
}

// GetConfig returns the application configuration
func (app *Application) GetConfig() *Config {
	return app.config
}

// ApplyDebugSettings applies debug settings to all components
func (app *Application) ApplyDebugSettings() {
	if app.config == nil {
		return
	}
	
	// Apply debug settings to PPU
	if app.bus != nil && app.bus.PPU != nil {
		app.bus.PPU.EnableBackgroundDebugLogging(app.config.Debug.EnableLogging)
		if app.config.Debug.EnableLogging {
			app.bus.PPU.SetBackgroundDebugVerbosity(2) // Medium verbosity
			fmt.Printf("[PPU_DEBUG] Debug logging enabled with verbosity 2\n")
		}
	}
	
	// Apply debug settings to input system
	if app.bus != nil {
		app.bus.EnableInputDebug(app.config.Debug.EnableLogging)
		if app.config.Debug.EnableLogging {
			fmt.Printf("[INPUT_DEBUG] Input debug logging enabled\n")
		}

		// Conditional debug categories with environment variables
		if app.config.Debug.EnableLogging && app.romPath != "" {
			// Memory monitoring (high performance impact)
			if os.Getenv("GONES_DEBUG_MEMORY") == "1" {
				app.bus.SetupSMBWatchpoints()
				app.bus.EnableWatchpointLogging(true)
				fmt.Printf("[DEBUG] Memory monitoring enabled (GONES_DEBUG_MEMORY=1)\n")
			}
			
			// Input debugging (medium performance impact)
			if os.Getenv("GONES_DEBUG_INPUT") == "1" {
				fmt.Printf("[DEBUG] Input debugging enabled (GONES_DEBUG_INPUT=1)\n")
			}
			
			// Rendering debugging (medium performance impact)  
			if os.Getenv("GONES_DEBUG_RENDER") == "1" {
				fmt.Printf("[DEBUG] Render debugging enabled (GONES_DEBUG_RENDER=1)\n")
			}
			
			// CPU debugging (very high performance impact - for debugging infinite loops)
			if os.Getenv("GONES_DEBUG_CPU") == "1" {
				app.bus.EnableCPUDebug(true)
				fmt.Printf("[DEBUG] CPU debug logging enabled (GONES_DEBUG_CPU=1)\n")
				fmt.Printf("[DEBUG] WARNING: CPU debugging has very high performance impact\n")
			}
			
			// Performance-optimized: all debug disabled by default
			if os.Getenv("GONES_DEBUG_MEMORY") != "1" && os.Getenv("GONES_DEBUG_INPUT") != "1" && os.Getenv("GONES_DEBUG_RENDER") != "1" && os.Getenv("GONES_DEBUG_CPU") != "1" {
				fmt.Printf("[DEBUG] All debug categories disabled for optimal performance\n")
				fmt.Printf("[DEBUG] Available categories: GONES_DEBUG_MEMORY, GONES_DEBUG_INPUT, GONES_DEBUG_RENDER, GONES_DEBUG_CPU\n")
			}
		}
	}
}


// Cleanup releases all resources and shuts down the application
func (app *Application) Cleanup() error {
	if app.config != nil && app.config.Debug.EnableLogging {
		fmt.Println("[APP_DEBUG] Cleaning up application resources...")
	}

	var lastErr error

	// Note: Audio cleanup will be handled by the graphics backend when audio is reimplemented

	// Clean up components
	if app.states != nil {
		if err := app.states.Cleanup(); err != nil {
			lastErr = err
			fmt.Printf("[APP_ERROR] State manager cleanup error: %v\n", err)
		}
	}

	// Note: UI cleanup removed with SDL2 dependency elimination

	if app.emulator != nil {
		if err := app.emulator.Cleanup(); err != nil {
			lastErr = err
			fmt.Printf("[APP_ERROR] Emulator cleanup error: %v\n", err)
		}
	}

	// Clean up graphics window
	if app.window != nil {
		if err := app.window.Cleanup(); err != nil {
			lastErr = err
			fmt.Printf("[APP_ERROR] Window cleanup error: %v\n", err)
		}
	}

	// Clean up graphics backend
	if app.graphicsBackend != nil {
		if err := app.graphicsBackend.Cleanup(); err != nil {
			lastErr = err
			fmt.Printf("[APP_ERROR] Graphics backend cleanup error: %v\n", err)
		}
	}

	// Note: Legacy SDL2 cleanup removed - using graphics backend cleanup only

	app.initialized = false
	if app.config != nil && app.config.Debug.EnableLogging {
		fmt.Println("[APP_DEBUG] Application cleanup complete")
	}

	return lastErr
}
