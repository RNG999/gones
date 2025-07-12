// Package app provides emulator integration for the main application.
package app

import (
	"fmt"
	"sync"
	"time"

	"gones/internal/bus"
)

// Emulator manages the emulation loop and timing
type Emulator struct {
	bus    *bus.Bus
	config *Config

	// Optimized timing control
	lastUpdateTime  time.Time
	accumulatedTime time.Duration
	targetFrameTime time.Duration
	cyclesPerFrame  uint64

	// Adaptive timing for smooth performance
	frameTiming  *AdaptiveFrameTiming
	timingBuffer *CircularTimingBuffer

	// Frame management with pooling
	frameComplete   bool
	frameBuffer     []uint32
	audioSamples    []float32
	frameBufferPool *FrameBufferPool

	// Enhanced performance monitoring
	actualFrameTime  time.Duration
	emulationTime    time.Duration
	cycleCount       uint64
	frameCount       uint64
	averageFrameTime time.Duration
	performanceStats *EmulatorPerformanceStats

	// State tracking
	isRunning     bool
	lastResetTime time.Time

	// Optimization flags
	adaptiveTimingEnabled bool
	performanceMode       PerformanceMode
}

// NewEmulator creates a new emulator instance with fixed timing for accuracy
func NewEmulator(bus *bus.Bus, config *Config) *Emulator {
	emulator := &Emulator{
		bus:                   bus,
		config:                config,
		targetFrameTime:       time.Duration(16666667) * time.Nanosecond, // Precise 60 FPS (16.666ms)
		cyclesPerFrame:        29781,                                     // NTSC: exactly 29,781 CPU cycles per frame
		frameBuffer:           make([]uint32, 256*240),
		audioSamples:          make([]float32, 0, 1024),
		isRunning:             false,
		lastResetTime:         time.Now(),
		adaptiveTimingEnabled: false,                                     // Disabled for consistent timing
		performanceMode:       PerformanceModeAccuracy,                   // Use accuracy mode for real-time speed
	}

	// Skip complex optimizations that can cause timing variance
	// emulator.initializeOptimizations()

	emulator.Reset()
	return emulator
}

// initializeOptimizations sets up performance optimization structures
func (e *Emulator) initializeOptimizations() {
	// Initialize adaptive frame timing
	e.frameTiming = &AdaptiveFrameTiming{
		targetFrameTime:    e.targetFrameTime,
		measuredFrameTimes: make([]time.Duration, 0, 60), // Store last 60 frames
		adaptationStrength: 0.1,                          // 10% adjustment strength
		stabilityThreshold: time.Microsecond * 100,       // 100μs stability threshold
		lastAdjustmentTime: time.Now(),
	}

	// Initialize timing buffer for performance analysis
	e.timingBuffer = NewCircularTimingBuffer(300) // 5 seconds at 60 FPS
	e.frameTiming.performanceHistory = e.timingBuffer

	// Initialize frame buffer pool for memory efficiency
	e.frameBufferPool = NewFrameBufferPool(3, 256*240) // Pool of 3 frame buffers

	// Initialize performance statistics
	e.performanceStats = &EmulatorPerformanceStats{
		frameTimeHistory:     NewCircularTimingBuffer(180), // 3 seconds of history
		emulationTimeHistory: NewCircularTimingBuffer(180),
		gcPauseHistory:       NewCircularTimingBuffer(60), // 1 second of GC history
		lastStatsUpdate:      time.Now(),
	}
}

// Reset resets the emulator state with simple initialization
func (e *Emulator) Reset() {
	e.lastUpdateTime = time.Now()
	e.frameComplete = false
	e.actualFrameTime = 0
	e.emulationTime = 0
	e.cycleCount = 0
	e.frameCount = 0
	e.averageFrameTime = 0
	e.lastResetTime = time.Now()

	// Clear frame buffer
	for i := range e.frameBuffer {
		e.frameBuffer[i] = 0
	}

	// Clear audio samples
	e.audioSamples = e.audioSamples[:0]
}

