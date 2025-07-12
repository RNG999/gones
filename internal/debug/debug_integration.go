// Package debug provides comprehensive debugging integration for color pipeline issues
package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ColorDebugSession represents a complete debugging session
type ColorDebugSession struct {
	outputDir       string
	pipelineDebug   *ColorPipelineDebugger
	frameDumper     *FrameDumper
	sessionID       string
	startTime       time.Time
	enabled         bool
	targetFrames    int
	currentFrame    uint64
}

// NewColorDebugSession creates a new comprehensive debugging session
func NewColorDebugSession(outputDir string) *ColorDebugSession {
	sessionID := fmt.Sprintf("debug_%s", time.Now().Format("20060102_150405"))
	sessionDir := filepath.Join(outputDir, sessionID)
	
	return &ColorDebugSession{
		outputDir:     sessionDir,
		pipelineDebug: NewColorPipelineDebugger(sessionDir),
		frameDumper:   NewFrameDumper(sessionDir),
		sessionID:     sessionID,
		startTime:     time.Now(),
		enabled:       false,
		targetFrames:  5, // Debug first 5 frames by default
		currentFrame:  0,
	}
}

// StartDebugging begins the debugging session
func (cds *ColorDebugSession) StartDebugging() error {
	if cds.enabled {
		return fmt.Errorf("debugging session already active")
	}
	
	// Create output directory
	if err := os.MkdirAll(cds.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create debug output directory: %v", err)
	}
	
	// Enable components
	cds.pipelineDebug.Enable()
	cds.frameDumper.Enable()
	
	// Configure for Super Mario Bros sky blue debugging
	cds.pipelineDebug.SetTargetColor(0x22) // Sky blue color index
	cds.frameDumper.SetMaxDumps(cds.targetFrames)
	cds.frameDumper.SetDumpInterval(1) // Dump every frame
	
	// Set global debugger
	debuggerMutex.Lock()
	globalDebugger = cds.pipelineDebug
	debuggerMutex.Unlock()
	
	cds.enabled = true
	
	// Create session info file
	return cds.createSessionInfo()
}

// StopDebugging ends the debugging session and generates reports
func (cds *ColorDebugSession) StopDebugging() error {
	if !cds.enabled {
		return fmt.Errorf("debugging session not active")
	}
	
	cds.enabled = false
	cds.pipelineDebug.Disable()
	cds.frameDumper.Disable()
	
	// Generate comprehensive report
	return cds.generateFinalReport()
}

// ProcessFrame processes a complete frame for debugging
func (cds *ColorDebugSession) ProcessFrame(frameBuffer [256 * 240]uint32, frameNum uint64) error {
	if !cds.enabled || frameNum >= uint64(cds.targetFrames) {
		return nil
	}
	
	cds.currentFrame = frameNum
	
	// Dump frame buffer
	if err := cds.frameDumper.DumpFrameBuffer(frameBuffer, frameNum); err != nil {
		return fmt.Errorf("failed to dump frame buffer: %v", err)
	}
	
	// Dump RGB breakdown
	if err := cds.frameDumper.DumpFrameBufferRGB(frameBuffer, frameNum); err != nil {
		return fmt.Errorf("failed to dump RGB frame buffer: %v", err)
	}
	
	// Dump color corruption analysis
	if err := cds.frameDumper.DumpColorCorruption(frameBuffer, frameNum); err != nil {
		return fmt.Errorf("failed to dump color corruption: %v", err)
	}
	
	return nil
}

