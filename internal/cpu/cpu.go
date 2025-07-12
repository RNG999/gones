// Package cpu implements the 6502 CPU emulation for the NES.
package cpu

import "fmt"

// Addressing modes
type AddressingMode int

const (
	Implied AddressingMode = iota
	Accumulator
	Immediate
	ZeroPage
	ZeroPageX
	ZeroPageY
	Relative
	Absolute
	AbsoluteX
	AbsoluteY
	Indirect
	IndexedIndirect // (zp,X)
	IndirectIndexed // (zp),Y
)

// CPU constants for performance optimization
const (
	// Stack base address
	stackBase = 0x0100
	// Status register bit masks
	nFlagMask  = 0x80
	vFlagMask  = 0x40
	unusedMask = 0x20
	bFlagMask  = 0x10
	dFlagMask  = 0x08
	iFlagMask  = 0x04
	zFlagMask  = 0x02
	cFlagMask  = 0x01
	// Zero page mask
	zeroPageMask = 0xFF
	// Page boundary mask
	pageMask = 0xFF00
	// Interrupt vectors
	nmiVector   = 0xFFFA
	irqVector   = 0xFFFE
	resetVector = 0xFFFC
)

// Instruction represents a 6502 instruction
type Instruction struct {
	Name   string
	Opcode uint8
	Bytes  uint8
	Cycles uint8
	Mode   AddressingMode
	// No function pointer needed - we'll use opcode switch
}

// CPU represents the 6502 processor used in the NES
type CPU struct {
	// Registers
	A  uint8  // Accumulator
	X  uint8  // X register
	Y  uint8  // Y register
	SP uint8  // Stack pointer
	PC uint16 // Program counter

	// Status register flags
	C bool // Carry
	Z bool // Zero
	I bool // Interrupt disable
	D bool // Decimal mode (not used in NES)
	B bool // Break
	V bool // Overflow
	N bool // Negative

	// Memory interface (to be implemented)
	memory MemoryInterface

	// Cycle counter
	cycles uint64

	// Instruction lookup table
	instructions [256]*Instruction

	// Interrupt flags
	nmiPending bool
	irqPending bool
	
	// NMI edge detection - track previous NMI state for edge detection
	nmiPrevious bool
	
	// Interrupt delay - interrupts are checked after instruction completion
	interruptDelay bool
	
	// Debug and loop detection fields
	enableDebugLogging  bool
	enableLoopDetection bool
	lastPC              uint16
	pcStayCount         int
}

// MemoryInterface defines the interface for CPU memory access
type MemoryInterface interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)
}

// New creates a new CPU instance
func New(memory MemoryInterface) *CPU {
	cpu := &CPU{
		memory: memory,
		SP:     0xFD, // Stack pointer initial value
		PC:     0,    // Will be set from reset vector
	}
	cpu.initInstructions()
	return cpu
}

// Reset performs a CPU reset following the precise 6502 reset sequence
func (cpu *CPU) Reset() {
	// 6502 Reset sequence takes 7 cycles total:
	// - 2 cycles for interrupt sequence start
	// - 3 cycles for stack operations (dummy writes)  
	// - 2 cycles to read reset vector
	
	// Initialize all registers to power-up state
	cpu.A = 0x00
	cpu.X = 0x00
	cpu.Y = 0x00
	cpu.SP = 0xFD
	
	// Set processor status to $34 (I=1, unused=1, others=0)
	// This matches real 6502 power-up state
	cpu.C = false // Carry = 0
	cpu.Z = false // Zero = 0  
	cpu.I = true  // Interrupt disable = 1
	cpu.D = false // Decimal = 0 (unused in NES anyway)
	cpu.B = true  // Break = 1 (unused bit, always 1)
	cpu.V = false // Overflow = 0
	cpu.N = false // Negative = 0
	
	// Perform 5 bus operations during reset (like Rgnes)
	// These are dummy reads/writes that occur during reset sequence
	for i := 0; i < 5; i++ {
		// Dummy read from current PC (before reset vector read)
		cpu.memory.Read(cpu.PC)
		cpu.cycles++
	}
	
	// Read reset vector from 0xFFFC-0xFFFD (2 more bus operations)
	low := uint16(cpu.memory.Read(resetVector))
	high := uint16(cpu.memory.Read(resetVector + 1))
	cpu.PC = (high << 8) | low
	
	// Add 2 more cycles for reset vector reads
	cpu.cycles += 2
	
	// Total: 7 cycles for complete reset sequence
}

// Step executes a single CPU instruction and returns cycles taken.
// This is the main execution loop called every CPU cycle.
func (cpu *CPU) Step() uint64 {
	// Capture PC for debugging
	currentPC := cpu.PC
	
	// Fetch instruction opcode from memory at PC
	opcode := cpu.memory.Read(cpu.PC)
	instruction := cpu.instructions[opcode]
	
	// Debug logging and loop detection
	if cpu.enableLoopDetection {
		cpu.detectInfiniteLoop(currentPC, opcode)
	}
	if cpu.enableDebugLogging {
		cpu.logInstruction(currentPC, opcode, instruction)
	}

	if instruction == nil {
		// This case should ideally not be hit if all illegal opcodes are defined.
		// It acts as a fallback.
		cpu.PC++
		cpu.cycles += 2
		return 2
	}

	// Get operand address based on addressing mode
	address, pageCrossed := cpu.getOperandAddress(instruction.Mode)

	// Execute instruction
	extraCycles := cpu.executeInstruction(opcode, address, pageCrossed)

	// Add page crossing penalty for certain instructions
	if pageCrossed {
		// Store instructions always take extra cycle for indexed modes
		if opcode == 0x9D || opcode == 0x99 || opcode == 0x91 {
			extraCycles++
		} else {
			// Check for read instructions that take a penalty on page cross
			switch opcode {
			// Official Opcodes
			case 0xBD, 0xB9, 0xB1, 0xBE, 0xBC, 0x7D, 0x79, 0x71, 0x3D, 0x39, 0x31, 0x1D, 0x19, 0x11, 0x5D, 0x59, 0x51, 0xDD, 0xD9, 0xD1:
				extraCycles++
			// Unofficial NOPs (Absolute,X)
			case 0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC:
				extraCycles++
			// Unofficial Read-type opcodes
			case 0xBF, 0xB3, 0xD3, 0xD7, 0xDF, 0xF3, 0xF7, 0xFF, 0x13, 0x17, 0x1F, 0x33, 0x37, 0x3F, 0x53, 0x57, 0x5F, 0x73, 0x77, 0x7F:
				extraCycles++
			}
		}
		// Branch instructions handle their own page crossing logic in the branch functions
	}

	totalCycles := uint64(instruction.Cycles + extraCycles)
	cpu.cycles += totalCycles
	
	// Check for pending interrupts after instruction completion
	// This implements the 1-instruction delay behavior
	cpu.ProcessPendingInterrupts()
	
	return totalCycles
}