// Start starts the emulator
func (e *Emulator) Start() {
	e.isRunning = true
	e.lastUpdateTime = time.Now()
}

// Stop stops the emulator
func (e *Emulator) Stop() {
	e.isRunning = false
}

// Update updates the emulator for exactly one frame with fixed timing
func (e *Emulator) Update() error {
	if !e.isRunning {
		return nil
	}

	frameStartTime := time.Now()

	// Run exactly one frame of emulation every time Update() is called
	// This ensures consistent timing when called at 60Hz by Ebitengine
	if err := e.runFrameFixed(); err != nil {
		return fmt.Errorf("frame execution error: %v", err)
	}

	// Update basic performance metrics
	e.actualFrameTime = time.Since(frameStartTime)
	e.updatePerformanceMetricsSimple(frameStartTime)

	return nil
}

// runFrameFixed executes exactly one frame worth of emulation with fixed timing
func (e *Emulator) runFrameFixed() error {
	emulationStart := time.Now()

	// Run emulation for exactly one frame (29,781 CPU cycles for NTSC)
	// This ensures consistent real-time emulation speed
	startCycles := e.bus.GetCycleCount()
	targetCycles := startCycles + e.cyclesPerFrame

	// Execute exactly the target number of cycles
	for e.bus.GetCycleCount() < targetCycles {
		e.bus.Step()
	}

	// Update frame count
	e.frameCount++

	// Get frame buffer from PPU
	nesFrameBuffer := e.bus.GetFrameBuffer()
	if len(nesFrameBuffer) == len(e.frameBuffer) {
		copy(e.frameBuffer, nesFrameBuffer)
	}

	// Get audio samples from APU
	nesSamples := e.bus.GetAudioSamples()
	if len(nesSamples) > 0 {
		e.updateAudioSamplesSimple(nesSamples)
	}

	// Update timing metrics
	e.emulationTime = time.Since(emulationStart)
	e.cycleCount = e.bus.GetCycleCount()

	return nil
}

// updateAudioSamplesSimple updates audio samples with simple copying
func (e *Emulator) updateAudioSamplesSimple(nesSamples []float32) {
	// Simple audio sample copying without complex optimizations
	if cap(e.audioSamples) < len(nesSamples) {
		e.audioSamples = make([]float32, len(nesSamples))
	} else {
		e.audioSamples = e.audioSamples[:len(nesSamples)]
	}
	copy(e.audioSamples, nesSamples)
}

// handleFrameDrop handles frame drops and timing adjustments
func (e *Emulator) handleFrameDrop() {
	if e.performanceStats != nil {
		e.performanceStats.mu.Lock()
		e.performanceStats.droppedFrames++
		e.performanceStats.mu.Unlock()
	}

	// Adjust timing if we're consistently dropping frames
	if e.adaptiveTimingEnabled && e.frameTiming != nil {
		e.frameTiming.HandleFrameDrop()
	}
}

// updatePerformanceMetricsSimple updates basic emulation performance metrics
func (e *Emulator) updatePerformanceMetricsSimple(frameStartTime time.Time) {
	// Simple average frame time calculation
	if e.averageFrameTime == 0 {
		e.averageFrameTime = e.actualFrameTime
	} else {
		// Simple weighted average
		e.averageFrameTime = time.Duration(
			float64(e.averageFrameTime)*0.95 + float64(e.actualFrameTime)*0.05,
		)
	}
}

// GetFrameBuffer returns the current frame buffer
func (e *Emulator) GetFrameBuffer() []uint32 {
	return e.frameBuffer
}

// GetAudioSamples returns the current audio samples
func (e *Emulator) GetAudioSamples() []float32 {
	return e.audioSamples
}

// IsFrameComplete returns whether the current frame is complete
func (e *Emulator) IsFrameComplete() bool {
	complete := e.frameComplete
	e.frameComplete = false // Reset flag
	return complete
}

