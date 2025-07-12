// Package apu implements the Audio Processing Unit for the NES.
package apu

// APU represents the NES Audio Processing Unit
type APU struct {
	// APU channels
	pulse1   PulseChannel
	pulse2   PulseChannel
	triangle TriangleChannel
	noise    NoiseChannel
	dmc      DMCChannel

	// Frame counter
	frameCounter     uint16
	frameMode        bool  // false = 4-step, true = 5-step
	frameIRQEnable   bool  // Frame counter IRQ enable
	frameCounterStep uint8 // Current step in frame counter
	frameIRQFlag     bool  // Frame counter IRQ flag

	// Channel enable flags
	channelEnable [5]bool // pulse1, pulse2, triangle, noise, dmc

	// Audio generation
	sampleBuffer     []float32
	sampleRate       int     // Target sample rate (e.g., 44100 Hz)
	cpuFrequency     float64 // NES CPU frequency
	cycleAccumulator float64 // For sample rate conversion

	// Timing
	cycles uint64
}

// PulseChannel represents a pulse wave channel
type PulseChannel struct {
	// Control registers
	dutyCycle       uint8 // 0-3 (12.5%, 25%, 50%, 75%)
	envelopeLoop    bool  // Length counter halt / envelope loop
	envelopeDisable bool  // Constant volume flag
	volume          uint8 // Volume/envelope (0-15)

	// Sweep unit
	sweepEnable  bool
	sweepPeriod  uint8 // 0-7
	sweepNegate  bool  // Pitch bend direction
	sweepShift   uint8 // 0-7
	sweepReload  bool  // Sweep reload flag
	sweepCounter uint8 // Internal sweep counter

	// Timer
	timer        uint16 // 11-bit timer
	timerCounter uint16 // Current timer value

	// Length counter
	lengthCounter uint8 // Length counter value
	lengthHalt    bool  // Length counter halt flag

	// Envelope
	envelopeStart   bool  // Start flag
	envelopeCounter uint8 // Envelope counter
	envelopeDivider uint8 // Envelope divider

	// Waveform generation
	dutyIndex    uint8 // Current position in duty cycle
	output       uint8 // Current output level
	sequencerPos uint8 // Position in 8-step sequencer
}

// TriangleChannel represents the triangle wave channel
type TriangleChannel struct {
	// Control register
	lengthCounterHalt bool  // Length counter halt / linear counter control
	linearCounterLoad uint8 // Linear counter reload value (0-127)

	// Timer
	timer        uint16 // 11-bit timer
	timerCounter uint16 // Current timer value

	// Length counter
	lengthCounter uint8 // Length counter value

	// Linear counter
	linearCounter       uint8 // Linear counter value
	linearCounterReload bool  // Linear counter reload flag

	// Waveform generation
	sequencerPos uint8 // Position in 32-step triangle sequence
	output       uint8 // Current output level
}

// NoiseChannel represents the noise channel
type NoiseChannel struct {
	// Control registers
	envelopeLoop    bool  // Length counter halt / envelope loop
	envelopeDisable bool  // Constant volume flag
	volume          uint8 // Volume/envelope (0-15)

	// Mode and period
	mode         bool   // false = 32k steps, true = 93 steps
	periodIndex  uint8  // Index into period table (0-15)
	timerCounter uint16 // Current timer value

	// Length counter
	lengthCounter uint8 // Length counter value
	lengthHalt    bool  // Length counter halt flag

	// Envelope
	envelopeStart   bool  // Start flag
	envelopeCounter uint8 // Envelope counter
	envelopeDivider uint8 // Envelope divider

	// Noise generation
	shiftRegister uint16 // 15-bit LFSR
	output        uint8  // Current output level
}

// DMCChannel represents the Delta Modulation Channel
type DMCChannel struct {
	// Control registers
	irqEnable bool  // IRQ enable flag
	loop      bool  // Loop flag
	rateIndex uint8 // Rate index (0-15)

	// Direct load
	outputLevel uint8 // 7-bit DAC value

	// Sample playback
	sampleAddress uint16 // Current sample address
	sampleLength  uint16 // Remaining sample bytes

	// Internal state
	timerCounter      uint16 // Current timer value
	sampleBuffer      uint8  // Current sample byte
	sampleBufferBits  uint8  // Remaining bits in sample buffer
	sampleBufferEmpty bool   // Sample buffer empty flag
	bytesRemaining    uint16 // Bytes remaining in sample
	currentAddress    uint16 // Current read address

	// IRQ flag
	irqFlag bool // DMC IRQ flag

	// Output
	output uint8 // Current output level
}

