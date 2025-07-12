# Color Pipeline Debugging Tools

This package provides comprehensive debugging tools to trace color values through the entire NES rendering pipeline, from PPU palette lookups to SDL2 rendering output. These tools are specifically designed to identify where color corruption occurs, particularly the issue where blue colors appear brown/yellow in Super Mario Bros.

## Overview

The debugging system consists of several components:

1. **ColorPipelineDebugger**: Traces color transformations through all pipeline stages
2. **FrameDumper**: Dumps frame buffer contents for analysis
3. **ColorDebugSession**: Manages complete debugging sessions
4. **Hook Functions**: Integration points in PPU and SDL code

## Color Pipeline Stages

The debugger tracks colors through these stages:

- `StageColorIndexLookup`: Palette RAM lookup (address → color index)
- `StageNESColorToRGB`: NES color index to RGB conversion
- `StageColorEmphasis`: Color emphasis application
- `StageFrameBuffer`: Frame buffer storage
- `StageSDLTextureUpdate`: SDL texture update
- `StageSDLRender`: Final SDL rendering

## Quick Start

### Enable Super Mario Bros Color Debugging

```go
import "gones/internal/debug"

// Start debugging session for Super Mario Bros sky blue issue
session, err := debug.EnableSuperMarioBrosColorDebugging()
if err != nil {
    log.Fatal(err)
}
defer session.StopDebugging()

// Process frames (this would happen in your main loop)
for frameNum := 0; frameNum < 5; frameNum++ {
    // ... get frame buffer from PPU ...
    session.ProcessFrame(frameBuffer, uint64(frameNum))
}
```

### Simple Color Environment Setup

```go
// Initialize debugging environment
debug.CreateColorDebugEnvironment("debug_output")

// Track specific color index (sky blue)
debug.TraceColorIndex0x22()

// Track specific pixel coordinates
debug.TracePixelAt(128, 120)

// Later, generate debug report
debug.DumpColorDebugReport()
```

## Debug Output Files

When debugging is enabled, several files are generated:

### Session Files
- `session_info.txt`: Session configuration and objectives
- `final_analysis_report.txt`: Comprehensive analysis with findings

### Event Logs
- `color_pipeline_events.log`: Detailed log of all color transformations
- `color_comparison_report_*.txt`: Expected vs actual color comparisons

### Frame Dumps
- `frame_*.txt`: Raw frame buffer dumps in hex format
- `frame_rgb_*.txt`: Frame buffers with RGB component breakdown
- `color_corruption_*.txt`: Analysis of corrupted colors

## Integration with Existing Code

The debugging hooks are already integrated into:

### PPU (internal/ppu/ppu.go)
- `nesColorToRGB()`: Traces color index to RGB conversion
- `getColor()`: Traces palette RAM lookups
- Frame buffer writes: Traces pixel storage

### SDL Renderer (internal/sdl/renderer.go)
- Texture updates: Traces color format conversions
- Surface rendering: Traces final pixel output

## Expected Color Values

For reference, here are the expected RGB values for common NES colors:

| Index | RGB      | Color Name | Usage in SMB |
|-------|----------|------------|--------------|
| 0x00  | #666666  | Gray       | Background   |
| 0x0F  | #000000  | Black      | Outlines     |
| 0x22  | #64B0FF  | Sky Blue   | Sky/Water    |
| 0x30  | #FFFEFF  | White      | Clouds       |

## Common Corruption Patterns

The debugger specifically looks for these corruption patterns:

- **0x22 → Brown (#8B4513)**: Color emphasis bug
- **0x22 → Red (#FF0000)**: Red screen bug  
- **Any color → Gray**: Greyscale mode incorrectly enabled

## Performance Considerations

The debugging system is designed to have minimal performance impact when disabled:

- **Disabled**: Near-zero overhead (simple boolean checks)
- **Enabled**: Moderate overhead suitable for debugging sessions
- **Filtering**: Only tracks specific colors/pixels to reduce data volume

## Configuration Options

### Target Color Tracking
```go
debugger.SetTargetColor(0x22) // Track only sky blue
debugger.SetTraceAllPixels(true) // Track all colors
```

### Pixel Region Tracking
```go
debugger.SetTargetPixel(x, y) // Track specific pixel
```

### Frame Dumping
```go
frameDumper.SetMaxDumps(10) // Limit to 10 frames
frameDumper.SetDumpInterval(5) // Dump every 5th frame
frameDumper.SetPixelFilter(CreateSkyBluePixelFilter()) // Filter pixels
```

## Analyzing Results

### Color Pipeline Events
The event log shows the transformation of colors through each stage:

```
Timestamp            Frame    Line Cyc  X    Y    Stage                Input      Output     Description
15:04:05.123        1        50   0    105  55   color_index_lookup   0x00003F01 0x00000022 Palette RAM lookup at 0x3F01 -> 0x22
15:04:05.123        1        50   0    105  55   nes_color_to_rgb     0x00000022 0x0064B0FF NES color 0x22 -> RGB(100,176,255)
15:04:05.123        1        50   0    105  55   frame_buffer         0x00000000 0x0064B0FF Frame buffer[13465] = RGB(100,176,255)
```

### Corruption Detection
The analyzer identifies where colors deviate from expected values:

```
Color Corruption Analysis:
Total Events: 150
Transformation Events: 45
Corruption by Stage:
  nes_color_to_rgb: 5 corruptions
  color_emphasis: 2 corruptions
```

### Frame Buffer Analysis
Frame dumps show the actual pixel values stored:

```
Color Frequency Analysis:
Color      | Count | Percentage
-----------|-------|----------
#666666   | 45000 | 73.24%   RGB(102,102,102)  // Background gray
#64B0FF   |  5000 |  8.14%   RGB(100,176,255)  // Correct sky blue
#8B4513   |   500 |  0.81%   RGB(139, 69, 19)  // Corrupted brown!
```

## Troubleshooting

### No Events Recorded
- Ensure `debug.EnableColorDebugging()` is called
- Check that the target color/pixel is actually being rendered
- Verify hooks are being called in the rendering pipeline

### Missing Output Files
- Check that the output directory is writable
- Ensure `session.StopDebugging()` is called to generate final reports
- Verify sufficient disk space for frame dumps

### High Memory Usage
- Reduce the number of target frames
- Use pixel filters to limit data collection
- Set specific target colors instead of tracing all pixels

## Example: Debugging Sky Blue Corruption

```go
// Start focused debugging session
session, err := debug.QuickSkyBlueDebugging("mario_debug")
if err != nil {
    log.Fatal(err)
}

// Process first few frames of Super Mario Bros
for frame := 0; frame < 3; frame++ {
    // ... run emulation for one frame ...
    frameBuffer := ppu.GetFrameBuffer()
    session.ProcessFrame(frameBuffer, uint64(frame))
}

// Generate analysis
session.StopDebugging()

// Print quick analysis
debug.AnalyzeColorPipeline()
```

This will generate a complete analysis showing exactly where the sky blue color (0x22) is being corrupted from the expected RGB(100,176,255) to brown/yellow colors.