// GetFrameCount returns the current frame count
func (e *Emulator) GetFrameCount() uint64 {
	return e.frameCount
}

// GetCycleCount returns the current CPU cycle count
func (e *Emulator) GetCycleCount() uint64 {
	return e.cycleCount
}

// GetEmulationTime returns the time spent in emulation for the last frame
func (e *Emulator) GetEmulationTime() time.Duration {
	return e.emulationTime
}

// GetActualFrameTime returns the actual frame time including rendering
func (e *Emulator) GetActualFrameTime() time.Duration {
	return e.actualFrameTime
}

// GetAverageFrameTime returns the average frame time
func (e *Emulator) GetAverageFrameTime() time.Duration {
	return e.averageFrameTime
}

// GetTargetFrameTime returns the target frame time (60 FPS)
func (e *Emulator) GetTargetFrameTime() time.Duration {
	return e.targetFrameTime
}

// GetEmulationSpeed returns the emulation speed as a percentage of real-time
func (e *Emulator) GetEmulationSpeed() float64 {
	if e.targetFrameTime == 0 {
		return 0.0
	}

	return float64(e.targetFrameTime) / float64(e.actualFrameTime) * 100.0
}

// GetCPUUsage returns the CPU usage percentage for emulation
func (e *Emulator) GetCPUUsage() float64 {
	if e.actualFrameTime == 0 {
		return 0.0
	}

	return float64(e.emulationTime) / float64(e.actualFrameTime) * 100.0
}

// IsRunning returns whether the emulator is running
func (e *Emulator) IsRunning() bool {
	return e.isRunning
}

// GetUptime returns the emulator uptime since last reset
func (e *Emulator) GetUptime() time.Duration {
	return time.Since(e.lastResetTime)
}

// SetTargetFrameRate sets the target frame rate
func (e *Emulator) SetTargetFrameRate(fps int) {
	if fps > 0 {
		e.targetFrameTime = time.Duration(1000000/fps) * time.Microsecond
	}
}

// SetCyclesPerFrame sets the number of CPU cycles per frame
func (e *Emulator) SetCyclesPerFrame(cycles uint64) {
	e.cyclesPerFrame = cycles
}

// StepFrame executes exactly one frame of emulation
func (e *Emulator) StepFrame() error {
	if e.bus == nil {
		return fmt.Errorf("bus not initialized")
	}

	emulationStart := time.Now()

	// Execute one frame worth of cycles
	startCycles := e.bus.GetCycleCount()
	targetCycles := startCycles + e.cyclesPerFrame

	for e.bus.GetCycleCount() < targetCycles {
		e.bus.Step()
	}

	// Update frame count
	e.frameCount++

	// Get updated frame buffer
	nesFrameBuffer := e.bus.GetFrameBuffer()
	if len(nesFrameBuffer) == len(e.frameBuffer) {
		copy(e.frameBuffer, nesFrameBuffer)
	}

	// Get updated audio samples
	nesSamples := e.bus.GetAudioSamples()
	if len(nesSamples) > 0 {
		if cap(e.audioSamples) < len(nesSamples) {
			e.audioSamples = make([]float32, len(nesSamples))
		} else {
			e.audioSamples = e.audioSamples[:len(nesSamples)]
		}
		copy(e.audioSamples, nesSamples)
	}

	// Update timing
	e.emulationTime = time.Since(emulationStart)
	e.cycleCount = e.bus.GetCycleCount()

	return nil
}

// StepInstruction executes one CPU instruction
func (e *Emulator) StepInstruction() error {
	if e.bus == nil {
		return fmt.Errorf("bus not initialized")
	}

	e.bus.Step()
	e.cycleCount = e.bus.GetCycleCount()

	return nil
}