// New creates a new APU instance
func New() *APU {
	apu := &APU{
		sampleBuffer:   make([]float32, 0, 4096),
		sampleRate:     44100,     // Standard audio sample rate
		cpuFrequency:   1789773.0, // NTSC CPU frequency
		frameMode:      false,     // Default to 4-step mode
		frameIRQEnable: true,      // Frame IRQ enabled by default
	}

	// Initialize noise shift register
	apu.noise.shiftRegister = 1

	return apu
}

// Reset resets the APU to its initial state
func (apu *APU) Reset() {
	// Reset all channels
	apu.pulse1 = PulseChannel{}
	apu.pulse2 = PulseChannel{}
	apu.triangle = TriangleChannel{}
	apu.noise = NoiseChannel{shiftRegister: 1} // Initialize LFSR
	apu.dmc = DMCChannel{}

	// Reset frame counter
	apu.frameCounter = 0
	apu.frameCounterStep = 0
	apu.frameMode = false
	apu.frameIRQEnable = true
	apu.frameIRQFlag = false

	// Reset channel enables
	for i := range apu.channelEnable {
		apu.channelEnable[i] = false
	}

	// Reset timing
	apu.cycles = 0
	apu.cycleAccumulator = 0

	// Clear sample buffer
	apu.sampleBuffer = apu.sampleBuffer[:0]
}

// Step advances the APU by one cycle
func (apu *APU) Step() {
	apu.cycles++

	// Step frame counter
	apu.stepFrameCounter()

	// Step each channel's timer
	apu.stepChannelTimers()

	// Generate audio sample if needed
	apu.generateSample()
}

// stepFrameCounter handles frame counter timing
func (apu *APU) stepFrameCounter() {
	apu.frameCounter++

	if apu.frameMode {
		// 5-step mode
		switch apu.frameCounter {
		case 7457:
			apu.clockEnvelopeAndLinear()
		case 14913:
			apu.clockEnvelopeAndLinear()
			apu.clockLengthAndSweep()
		case 22371:
			apu.clockEnvelopeAndLinear()
		case 37281:
			apu.clockEnvelopeAndLinear()
			apu.clockLengthAndSweep()
			apu.frameCounter = 0
			apu.frameCounterStep = 0
		}
	} else {
		// 4-step mode
		switch apu.frameCounter {
		case 7457:
			apu.clockEnvelopeAndLinear()
		case 14913:
			apu.clockEnvelopeAndLinear()
			apu.clockLengthAndSweep()
		case 22371:
			apu.clockEnvelopeAndLinear()
		case 29829:
			apu.clockEnvelopeAndLinear()
			apu.clockLengthAndSweep()
		case 29830:
			// Frame IRQ
			if apu.frameIRQEnable {
				apu.frameIRQFlag = true
			}
			apu.frameCounter = 0
			apu.frameCounterStep = 0
		}
	}
}

// clockEnvelopeAndLinear clocks envelope and linear counter units
func (apu *APU) clockEnvelopeAndLinear() {
	apu.clockPulseEnvelope(&apu.pulse1)
	apu.clockPulseEnvelope(&apu.pulse2)
	apu.clockNoiseEnvelope(&apu.noise)
	apu.clockTriangleLinear(&apu.triangle)
}

// clockLengthAndSweep clocks length counters and sweep units
func (apu *APU) clockLengthAndSweep() {
	apu.clockPulseLength(&apu.pulse1)
	apu.clockPulseSweep(&apu.pulse1, true) // Pulse 1 has different sweep behavior
	apu.clockPulseLength(&apu.pulse2)
	apu.clockPulseSweep(&apu.pulse2, false) // Pulse 2
	apu.clockTriangleLength(&apu.triangle)
	apu.clockNoiseLength(&apu.noise)
}

// stepChannelTimers steps the timer for each channel
func (apu *APU) stepChannelTimers() {
	if apu.channelEnable[0] {
		apu.stepPulseTimer(&apu.pulse1)
	}
	if apu.channelEnable[1] {
		apu.stepPulseTimer(&apu.pulse2)
	}
	if apu.channelEnable[2] {
		apu.stepTriangleTimer(&apu.triangle)
	}
	if apu.channelEnable[3] {
		apu.stepNoiseTimer(&apu.noise)
	}
	if apu.channelEnable[4] {
		apu.stepDMCTimer(&apu.dmc)
	}
}

