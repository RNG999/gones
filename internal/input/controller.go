// Package input implements controller handling for the NES.
package input

import (
	"log"
)

// Button represents NES controller buttons
type Button uint8

const (
	ButtonA Button = 1 << iota
	ButtonB
	ButtonSelect
	ButtonStart
	ButtonUp
	ButtonDown
	ButtonLeft
	ButtonRight
)

// Convenience constants for shorter names used in SDL integration
const (
	A      = ButtonA
	B      = ButtonB
	Select = ButtonSelect
	Start  = ButtonStart
	Up     = ButtonUp
	Down   = ButtonDown
	Left   = ButtonLeft
	Right  = ButtonRight
)

// Controller represents a NES controller
type Controller struct {
	// Current button states (8 buttons: A, B, Select, Start, Up, Down, Left, Right)
	buttons uint8

	// Shift register for serial reading
	shiftRegister uint8
	strobe        bool

	// Snapshot of button states when strobe was activated
	buttonSnapshot uint8
	
	// Bit position tracking for proper NES controller protocol
	bitPosition uint8  // Tracks which bit we're reading (0-7 for buttons, 8+ for extended reads)
	
	// Debug tracking
	readCount    uint64
	writeCount   uint64
	debugEnabled bool
}

// New creates a new Controller instance
func New() *Controller {
	return &Controller{}
}

// SetButton sets the state of a button (simplified approach like other NES emulators)
func (c *Controller) SetButton(button Button, pressed bool) {
	oldButtons := c.buttons
	
	if pressed {
		c.buttons |= uint8(button)
	} else {
		c.buttons &^= uint8(button)
	}
	
	// Debug log for button state changes
	if c.debugEnabled {
		log.Printf("[BUTTON_DEBUG] SetButton: button=%d, pressed=%t, oldButtons=0x%02X, newButtons=0x%02X", 
			uint8(button), pressed, oldButtons, c.buttons)
	}
}

// SetButtons sets all button states at once (array approach like ChibiNES/Fogleman NES)
func (c *Controller) SetButtons(buttons [8]bool) {
	oldButtons := c.buttons
	
	// Convert boolean array to bit pattern for input state
	// NES button order: A, B, Select, Start, Up, Down, Left, Right
	c.buttons = 0
	if buttons[0] { c.buttons |= uint8(ButtonA) }
	if buttons[1] { c.buttons |= uint8(ButtonB) }
	if buttons[2] { c.buttons |= uint8(ButtonSelect) }
	if buttons[3] { c.buttons |= uint8(ButtonStart) }
	if buttons[4] { c.buttons |= uint8(ButtonUp) }
	if buttons[5] { c.buttons |= uint8(ButtonDown) }
	if buttons[6] { c.buttons |= uint8(ButtonLeft) }
	if buttons[7] { c.buttons |= uint8(ButtonRight) }
	
	// Debug log for button state changes
	if c.debugEnabled {
		log.Printf("[BUTTON_DEBUG] SetButtons: [A:%t B:%t Sel:%t Start:%t U:%t D:%t L:%t R:%t] oldButtons=0x%02X, newButtons=0x%02X", 
			buttons[0], buttons[1], buttons[2], buttons[3], buttons[4], buttons[5], buttons[6], buttons[7],
			oldButtons, c.buttons)
	}
}

// IsPressed returns true if the button is currently pressed
func (c *Controller) IsPressed(button Button) bool {
	return (c.buttons & uint8(button)) != 0
}

// Write handles writes to the controller register ($4016)
func (c *Controller) Write(value uint8) {
	c.writeCount++
	wasStrobe := c.strobe
	c.strobe = (value & 1) != 0

	if c.strobe {
		// Strobe is active - capture current button state immediately
		c.buttonSnapshot = c.buttons
		c.shiftRegister = c.buttons // Set shift register immediately for compatibility
		c.bitPosition = 0           // Reset bit position for new read sequence
		if c.debugEnabled {
			log.Printf("[CONTROLLER_DEBUG] Strobe activated: buttons=0x%02X, snapshot=0x%02X, bitPos=0", 
				c.buttons, c.buttonSnapshot)
		}
	} else if wasStrobe {
		// Strobe was just deactivated - capture current button state and load into shift register
		c.buttonSnapshot = c.buttons  // Update snapshot with current button state
		c.shiftRegister = c.buttonSnapshot
		c.bitPosition = 0 // Reset bit position for new read sequence
		if c.debugEnabled {
			log.Printf("[CONTROLLER_DEBUG] Strobe deactivated: captured buttons=0x%02X, snapshot=0x%02X, shiftRegister=0x%02X, bitPos=0", 
				c.buttons, c.buttonSnapshot, c.shiftRegister)
		}
	}
}