// GetCPUState returns the current CPU state for debugging
func (e *Emulator) GetCPUState() bus.CPUState {
	if e.bus == nil {
		return bus.CPUState{}
	}

	return e.bus.GetCPUState()
}

// GetPPUState returns the current PPU state for debugging
func (e *Emulator) GetPPUState() bus.PPUState {
	if e.bus == nil {
		return bus.PPUState{}
	}

	return e.bus.GetPPUState()
}

// GetPerformanceStats returns comprehensive performance statistics with optimizations
func (e *Emulator) GetPerformanceStats() EmulatorStats {
	stats := EmulatorStats{
		FrameCount:       e.frameCount,
		CycleCount:       e.cycleCount,
		EmulationTime:    e.emulationTime,
		ActualFrameTime:  e.actualFrameTime,
		AverageFrameTime: e.averageFrameTime,
		TargetFrameTime:  e.targetFrameTime,
		EmulationSpeed:   e.GetEmulationSpeed(),
		CPUUsage:         e.GetCPUUsage(),
		Uptime:           e.GetUptime(),
		IsRunning:        e.isRunning,
	}

	// Add enhanced performance metrics
	if e.performanceStats != nil {
		e.performanceStats.mu.RLock()
		stats.FrameJitter = e.performanceStats.CalculateJitter()
		stats.DroppedFrames = e.performanceStats.droppedFrames
		stats.AdaptationCount = e.performanceStats.adaptationCount
		stats.GCImpact = e.performanceStats.GetAverageGCPause()
		stats.MemoryEfficiency = e.calculateMemoryEfficiency()
		e.performanceStats.mu.RUnlock()
	}

	return stats
}

// calculateMemoryEfficiency calculates memory usage efficiency
func (e *Emulator) calculateMemoryEfficiency() float64 {
	// Calculate efficiency based on allocation patterns and GC frequency
	// Higher values indicate better memory efficiency
	if e.performanceStats == nil {
		return 1.0
	}

	// Simple efficiency metric: lower GC impact = higher efficiency
	gcImpact := e.performanceStats.GetAverageGCPause()
	if gcImpact == 0 {
		return 1.0
	}

	// Calculate efficiency as inverse of GC impact relative to frame time
	frameTime := float64(e.targetFrameTime.Nanoseconds())
	gcRatio := float64(gcImpact.Nanoseconds()) / frameTime

	return 1.0 / (1.0 + gcRatio)
}

// SetPerformanceMode sets the emulator performance optimization mode
func (e *Emulator) SetPerformanceMode(mode PerformanceMode) {
	e.performanceMode = mode

	// Adjust optimizations based on performance mode
	switch mode {
	case PerformanceModeAccuracy:
		e.adaptiveTimingEnabled = false
		if e.frameTiming != nil {
			e.frameTiming.SetAdaptationStrength(0.05) // Very conservative
		}
	case PerformanceModeBalanced:
		e.adaptiveTimingEnabled = true
		if e.frameTiming != nil {
			e.frameTiming.SetAdaptationStrength(0.1) // Moderate adaptation
		}
	case PerformanceModeSpeed:
		e.adaptiveTimingEnabled = true
		if e.frameTiming != nil {
			e.frameTiming.SetAdaptationStrength(0.2) // Aggressive adaptation
		}
	}
}

// GetPerformanceMode returns the current performance mode
func (e *Emulator) GetPerformanceMode() PerformanceMode {
	return e.performanceMode
}

// EnableAdaptiveTiming enables or disables adaptive timing
func (e *Emulator) EnableAdaptiveTiming(enabled bool) {
	e.adaptiveTimingEnabled = enabled
}

// PerformanceMode defines emulator performance optimization levels
type PerformanceMode int

const (
	PerformanceModeAccuracy PerformanceMode = iota // Prioritize accuracy over speed
	PerformanceModeBalanced                        // Balance accuracy and performance
	PerformanceModeSpeed                           // Prioritize speed over accuracy
)