// getOperandAddress returns the effective address for the given addressing mode.
// Returns the address and whether a page boundary was crossed (affects cycle timing).
func (cpu *CPU) getOperandAddress(mode AddressingMode) (uint16, bool) {
	pageCrossed := false

	switch mode {
	case Implied, Accumulator:
		cpu.PC += 1 // Single byte instruction
		return 0, false

	case Immediate:
		address := cpu.PC + 1
		cpu.PC += 2
		return address, false

	case ZeroPage:
		address := uint16(cpu.memory.Read(cpu.PC + 1))
		cpu.PC += 2
		return address, false

	case ZeroPageX:
		base := cpu.memory.Read(cpu.PC + 1)
		address := uint16((base + cpu.X) & zeroPageMask) // Wrap within zero page
		cpu.PC += 2
		return address, false

	case ZeroPageY:
		base := cpu.memory.Read(cpu.PC + 1)
		address := uint16((base + cpu.Y) & zeroPageMask) // Wrap within zero page
		cpu.PC += 2
		return address, false

	case Relative:
		offset := int8(cpu.memory.Read(cpu.PC + 1))
		oldPC := cpu.PC + 2
		newPC := uint16(int32(oldPC) + int32(offset))
		cpu.PC = oldPC // Will be updated by branch instruction if taken
		// Check for page boundary crossing
		pageCrossed = (oldPC & pageMask) != (newPC & pageMask)
		return newPC, pageCrossed

	case Absolute:
		low := uint16(cpu.memory.Read(cpu.PC + 1))
		high := uint16(cpu.memory.Read(cpu.PC + 2))
		address := (high << 8) | low
		cpu.PC += 3
		return address, false

	case AbsoluteX:
		low := uint16(cpu.memory.Read(cpu.PC + 1))
		high := uint16(cpu.memory.Read(cpu.PC + 2))
		base := (high << 8) | low
		address := base + uint16(cpu.X)
		cpu.PC += 3
		// Check for page boundary crossing
		pageCrossed = (base & pageMask) != (address & pageMask)
		return address, pageCrossed

	case AbsoluteY:
		low := uint16(cpu.memory.Read(cpu.PC + 1))
		high := uint16(cpu.memory.Read(cpu.PC + 2))
		base := (high << 8) | low
		address := base + uint16(cpu.Y)
		cpu.PC += 3
		// Check for page boundary crossing
		pageCrossed = (base & pageMask) != (address & pageMask)
		return address, pageCrossed

	case Indirect: // Only used by JMP
		lowPtr := uint16(cpu.memory.Read(cpu.PC + 1))
		highPtr := uint16(cpu.memory.Read(cpu.PC + 2))
		ptr := (highPtr << 8) | lowPtr

		// Handle page boundary bug: if low byte is 0xFF,
		// high byte is read from beginning of same page
		var address uint16
		if (ptr & zeroPageMask) == zeroPageMask {
			low := uint16(cpu.memory.Read(ptr))
			high := uint16(cpu.memory.Read(ptr & pageMask)) // Bug: wraps to start of page
			address = (high << 8) | low
		} else {
			low := uint16(cpu.memory.Read(ptr))
			high := uint16(cpu.memory.Read(ptr + 1))
			address = (high << 8) | low
		}
		cpu.PC += 3
		return address, false

	case IndexedIndirect: // (zp,X)
		base := cpu.memory.Read(cpu.PC + 1)
		ptr := (base + cpu.X) & zeroPageMask // Wrap within zero page
		low := uint16(cpu.memory.Read(uint16(ptr)))
		high := uint16(cpu.memory.Read(uint16((ptr + 1) & zeroPageMask))) // Wrap within zero page
		address := (high << 8) | low
		cpu.PC += 2
		return address, false

	case IndirectIndexed: // (zp),Y
		ptr := uint16(cpu.memory.Read(cpu.PC + 1))
		low := uint16(cpu.memory.Read(ptr))
		high := uint16(cpu.memory.Read((ptr + 1) & zeroPageMask)) // Wrap within zero page
		base := (high << 8) | low
		address := base + uint16(cpu.Y)
		cpu.PC += 2
		// Check for page boundary crossing
		pageCrossed = (base & pageMask) != (address & pageMask)
		return address, pageCrossed

	default:
		return 0, false
	}
}

// Stack operations - optimized with constant
func (cpu *CPU) push(value uint8) {
	cpu.memory.Write(stackBase+uint16(cpu.SP), value)
	cpu.SP--
}

func (cpu *CPU) pop() uint8 {
	cpu.SP++
	return cpu.memory.Read(stackBase + uint16(cpu.SP))
}

func (cpu *CPU) pushWord(value uint16) {
	cpu.push(uint8(value >> 8))   // High byte first
	cpu.push(uint8(value & 0xFF)) // Low byte second
}

func (cpu *CPU) popWord() uint16 {
	low := uint16(cpu.pop())
	high := uint16(cpu.pop())
	return (high << 8) | low
}

// Flag operations
// setZN sets Zero and Negative flags based on value - optimized with constant
func (cpu *CPU) setZN(value uint8) {
	cpu.Z = value == 0
	cpu.N = (value & nFlagMask) != 0
}

// Interrupt handling - optimized with constants
func (cpu *CPU) handleNMI() {
	cpu.pushWord(cpu.PC)
	// Push status with B=0 for hardware interrupt
	status := cpu.GetStatusByte() & (^uint8(bFlagMask)) // Clear B flag (bit 4)
	status |= unusedMask                                // Set unused bit (bit 5)
	cpu.push(status)
	cpu.I = true
	low := uint16(cpu.memory.Read(nmiVector))
	high := uint16(cpu.memory.Read(nmiVector + 1))
	cpu.PC = (high << 8) | low
	cpu.cycles += 7 // NMI takes 7 cycles
}

func (cpu *CPU) handleIRQ() {
	cpu.pushWord(cpu.PC)
	// Push status with B=0 for hardware interrupt
	status := cpu.GetStatusByte() & (^uint8(bFlagMask)) // Clear B flag (bit 4)
	status |= unusedMask                                // Set unused bit (bit 5)
	cpu.push(status)
	cpu.I = true
	low := uint16(cpu.memory.Read(irqVector))
	high := uint16(cpu.memory.Read(irqVector + 1))
	cpu.PC = (high << 8) | low
	cpu.cycles += 7 // IRQ takes 7 cycles
}

// SetNMI sets the NMI line state for edge detection
// NMI triggers on falling edge (true -> false transition)
func (cpu *CPU) SetNMI(state bool) {
	// Check for falling edge (previous=true, current=false)
	if cpu.nmiPrevious && !state {
		cpu.nmiPending = true
	}
	cpu.nmiPrevious = state
}

// SetIRQ sets the IRQ line state
func (cpu *CPU) SetIRQ(state bool) {
	cpu.irqPending = state
}

// ProcessPendingInterrupts checks and processes any pending interrupts
// This should be called after each instruction completion
func (cpu *CPU) ProcessPendingInterrupts() {
	// NMI has highest priority and cannot be disabled
	if cpu.nmiPending {
		cpu.nmiPending = false
		cpu.handleNMI()
		return
	}
	
	// IRQ can be disabled by the I flag
	if cpu.irqPending && !cpu.I {
		cpu.handleIRQ()
		return
	}
}

// Legacy methods for backward compatibility
func (cpu *CPU) TriggerNMI() {
	cpu.nmiPending = true
}

func (cpu *CPU) TriggerIRQ() {
	cpu.irqPending = true
}

// GetStatusByte returns the status register as a byte - optimized with bit masks
func (cpu *CPU) GetStatusByte() uint8 {
	var status uint8
	if cpu.N {
		status |= nFlagMask
	}
	if cpu.V {
		status |= vFlagMask
	}
	// Bit 5 is always set (unused)
	status |= unusedMask
	if cpu.B {
		status |= bFlagMask
	}
	if cpu.D {
		status |= dFlagMask
	}
	if cpu.I {
		status |= iFlagMask
	}
	if cpu.Z {
		status |= zFlagMask
	}
	if cpu.C {
		status |= cFlagMask
	}
	return status
}

// SetStatusByte sets the status register from a byte - optimized with bit masks
func (cpu *CPU) SetStatusByte(status uint8) {
	cpu.N = (status & nFlagMask) != 0
	cpu.V = (status & vFlagMask) != 0
	cpu.B = (status & bFlagMask) != 0
	cpu.D = (status & dFlagMask) != 0
	cpu.I = (status & iFlagMask) != 0
	cpu.Z = (status & zFlagMask) != 0
	cpu.C = (status & cFlagMask) != 0
}

// Instruction operations

// Load operations
func (cpu *CPU) lda(address uint16) uint8 {
	cpu.A = cpu.memory.Read(address)
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) ldx(address uint16) uint8 {
	cpu.X = cpu.memory.Read(address)
	cpu.setZN(cpu.X)
	return 0
}

func (cpu *CPU) ldy(address uint16) uint8 {
	cpu.Y = cpu.memory.Read(address)
	cpu.setZN(cpu.Y)
	return 0
}

// Store operations
func (cpu *CPU) sta(address uint16) uint8 {
	cpu.memory.Write(address, cpu.A)
	return 0
}

func (cpu *CPU) stx(address uint16) uint8 {
	cpu.memory.Write(address, cpu.X)
	return 0
}

func (cpu *CPU) sty(address uint16) uint8 {
	cpu.memory.Write(address, cpu.Y)
	return 0
}