// generateSample generates an audio sample and adds it to the buffer
func (apu *APU) generateSample() {
	// Convert from CPU frequency to sample rate
	apu.cycleAccumulator += float64(apu.sampleRate) / apu.cpuFrequency

	if apu.cycleAccumulator >= 1.0 {
		apu.cycleAccumulator -= 1.0

		// Mix all channels
		pulse1Out := apu.getPulseOutput(&apu.pulse1)
		pulse2Out := apu.getPulseOutput(&apu.pulse2)
		triangleOut := apu.getTriangleOutput(&apu.triangle)
		noiseOut := apu.getNoiseOutput(&apu.noise)
		dmcOut := apu.getDMCOutput(&apu.dmc)

		// Apply NES mixer formula
		sample := apu.mixChannels(pulse1Out, pulse2Out, triangleOut, noiseOut, dmcOut)

		// Add to sample buffer
		apu.sampleBuffer = append(apu.sampleBuffer, sample)
	}
}

// WriteRegister writes to an APU register
func (apu *APU) WriteRegister(address uint16, value uint8) {
	switch address {
	// Pulse Channel 1
	case 0x4000:
		apu.writePulseControl(&apu.pulse1, value)
	case 0x4001:
		apu.writePulseSweep(&apu.pulse1, value)
	case 0x4002:
		apu.writePulseTimerLow(&apu.pulse1, value)
	case 0x4003:
		apu.writePulseTimerHigh(&apu.pulse1, value)

	// Pulse Channel 2
	case 0x4004:
		apu.writePulseControl(&apu.pulse2, value)
	case 0x4005:
		apu.writePulseSweep(&apu.pulse2, value)
	case 0x4006:
		apu.writePulseTimerLow(&apu.pulse2, value)
	case 0x4007:
		apu.writePulseTimerHigh(&apu.pulse2, value)

	// Triangle Channel
	case 0x4008:
		apu.writeTriangleControl(value)
	case 0x400A:
		apu.writeTriangleTimerLow(value)
	case 0x400B:
		apu.writeTriangleTimerHigh(value)

	// Noise Channel
	case 0x400C:
		apu.writeNoiseControl(value)
	case 0x400E:
		apu.writeNoisePeriod(value)
	case 0x400F:
		apu.writeNoiseLength(value)

	// DMC Channel
	case 0x4010:
		apu.writeDMCControl(value)
	case 0x4011:
		apu.writeDMCDirectLoad(value)
	case 0x4012:
		apu.writeDMCSampleAddress(value)
	case 0x4013:
		apu.writeDMCSampleLength(value)

	// Control registers
	case 0x4015:
		apu.writeChannelEnable(value)
	case 0x4017:
		apu.writeFrameCounter(value)
	}
}

// GetSamples returns the current audio samples
func (apu *APU) GetSamples() []float32 {
	samples := make([]float32, len(apu.sampleBuffer))
	copy(samples, apu.sampleBuffer)
	apu.sampleBuffer = apu.sampleBuffer[:0]
	return samples
}

// ReadStatus reads the APU status register ($4015)
func (apu *APU) ReadStatus() uint8 {
	status := uint8(0)

	// Channel length counter status
	if apu.pulse1.lengthCounter > 0 {
		status |= 0x01
	}
	if apu.pulse2.lengthCounter > 0 {
		status |= 0x02
	}
	if apu.triangle.lengthCounter > 0 {
		status |= 0x04
	}
	if apu.noise.lengthCounter > 0 {
		status |= 0x08
	}
	if apu.dmc.bytesRemaining > 0 {
		status |= 0x10
	}

	// Frame IRQ flag
	if apu.frameIRQFlag {
		status |= 0x40
	}

	// DMC IRQ flag
	if apu.dmc.irqFlag {
		status |= 0x80
	}

	// Reading $4015 clears the frame IRQ flag
	apu.frameIRQFlag = false

	return status
}

// Length counter lookup table
var lengthTable = [32]uint8{
	10, 254, 20, 2, 40, 4, 80, 6,
	160, 8, 60, 10, 14, 12, 26, 14,
	12, 16, 24, 8, 48, 6, 96, 4,
	192, 2, 72, 16, 28, 32, 52, 2,
}

