// Package app provides save state functionality for the NES emulator.
package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gones/internal/bus"
)

// StateManager manages save states
type StateManager struct {
	saveDirectory string
	maxSlots      int
	initialized   bool
}

// SaveState represents a saved emulator state
type SaveState struct {
	// Metadata
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	ROMPath     string    `json:"rom_path"`
	ROMChecksum string    `json:"rom_checksum"`
	SlotNumber  int       `json:"slot_number"`
	Description string    `json:"description"`

	// Emulator state
	CPUState    CPUStateData `json:"cpu_state"`
	PPUState    PPUStateData `json:"ppu_state"`
	APUState    APUStateData `json:"apu_state"`
	MemoryState MemoryData   `json:"memory_state"`

	// Frame information
	FrameCount uint64 `json:"frame_count"`
	CycleCount uint64 `json:"cycle_count"`

	// Screenshot (base64 encoded)
	Screenshot string `json:"screenshot,omitempty"`
}

// CPUStateData represents CPU state for save files
type CPUStateData struct {
	PC     uint16       `json:"pc"`
	A      uint8        `json:"a"`
	X      uint8        `json:"x"`
	Y      uint8        `json:"y"`
	SP     uint8        `json:"sp"`
	Cycles uint64       `json:"cycles"`
	Flags  CPUFlagsData `json:"flags"`
}

// CPUFlagsData represents CPU flags for save files
type CPUFlagsData struct {
	N bool `json:"n"`
	V bool `json:"v"`
	B bool `json:"b"`
	D bool `json:"d"`
	I bool `json:"i"`
	Z bool `json:"z"`
	C bool `json:"c"`
}

// PPUStateData represents PPU state for save files
type PPUStateData struct {
	Scanline    int    `json:"scanline"`
	Cycle       int    `json:"cycle"`
	FrameCount  uint64 `json:"frame_count"`
	VBlankFlag  bool   `json:"vblank_flag"`
	RenderingOn bool   `json:"rendering_on"`
	NMIEnabled  bool   `json:"nmi_enabled"`
	// Additional PPU registers and state would go here
}

// APUStateData represents APU state for save files
type APUStateData struct {
	// Simplified APU state - in a full implementation,
	// this would include all channel states, registers, etc.
	Enabled    bool `json:"enabled"`
	SampleRate int  `json:"sample_rate"`
	// Channel states would go here
}

// MemoryData represents memory state for save files
type MemoryData struct {
	// This is a simplified representation - in a full implementation,
	// you would serialize all relevant memory regions
	RAMData  []uint8 `json:"ram_data"`
	VRAMData []uint8 `json:"vram_data"`
	OAMData  []uint8 `json:"oam_data"`
	// Mapper state would go here
}

