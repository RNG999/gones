//go:build headless
// +build headless

package graphics

import "fmt"

// EbitengineBackend stub for headless builds
type EbitengineBackend struct{}

// EbitengineWindow stub for headless builds  
type EbitengineWindow struct{}

// NewEbitengineBackend creates a stub backend for headless builds
func NewEbitengineBackend() Backend {
	return &EbitengineBackend{}
}

// Stub implementations for EbitengineBackend
func (b *EbitengineBackend) Initialize(config Config) error {
	return fmt.Errorf("Ebitengine backend not available in headless build")
}

func (b *EbitengineBackend) CreateWindow(title string, width, height int) (Window, error) {
	return nil, fmt.Errorf("Ebitengine backend not available in headless build")
}

func (b *EbitengineBackend) Cleanup() error {
	return nil
}

func (b *EbitengineBackend) IsHeadless() bool {
	return true
}

func (b *EbitengineBackend) GetName() string {
	return "Ebitengine-Stub"
}

// Stub implementations for EbitengineWindow
func (w *EbitengineWindow) SetTitle(title string) {}
func (w *EbitengineWindow) GetSize() (width, height int) { return 0, 0 }
func (w *EbitengineWindow) ShouldClose() bool { return true }
func (w *EbitengineWindow) SwapBuffers() {}
func (w *EbitengineWindow) PollEvents() []InputEvent { return nil }
func (w *EbitengineWindow) RenderFrame(frameBuffer [256 * 240]uint32) error {
	return fmt.Errorf("Ebitengine backend not available in headless build")
}
func (w *EbitengineWindow) Cleanup() error { return nil }
func (w *EbitengineWindow) Run() error {
	return fmt.Errorf("Ebitengine backend not available in headless build")
}
func (w *EbitengineWindow) SetEmulatorUpdateFunc(updateFunc func() error) {}