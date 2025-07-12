// Package app provides configuration management for the NES emulator.
package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all application configuration
type Config struct {
	Window    WindowConfig    `json:"window"`
	Video     VideoConfig     `json:"video"`
	Audio     AudioConfig     `json:"audio"`
	Input     InputConfig     `json:"input"`
	Emulation EmulationConfig `json:"emulation"`
	Debug     DebugConfig     `json:"debug"`
	Paths     PathsConfig     `json:"paths"`

	// Internal state
	configPath string
	loaded     bool
}

// WindowConfig contains window-related configuration
type WindowConfig struct {
	Width      int  `json:"width"`
	Height     int  `json:"height"`
	Fullscreen bool `json:"fullscreen"`
	Resizable  bool `json:"resizable"`
	Centered   bool `json:"centered"`
	Scale      int  `json:"scale"` // NES resolution multiplier
}

// VideoConfig contains video rendering configuration
type VideoConfig struct {
	VSync        bool    `json:"vsync"`
	FrameSkip    int     `json:"frame_skip"`
	AspectRatio  string  `json:"aspect_ratio"` // "4:3", "16:9", "original"
	Filter       string  `json:"filter"`       // "nearest", "linear", "cubic"
	Backend      string  `json:"backend"`      // "ebitengine", "sdl2", "headless", "terminal"
	Brightness   float32 `json:"brightness"`
	Contrast     float32 `json:"contrast"`
	Saturation   float32 `json:"saturation"`
	ShowOverscan bool    `json:"show_overscan"`
	CropOverscan bool    `json:"crop_overscan"`
}

// AudioConfig contains audio configuration
type AudioConfig struct {
	Enabled    bool    `json:"enabled"`
	SampleRate int     `json:"sample_rate"`
	BufferSize int     `json:"buffer_size"`
	Volume     float32 `json:"volume"`
	Channels   int     `json:"channels"`
	Latency    int     `json:"latency"` // Target latency in milliseconds
}

// InputConfig contains input configuration
type InputConfig struct {
	Player1Keys        KeyMapping `json:"player1_keys"`
	Player2Keys        KeyMapping `json:"player2_keys"`
	ControllerDeadzone float32    `json:"controller_deadzone"`
	AutofireRate       int        `json:"autofire_rate"`
	EnableAutofire     bool       `json:"enable_autofire"`
}

// KeyMapping represents keyboard key mappings for NES controller
type KeyMapping struct {
	Up     string `json:"up"`
	Down   string `json:"down"`
	Left   string `json:"left"`
	Right  string `json:"right"`
	A      string `json:"a"`
	B      string `json:"b"`
	Start  string `json:"start"`
	Select string `json:"select"`
}

// EmulationConfig contains emulation-specific settings
type EmulationConfig struct {
	Region           string  `json:"region"`         // "NTSC", "PAL", "Dendy"
	FrameRate        float64 `json:"frame_rate"`     // Target frame rate
	CycleAccuracy    bool    `json:"cycle_accuracy"` // Cycle-accurate emulation
	EnableSound      bool    `json:"enable_sound"`
	RewindBuffer     int     `json:"rewind_buffer"`    // Rewind buffer size in seconds
	SaveStateSlots   int     `json:"save_state_slots"` // Number of save state slots
	AutoSave         bool    `json:"auto_save"`        // Auto-save state on exit
	PauseOnFocusLoss bool    `json:"pause_on_focus_loss"`
}

// DebugConfig contains debugging and development options
type DebugConfig struct {
	ShowFPS         bool   `json:"show_fps"`
	ShowDebugInfo   bool   `json:"show_debug_info"`
	EnableLogging   bool   `json:"enable_logging"`
	LogLevel        string `json:"log_level"` // "DEBUG", "INFO", "WARN", "ERROR"
	CPUTracing      bool   `json:"cpu_tracing"`
	PPUDebugging    bool   `json:"ppu_debugging"`
	MemoryDebugging bool   `json:"memory_debugging"`
}

