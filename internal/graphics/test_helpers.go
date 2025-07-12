//go:build !headless
// +build !headless

package graphics

// Test helper methods for accessing internal state during testing

// GetFrameBufferForTesting returns the internal frame buffer for testing purposes
func (w *EbitengineWindow) GetFrameBufferForTesting() [256 * 240]uint32 {
	if w.game == nil {
		return [256 * 240]uint32{}
	}
	return w.game.frameBuffer
}

// GetGameForTesting returns the internal game instance for testing purposes
func (w *EbitengineWindow) GetGameForTesting() *EbitengineGame {
	return w.game
}

// GetEmulatorUpdateFuncForTesting returns the emulator update function for testing
func (w *EbitengineWindow) GetEmulatorUpdateFuncForTesting() func() error {
	return w.emulatorUpdateFunc
}