// Arithmetic operations
func (cpu *CPU) adc(address uint16) uint8 {
	value := cpu.memory.Read(address)
	carry := uint8(0)
	if cpu.C {
		carry = 1
	}

	result := uint16(cpu.A) + uint16(value) + uint16(carry)

	// Set overflow flag - occurs when sign of result differs from inputs
	cpu.V = ((cpu.A^uint8(result))&0x80) != 0 && ((cpu.A^value)&0x80) == 0

	cpu.C = result > 0xFF
	cpu.A = uint8(result)
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) sbc(address uint16) uint8 {
	value := cpu.memory.Read(address) ^ 0xFF // Invert bits for subtraction
	carry := uint8(0)
	if cpu.C {
		carry = 1
	}

	result := uint16(cpu.A) + uint16(value) + uint16(carry)

	// Set overflow flag
	cpu.V = ((cpu.A^uint8(result))&0x80) != 0 && ((cpu.A^value)&0x80) == 0

	cpu.C = result > 0xFF
	cpu.A = uint8(result)
	cpu.setZN(cpu.A)
	return 0
}

// Logical operations
func (cpu *CPU) and(address uint16) uint8 {
	cpu.A &= cpu.memory.Read(address)
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) ora(address uint16) uint8 {
	cpu.A |= cpu.memory.Read(address)
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) eor(address uint16) uint8 {
	cpu.A ^= cpu.memory.Read(address)
	cpu.setZN(cpu.A)
	return 0
}

// Shift and rotate operations (Memory versions)
func (cpu *CPU) asl(address uint16) uint8 {
	value := cpu.memory.Read(address)
	cpu.C = (value & 0x80) != 0
	value <<= 1
	cpu.memory.Write(address, value)
	cpu.setZN(value)
	return 0
}

func (cpu *CPU) lsr(address uint16) uint8 {
	value := cpu.memory.Read(address)
	cpu.C = (value & 0x01) != 0
	value >>= 1
	cpu.memory.Write(address, value)
	cpu.setZN(value)
	return 0
}

func (cpu *CPU) rol(address uint16) uint8 {
	value := cpu.memory.Read(address)
	oldCarry := cpu.C
	cpu.C = (value & 0x80) != 0
	value <<= 1
	if oldCarry {
		value |= 0x01
	}
	cpu.memory.Write(address, value)
	cpu.setZN(value)
	return 0
}

func (cpu *CPU) ror(address uint16) uint8 {
	value := cpu.memory.Read(address)
	oldCarry := cpu.C
	cpu.C = (value & 0x01) != 0
	value >>= 1
	if oldCarry {
		value |= 0x80
	}
	cpu.memory.Write(address, value)
	cpu.setZN(value)
	return 0
}

// Comparison operations
func (cpu *CPU) cmp(address uint16) uint8 {
	value := cpu.memory.Read(address)
	result := cpu.A - value
	cpu.C = cpu.A >= value
	cpu.setZN(result)
	return 0
}

func (cpu *CPU) cpx(address uint16) uint8 {
	value := cpu.memory.Read(address)
	result := cpu.X - value
	cpu.C = cpu.X >= value
	cpu.setZN(result)
	return 0
}

func (cpu *CPU) cpy(address uint16) uint8 {
	value := cpu.memory.Read(address)
	result := cpu.Y - value
	cpu.C = cpu.Y >= value
	cpu.setZN(result)
	return 0
}

// Increment/Decrement operations
func (cpu *CPU) inc(address uint16) uint8 {
	value := cpu.memory.Read(address) + 1
	cpu.memory.Write(address, value)
	cpu.setZN(value)
	return 0
}

func (cpu *CPU) dec(address uint16) uint8 {
	value := cpu.memory.Read(address) - 1
	cpu.memory.Write(address, value)
	cpu.setZN(value)
	return 0
}

func (cpu *CPU) inx(address uint16) uint8 {
	cpu.X++
	cpu.setZN(cpu.X)
	return 0
}

func (cpu *CPU) dex(address uint16) uint8 {
	cpu.X--
	cpu.setZN(cpu.X)
	return 0
}

func (cpu *CPU) iny(address uint16) uint8 {
	cpu.Y++
	cpu.setZN(cpu.Y)
	return 0
}

func (cpu *CPU) dey(address uint16) uint8 {
	cpu.Y--
	cpu.setZN(cpu.Y)
	return 0
}

// Transfer operations
func (cpu *CPU) tax(address uint16) uint8 {
	cpu.X = cpu.A
	cpu.setZN(cpu.X)
	return 0
}

func (cpu *CPU) txa(address uint16) uint8 {
	cpu.A = cpu.X
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) tay(address uint16) uint8 {
	cpu.Y = cpu.A
	cpu.setZN(cpu.Y)
	return 0
}

func (cpu *CPU) tya(address uint16) uint8 {
	cpu.A = cpu.Y
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) tsx(address uint16) uint8 {
	cpu.X = cpu.SP
	cpu.setZN(cpu.X)
	return 0
}

func (cpu *CPU) txs(address uint16) uint8 {
	cpu.SP = cpu.X
	return 0
}

// Stack operations
func (cpu *CPU) pha(address uint16) uint8 {
	cpu.push(cpu.A)
	return 0
}

func (cpu *CPU) pla(address uint16) uint8 {
	cpu.A = cpu.pop()
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) php(address uint16) uint8 {
	cpu.push(cpu.GetStatusByte() | bFlagMask) // B flag set for PHP
	return 0
}

func (cpu *CPU) plp(address uint16) uint8 {
	status := cpu.pop()
	cpu.SetStatusByte(status)
	return 0
}

// Flag operations
func (cpu *CPU) clc(address uint16) uint8 {
	cpu.C = false
	return 0
}

func (cpu *CPU) sec(address uint16) uint8 {
	cpu.C = true
	return 0
}

func (cpu *CPU) cli(address uint16) uint8 {
	cpu.I = false
	return 0
}

func (cpu *CPU) sei(address uint16) uint8 {
	cpu.I = true
	return 0
}

func (cpu *CPU) clv(address uint16) uint8 {
	cpu.V = false
	return 0
}

func (cpu *CPU) cld(address uint16) uint8 {
	cpu.D = false
	return 0
}

func (cpu *CPU) sed(address uint16) uint8 {
	cpu.D = true
	return 0
}

// Control flow operations
func (cpu *CPU) jmp(address uint16) uint8 {
	cpu.PC = address
	return 0
}

func (cpu *CPU) jsr(address uint16) uint8 {
	// Push return address - 1 (JSR pushes PC-1)
	cpu.pushWord(cpu.PC - 1)
	cpu.PC = address
	return 0
}

func (cpu *CPU) rts(address uint16) uint8 {
	cpu.PC = cpu.popWord() + 1 // RTS adds 1 to popped address
	return 0
}

func (cpu *CPU) rti(address uint16) uint8 {
	cpu.SetStatusByte(cpu.pop())
	cpu.PC = cpu.popWord()
	return 0
}

// Branch operations
func (cpu *CPU) bcc(address uint16, pageCrossed bool) uint8 {
	if !cpu.C {
		cpu.PC = address
		if pageCrossed {
			return 2 // 1 for taken + 1 for page crossing
		}
		return 1 // 1 for taken branch
	}
	return 0
}

func (cpu *CPU) bcs(address uint16, pageCrossed bool) uint8 {
	if cpu.C {
		cpu.PC = address
		if pageCrossed {
			return 2 // 1 for taken + 1 for page crossing
		}
		return 1 // 1 for taken branch
	}
	return 0
}

func (cpu *CPU) bne(address uint16, pageCrossed bool) uint8 {
	if !cpu.Z {
		cpu.PC = address
		if pageCrossed {
			return 2 // 1 for taken + 1 for page crossing
		}
		return 1 // 1 for taken branch
	}
	return 0
}

func (cpu *CPU) beq(address uint16, pageCrossed bool) uint8 {
	if cpu.Z {
		cpu.PC = address
		if pageCrossed {
			return 2 // 1 for taken + 1 for page crossing
		}
		return 1 // 1 for taken branch
	}
	return 0
}

func (cpu *CPU) bpl(address uint16, pageCrossed bool) uint8 {
	if !cpu.N {
		cpu.PC = address
		if pageCrossed {
			return 2 // 1 for taken + 1 for page crossing
		}
		return 1 // 1 for taken branch
	}
	return 0
}