// PathsConfig contains file and directory paths
type PathsConfig struct {
	ROMs        string `json:"roms"`
	SaveData    string `json:"save_data"`
	SaveStates  string `json:"save_states"`
	Screenshots string `json:"screenshots"`
	Config      string `json:"config"`
	Logs        string `json:"logs"`
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	config := &Config{
		Window: WindowConfig{
			Width:      800,
			Height:     600,
			Fullscreen: false,
			Resizable:  true,
			Centered:   true,
			Scale:      2, // 512x480 (256x240 * 2)
		},
		Video: VideoConfig{
			VSync:        true,
			FrameSkip:    0,
			AspectRatio:  "4:3",
			Filter:       "nearest",
			Backend:      "ebitengine", // Default to Ebitengine for GUI mode
			Brightness:   1.0,
			Contrast:     1.0,
			Saturation:   1.0,
			ShowOverscan: false,
			CropOverscan: true,
		},
		Audio: AudioConfig{
			Enabled:    true,
			SampleRate: 44100,
			BufferSize: 1024,
			Volume:     0.8,
			Channels:   2,
			Latency:    50,
		},
		Input: InputConfig{
			Player1Keys: KeyMapping{
				Up:     "W",
				Down:   "S",
				Left:   "A",
				Right:  "D",
				A:      "J",
				B:      "K",
				Start:  "Return",
				Select: "Space",
			},
			Player2Keys: KeyMapping{
				Up:     "Up",
				Down:   "Down",
				Left:   "Left",
				Right:  "Right",
				A:      "N",
				B:      "M",
				Start:  "RShift",
				Select: "RCtrl",
			},
			ControllerDeadzone: 0.1,
			AutofireRate:       10,
			EnableAutofire:     false,
		},
		Emulation: EmulationConfig{
			Region:           "NTSC",
			FrameRate:        60.0,
			CycleAccuracy:    true,
			EnableSound:      true,
			RewindBuffer:     30,
			SaveStateSlots:   10,
			AutoSave:         true,
			PauseOnFocusLoss: true,
		},
		Debug: DebugConfig{
			ShowFPS:         false,
			ShowDebugInfo:   false,
			EnableLogging:   false,
			LogLevel:        "INFO",
			CPUTracing:      false,
			PPUDebugging:    false,
			MemoryDebugging: false,
		},
		Paths: PathsConfig{
			ROMs:        "./roms",
			SaveData:    "./saves",
			SaveStates:  "./states",
			Screenshots: "./screenshots",
			Config:      "./config",
			Logs:        "./logs",
		},
		loaded: false,
	}

	return config
}

// LoadFromFile loads configuration from a JSON file
func (c *Config) LoadFromFile(path string) error {
	c.configPath = path

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist - save default config and return
		return c.SaveToFile(path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate configuration
	if err := c.validate(); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	// Ensure required directories exist
	if err := c.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	c.loaded = true
	return nil
}

// SaveToFile saves configuration to a JSON file
func (c *Config) SaveToFile(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	c.configPath = path
	return nil
}

// Save saves the configuration to the current config file
func (c *Config) Save() error {
	if c.configPath == "" {
		return fmt.Errorf("no config file path set")
	}

	return c.SaveToFile(c.configPath)
}

// validate validates the configuration values
func (c *Config) validate() error {
	// Validate window configuration
	if c.Window.Width <= 0 || c.Window.Height <= 0 {
		return fmt.Errorf("invalid window dimensions: %dx%d", c.Window.Width, c.Window.Height)
	}

	if c.Window.Scale <= 0 {
		c.Window.Scale = 1
	}

	// Validate video configuration
	if c.Video.Brightness < 0.1 || c.Video.Brightness > 3.0 {
		c.Video.Brightness = 1.0
	}

	if c.Video.Contrast < 0.1 || c.Video.Contrast > 3.0 {
		c.Video.Contrast = 1.0
	}

	if c.Video.Saturation < 0.0 || c.Video.Saturation > 3.0 {
		c.Video.Saturation = 1.0
	}

	// Validate audio configuration
	if c.Audio.SampleRate <= 0 {
		c.Audio.SampleRate = 44100
	}

	if c.Audio.BufferSize <= 0 {
		c.Audio.BufferSize = 1024
	}

	if c.Audio.Volume < 0.0 || c.Audio.Volume > 1.0 {
		c.Audio.Volume = 0.8
	}

	if c.Audio.Channels != 1 && c.Audio.Channels != 2 {
		c.Audio.Channels = 2
	}

	// Validate emulation configuration
	if c.Emulation.FrameRate <= 0 {
		c.Emulation.FrameRate = 60.0
	}

	if c.Emulation.RewindBuffer < 0 {
		c.Emulation.RewindBuffer = 0
	}

	if c.Emulation.SaveStateSlots <= 0 {
		c.Emulation.SaveStateSlots = 10
	}

	// Validate input configuration
	if c.Input.ControllerDeadzone < 0.0 || c.Input.ControllerDeadzone > 1.0 {
		c.Input.ControllerDeadzone = 0.1
	}

	if c.Input.AutofireRate <= 0 {
		c.Input.AutofireRate = 10
	}

	return nil
}

// createDirectories creates required directories
func (c *Config) createDirectories() error {
	dirs := []string{
		c.Paths.ROMs,
		c.Paths.SaveData,
		c.Paths.SaveStates,
		c.Paths.Screenshots,
		c.Paths.Config,
		c.Paths.Logs,
	}

	for _, dir := range dirs {
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", dir, err)
			}
		}
	}

	return nil
}

