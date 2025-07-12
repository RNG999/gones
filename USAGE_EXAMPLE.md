# gones - Complete GUI Application Usage Example

## Overview

The gones NES emulator now includes a complete GUI application with:

- **Main Application Loop**: 60 FPS emulation with SDL2 integration
- **User Interface**: Menu system, pause/resume, configuration
- **Save States**: Save and load emulator states
- **Configuration Management**: JSON-based configuration with defaults
- **Audio/Video/Input Integration**: Complete SDL2 subsystem coordination

## Application Architecture

```
cmd/gones/main.go           # Main executable with CLI and GUI modes
internal/app/
├── app.go                  # Main application lifecycle and integration
├── emulator.go             # Emulation loop with timing control
├── ui.go                   # User interface and menu system
├── config.go               # Configuration management
└── states.go               # Save state functionality
```

## Building and Running

```bash
# Build the complete application
go build -o gones ./cmd/gones

# Run GUI mode (load ROM from menu)
./gones

# Run with ROM pre-loaded
./gones -rom game.nes

# Run with debug information
./gones -rom game.nes -debug

# Run with custom configuration
./gones -config custom.json

# Show help
./gones -help

# Show version information  
./gones -version
```

## Key Features Implemented

### 1. Complete Application Integration

The `Application` struct in `app.go` integrates all components:

```go
type Application struct {
    // Core emulation components
    bus *bus.Bus
    
    // SDL2 subsystems
    sdlManager    *sdl.Manager
    window        *sdl.Window
    renderer      *sdl.Renderer
    audioDevice   *sdl.AudioDevice
    inputManager  *sdl.InputManager
    
    // Application subsystems
    config        *Config
    ui            *UI
    emulator      *Emulator
    states        *StateManager
}
```

### 2. 60 FPS Emulation Loop

The `Emulator` struct provides precise timing:

```go
// Target 60 FPS with cycle-accurate emulation
targetFrameTime: time.Duration(1000000/60) * time.Microsecond
cyclesPerFrame:  29781 // NTSC: ~29,781 CPU cycles per frame

// Main emulation loop with timing control
func (e *Emulator) Update() error {
    // Accumulate time for smooth frame timing
    e.accumulatedTime += deltaTime
    
    if e.accumulatedTime >= e.targetFrameTime {
        e.runFrame() // Execute one frame of emulation
    }
}
```

### 3. Complete SDL2 Integration

- **Graphics**: NES frame buffer → SDL2 renderer → window display
- **Audio**: APU samples → SDL2 audio callback → audio device  
- **Input**: SDL2 events → NES controller interface → emulator

### 4. Configuration System

JSON-based configuration with validation:

```json
{
  "window": {
    "width": 800,
    "height": 600,
    "scale": 2,
    "resizable": true
  },
  "video": {
    "vsync": true,
    "filter": "nearest",
    "aspect_ratio": "4:3"
  },
  "audio": {
    "enabled": true,
    "sample_rate": 44100,
    "volume": 0.8
  },
  "emulation": {
    "region": "NTSC",
    "frame_rate": 60.0,
    "cycle_accuracy": true
  }
}
```

### 5. Save State System

Complete state serialization with metadata:

```go
type SaveState struct {
    Version      string       `json:"version"`
    Timestamp    time.Time    `json:"timestamp"`
    ROMPath      string       `json:"rom_path"`
    CPUState     CPUStateData `json:"cpu_state"`
    PPUState     PPUStateData `json:"ppu_state"`
    MemoryState  MemoryData   `json:"memory_state"`
    FrameCount   uint64       `json:"frame_count"`
}
```

### 6. User Interface Framework

Menu system with navigation and actions:

```go
// Menu actions
MenuActionLoadROM
MenuActionSaveState  
MenuActionLoadState
MenuActionReset
MenuActionSettings
MenuActionExit

// Input handling
func (ui *UI) HandleMenuInput(action MenuInputAction) MenuAction
```

## Main Application Flow

```
1. Parse command line arguments
2. Create Application with configuration
3. Initialize SDL2 subsystems (video, audio, input)
4. Load ROM (if specified)
5. Start main application loop:
   - Process input events
   - Update emulator (60 FPS)
   - Render frame
   - Handle UI (menus, overlays)
   - Maintain target frame rate
6. Cleanup resources on shutdown
```

## Integration Points

### Bus ↔ Emulator
- `bus.Step()` for CPU instruction execution  
- `bus.GetFrameBuffer()` for graphics data
- `bus.GetAudioSamples()` for sound data
- `bus.SetControllerButton()` for input

### Emulator ↔ Application  
- `emulator.Update()` called each frame
- Frame completion signaling
- Performance metrics tracking

### SDL2 ↔ Application
- Event polling and input mapping
- Frame buffer rendering
- Audio callback integration
- Window lifecycle management

## Error Handling

Comprehensive error handling with graceful degradation:

```go
type ApplicationError struct {
    Component string
    Operation string
    Err       error
}

// Audio failure doesn't stop emulation
if err := app.initializeAudio(); err != nil {
    log.Printf("Warning: Audio initialization failed: %v", err)
}
```

## Performance Features

- **Frame Rate Control**: Precise 60 FPS timing with vsync
- **Performance Monitoring**: FPS counter, timing metrics
- **Resource Management**: Proper cleanup and memory management
- **Graceful Shutdown**: Signal handling for clean exit

## Usage Examples

### Basic Emulation
```bash
# Start GUI, load ROM from menu
./gones

# Quick start with ROM
./gones -rom mario.nes
```

### Development/Testing
```bash
# Debug mode with performance info
./gones -rom test.nes -debug

# Custom configuration for testing
./gones -config test-config.json -rom test.nes
```

### Headless Mode
```bash
# Automated testing (when implemented)
./gones -nogui -rom test.nes
```

This complete implementation provides a professional-grade NES emulator with modern application architecture, comprehensive error handling, and extensible design for future enhancements.