// Duty cycle lookup table (8 steps each)
var dutyTable = [4][8]uint8{
	{0, 1, 0, 0, 0, 0, 0, 0}, // 12.5%
	{0, 1, 1, 0, 0, 0, 0, 0}, // 25%
	{0, 1, 1, 1, 1, 0, 0, 0}, // 50%
	{1, 0, 0, 1, 1, 1, 1, 1}, // 75%
}

// Triangle wave sequence (32 steps)
var triangleTable = [32]uint8{
	15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
}

// Noise period table (NTSC)
var noisePeriodTable = [16]uint16{
	4, 8, 16, 32, 64, 96, 128, 160,
	202, 254, 380, 508, 762, 1016, 2034, 4068,
}

// DMC rate table (NTSC)
var dmcRateTable = [16]uint16{
	428, 380, 340, 320, 286, 254, 226, 214,
	190, 160, 142, 128, 106, 84, 72, 54,
}

// Pulse channel register write methods

// writePulseControl writes to pulse control register ($4000/$4004)
func (apu *APU) writePulseControl(pulse *PulseChannel, value uint8) {
	pulse.dutyCycle = (value >> 6) & 0x03
	pulse.envelopeLoop = (value & 0x20) != 0
	pulse.lengthHalt = pulse.envelopeLoop
	pulse.envelopeDisable = (value & 0x10) != 0
	pulse.volume = value & 0x0F
	pulse.envelopeStart = true
}

// writePulseSweep writes to pulse sweep register ($4001/$4005)
func (apu *APU) writePulseSweep(pulse *PulseChannel, value uint8) {
	pulse.sweepEnable = (value & 0x80) != 0
	pulse.sweepPeriod = (value >> 4) & 0x07
	pulse.sweepNegate = (value & 0x08) != 0
	pulse.sweepShift = value & 0x07
	pulse.sweepReload = true
}

// writePulseTimerLow writes to pulse timer low register ($4002/$4006)
func (apu *APU) writePulseTimerLow(pulse *PulseChannel, value uint8) {
	pulse.timer = (pulse.timer & 0xFF00) | uint16(value)
}

// writePulseTimerHigh writes to pulse timer high register ($4003/$4007)
func (apu *APU) writePulseTimerHigh(pulse *PulseChannel, value uint8) {
	pulse.timer = (pulse.timer & 0x00FF) | (uint16(value&0x07) << 8)
	pulse.lengthCounter = lengthTable[(value>>3)&0x1F]
	pulse.envelopeStart = true
	pulse.dutyIndex = 0 // Reset duty cycle position
}

// stepPulseTimer steps the pulse channel timer
func (apu *APU) stepPulseTimer(pulse *PulseChannel) {
	if pulse.timerCounter == 0 {
		pulse.timerCounter = pulse.timer
		pulse.sequencerPos = (pulse.sequencerPos + 1) & 0x07
	} else {
		pulse.timerCounter--
	}
}

// clockPulseEnvelope clocks the pulse envelope unit
func (apu *APU) clockPulseEnvelope(pulse *PulseChannel) {
	if pulse.envelopeStart {
		pulse.envelopeStart = false
		pulse.envelopeCounter = 15
		pulse.envelopeDivider = pulse.volume
	} else if pulse.envelopeDivider == 0 {
		pulse.envelopeDivider = pulse.volume
		if pulse.envelopeCounter > 0 {
			pulse.envelopeCounter--
		} else if pulse.envelopeLoop {
			pulse.envelopeCounter = 15
		}
	} else {
		pulse.envelopeDivider--
	}
}

// clockPulseLength clocks the pulse length counter
func (apu *APU) clockPulseLength(pulse *PulseChannel) {
	if !pulse.lengthHalt && pulse.lengthCounter > 0 {
		pulse.lengthCounter--
	}
}

// clockPulseSweep clocks the pulse sweep unit
func (apu *APU) clockPulseSweep(pulse *PulseChannel, isPulse1 bool) {
	if pulse.sweepCounter == 0 && pulse.sweepEnable && pulse.sweepShift > 0 {
		changeAmount := pulse.timer >> pulse.sweepShift
		if pulse.sweepNegate {
			if isPulse1 {
				// Pulse 1 uses one's complement
				pulse.timer = pulse.timer - changeAmount - 1
			} else {
				// Pulse 2 uses two's complement
				pulse.timer = pulse.timer - changeAmount
			}
		} else {
			pulse.timer = pulse.timer + changeAmount
		}
	}

	if pulse.sweepCounter == 0 || pulse.sweepReload {
		pulse.sweepCounter = pulse.sweepPeriod
		pulse.sweepReload = false
	} else {
		pulse.sweepCounter--
	}
}