func (cpu *CPU) bmi(address uint16, pageCrossed bool) uint8 {
	if cpu.N {
		cpu.PC = address
		if pageCrossed {
			return 2 // 1 for taken + 1 for page crossing
		}
		return 1 // 1 for taken branch
	}
	return 0
}

func (cpu *CPU) bvc(address uint16, pageCrossed bool) uint8 {
	if !cpu.V {
		cpu.PC = address
		if pageCrossed {
			return 2 // 1 for taken + 1 for page crossing
		}
		return 1 // 1 for taken branch
	}
	return 0
}

func (cpu *CPU) bvs(address uint16, pageCrossed bool) uint8 {
	if cpu.V {
		cpu.PC = address
		if pageCrossed {
			return 2 // 1 for taken + 1 for page crossing
		}
		return 1 // 1 for taken branch
	}
	return 0
}

// Miscellaneous operations
func (cpu *CPU) bit(address uint16) uint8 {
	value := cpu.memory.Read(address)
	cpu.N = (value & nFlagMask) != 0 // Bit 7 of memory
	cpu.V = (value & vFlagMask) != 0 // Bit 6 of memory
	cpu.Z = (cpu.A & value) == 0     // Zero if A AND memory == 0
	return 0
}

func (cpu *CPU) nop(address uint16) uint8 {
	return 0
}

func (cpu *CPU) brk(address uint16) uint8 {
	// BRK is a 1-byte instruction, but it pushes PC+2 to the stack.
	// getOperandAddress for Implied mode has already incremented PC by 1.
	cpu.PC++ // Manually increment for the 'padding' byte
	cpu.pushWord(cpu.PC)

	cpu.push(cpu.GetStatusByte() | bFlagMask) // B flag is set when pushed by BRK/PHP
	cpu.I = true                              // Disable interrupts

	// Load IRQ vector into PC
	low := uint16(cpu.memory.Read(irqVector))
	high := uint16(cpu.memory.Read(irqVector + 1))
	cpu.PC = (high << 8) | low
	return 0
}

// --- Unofficial Opcodes ---

func (cpu *CPU) lax(address uint16) uint8 {
	cpu.A = cpu.memory.Read(address)
	cpu.X = cpu.A
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) sax(address uint16) uint8 {
	cpu.memory.Write(address, cpu.A&cpu.X)
	return 0
}

func (cpu *CPU) dcp(address uint16) uint8 {
	value := cpu.memory.Read(address) - 1
	cpu.memory.Write(address, value)
	result := cpu.A - value
	cpu.C = cpu.A >= value
	cpu.setZN(result)
	return 0
}

func (cpu *CPU) isb(address uint16) uint8 {
	value := cpu.memory.Read(address) + 1
	cpu.memory.Write(address, value)
	// Now perform SBC with the incremented value
	cpu.sbc(address) // Re-use SBC logic, but it will read the already-incremented value
	return 0
}

func (cpu *CPU) slo(address uint16) uint8 {
	// ASL part
	value := cpu.memory.Read(address)
	cpu.C = (value & 0x80) != 0
	value <<= 1
	cpu.memory.Write(address, value)
	// ORA part
	cpu.A |= value
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) rla(address uint16) uint8 {
	// ROL part
	value := cpu.memory.Read(address)
	oldCarry := cpu.C
	cpu.C = (value & 0x80) != 0
	value <<= 1
	if oldCarry {
		value |= 0x01
	}
	cpu.memory.Write(address, value)
	// AND part
	cpu.A &= value
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) sre(address uint16) uint8 {
	// LSR part
	value := cpu.memory.Read(address)
	cpu.C = (value & 0x01) != 0
	value >>= 1
	cpu.memory.Write(address, value)
	// EOR part
	cpu.A ^= value
	cpu.setZN(cpu.A)
	return 0
}

func (cpu *CPU) rra(address uint16) uint8 {
	// ROR part
	value := cpu.memory.Read(address)
	oldCarry := cpu.C
	cpu.C = (value & 0x01) != 0
	value >>= 1
	if oldCarry {
		value |= 0x80
	}
	cpu.memory.Write(address, value)
	// ADC part
	cpu.adc(address) // Re-use ADC logic, but it will read the already-rotated value
	return 0
}

// executeInstruction executes the given opcode with the provided address.
// Returns extra cycles taken beyond the base instruction cycle count.
func (cpu *CPU) executeInstruction(opcode uint8, address uint16, pageCrossed bool) uint8 {
	switch opcode {
	// Load/Store Instructions
	case 0xA9, 0xA5, 0xB5, 0xAD, 0xBD, 0xB9, 0xA1, 0xB1: // LDA
		return cpu.lda(address)
	case 0xA2, 0xA6, 0xB6, 0xAE, 0xBE: // LDX
		return cpu.ldx(address)
	case 0xA0, 0xA4, 0xB4, 0xAC, 0xBC: // LDY
		return cpu.ldy(address)
	case 0x85, 0x95, 0x8D, 0x9D, 0x99, 0x81, 0x91: // STA
		return cpu.sta(address)
	case 0x86, 0x96, 0x8E: // STX
		return cpu.stx(address)
	case 0x84, 0x94, 0x8C: // STY
		return cpu.sty(address)

	// Arithmetic Instructions
	case 0x69, 0x65, 0x75, 0x6D, 0x7D, 0x79, 0x61, 0x71: // ADC
		return cpu.adc(address)
	case 0xE9, 0xEB, 0xE5, 0xF5, 0xED, 0xFD, 0xF9, 0xE1, 0xF1: // SBC (0xEB is unofficial)
		return cpu.sbc(address)

	// Logical Instructions
	case 0x29, 0x25, 0x35, 0x2D, 0x3D, 0x39, 0x21, 0x31: // AND
		return cpu.and(address)
	case 0x09, 0x05, 0x15, 0x0D, 0x1D, 0x19, 0x01, 0x11: // ORA
		return cpu.ora(address)
	case 0x49, 0x45, 0x55, 0x4D, 0x5D, 0x59, 0x41, 0x51: // EOR
		return cpu.eor(address)

	// Shift and Rotate Instructions
	case 0x0A: // ASL Accumulator
		cpu.C = (cpu.A & 0x80) != 0
		cpu.A <<= 1
		cpu.setZN(cpu.A)
		return 0
	case 0x06, 0x16, 0x0E, 0x1E: // ASL Memory
		return cpu.asl(address)
	case 0x4A: // LSR Accumulator
		cpu.C = (cpu.A & 0x01) != 0
		cpu.A >>= 1
		cpu.setZN(cpu.A)
		return 0
	case 0x46, 0x56, 0x4E, 0x5E: // LSR Memory
		return cpu.lsr(address)
	case 0x2A: // ROL Accumulator
		oldCarry := cpu.C
		cpu.C = (cpu.A & 0x80) != 0
		cpu.A <<= 1
		if oldCarry {
			cpu.A |= 0x01
		}
		cpu.setZN(cpu.A)
		return 0
	case 0x26, 0x36, 0x2E, 0x3E: // ROL Memory
		return cpu.rol(address)
	case 0x6A: // ROR Accumulator
		oldCarry := cpu.C
		cpu.C = (cpu.A & 0x01) != 0
		cpu.A >>= 1
		if oldCarry {
			cpu.A |= 0x80
		}
		cpu.setZN(cpu.A)
		return 0
	case 0x66, 0x76, 0x6E, 0x7E: // ROR Memory
		return cpu.ror(address)

	// Comparison Instructions
	case 0xC9, 0xC5, 0xD5, 0xCD, 0xDD, 0xD9, 0xC1, 0xD1: // CMP
		return cpu.cmp(address)
	case 0xE0, 0xE4, 0xEC: // CPX
		return cpu.cpx(address)
	case 0xC0, 0xC4, 0xCC: // CPY
		return cpu.cpy(address)

	// Increment/Decrement Instructions
	case 0xE6, 0xF6, 0xEE, 0xFE: // INC
		return cpu.inc(address)
	case 0xC6, 0xD6, 0xCE, 0xDE: // DEC
		return cpu.dec(address)
	case 0xE8: // INX
		return cpu.inx(address)
	case 0xCA: // DEX
		return cpu.dex(address)
	case 0xC8: // INY
		return cpu.iny(address)
	case 0x88: // DEY
		return cpu.dey(address)

	// Transfer Instructions
	case 0xAA: // TAX
		return cpu.tax(address)
	case 0x8A: // TXA
		return cpu.txa(address)
	case 0xA8: // TAY
		return cpu.tay(address)
	case 0x98: // TYA
		return cpu.tya(address)
	case 0xBA: // TSX
		return cpu.tsx(address)
	case 0x9A: // TXS
		return cpu.txs(address)

	// Stack Instructions
	case 0x48: // PHA
		return cpu.pha(address)
	case 0x68: // PLA
		return cpu.pla(address)
	case 0x08: // PHP
		return cpu.php(address)
	case 0x28: // PLP
		return cpu.plp(address)

	// Flag Instructions
	case 0x18: // CLC
		return cpu.clc(address)
	case 0x38: // SEC
		return cpu.sec(address)
	case 0x58: // CLI
		return cpu.cli(address)
	case 0x78: // SEI
		return cpu.sei(address)
	case 0xB8: // CLV
		return cpu.clv(address)
	case 0xD8: // CLD
		return cpu.cld(address)
	case 0xF8: // SED
		return cpu.sed(address)

	// Control Flow Instructions
	case 0x4C, 0x6C: // JMP
		return cpu.jmp(address)
	case 0x20: // JSR
		return cpu.jsr(address)
	case 0x60: // RTS
		return cpu.rts(address)
	case 0x40: // RTI
		return cpu.rti(address)

	// Branch Instructions
	case 0x90: // BCC
		return cpu.bcc(address, pageCrossed)
	case 0xB0: // BCS
		return cpu.bcs(address, pageCrossed)
	case 0xD0: // BNE
		return cpu.bne(address, pageCrossed)
	case 0xF0: // BEQ
		return cpu.beq(address, pageCrossed)
	case 0x10: // BPL
		return cpu.bpl(address, pageCrossed)
	case 0x30: // BMI
		return cpu.bmi(address, pageCrossed)
	case 0x50: // BVC
		return cpu.bvc(address, pageCrossed)
	case 0x70: // BVS
		return cpu.bvs(address, pageCrossed)

	// Miscellaneous Instructions
	case 0x24, 0x2C: // BIT
		return cpu.bit(address)
	case 0x00: // BRK
		return cpu.brk(address)

	// Unofficial NOPs
	case 0xEA, 0x1A, 0x3A, 0x5A, 0x7A, 0xDA, 0xFA, 0x80, 0x82, 0x89, 0xC2, 0xE2, 0x04, 0x44, 0x64, 0x14, 0x34, 0x54, 0x74, 0xD4, 0xF4, 0x0C, 0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC:
		return cpu.nop(address)

	// Unofficial Opcodes
	case 0xA3, 0xA7, 0xAF, 0xB3, 0xB7, 0xBF: // LAX
		return cpu.lax(address)
	case 0x83, 0x87, 0x8F, 0x97: // SAX
		return cpu.sax(address)
	case 0xC3, 0xC7, 0xCF, 0xD3, 0xD7, 0xDF, 0xDB: // DCP
		return cpu.dcp(address)
	case 0xE3, 0xE7, 0xEF, 0xF3, 0xF7, 0xFF, 0xFB: // ISB
		return cpu.isb(address)
	case 0x03, 0x07, 0x0F, 0x13, 0x17, 0x1F, 0x1B: // SLO
		return cpu.slo(address)
	case 0x23, 0x27, 0x2F, 0x33, 0x37, 0x3F, 0x3B: // RLA
		return cpu.rla(address)
	case 0x43, 0x47, 0x4F, 0x53, 0x57, 0x5F, 0x5B: // SRE
		return cpu.sre(address)
	case 0x63, 0x67, 0x6F, 0x73, 0x77, 0x7F, 0x7B: // RRA
		return cpu.rra(address)

	default:
		// Should not be reached if all opcodes are mapped
		return 0
	}
}