// createSessionInfo creates a session information file
func (cds *ColorDebugSession) createSessionInfo() error {
	infoPath := filepath.Join(cds.outputDir, "session_info.txt")
	file, err := os.Create(infoPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	fmt.Fprintf(file, "Color Pipeline Debug Session\n")
	fmt.Fprintf(file, "===========================\n\n")
	fmt.Fprintf(file, "Session ID: %s\n", cds.sessionID)
	fmt.Fprintf(file, "Start Time: %s\n", cds.startTime.Format(time.RFC3339))
	fmt.Fprintf(file, "Output Directory: %s\n", cds.outputDir)
	fmt.Fprintf(file, "Target Frames: %d\n", cds.targetFrames)
	fmt.Fprintf(file, "\nObjective:\n")
	fmt.Fprintf(file, "Trace color index 0x22 (sky blue) through the rendering pipeline\n")
	fmt.Fprintf(file, "to identify where blue colors are being transformed to brown/yellow.\n")
	fmt.Fprintf(file, "\nExpected Results:\n")
	fmt.Fprintf(file, "- Color index 0x22 should map to RGB(100, 176, 255) = #64B0FF\n")
	fmt.Fprintf(file, "- This should appear as sky blue in Super Mario Bros\n")
	fmt.Fprintf(file, "- Any brown/yellow colors indicate pipeline corruption\n")
	
	return nil
}

// generateFinalReport creates a comprehensive analysis report
func (cds *ColorDebugSession) generateFinalReport() error {
	reportPath := filepath.Join(cds.outputDir, "final_analysis_report.txt")
	file, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	endTime := time.Now()
	duration := endTime.Sub(cds.startTime)
	
	fmt.Fprintf(file, "Color Pipeline Debug Session - Final Report\n")
	fmt.Fprintf(file, "==========================================\n\n")
	fmt.Fprintf(file, "Session ID: %s\n", cds.sessionID)
	fmt.Fprintf(file, "Duration: %v\n", duration)
	fmt.Fprintf(file, "End Time: %s\n", endTime.Format(time.RFC3339))
	fmt.Fprintf(file, "Frames Processed: %d\n", cds.currentFrame+1)
	
	// Analyze pipeline events
	events := cds.pipelineDebug.GetEvents()
	fmt.Fprintf(file, "\nPipeline Events Analysis:\n")
	fmt.Fprintf(file, "Total Events Recorded: %d\n", len(events))
	
	// Count events by stage
	stageCount := make(map[ColorStage]int)
	for _, event := range events {
		stageCount[event.Stage]++
	}
	
	fmt.Fprintf(file, "\nEvents by Stage:\n")
	for stage, count := range stageCount {
		fmt.Fprintf(file, "  %s: %d\n", stage, count)
	}
	
	// Analyze color corruption
	analysis := cds.pipelineDebug.AnalyzeColorCorruption()
	if analysis != nil {
		fmt.Fprintf(file, "\nColor Corruption Analysis:\n")
		fmt.Fprintf(file, "Total Transformation Events: %d\n", analysis.TransformationEvents)
		fmt.Fprintf(file, "Corruption by Stage:\n")
		for stage, count := range analysis.CorruptionStages {
			fmt.Fprintf(file, "  %s: %d corruptions\n", stage, count)
		}
		
		if len(analysis.SampleEvents) > 0 {
			fmt.Fprintf(file, "\nSample Corruption Events:\n")
			for i, event := range analysis.SampleEvents {
				if i >= 5 { // Limit to first 5 samples
					break
				}
				fmt.Fprintf(file, "  Frame %d, Pixel (%d,%d): 0x%08X -> 0x%08X (%s)\n",
					event.Frame, event.PixelX, event.PixelY,
					event.InputValue, event.OutputValue, event.Description)
			}
		}
	}
	
	// Key findings and recommendations
	fmt.Fprintf(file, "\nKey Findings:\n")
	
	skyBlueEvents := 0
	corruptedSkyBlue := 0
	
	for _, event := range events {
		if event.Stage == StageNESColorToRGB && event.InputValue == 0x22 {
			skyBlueEvents++
			if event.OutputValue != 0x64B0FF {
				corruptedSkyBlue++
				fmt.Fprintf(file, "- Sky blue corruption detected: 0x22 -> 0x%06X (expected 0x64B0FF)\n",
					event.OutputValue)
			}
		}
	}
	
	if skyBlueEvents > 0 {
		corruptionRate := float64(corruptedSkyBlue) / float64(skyBlueEvents) * 100
		fmt.Fprintf(file, "- Sky blue (0x22) events: %d\n", skyBlueEvents)
		fmt.Fprintf(file, "- Corrupted sky blue events: %d (%.1f%%)\n", corruptedSkyBlue, corruptionRate)
	}
	
	fmt.Fprintf(file, "\nRecommendations:\n")
	if corruptedSkyBlue > 0 {
		fmt.Fprintf(file, "- Color corruption detected in NES color to RGB conversion\n")
		fmt.Fprintf(file, "- Check nesColorToRGB function for palette accuracy\n")
		fmt.Fprintf(file, "- Verify color emphasis application is not causing corruption\n")
	} else {
		fmt.Fprintf(file, "- NES color conversion appears correct\n")
		fmt.Fprintf(file, "- Check SDL texture format and pixel format conversion\n")
		fmt.Fprintf(file, "- Verify frame buffer to SDL texture transfer\n")
	}
	
	fmt.Fprintf(file, "\nGenerated Files:\n")
	fmt.Fprintf(file, "- session_info.txt: Session configuration\n")
	fmt.Fprintf(file, "- color_pipeline_events.log: Detailed event log\n")
	fmt.Fprintf(file, "- color_comparison_report_*.txt: Expected vs actual colors\n")
	fmt.Fprintf(file, "- frame_*.txt: Frame buffer dumps\n")
	fmt.Fprintf(file, "- frame_rgb_*.txt: RGB component analysis\n")
	fmt.Fprintf(file, "- color_corruption_*.txt: Corruption detection\n")
	
	// Export pipeline events
	if err := cds.pipelineDebug.ExportEventsToFile("color_pipeline_events.log"); err != nil {
		fmt.Fprintf(file, "\nWarning: Failed to export pipeline events: %v\n", err)
	}
	
	// Create comparison report
	if err := cds.pipelineDebug.CreateColorComparisonReport(); err != nil {
		fmt.Fprintf(file, "\nWarning: Failed to create comparison report: %v\n", err)
	}
	
	return nil
}

// QuickSkyBlueDebugging sets up debugging specifically for the sky blue corruption issue
func QuickSkyBlueDebugging(outputDir string) (*ColorDebugSession, error) {
	session := NewColorDebugSession(outputDir)
	
	// Configure for sky blue debugging
	session.targetFrames = 3 // Just debug first 3 frames
	session.pipelineDebug.SetTargetColor(0x22) // Sky blue
	session.pipelineDebug.SetTraceAllPixels(false) // Only trace target color
	
	// Set up frame dumper with sky blue filter
	session.frameDumper.SetPixelFilter(CreateSkyBluePixelFilter())
	
	if err := session.StartDebugging(); err != nil {
		return nil, err
	}
	
	return session, nil
}

// GetSessionOutputDir returns the output directory for this session
func (cds *ColorDebugSession) GetSessionOutputDir() string {
	return cds.outputDir
}

// IsEnabled returns whether debugging is currently enabled
func (cds *ColorDebugSession) IsEnabled() bool {
	return cds.enabled
}