// getPulseOutput gets the current pulse channel output
func (apu *APU) getPulseOutput(pulse *PulseChannel) uint8 {
	if pulse.lengthCounter == 0 || pulse.timer < 8 || pulse.timer > 0x7FF {
		return 0
	}

	if dutyTable[pulse.dutyCycle][pulse.sequencerPos] == 0 {
		return 0
	}

	if pulse.envelopeDisable {
		return pulse.volume
	}
	return pulse.envelopeCounter
}

// Triangle channel register write methods

// writeTriangleControl writes to triangle control register ($4008)
func (apu *APU) writeTriangleControl(value uint8) {
	apu.triangle.lengthCounterHalt = (value & 0x80) != 0
	apu.triangle.linearCounterLoad = value & 0x7F
	apu.triangle.linearCounterReload = true
}

// writeTriangleTimerLow writes to triangle timer low register ($400A)
func (apu *APU) writeTriangleTimerLow(value uint8) {
	apu.triangle.timer = (apu.triangle.timer & 0xFF00) | uint16(value)
}

// writeTriangleTimerHigh writes to triangle timer high register ($400B)
func (apu *APU) writeTriangleTimerHigh(value uint8) {
	apu.triangle.timer = (apu.triangle.timer & 0x00FF) | (uint16(value&0x07) << 8)
	apu.triangle.lengthCounter = lengthTable[(value>>3)&0x1F]
	apu.triangle.linearCounterReload = true
}

// stepTriangleTimer steps the triangle channel timer
func (apu *APU) stepTriangleTimer(triangle *TriangleChannel) {
	if triangle.timerCounter == 0 {
		triangle.timerCounter = triangle.timer
		if triangle.lengthCounter > 0 && triangle.linearCounter > 0 {
			triangle.sequencerPos = (triangle.sequencerPos + 1) & 0x1F
		}
	} else {
		triangle.timerCounter--
	}
}

// clockTriangleLinear clocks the triangle linear counter
func (apu *APU) clockTriangleLinear(triangle *TriangleChannel) {
	if triangle.linearCounterReload {
		triangle.linearCounter = triangle.linearCounterLoad
	} else if triangle.linearCounter > 0 {
		triangle.linearCounter--
	}

	if !triangle.lengthCounterHalt {
		triangle.linearCounterReload = false
	}
}

// clockTriangleLength clocks the triangle length counter
func (apu *APU) clockTriangleLength(triangle *TriangleChannel) {
	if !triangle.lengthCounterHalt && triangle.lengthCounter > 0 {
		triangle.lengthCounter--
	}
}

// getTriangleOutput gets the current triangle channel output
func (apu *APU) getTriangleOutput(triangle *TriangleChannel) uint8 {
	if triangle.lengthCounter == 0 || triangle.linearCounter == 0 || triangle.timer < 2 {
		return 0
	}
	return triangleTable[triangle.sequencerPos]
}

// Noise channel register write methods

// writeNoiseControl writes to noise control register ($400C)
func (apu *APU) writeNoiseControl(value uint8) {
	apu.noise.envelopeLoop = (value & 0x20) != 0
	apu.noise.lengthHalt = apu.noise.envelopeLoop
	apu.noise.envelopeDisable = (value & 0x10) != 0
	apu.noise.volume = value & 0x0F
	apu.noise.envelopeStart = true
}

// writeNoisePeriod writes to noise period register ($400E)
func (apu *APU) writeNoisePeriod(value uint8) {
	apu.noise.mode = (value & 0x80) != 0
	apu.noise.periodIndex = value & 0x0F
}

// writeNoiseLength writes to noise length register ($400F)
func (apu *APU) writeNoiseLength(value uint8) {
	apu.noise.lengthCounter = lengthTable[(value>>3)&0x1F]
	apu.noise.envelopeStart = true
}