// initInstructions populates the instruction lookup table with all valid 6502 opcodes.
// This creates a direct opcode-to-instruction mapping for fast dispatch.
func (cpu *CPU) initInstructions() {
	// Initialize all entries to nil first
	for i := range cpu.instructions {
		cpu.instructions[i] = nil
	}

	// Load/Store Instructions
	cpu.instructions[0xA9] = &Instruction{"LDA", 0xA9, 2, 2, Immediate}
	cpu.instructions[0xA5] = &Instruction{"LDA", 0xA5, 2, 3, ZeroPage}
	cpu.instructions[0xB5] = &Instruction{"LDA", 0xB5, 2, 4, ZeroPageX}
	cpu.instructions[0xAD] = &Instruction{"LDA", 0xAD, 3, 4, Absolute}
	cpu.instructions[0xBD] = &Instruction{"LDA", 0xBD, 3, 4, AbsoluteX}
	cpu.instructions[0xB9] = &Instruction{"LDA", 0xB9, 3, 4, AbsoluteY}
	cpu.instructions[0xA1] = &Instruction{"LDA", 0xA1, 2, 6, IndexedIndirect}
	cpu.instructions[0xB1] = &Instruction{"LDA", 0xB1, 2, 5, IndirectIndexed}

	cpu.instructions[0xA2] = &Instruction{"LDX", 0xA2, 2, 2, Immediate}
	cpu.instructions[0xA6] = &Instruction{"LDX", 0xA6, 2, 3, ZeroPage}
	cpu.instructions[0xB6] = &Instruction{"LDX", 0xB6, 2, 4, ZeroPageY}
	cpu.instructions[0xAE] = &Instruction{"LDX", 0xAE, 3, 4, Absolute}
	cpu.instructions[0xBE] = &Instruction{"LDX", 0xBE, 3, 4, AbsoluteY}

	cpu.instructions[0xA0] = &Instruction{"LDY", 0xA0, 2, 2, Immediate}
	cpu.instructions[0xA4] = &Instruction{"LDY", 0xA4, 2, 3, ZeroPage}
	cpu.instructions[0xB4] = &Instruction{"LDY", 0xB4, 2, 4, ZeroPageX}
	cpu.instructions[0xAC] = &Instruction{"LDY", 0xAC, 3, 4, Absolute}
	cpu.instructions[0xBC] = &Instruction{"LDY", 0xBC, 3, 4, AbsoluteX}

	cpu.instructions[0x85] = &Instruction{"STA", 0x85, 2, 3, ZeroPage}
	cpu.instructions[0x95] = &Instruction{"STA", 0x95, 2, 4, ZeroPageX}
	cpu.instructions[0x8D] = &Instruction{"STA", 0x8D, 3, 4, Absolute}
	cpu.instructions[0x9D] = &Instruction{"STA", 0x9D, 3, 5, AbsoluteX}
	cpu.instructions[0x99] = &Instruction{"STA", 0x99, 3, 5, AbsoluteY}
	cpu.instructions[0x81] = &Instruction{"STA", 0x81, 2, 6, IndexedIndirect}
	cpu.instructions[0x91] = &Instruction{"STA", 0x91, 2, 6, IndirectIndexed}

	cpu.instructions[0x86] = &Instruction{"STX", 0x86, 2, 3, ZeroPage}
	cpu.instructions[0x96] = &Instruction{"STX", 0x96, 2, 4, ZeroPageY}
	cpu.instructions[0x8E] = &Instruction{"STX", 0x8E, 3, 4, Absolute}

	cpu.instructions[0x84] = &Instruction{"STY", 0x84, 2, 3, ZeroPage}
	cpu.instructions[0x94] = &Instruction{"STY", 0x94, 2, 4, ZeroPageX}
	cpu.instructions[0x8C] = &Instruction{"STY", 0x8C, 3, 4, Absolute}

	// Arithmetic Instructions
	cpu.instructions[0x69] = &Instruction{"ADC", 0x69, 2, 2, Immediate}
	cpu.instructions[0x65] = &Instruction{"ADC", 0x65, 2, 3, ZeroPage}
	cpu.instructions[0x75] = &Instruction{"ADC", 0x75, 2, 4, ZeroPageX}
	cpu.instructions[0x6D] = &Instruction{"ADC", 0x6D, 3, 4, Absolute}
	cpu.instructions[0x7D] = &Instruction{"ADC", 0x7D, 3, 4, AbsoluteX}
	cpu.instructions[0x79] = &Instruction{"ADC", 0x79, 3, 4, AbsoluteY}
	cpu.instructions[0x61] = &Instruction{"ADC", 0x61, 2, 6, IndexedIndirect}
	cpu.instructions[0x71] = &Instruction{"ADC", 0x71, 2, 5, IndirectIndexed}

	cpu.instructions[0xE9] = &Instruction{"SBC", 0xE9, 2, 2, Immediate}
	cpu.instructions[0xE5] = &Instruction{"SBC", 0xE5, 2, 3, ZeroPage}
	cpu.instructions[0xF5] = &Instruction{"SBC", 0xF5, 2, 4, ZeroPageX}
	cpu.instructions[0xED] = &Instruction{"SBC", 0xED, 3, 4, Absolute}
	cpu.instructions[0xFD] = &Instruction{"SBC", 0xFD, 3, 4, AbsoluteX}
	cpu.instructions[0xF9] = &Instruction{"SBC", 0xF9, 3, 4, AbsoluteY}
	cpu.instructions[0xE1] = &Instruction{"SBC", 0xE1, 2, 6, IndexedIndirect}
	cpu.instructions[0xF1] = &Instruction{"SBC", 0xF1, 2, 5, IndirectIndexed}

	// Logical Instructions
	cpu.instructions[0x29] = &Instruction{"AND", 0x29, 2, 2, Immediate}
	cpu.instructions[0x25] = &Instruction{"AND", 0x25, 2, 3, ZeroPage}
	cpu.instructions[0x35] = &Instruction{"AND", 0x35, 2, 4, ZeroPageX}
	cpu.instructions[0x2D] = &Instruction{"AND", 0x2D, 3, 4, Absolute}
	cpu.instructions[0x3D] = &Instruction{"AND", 0x3D, 3, 4, AbsoluteX}
	cpu.instructions[0x39] = &Instruction{"AND", 0x39, 3, 4, AbsoluteY}
	cpu.instructions[0x21] = &Instruction{"AND", 0x21, 2, 6, IndexedIndirect}
	cpu.instructions[0x31] = &Instruction{"AND", 0x31, 2, 5, IndirectIndexed}

	cpu.instructions[0x09] = &Instruction{"ORA", 0x09, 2, 2, Immediate}
	cpu.instructions[0x05] = &Instruction{"ORA", 0x05, 2, 3, ZeroPage}
	cpu.instructions[0x15] = &Instruction{"ORA", 0x15, 2, 4, ZeroPageX}
	cpu.instructions[0x0D] = &Instruction{"ORA", 0x0D, 3, 4, Absolute}
	cpu.instructions[0x1D] = &Instruction{"ORA", 0x1D, 3, 4, AbsoluteX}
	cpu.instructions[0x19] = &Instruction{"ORA", 0x19, 3, 4, AbsoluteY}
	cpu.instructions[0x01] = &Instruction{"ORA", 0x01, 2, 6, IndexedIndirect}
	cpu.instructions[0x11] = &Instruction{"ORA", 0x11, 2, 5, IndirectIndexed}

	cpu.instructions[0x49] = &Instruction{"EOR", 0x49, 2, 2, Immediate}
	cpu.instructions[0x45] = &Instruction{"EOR", 0x45, 2, 3, ZeroPage}
	cpu.instructions[0x55] = &Instruction{"EOR", 0x55, 2, 4, ZeroPageX}
	cpu.instructions[0x4D] = &Instruction{"EOR", 0x4D, 3, 4, Absolute}
	cpu.instructions[0x5D] = &Instruction{"EOR", 0x5D, 3, 4, AbsoluteX}
	cpu.instructions[0x59] = &Instruction{"EOR", 0x59, 3, 4, AbsoluteY}
	cpu.instructions[0x41] = &Instruction{"EOR", 0x41, 2, 6, IndexedIndirect}
	cpu.instructions[0x51] = &Instruction{"EOR", 0x51, 2, 5, IndirectIndexed}

	// Shift and Rotate Instructions
	cpu.instructions[0x0A] = &Instruction{"ASL", 0x0A, 1, 2, Accumulator}
	cpu.instructions[0x06] = &Instruction{"ASL", 0x06, 2, 5, ZeroPage}
	cpu.instructions[0x16] = &Instruction{"ASL", 0x16, 2, 6, ZeroPageX}
	cpu.instructions[0x0E] = &Instruction{"ASL", 0x0E, 3, 6, Absolute}
	cpu.instructions[0x1E] = &Instruction{"ASL", 0x1E, 3, 7, AbsoluteX}

	cpu.instructions[0x4A] = &Instruction{"LSR", 0x4A, 1, 2, Accumulator}
	cpu.instructions[0x46] = &Instruction{"LSR", 0x46, 2, 5, ZeroPage}
	cpu.instructions[0x56] = &Instruction{"LSR", 0x56, 2, 6, ZeroPageX}
	cpu.instructions[0x4E] = &Instruction{"LSR", 0x4E, 3, 6, Absolute}
	cpu.instructions[0x5E] = &Instruction{"LSR", 0x5E, 3, 7, AbsoluteX}

	cpu.instructions[0x2A] = &Instruction{"ROL", 0x2A, 1, 2, Accumulator}
	cpu.instructions[0x26] = &Instruction{"ROL", 0x26, 2, 5, ZeroPage}
	cpu.instructions[0x36] = &Instruction{"ROL", 0x36, 2, 6, ZeroPageX}
	cpu.instructions[0x2E] = &Instruction{"ROL", 0x2E, 3, 6, Absolute}
	cpu.instructions[0x3E] = &Instruction{"ROL", 0x3E, 3, 7, AbsoluteX}

	cpu.instructions[0x6A] = &Instruction{"ROR", 0x6A, 1, 2, Accumulator}
	cpu.instructions[0x66] = &Instruction{"ROR", 0x66, 2, 5, ZeroPage}
	cpu.instructions[0x76] = &Instruction{"ROR", 0x76, 2, 6, ZeroPageX}
	cpu.instructions[0x6E] = &Instruction{"ROR", 0x6E, 3, 6, Absolute}
	cpu.instructions[0x7E] = &Instruction{"ROR", 0x7E, 3, 7, AbsoluteX}

	// Comparison Instructions
	cpu.instructions[0xC9] = &Instruction{"CMP", 0xC9, 2, 2, Immediate}
	cpu.instructions[0xC5] = &Instruction{"CMP", 0xC5, 2, 3, ZeroPage}
	cpu.instructions[0xD5] = &Instruction{"CMP", 0xD5, 2, 4, ZeroPageX}
	cpu.instructions[0xCD] = &Instruction{"CMP", 0xCD, 3, 4, Absolute}
	cpu.instructions[0xDD] = &Instruction{"CMP", 0xDD, 3, 4, AbsoluteX}
	cpu.instructions[0xD9] = &Instruction{"CMP", 0xD9, 3, 4, AbsoluteY}
	cpu.instructions[0xC1] = &Instruction{"CMP", 0xC1, 2, 6, IndexedIndirect}
	cpu.instructions[0xD1] = &Instruction{"CMP", 0xD1, 2, 5, IndirectIndexed}

	cpu.instructions[0xE0] = &Instruction{"CPX", 0xE0, 2, 2, Immediate}
	cpu.instructions[0xE4] = &Instruction{"CPX", 0xE4, 2, 3, ZeroPage}
	cpu.instructions[0xEC] = &Instruction{"CPX", 0xEC, 3, 4, Absolute}

	cpu.instructions[0xC0] = &Instruction{"CPY", 0xC0, 2, 2, Immediate}
	cpu.instructions[0xC4] = &Instruction{"CPY", 0xC4, 2, 3, ZeroPage}
	cpu.instructions[0xCC] = &Instruction{"CPY", 0xCC, 3, 4, Absolute}

	// Increment/Decrement Instructions
	cpu.instructions[0xE6] = &Instruction{"INC", 0xE6, 2, 5, ZeroPage}
	cpu.instructions[0xF6] = &Instruction{"INC", 0xF6, 2, 6, ZeroPageX}
	cpu.instructions[0xEE] = &Instruction{"INC", 0xEE, 3, 6, Absolute}
	cpu.instructions[0xFE] = &Instruction{"INC", 0xFE, 3, 7, AbsoluteX}

	cpu.instructions[0xC6] = &Instruction{"DEC", 0xC6, 2, 5, ZeroPage}
	cpu.instructions[0xD6] = &Instruction{"DEC", 0xD6, 2, 6, ZeroPageX}
	cpu.instructions[0xCE] = &Instruction{"DEC", 0xCE, 3, 6, Absolute}
	cpu.instructions[0xDE] = &Instruction{"DEC", 0xDE, 3, 7, AbsoluteX}

	cpu.instructions[0xE8] = &Instruction{"INX", 0xE8, 1, 2, Implied}
	cpu.instructions[0xCA] = &Instruction{"DEX", 0xCA, 1, 2, Implied}
	cpu.instructions[0xC8] = &Instruction{"INY", 0xC8, 1, 2, Implied}
	cpu.instructions[0x88] = &Instruction{"DEY", 0x88, 1, 2, Implied}

	// Transfer Instructions
	cpu.instructions[0xAA] = &Instruction{"TAX", 0xAA, 1, 2, Implied}
	cpu.instructions[0x8A] = &Instruction{"TXA", 0x8A, 1, 2, Implied}
	cpu.instructions[0xA8] = &Instruction{"TAY", 0xA8, 1, 2, Implied}
	cpu.instructions[0x98] = &Instruction{"TYA", 0x98, 1, 2, Implied}
	cpu.instructions[0xBA] = &Instruction{"TSX", 0xBA, 1, 2, Implied}
	cpu.instructions[0x9A] = &Instruction{"TXS", 0x9A, 1, 2, Implied}

	// Stack Instructions
	cpu.instructions[0x48] = &Instruction{"PHA", 0x48, 1, 3, Implied}
	cpu.instructions[0x68] = &Instruction{"PLA", 0x68, 1, 4, Implied}
	cpu.instructions[0x08] = &Instruction{"PHP", 0x08, 1, 3, Implied}
	cpu.instructions[0x28] = &Instruction{"PLP", 0x28, 1, 4, Implied}

	// Flag Instructions
	cpu.instructions[0x18] = &Instruction{"CLC", 0x18, 1, 2, Implied}
	cpu.instructions[0x38] = &Instruction{"SEC", 0x38, 1, 2, Implied}
	cpu.instructions[0x58] = &Instruction{"CLI", 0x58, 1, 2, Implied}
	cpu.instructions[0x78] = &Instruction{"SEI", 0x78, 1, 2, Implied}
	cpu.instructions[0xB8] = &Instruction{"CLV", 0xB8, 1, 2, Implied}
	cpu.instructions[0xD8] = &Instruction{"CLD", 0xD8, 1, 2, Implied}
	cpu.instructions[0xF8] = &Instruction{"SED", 0xF8, 1, 2, Implied}

	// Control Flow Instructions
	cpu.instructions[0x4C] = &Instruction{"JMP", 0x4C, 3, 3, Absolute}
	cpu.instructions[0x6C] = &Instruction{"JMP", 0x6C, 3, 5, Indirect}
	cpu.instructions[0x20] = &Instruction{"JSR", 0x20, 3, 6, Absolute}
	cpu.instructions[0x60] = &Instruction{"RTS", 0x60, 1, 6, Implied}
	cpu.instructions[0x40] = &Instruction{"RTI", 0x40, 1, 6, Implied}

	// Branch Instructions
	cpu.instructions[0x90] = &Instruction{"BCC", 0x90, 2, 2, Relative}
	cpu.instructions[0xB0] = &Instruction{"BCS", 0xB0, 2, 2, Relative}
	cpu.instructions[0xD0] = &Instruction{"BNE", 0xD0, 2, 2, Relative}
	cpu.instructions[0xF0] = &Instruction{"BEQ", 0xF0, 2, 2, Relative}
	cpu.instructions[0x10] = &Instruction{"BPL", 0x10, 2, 2, Relative}
	cpu.instructions[0x30] = &Instruction{"BMI", 0x30, 2, 2, Relative}
	cpu.instructions[0x50] = &Instruction{"BVC", 0x50, 2, 2, Relative}
	cpu.instructions[0x70] = &Instruction{"BVS", 0x70, 2, 2, Relative}

	// Miscellaneous Instructions
	cpu.instructions[0x24] = &Instruction{"BIT", 0x24, 2, 3, ZeroPage}
	cpu.instructions[0x2C] = &Instruction{"BIT", 0x2C, 3, 4, Absolute}
	cpu.instructions[0xEA] = &Instruction{"NOP", 0xEA, 1, 2, Implied}
	cpu.instructions[0x00] = &Instruction{"BRK", 0x00, 1, 7, Implied} // Bytes=1, but PC is handled specially

	// Unofficial NOPs
	cpu.instructions[0x1A] = &Instruction{"NOP", 0x1A, 1, 2, Implied}
	cpu.instructions[0x3A] = &Instruction{"NOP", 0x3A, 1, 2, Implied}
	cpu.instructions[0x5A] = &Instruction{"NOP", 0x5A, 1, 2, Implied}
	cpu.instructions[0x7A] = &Instruction{"NOP", 0x7A, 1, 2, Implied}
	cpu.instructions[0xDA] = &Instruction{"NOP", 0xDA, 1, 2, Implied}
	cpu.instructions[0xFA] = &Instruction{"NOP", 0xFA, 1, 2, Implied}
	cpu.instructions[0x80] = &Instruction{"NOP", 0x80, 2, 2, Immediate}
	cpu.instructions[0x82] = &Instruction{"NOP", 0x82, 2, 2, Immediate}
	cpu.instructions[0x89] = &Instruction{"NOP", 0x89, 2, 2, Immediate}
	cpu.instructions[0xC2] = &Instruction{"NOP", 0xC2, 2, 2, Immediate}
	cpu.instructions[0xE2] = &Instruction{"NOP", 0xE2, 2, 2, Immediate}
	cpu.instructions[0x04] = &Instruction{"NOP", 0x04, 2, 3, ZeroPage}
	cpu.instructions[0x44] = &Instruction{"NOP", 0x44, 2, 3, ZeroPage}
	cpu.instructions[0x64] = &Instruction{"NOP", 0x64, 2, 3, ZeroPage}
	cpu.instructions[0x14] = &Instruction{"NOP", 0x14, 2, 4, ZeroPageX}
	cpu.instructions[0x34] = &Instruction{"NOP", 0x34, 2, 4, ZeroPageX}
	cpu.instructions[0x54] = &Instruction{"NOP", 0x54, 2, 4, ZeroPageX}
	cpu.instructions[0x74] = &Instruction{"NOP", 0x74, 2, 4, ZeroPageX}
	cpu.instructions[0xD4] = &Instruction{"NOP", 0xD4, 2, 4, ZeroPageX}
	cpu.instructions[0xF4] = &Instruction{"NOP", 0xF4, 2, 4, ZeroPageX}
	cpu.instructions[0x0C] = &Instruction{"NOP", 0x0C, 3, 4, Absolute}
	cpu.instructions[0x1C] = &Instruction{"NOP", 0x1C, 3, 4, AbsoluteX}
	cpu.instructions[0x3C] = &Instruction{"NOP", 0x3C, 3, 4, AbsoluteX}
	cpu.instructions[0x5C] = &Instruction{"NOP", 0x5C, 3, 4, AbsoluteX}
	cpu.instructions[0x7C] = &Instruction{"NOP", 0x7C, 3, 4, AbsoluteX}
	cpu.instructions[0xDC] = &Instruction{"NOP", 0xDC, 3, 4, AbsoluteX}
	cpu.instructions[0xFC] = &Instruction{"NOP", 0xFC, 3, 4, AbsoluteX}

	// Unofficial Opcodes
	cpu.instructions[0xA7] = &Instruction{"LAX", 0xA7, 2, 3, ZeroPage}
	cpu.instructions[0xB7] = &Instruction{"LAX", 0xB7, 2, 4, ZeroPageY}
	cpu.instructions[0xAF] = &Instruction{"LAX", 0xAF, 3, 4, Absolute}
	cpu.instructions[0xBF] = &Instruction{"LAX", 0xBF, 3, 4, AbsoluteY}
	cpu.instructions[0xA3] = &Instruction{"LAX", 0xA3, 2, 6, IndexedIndirect}
	cpu.instructions[0xB3] = &Instruction{"LAX", 0xB3, 2, 5, IndirectIndexed}

	cpu.instructions[0x87] = &Instruction{"SAX", 0x87, 2, 3, ZeroPage}
	cpu.instructions[0x97] = &Instruction{"SAX", 0x97, 2, 4, ZeroPageY}
	cpu.instructions[0x8F] = &Instruction{"SAX", 0x8F, 3, 4, Absolute}
	cpu.instructions[0x83] = &Instruction{"SAX", 0x83, 2, 6, IndexedIndirect}

	cpu.instructions[0xEB] = &Instruction{"SBC", 0xEB, 2, 2, Immediate}

	cpu.instructions[0xC7] = &Instruction{"DCP", 0xC7, 2, 5, ZeroPage}
	cpu.instructions[0xD7] = &Instruction{"DCP", 0xD7, 2, 6, ZeroPageX}
	cpu.instructions[0xCF] = &Instruction{"DCP", 0xCF, 3, 6, Absolute}
	cpu.instructions[0xDF] = &Instruction{"DCP", 0xDF, 3, 7, AbsoluteX}
	cpu.instructions[0xDB] = &Instruction{"DCP", 0xDB, 3, 7, AbsoluteY}
	cpu.instructions[0xC3] = &Instruction{"DCP", 0xC3, 2, 8, IndexedIndirect}
	cpu.instructions[0xD3] = &Instruction{"DCP", 0xD3, 2, 8, IndirectIndexed}

	cpu.instructions[0xE7] = &Instruction{"ISB", 0xE7, 2, 5, ZeroPage}
	cpu.instructions[0xF7] = &Instruction{"ISB", 0xF7, 2, 6, ZeroPageX}
	cpu.instructions[0xEF] = &Instruction{"ISB", 0xEF, 3, 6, Absolute}
	cpu.instructions[0xFF] = &Instruction{"ISB", 0xFF, 3, 7, AbsoluteX}
	cpu.instructions[0xFB] = &Instruction{"ISB", 0xFB, 3, 7, AbsoluteY}
	cpu.instructions[0xE3] = &Instruction{"ISB", 0xE3, 2, 8, IndexedIndirect}
	cpu.instructions[0xF3] = &Instruction{"ISB", 0xF3, 2, 8, IndirectIndexed}

	cpu.instructions[0x07] = &Instruction{"SLO", 0x07, 2, 5, ZeroPage}
	cpu.instructions[0x17] = &Instruction{"SLO", 0x17, 2, 6, ZeroPageX}
	cpu.instructions[0x0F] = &Instruction{"SLO", 0x0F, 3, 6, Absolute}
	cpu.instructions[0x1F] = &Instruction{"SLO", 0x1F, 3, 7, AbsoluteX}
	cpu.instructions[0x1B] = &Instruction{"SLO", 0x1B, 3, 7, AbsoluteY}
	cpu.instructions[0x03] = &Instruction{"SLO", 0x03, 2, 8, IndexedIndirect}
	cpu.instructions[0x13] = &Instruction{"SLO", 0x13, 2, 8, IndirectIndexed}

	cpu.instructions[0x27] = &Instruction{"RLA", 0x27, 2, 5, ZeroPage}
	cpu.instructions[0x37] = &Instruction{"RLA", 0x37, 2, 6, ZeroPageX}
	cpu.instructions[0x2F] = &Instruction{"RLA", 0x2F, 3, 6, Absolute}
	cpu.instructions[0x3F] = &Instruction{"RLA", 0x3F, 3, 7, AbsoluteX}
	cpu.instructions[0x3B] = &Instruction{"RLA", 0x3B, 3, 7, AbsoluteY}
	cpu.instructions[0x23] = &Instruction{"RLA", 0x23, 2, 8, IndexedIndirect}
	cpu.instructions[0x33] = &Instruction{"RLA", 0x33, 2, 8, IndirectIndexed}

	cpu.instructions[0x47] = &Instruction{"SRE", 0x47, 2, 5, ZeroPage}
	cpu.instructions[0x57] = &Instruction{"SRE", 0x57, 2, 6, ZeroPageX}
	cpu.instructions[0x4F] = &Instruction{"SRE", 0x4F, 3, 6, Absolute}
	cpu.instructions[0x5F] = &Instruction{"SRE", 0x5F, 3, 7, AbsoluteX}
	cpu.instructions[0x5B] = &Instruction{"SRE", 0x5B, 3, 7, AbsoluteY}
	cpu.instructions[0x43] = &Instruction{"SRE", 0x43, 2, 8, IndexedIndirect}
	cpu.instructions[0x53] = &Instruction{"SRE", 0x53, 2, 8, IndirectIndexed}

	cpu.instructions[0x67] = &Instruction{"RRA", 0x67, 2, 5, ZeroPage}
	cpu.instructions[0x77] = &Instruction{"RRA", 0x77, 2, 6, ZeroPageX}
	cpu.instructions[0x6F] = &Instruction{"RRA", 0x6F, 3, 6, Absolute}
	cpu.instructions[0x7F] = &Instruction{"RRA", 0x7F, 3, 7, AbsoluteX}
	cpu.instructions[0x7B] = &Instruction{"RRA", 0x7B, 3, 7, AbsoluteY}
	cpu.instructions[0x63] = &Instruction{"RRA", 0x63, 2, 8, IndexedIndirect}
	cpu.instructions[0x73] = &Instruction{"RRA", 0x73, 2, 8, IndirectIndexed}
}