// StateSlotInfo contains information about a save state slot
type StateSlotInfo struct {
	SlotNumber  int       `json:"slot_number"`
	Used        bool      `json:"used"`
	Timestamp   time.Time `json:"timestamp"`
	ROMPath     string    `json:"rom_path"`
	Description string    `json:"description"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
}

// NewStateManager creates a new state manager
func NewStateManager(saveDirectory string) *StateManager {
	manager := &StateManager{
		saveDirectory: saveDirectory,
		maxSlots:      10, // Default to 10 save slots
		initialized:   false,
	}

	if err := manager.initialize(); err != nil {
		// Log error but continue
		fmt.Printf("Warning: State manager initialization failed: %v\n", err)
	}

	return manager
}

// initialize initializes the state manager
func (sm *StateManager) initialize() error {
	// Create save directory if it doesn't exist
	if err := os.MkdirAll(sm.saveDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %v", err)
	}

	sm.initialized = true
	return nil
}

// SaveState saves the current emulator state to a slot
func (sm *StateManager) SaveState(bus *bus.Bus, slot int, romPath string) error {
	if !sm.initialized {
		return fmt.Errorf("state manager not initialized")
	}

	if slot < 0 || slot >= sm.maxSlots {
		return fmt.Errorf("invalid save slot: %d (must be 0-%d)", slot, sm.maxSlots-1)
	}

	if bus == nil {
		return fmt.Errorf("bus cannot be nil")
	}

	// Create save state
	saveState := &SaveState{
		Version:     "1.0",
		Timestamp:   time.Now(),
		ROMPath:     romPath,
		ROMChecksum: sm.calculateROMChecksum(romPath),
		SlotNumber:  slot,
		Description: fmt.Sprintf("Auto-save %s", time.Now().Format("2006-01-02 15:04:05")),
		FrameCount:  bus.GetFrameCount(),
		CycleCount:  bus.GetCycleCount(),
	}

	// Capture CPU state
	cpuState := bus.GetCPUState()
	saveState.CPUState = CPUStateData{
		PC:     cpuState.PC,
		A:      cpuState.A,
		X:      cpuState.X,
		Y:      cpuState.Y,
		SP:     cpuState.SP,
		Cycles: cpuState.Cycles,
		Flags: CPUFlagsData{
			N: cpuState.Flags.N,
			V: cpuState.Flags.V,
			B: cpuState.Flags.B,
			D: cpuState.Flags.D,
			I: cpuState.Flags.I,
			Z: cpuState.Flags.Z,
			C: cpuState.Flags.C,
		},
	}

	// Capture PPU state
	ppuState := bus.GetPPUState()
	saveState.PPUState = PPUStateData{
		Scanline:    ppuState.Scanline,
		Cycle:       ppuState.Cycle,
		FrameCount:  ppuState.FrameCount,
		VBlankFlag:  ppuState.VBlankFlag,
		RenderingOn: ppuState.RenderingOn,
		NMIEnabled:  ppuState.NMIEnabled,
	}

	// Simplified APU state
	saveState.APUState = APUStateData{
		Enabled:    true,  // Simplified
		SampleRate: 44100, // Would get from actual APU
	}

	// Simplified memory state - in a full implementation,
	// you would serialize all relevant memory regions
	saveState.MemoryState = MemoryData{
		RAMData:  make([]uint8, 2048), // NES has 2KB RAM
		VRAMData: make([]uint8, 2048), // 2KB VRAM
		OAMData:  make([]uint8, 256),  // 256 bytes OAM
	}

	// TODO: Actually read memory from bus
	// This is simplified - you would need methods to extract memory data

	// Generate file path
	filePath := sm.getSlotFilePath(slot, romPath)

	// Save to file
	if err := sm.saveToFile(saveState, filePath); err != nil {
		return fmt.Errorf("failed to save state: %v", err)
	}

	return nil
}

// LoadState loads a saved state from a slot
func (sm *StateManager) LoadState(bus *bus.Bus, slot int, romPath string) error {
	if !sm.initialized {
		return fmt.Errorf("state manager not initialized")
	}

	if slot < 0 || slot >= sm.maxSlots {
		return fmt.Errorf("invalid save slot: %d (must be 0-%d)", slot, sm.maxSlots-1)
	}

	if bus == nil {
		return fmt.Errorf("bus cannot be nil")
	}

	// Generate file path
	filePath := sm.getSlotFilePath(slot, romPath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("save state not found in slot %d", slot)
	}

	// Load from file
	saveState, err := sm.loadFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load state: %v", err)
	}

	// Validate save state
	if err := sm.validateSaveState(saveState, romPath); err != nil {
		return fmt.Errorf("invalid save state: %v", err)
	}

	// Restore state to bus
	if err := sm.restoreState(bus, saveState); err != nil {
		return fmt.Errorf("failed to restore state: %v", err)
	}

	return nil
}

// saveToFile saves a state to a file
func (sm *StateManager) saveToFile(state *SaveState, filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// loadFromFile loads a state from a file
func (sm *StateManager) loadFromFile(filePath string) (*SaveState, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Unmarshal JSON
	var state SaveState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %v", err)
	}

	return &state, nil
}

// validateSaveState validates a loaded save state
func (sm *StateManager) validateSaveState(state *SaveState, currentROMPath string) error {
	if state.Version == "" {
		return fmt.Errorf("missing version information")
	}

	// Check ROM compatibility (simplified)
	if state.ROMPath != currentROMPath {
		// In a more sophisticated implementation, you might allow loading
		// states from the same ROM with a different path
		return fmt.Errorf("save state is for a different ROM")
	}

	// Additional validation could include:
	// - Version compatibility checks
	// - Checksum verification
	// - State integrity checks

	return nil
}

// restoreState restores emulator state from a save state
func (sm *StateManager) restoreState(bus *bus.Bus, state *SaveState) error {
	// This is a simplified implementation - in a full implementation,
	// you would need methods to restore all emulator state

	// Reset the bus first
	bus.Reset()

	// TODO: Restore actual state
	// This would require methods to:
	// 1. Set CPU registers and state
	// 2. Restore PPU registers and VRAM
	// 3. Restore APU state
	// 4. Restore memory contents
	// 5. Restore mapper state

	fmt.Printf("State restore not fully implemented - would restore frame %d, cycle %d\n",
		state.FrameCount, state.CycleCount)

	return nil
}

// getSlotFilePath generates the file path for a save slot
func (sm *StateManager) getSlotFilePath(slot int, romPath string) string {
	romName := filepath.Base(romPath)
	romNameWithoutExt := romName[:len(romName)-len(filepath.Ext(romName))]
	fileName := fmt.Sprintf("%s_slot_%d.save", romNameWithoutExt, slot)
	return filepath.Join(sm.saveDirectory, fileName)
}

// calculateROMChecksum calculates a checksum for ROM verification
func (sm *StateManager) calculateROMChecksum(romPath string) string {
	// Simplified checksum - in a real implementation,
	// you would calculate MD5/SHA256 of the ROM file
	return fmt.Sprintf("checksum_%s", filepath.Base(romPath))
}

// GetSlotInfo returns information about all save slots
func (sm *StateManager) GetSlotInfo(romPath string) []StateSlotInfo {
	slots := make([]StateSlotInfo, sm.maxSlots)

	for i := 0; i < sm.maxSlots; i++ {
		slotInfo := StateSlotInfo{
			SlotNumber: i,
			Used:       false,
		}

		filePath := sm.getSlotFilePath(i, romPath)
		if stat, err := os.Stat(filePath); err == nil {
			// File exists
			slotInfo.Used = true
			slotInfo.FilePath = filePath
			slotInfo.FileSize = stat.Size()
			slotInfo.Timestamp = stat.ModTime()

			// Try to load basic info from the save state
			if state, err := sm.loadFromFile(filePath); err == nil {
				slotInfo.ROMPath = state.ROMPath
				slotInfo.Description = state.Description
				slotInfo.Timestamp = state.Timestamp
			}
		}

		slots[i] = slotInfo
	}

	return slots
}

// DeleteState deletes a save state from a slot
func (sm *StateManager) DeleteState(slot int, romPath string) error {
	if !sm.initialized {
		return fmt.Errorf("state manager not initialized")
	}

	if slot < 0 || slot >= sm.maxSlots {
		return fmt.Errorf("invalid save slot: %d", slot)
	}

	filePath := sm.getSlotFilePath(slot, romPath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("save state not found in slot %d", slot)
	}

	// Delete file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete save state: %v", err)
	}

	return nil
}

// HasSaveState checks if a save state exists in a slot
func (sm *StateManager) HasSaveState(slot int, romPath string) bool {
	if slot < 0 || slot >= sm.maxSlots {
		return false
	}

	filePath := sm.getSlotFilePath(slot, romPath)
	_, err := os.Stat(filePath)
	return err == nil
}

// GetMaxSlots returns the maximum number of save slots
func (sm *StateManager) GetMaxSlots() int {
	return sm.maxSlots
}

// SetMaxSlots sets the maximum number of save slots
func (sm *StateManager) SetMaxSlots(slots int) {
	if slots > 0 {
		sm.maxSlots = slots
	}
}

// GetSaveDirectory returns the save directory path
func (sm *StateManager) GetSaveDirectory() string {
	return sm.saveDirectory
}

// SetSaveDirectory sets the save directory path
func (sm *StateManager) SetSaveDirectory(directory string) error {
	sm.saveDirectory = directory
	return sm.initialize()
}

// ExportState exports a save state to a specific file
func (sm *StateManager) ExportState(bus *bus.Bus, filePath string, romPath string) error {
	// Create temporary save state
	saveState := &SaveState{
		Version:     "1.0",
		Timestamp:   time.Now(),
		ROMPath:     romPath,
		ROMChecksum: sm.calculateROMChecksum(romPath),
		SlotNumber:  -1, // Export doesn't use slots
		Description: fmt.Sprintf("Export %s", time.Now().Format("2006-01-02 15:04:05")),
		FrameCount:  bus.GetFrameCount(),
		CycleCount:  bus.GetCycleCount(),
	}

	// Fill in state data (simplified)
	cpuState := bus.GetCPUState()
	saveState.CPUState = CPUStateData{
		PC:     cpuState.PC,
		A:      cpuState.A,
		X:      cpuState.X,
		Y:      cpuState.Y,
		SP:     cpuState.SP,
		Cycles: cpuState.Cycles,
		Flags: CPUFlagsData{
			N: cpuState.Flags.N,
			V: cpuState.Flags.V,
			B: cpuState.Flags.B,
			D: cpuState.Flags.D,
			I: cpuState.Flags.I,
			Z: cpuState.Flags.Z,
			C: cpuState.Flags.C,
		},
	}

	// Save to specified file
	return sm.saveToFile(saveState, filePath)
}

// ImportState imports a save state from a specific file
func (sm *StateManager) ImportState(bus *bus.Bus, filePath string, romPath string) error {
	// Load from file
	saveState, err := sm.loadFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to import state: %v", err)
	}

	// Validate and restore
	if err := sm.validateSaveState(saveState, romPath); err != nil {
		return fmt.Errorf("invalid imported state: %v", err)
	}

	return sm.restoreState(bus, saveState)
}

// Cleanup cleans up state manager resources
func (sm *StateManager) Cleanup() error {
	sm.initialized = false
	return nil
}

// GetStateManagerStats returns statistics about the state manager
func (sm *StateManager) GetStateManagerStats(romPath string) StateManagerStats {
	slots := sm.GetSlotInfo(romPath)

	var usedSlots int
	var totalSize int64
	for _, slot := range slots {
		if slot.Used {
			usedSlots++
			totalSize += slot.FileSize
		}
	}

	return StateManagerStats{
		MaxSlots:      sm.maxSlots,
		UsedSlots:     usedSlots,
		FreeSlots:     sm.maxSlots - usedSlots,
		TotalSize:     totalSize,
		SaveDirectory: sm.saveDirectory,
		Initialized:   sm.initialized,
	}
}

// StateManagerStats contains state manager statistics
type StateManagerStats struct {
	MaxSlots      int    `json:"max_slots"`
	UsedSlots     int    `json:"used_slots"`
	FreeSlots     int    `json:"free_slots"`
	TotalSize     int64  `json:"total_size"`
	SaveDirectory string `json:"save_directory"`
	Initialized   bool   `json:"initialized"`
}