// stepNoiseTimer steps the noise channel timer
func (apu *APU) stepNoiseTimer(noise *NoiseChannel) {
	if noise.timerCounter == 0 {
		noise.timerCounter = noisePeriodTable[noise.periodIndex]

		// Clock shift register
		feedback := noise.shiftRegister & 0x01
		if noise.mode {
			// Mode 1: feedback from bits 0 and 6
			feedback ^= (noise.shiftRegister >> 6) & 0x01
		} else {
			// Mode 0: feedback from bits 0 and 1
			feedback ^= (noise.shiftRegister >> 1) & 0x01
		}

		noise.shiftRegister = (noise.shiftRegister >> 1) | (feedback << 14)
	} else {
		noise.timerCounter--
	}
}

// clockNoiseEnvelope clocks the noise envelope unit
func (apu *APU) clockNoiseEnvelope(noise *NoiseChannel) {
	if noise.envelopeStart {
		noise.envelopeStart = false
		noise.envelopeCounter = 15
		noise.envelopeDivider = noise.volume
	} else if noise.envelopeDivider == 0 {
		noise.envelopeDivider = noise.volume
		if noise.envelopeCounter > 0 {
			noise.envelopeCounter--
		} else if noise.envelopeLoop {
			noise.envelopeCounter = 15
		}
	} else {
		noise.envelopeDivider--
	}
}

// clockNoiseLength clocks the noise length counter
func (apu *APU) clockNoiseLength(noise *NoiseChannel) {
	if !noise.lengthHalt && noise.lengthCounter > 0 {
		noise.lengthCounter--
	}
}

// getNoiseOutput gets the current noise channel output
func (apu *APU) getNoiseOutput(noise *NoiseChannel) uint8 {
	if noise.lengthCounter == 0 || (noise.shiftRegister&0x01) != 0 {
		return 0
	}

	if noise.envelopeDisable {
		return noise.volume
	}
	return noise.envelopeCounter
}

// DMC channel register write methods

// writeDMCControl writes to DMC control register ($4010)
func (apu *APU) writeDMCControl(value uint8) {
	apu.dmc.irqEnable = (value & 0x80) != 0
	apu.dmc.loop = (value & 0x40) != 0
	apu.dmc.rateIndex = value & 0x0F

	if !apu.dmc.irqEnable {
		apu.dmc.irqFlag = false
	}
}

// writeDMCDirectLoad writes to DMC direct load register ($4011)
func (apu *APU) writeDMCDirectLoad(value uint8) {
	apu.dmc.outputLevel = value & 0x7F
}

// writeDMCSampleAddress writes to DMC sample address register ($4012)
func (apu *APU) writeDMCSampleAddress(value uint8) {
	apu.dmc.sampleAddress = 0xC000 + (uint16(value) << 6)
}

// writeDMCSampleLength writes to DMC sample length register ($4013)
func (apu *APU) writeDMCSampleLength(value uint8) {
	apu.dmc.sampleLength = (uint16(value) << 4) + 1
}

// stepDMCTimer steps the DMC channel timer
func (apu *APU) stepDMCTimer(dmc *DMCChannel) {
	if dmc.timerCounter == 0 {
		dmc.timerCounter = dmcRateTable[dmc.rateIndex]

		if !dmc.sampleBufferEmpty {
			// Clock output unit
			if dmc.sampleBufferBits == 0 {
				// No more bits in buffer
				dmc.sampleBufferEmpty = true

				if dmc.bytesRemaining > 0 {
					// Load next sample byte
					// TODO: Implement CPU memory access for DMC
					dmc.sampleBuffer = 0 // Placeholder
					dmc.sampleBufferBits = 8
					dmc.sampleBufferEmpty = false
					dmc.bytesRemaining--
					dmc.currentAddress++

					if dmc.bytesRemaining == 0 {
						if dmc.loop {
							// Restart sample
							dmc.currentAddress = dmc.sampleAddress
							dmc.bytesRemaining = dmc.sampleLength
						} else if dmc.irqEnable {
							dmc.irqFlag = true
						}
					}
				}
			} else {
				// Process next bit
				if (dmc.sampleBuffer & 0x01) != 0 {
					if dmc.outputLevel <= 125 {
						dmc.outputLevel += 2
					}
				} else {
					if dmc.outputLevel >= 2 {
						dmc.outputLevel -= 2
					}
				}

				dmc.sampleBuffer >>= 1
				dmc.sampleBufferBits--
			}
		}
	} else {
		dmc.timerCounter--
	}
}

// getDMCOutput gets the current DMC channel output
func (apu *APU) getDMCOutput(dmc *DMCChannel) uint8 {
	return dmc.outputLevel
}