// AdaptiveFrameTiming provides intelligent frame timing adjustments
type AdaptiveFrameTiming struct {
	mu                 sync.RWMutex
	targetFrameTime    time.Duration
	measuredFrameTimes []time.Duration
	currentAdjustment  time.Duration
	adaptationStrength float64
	stabilityThreshold time.Duration
	lastAdjustmentTime time.Time
	performanceHistory *CircularTimingBuffer
}

// CircularTimingBuffer efficiently stores timing measurements
type CircularTimingBuffer struct {
	buffer   []time.Duration
	index    int
	size     int
	capacity int
	mu       sync.RWMutex
}

// FrameBufferPool manages frame buffer reuse to reduce allocations
type FrameBufferPool struct {
	pool chan []uint32
	size int
}

// EmulatorPerformanceStats contains detailed performance metrics
type EmulatorPerformanceStats struct {
	mu                   sync.RWMutex
	frameTimeHistory     *CircularTimingBuffer
	emulationTimeHistory *CircularTimingBuffer
	gcPauseHistory       *CircularTimingBuffer
	droppedFrames        uint64
	targetMissCount      uint64
	adaptationCount      uint64
	lastGCTime           time.Time
	lastStatsUpdate      time.Time
}

// EmulatorStats contains emulator performance statistics
type EmulatorStats struct {
	FrameCount       uint64
	CycleCount       uint64
	EmulationTime    time.Duration
	ActualFrameTime  time.Duration
	AverageFrameTime time.Duration
	TargetFrameTime  time.Duration
	EmulationSpeed   float64
	CPUUsage         float64
	Uptime           time.Duration
	IsRunning        bool

	// Enhanced performance metrics
	FrameJitter      time.Duration
	DroppedFrames    uint64
	AdaptationCount  uint64
	GCImpact         time.Duration
	MemoryEfficiency float64
}

// Cleanup cleans up emulator resources with optimization cleanup
func (e *Emulator) Cleanup() error {
	e.Stop()

	// Clear buffers
	e.frameBuffer = nil
	e.audioSamples = nil

	// Cleanup optimization structures
	if e.frameBufferPool != nil {
		e.frameBufferPool.Cleanup()
		e.frameBufferPool = nil
	}

	if e.frameTiming != nil {
		e.frameTiming = nil
	}

	if e.timingBuffer != nil {
		e.timingBuffer = nil
	}

	if e.performanceStats != nil {
		e.performanceStats = nil
	}

	return nil
}

// AdaptiveFrameTiming implementation

// NewAdaptiveFrameTiming creates a new adaptive frame timing system
func NewAdaptiveFrameTiming(targetFrameTime time.Duration) *AdaptiveFrameTiming {
	return &AdaptiveFrameTiming{
		targetFrameTime:    targetFrameTime,
		measuredFrameTimes: make([]time.Duration, 0, 60),
		adaptationStrength: 0.1,
		stabilityThreshold: time.Microsecond * 100,
		lastAdjustmentTime: time.Now(),
	}
}

// GetAdjustedFrameTime returns the current adjusted frame time
func (aft *AdaptiveFrameTiming) GetAdjustedFrameTime() time.Duration {
	aft.mu.RLock()
	defer aft.mu.RUnlock()
	return aft.targetFrameTime + aft.currentAdjustment
}

// RecordFrameTime records a frame time measurement for adaptive adjustments
func (aft *AdaptiveFrameTiming) RecordFrameTime(frameTime time.Duration) {
	aft.mu.Lock()
	defer aft.mu.Unlock()

	// Add to measurement history
	aft.measuredFrameTimes = append(aft.measuredFrameTimes, frameTime)
	if len(aft.measuredFrameTimes) > 60 {
		aft.measuredFrameTimes = aft.measuredFrameTimes[1:]
	}

	// Adapt timing if we have enough measurements
	if len(aft.measuredFrameTimes) >= 10 && time.Since(aft.lastAdjustmentTime) > time.Millisecond*100 {
		aft.adaptTiming()
		aft.lastAdjustmentTime = time.Now()
	}
}