// Read handles reads from the controller register ($4016/$4017)
func (c *Controller) Read() uint8 {
	c.readCount++
	
	if c.strobe {
		// When strobe is active, always return button A state and reset to position 0
		// This matches rgnes/fogleman behavior: reset index during read if strobe is high
		c.bitPosition = 0
		buttonBit := uint8(c.buttonSnapshot & 1)
		result := buttonBit  // Only bit 0 contains button data
		if c.debugEnabled && c.readCount%10 == 0 {
			log.Printf("[CONTROLLER_DEBUG] Read during strobe: result=0x%02X (bits 0,1=%d), buttonSnapshot=0x%02X, bitPos reset to 0", 
				result, buttonBit, c.buttonSnapshot)
		}
		return result
	}

	var result uint8
	
	if c.bitPosition < 8 {
		// Reading bits 0-7: Normal button sequence
		buttonBit := uint8(c.shiftRegister & 1)
		result = buttonBit  // Only bit 0 contains button data
		c.shiftRegister >>= 1
		c.bitPosition++
		
		if c.debugEnabled && c.readCount%10 == 0 {
			log.Printf("[CONTROLLER_DEBUG] Read bit %d: result=0x%02X (bits 0,1=%d), shiftRegister=0x%02X", 
				c.bitPosition-1, result, buttonBit, c.shiftRegister)
		}
	} else {
		// Reading bit 8+: Return 0 (matches rgnes/fogleman NES behavior)
		result = 0
		
		if c.debugEnabled && c.readCount%10 == 0 {
			log.Printf("[CONTROLLER_DEBUG] Extended read (bit %d): result=0x%02X", 
				c.bitPosition, result)
		}
		c.bitPosition++ // Continue incrementing for debug purposes
	}
	
	return result
}

// Reset resets the controller state
func (c *Controller) Reset() {
	c.buttons = 0
	c.shiftRegister = 0
	c.strobe = false
	c.buttonSnapshot = 0
	c.bitPosition = 0
	c.readCount = 0
	c.writeCount = 0
}

// EnableDebug enables debug logging for this controller
func (c *Controller) EnableDebug(enable bool) {
	c.debugEnabled = enable
}

// GetBitPosition returns the current bit position (for testing)
func (c *Controller) GetBitPosition() uint8 {
	return c.bitPosition
}


// InputState represents the state of all input devices
type InputState struct {
	Controller1 *Controller
	Controller2 *Controller
}

// NewInputState creates a new input state with two controllers
func NewInputState() *InputState {
	return &InputState{
		Controller1: New(),
		Controller2: New(),
	}
}

// Reset resets all input devices
func (is *InputState) Reset() {
	is.Controller1.Reset()
	is.Controller2.Reset()
}

// EnableDebug enables debug logging for all controllers
func (is *InputState) EnableDebug(enable bool) {
	is.Controller1.EnableDebug(enable)
	is.Controller2.EnableDebug(enable)
}

// SetButtons1 sets all button states for controller 1 (array approach)
func (is *InputState) SetButtons1(buttons [8]bool) {
	is.Controller1.SetButtons(buttons)
}

// SetButtons2 sets all button states for controller 2 (array approach)
func (is *InputState) SetButtons2(buttons [8]bool) {
	is.Controller2.SetButtons(buttons)
}


// Read reads from controller ports
func (is *InputState) Read(address uint16) uint8 {
	switch address {
	case 0x4016:
		result := is.Controller1.Read()
		if is.Controller1.debugEnabled {
			log.Printf("[INPUT_TRACE] $4016 read: result=0x%02X, readCount=%d", result, is.Controller1.readCount)
		}
		return result
	case 0x4017:
		// Controller 2 - Independent controller with its own bitPosition tracking
		// Critical for SMB title screen - Controller 2 must be completely independent
		result := is.Controller2.Read()
		
		// Controller 2 returns bit 6 set (0x40) as per NES hardware behavior
		// This is due to open bus behavior on the NES
		result |= 0x40
		
		if is.Controller2.debugEnabled {
			log.Printf("[INPUT_TRACE] $4017 read: result=0x%02X, buttons=0x%02X, bitPos=%d", 
				result, is.Controller2.buttons, is.Controller2.bitPosition)
		}
		return result
	default:
		return 0
	}
}

// Write writes to controller ports
func (is *InputState) Write(address uint16, value uint8) {
	if address == 0x4016 {
		if is.Controller1.debugEnabled {
			log.Printf("[INPUT_TRACE] $4016 write: value=0x%02X, strobe=%t, writeCount=%d", 
				value, (value&1) != 0, is.Controller1.writeCount+1)
		}
		// Both controllers receive strobe signals
		is.Controller1.Write(value)
		is.Controller2.Write(value)
	}
}