// Control register methods

// writeChannelEnable writes to channel enable register ($4015)
func (apu *APU) writeChannelEnable(value uint8) {
	apu.channelEnable[0] = (value & 0x01) != 0 // Pulse 1
	apu.channelEnable[1] = (value & 0x02) != 0 // Pulse 2
	apu.channelEnable[2] = (value & 0x04) != 0 // Triangle
	apu.channelEnable[3] = (value & 0x08) != 0 // Noise
	apu.channelEnable[4] = (value & 0x10) != 0 // DMC

	// Clear length counters for disabled channels
	if !apu.channelEnable[0] {
		apu.pulse1.lengthCounter = 0
	}
	if !apu.channelEnable[1] {
		apu.pulse2.lengthCounter = 0
	}
	if !apu.channelEnable[2] {
		apu.triangle.lengthCounter = 0
	}
	if !apu.channelEnable[3] {
		apu.noise.lengthCounter = 0
	}
	if !apu.channelEnable[4] {
		apu.dmc.bytesRemaining = 0
	} else if apu.dmc.bytesRemaining == 0 {
		// Start DMC if enabled and no bytes remaining
		apu.dmc.currentAddress = apu.dmc.sampleAddress
		apu.dmc.bytesRemaining = apu.dmc.sampleLength
	}

	// Clear DMC IRQ flag
	apu.dmc.irqFlag = false
}

// writeFrameCounter writes to frame counter register ($4017)
func (apu *APU) writeFrameCounter(value uint8) {
	apu.frameMode = (value & 0x80) != 0
	apu.frameIRQEnable = (value & 0x40) == 0

	if !apu.frameIRQEnable {
		apu.frameIRQFlag = false
	}

	// Reset frame counter
	apu.frameCounter = 0
	apu.frameCounterStep = 0

	// If 5-step mode, immediately clock all units
	if apu.frameMode {
		apu.clockEnvelopeAndLinear()
		apu.clockLengthAndSweep()
	}
}

// mixChannels applies the NES audio mixer formula
func (apu *APU) mixChannels(pulse1, pulse2, triangle, noise, dmc uint8) float32 {
	// Pulse mixing
	pulseSum := float64(pulse1 + pulse2)
	var pulseOut float64
	if pulseSum != 0 {
		pulseOut = 95.88 / ((8128.0 / pulseSum) + 100.0)
	}

	// TND mixing
	tndSum := (float64(triangle) / 8227.0) + (float64(noise) / 12241.0) + (float64(dmc) / 22638.0)
	var tndOut float64
	if tndSum != 0 {
		tndOut = 159.79 / ((1.0 / tndSum) + 100.0)
	}

	// Final output
	output := pulseOut + tndOut

	// Scale to -1.0 to 1.0 range
	return float32(output/30.0 - 1.0)
}

// GetFrameIRQ returns the current frame counter IRQ flag
func (apu *APU) GetFrameIRQ() bool {
	return apu.frameIRQFlag
}

// GetDMCIRQ returns the current DMC IRQ flag
func (apu *APU) GetDMCIRQ() bool {
	return apu.dmc.irqFlag
}

// SetSampleRate sets the target audio sample rate
func (apu *APU) SetSampleRate(rate int) {
	apu.sampleRate = rate
	apu.cycleAccumulator = 0 // Reset accumulator when sample rate changes
}

// GetSampleRate returns the current sample rate
func (apu *APU) GetSampleRate() int {
	return apu.sampleRate
}

// GetChannelOutput returns the output level for a specific channel (for debugging)
func (apu *APU) GetChannelOutput(channel int) uint8 {
	if !apu.channelEnable[channel] {
		return 0
	}

	switch channel {
	case 0:
		return apu.getPulseOutput(&apu.pulse1)
	case 1:
		return apu.getPulseOutput(&apu.pulse2)
	case 2:
		return apu.getTriangleOutput(&apu.triangle)
	case 3:
		return apu.getNoiseOutput(&apu.noise)
	case 4:
		return apu.getDMCOutput(&apu.dmc)
	default:
		return 0
	}
}

// IsChannelEnabled returns whether a channel is enabled
func (apu *APU) IsChannelEnabled(channel int) bool {
	if channel < 0 || channel >= len(apu.channelEnable) {
		return false
	}
	return apu.channelEnable[channel]
}