// CPU Debug Methods

// EnableDebugLogging enables/disables CPU instruction logging
func (cpu *CPU) EnableDebugLogging(enable bool) {
	cpu.enableDebugLogging = enable
}

// EnableLoopDetection enables/disables infinite loop detection
func (cpu *CPU) EnableLoopDetection(enable bool) {
	cpu.enableLoopDetection = enable
}

// detectInfiniteLoop detects when CPU is stuck at the same PC
func (cpu *CPU) detectInfiniteLoop(pc uint16, opcode uint8) {
	if pc == cpu.lastPC {
		cpu.pcStayCount++
		if cpu.pcStayCount > 100 { // Lower threshold for faster detection
			fmt.Printf("[CPU_LOOP] CPU stuck at PC=$%04X executing opcode=0x%02X for %d cycles\n",
				pc, opcode, cpu.pcStayCount)
			if cpu.pcStayCount%1000 == 0 { // Log every 1000 cycles
				cpu.logCPUState(pc, opcode)
			}
		}
	} else {
		cpu.pcStayCount = 0
	}
	cpu.lastPC = pc
}

// logInstruction logs CPU instruction execution
func (cpu *CPU) logInstruction(pc uint16, opcode uint8, instruction *Instruction) {
	name := "UNK"
	if instruction != nil {
		name = instruction.Name
	}
	
	fmt.Printf("[CPU_DEBUG] PC=$%04X: %s (0x%02X) | A=$%02X X=$%02X Y=$%02X SP=$%02X | %s\n",
		pc, name, opcode, cpu.A, cpu.X, cpu.Y, cpu.SP, cpu.getFlagsString())
}