// adaptTiming adjusts frame timing based on performance measurements
func (aft *AdaptiveFrameTiming) adaptTiming() {
	if len(aft.measuredFrameTimes) == 0 {
		return
	}

	// Calculate average frame time
	var total time.Duration
	for _, ft := range aft.measuredFrameTimes {
		total += ft
	}
	avgFrameTime := total / time.Duration(len(aft.measuredFrameTimes))

	// Calculate deviation from target
	deviation := avgFrameTime - aft.targetFrameTime

	// Only adjust if deviation is significant
	if deviation > aft.stabilityThreshold || deviation < -aft.stabilityThreshold {
		// Calculate adjustment with adaptive strength
		adjustment := time.Duration(float64(deviation) * aft.adaptationStrength)

		// Apply adjustment with bounds checking
		maxAdjustment := aft.targetFrameTime / 10 // Max 10% adjustment
		if adjustment > maxAdjustment {
			adjustment = maxAdjustment
		} else if adjustment < -maxAdjustment {
			adjustment = -maxAdjustment
		}

		aft.currentAdjustment -= adjustment
	}
}

// HandleFrameDrop handles frame drop events
func (aft *AdaptiveFrameTiming) HandleFrameDrop() {
	aft.mu.Lock()
	defer aft.mu.Unlock()

	// Slightly relax timing to prevent future drops
	relaxation := time.Microsecond * 50
	aft.currentAdjustment += relaxation

	// Bound the adjustment
	maxAdjustment := aft.targetFrameTime / 10
	if aft.currentAdjustment > maxAdjustment {
		aft.currentAdjustment = maxAdjustment
	}
}

// SetAdaptationStrength sets the adaptation strength
func (aft *AdaptiveFrameTiming) SetAdaptationStrength(strength float64) {
	aft.mu.Lock()
	defer aft.mu.Unlock()
	aft.adaptationStrength = strength
}

// Reset resets the adaptive timing system
func (aft *AdaptiveFrameTiming) Reset() {
	aft.mu.Lock()
	defer aft.mu.Unlock()
	aft.measuredFrameTimes = aft.measuredFrameTimes[:0]
	aft.currentAdjustment = 0
	aft.lastAdjustmentTime = time.Now()
}

// CircularTimingBuffer implementation

// NewCircularTimingBuffer creates a new circular timing buffer
func NewCircularTimingBuffer(capacity int) *CircularTimingBuffer {
	return &CircularTimingBuffer{
		buffer:   make([]time.Duration, capacity),
		capacity: capacity,
		index:    0,
		size:     0,
	}
}

// Add adds a timing measurement to the buffer
func (ctb *CircularTimingBuffer) Add(duration time.Duration) {
	ctb.mu.Lock()
	defer ctb.mu.Unlock()

	ctb.buffer[ctb.index] = duration
	ctb.index = (ctb.index + 1) % ctb.capacity

	if ctb.size < ctb.capacity {
		ctb.size++
	}
}

// GetAverage calculates the average of stored durations
func (ctb *CircularTimingBuffer) GetAverage() time.Duration {
	ctb.mu.RLock()
	defer ctb.mu.RUnlock()

	if ctb.size == 0 {
		return 0
	}

	var total time.Duration
	for i := 0; i < ctb.size; i++ {
		total += ctb.buffer[i]
	}

	return total / time.Duration(ctb.size)
}

// GetVariance calculates the variance of stored durations
func (ctb *CircularTimingBuffer) GetVariance() time.Duration {
	ctb.mu.RLock()
	defer ctb.mu.RUnlock()

	if ctb.size < 2 {
		return 0
	}

	avg := ctb.GetAverage()
	var variance int64

	for i := 0; i < ctb.size; i++ {
		diff := int64(ctb.buffer[i] - avg)
		variance += diff * diff
	}

	return time.Duration(variance / int64(ctb.size))
}