// GetNESResolution returns the native NES resolution
func (c *Config) GetNESResolution() (int, int) {
	return 256, 240
}

// GetWindowResolution returns the window resolution based on scale
func (c *Config) GetWindowResolution() (int, int) {
	nesWidth, nesHeight := c.GetNESResolution()
	return nesWidth * c.Window.Scale, nesHeight * c.Window.Scale
}

// GetAspectRatio returns the aspect ratio as a float
func (c *Config) GetAspectRatio() float32 {
	switch c.Video.AspectRatio {
	case "4:3":
		return 4.0 / 3.0
	case "16:9":
		return 16.0 / 9.0
	case "original":
		nesWidth, nesHeight := c.GetNESResolution()
		return float32(nesWidth) / float32(nesHeight)
	default:
		return 4.0 / 3.0 // Default to 4:3
	}
}

// IsLoaded returns whether the configuration was loaded from file
func (c *Config) IsLoaded() bool {
	return c.loaded
}

// GetConfigPath returns the path to the config file
func (c *Config) GetConfigPath() string {
	return c.configPath
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	// Marshal to JSON and back to create deep copy
	data, err := json.Marshal(c)
	if err != nil {
		return NewConfig() // Return default config on error
	}

	clone := &Config{}
	if err := json.Unmarshal(data, clone); err != nil {
		return NewConfig() // Return default config on error
	}

	// Copy non-serialized fields
	clone.configPath = c.configPath
	clone.loaded = c.loaded

	return clone
}

// UpdateWindow updates window configuration
func (c *Config) UpdateWindow(width, height int, fullscreen bool) {
	c.Window.Width = width
	c.Window.Height = height
	c.Window.Fullscreen = fullscreen
}

// UpdateVideo updates video configuration
func (c *Config) UpdateVideo(vsync bool, filter string, brightness, contrast, saturation float32) {
	c.Video.VSync = vsync
	c.Video.Filter = filter
	c.Video.Brightness = brightness
	c.Video.Contrast = contrast
	c.Video.Saturation = saturation
}

// UpdateAudio updates audio configuration
func (c *Config) UpdateAudio(enabled bool, volume float32, sampleRate int) {
	c.Audio.Enabled = enabled
	c.Audio.Volume = volume
	c.Audio.SampleRate = sampleRate
}

// UpdateEmulation updates emulation configuration
func (c *Config) UpdateEmulation(region string, frameRate float64, cycleAccuracy bool) {
	c.Emulation.Region = region
	c.Emulation.FrameRate = frameRate
	c.Emulation.CycleAccuracy = cycleAccuracy
}

// UpdateDebug updates debug configuration
func (c *Config) UpdateDebug(showFPS, showDebugInfo, enableLogging bool) {
	c.Debug.ShowFPS = showFPS
	c.Debug.ShowDebugInfo = showDebugInfo
	c.Debug.EnableLogging = enableLogging
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	return "./config/gones.json"
}

// GetDefaultConfigDir returns the default configuration directory
func GetDefaultConfigDir() string {
	return "./config"
}

// ConfigError represents configuration-related errors
type ConfigError struct {
	Field string
	Value interface{}
	Err   error
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error in field '%s' with value '%v': %v", e.Field, e.Value, e.Err)
}