// logCPUState logs detailed CPU state during infinite loops
func (cpu *CPU) logCPUState(pc uint16, opcode uint8) {
	instruction := cpu.instructions[opcode]
	name := "UNK"
	if instruction != nil {
		name = instruction.Name
	}
	
	// Read memory around PC for context
	mem1 := cpu.memory.Read(pc + 1)
	mem2 := cpu.memory.Read(pc + 2)
	
	fmt.Printf("[CPU_STATE] PC=$%04X: %s (0x%02X %02X %02X) | A=$%02X X=$%02X Y=$%02X SP=$%02X | %s | Cycles=%d\n",
		pc, name, opcode, mem1, mem2, cpu.A, cpu.X, cpu.Y, cpu.SP, cpu.getFlagsString(), cpu.cycles)
}

// getFlagsString returns CPU flags as string
func (cpu *CPU) getFlagsString() string {
	flags := ""
	if cpu.N { flags += "N" } else { flags += "-" }
	if cpu.V { flags += "V" } else { flags += "-" }
	flags += "-" // Unused flag
	if cpu.B { flags += "B" } else { flags += "-" }
	if cpu.D { flags += "D" } else { flags += "-" }
	if cpu.I { flags += "I" } else { flags += "-" }
	if cpu.Z { flags += "Z" } else { flags += "-" }
	if cpu.C { flags += "C" } else { flags += "-" }
	return flags
}