// Reset clears the buffer
func (ctb *CircularTimingBuffer) Reset() {
	ctb.mu.Lock()
	defer ctb.mu.Unlock()
	ctb.index = 0
	ctb.size = 0
}

// FrameBufferPool implementation

// NewFrameBufferPool creates a new frame buffer pool
func NewFrameBufferPool(poolSize, bufferSize int) *FrameBufferPool {
	pool := &FrameBufferPool{
		pool: make(chan []uint32, poolSize),
		size: bufferSize,
	}

	// Pre-allocate buffers
	for i := 0; i < poolSize; i++ {
		pool.pool <- make([]uint32, bufferSize)
	}

	return pool
}

// Get retrieves a frame buffer from the pool
func (fbp *FrameBufferPool) Get() []uint32 {
	select {
	case buffer := <-fbp.pool:
		return buffer
	default:
		// Pool is empty, create new buffer
		return make([]uint32, fbp.size)
	}
}

// Put returns a frame buffer to the pool
func (fbp *FrameBufferPool) Put(buffer []uint32) {
	if len(buffer) != fbp.size {
		return // Wrong size, don't return to pool
	}

	select {
	case fbp.pool <- buffer:
		// Successfully returned to pool
	default:
		// Pool is full, let GC handle the buffer
	}
}

// Cleanup closes the pool
func (fbp *FrameBufferPool) Cleanup() {
	close(fbp.pool)
}

// EmulatorPerformanceStats implementation

// RecordFrameTime records a frame time measurement
func (eps *EmulatorPerformanceStats) RecordFrameTime(frameTime time.Duration) {
	eps.mu.Lock()
	defer eps.mu.Unlock()

	if eps.frameTimeHistory != nil {
		eps.frameTimeHistory.Add(frameTime)
	}
}

// RecordEmulationTime records an emulation time measurement
func (eps *EmulatorPerformanceStats) RecordEmulationTime(emulationTime time.Duration) {
	eps.mu.Lock()
	defer eps.mu.Unlock()

	if eps.emulationTimeHistory != nil {
		eps.emulationTimeHistory.Add(emulationTime)
	}
}

// CalculateJitter calculates frame timing jitter
func (eps *EmulatorPerformanceStats) CalculateJitter() time.Duration {
	eps.mu.RLock()
	defer eps.mu.RUnlock()

	if eps.frameTimeHistory != nil {
		return eps.frameTimeHistory.GetVariance()
	}
	return 0
}

// GetAverageGCPause returns the average GC pause time
func (eps *EmulatorPerformanceStats) GetAverageGCPause() time.Duration {
	eps.mu.RLock()
	defer eps.mu.RUnlock()

	if eps.gcPauseHistory != nil {
		return eps.gcPauseHistory.GetAverage()
	}
	return 0
}

// UpdateStats updates performance statistics
func (eps *EmulatorPerformanceStats) UpdateStats() {
	eps.mu.Lock()
	defer eps.mu.Unlock()

	// Update statistics periodically
	if time.Since(eps.lastStatsUpdate) > time.Second {
		// This would typically collect GC stats, memory usage, etc.
		eps.lastStatsUpdate = time.Now()
	}
}

// Reset resets performance statistics
func (eps *EmulatorPerformanceStats) Reset() {
	eps.mu.Lock()
	defer eps.mu.Unlock()

	if eps.frameTimeHistory != nil {
		eps.frameTimeHistory.Reset()
	}
	if eps.emulationTimeHistory != nil {
		eps.emulationTimeHistory.Reset()
	}
	if eps.gcPauseHistory != nil {
		eps.gcPauseHistory.Reset()
	}

	eps.droppedFrames = 0
	eps.targetMissCount = 0
	eps.adaptationCount = 0
	eps.lastGCTime = time.Time{}
	eps.lastStatsUpdate = time.Now()
}
