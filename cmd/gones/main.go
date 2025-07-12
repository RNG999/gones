// Package main implements the gones NES emulator executable.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gones/internal/app"
	"gones/internal/version"
)

func main() {
	// Parse command line flags
	var (
		romFile    = flag.String("rom", "", "Path to NES ROM file (optional for GUI mode)")
		configFile = flag.String("config", "", "Path to configuration file")
		debug      = flag.Bool("debug", false, "Enable debug mode")
		nogui      = flag.Bool("nogui", false, "Run without GUI (headless mode)")
		help       = flag.Bool("help", false, "Show help message")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	if *version {
		printVersion()
		os.Exit(0)
	}

	// Set up graceful shutdown
	setupGracefulShutdown()

	fmt.Println("🎮 gones - Go NES Emulator Starting...")

	// Determine config file path
	configPath := *configFile
	if configPath == "" {
		configPath = app.GetDefaultConfigPath()
	}

	// Create application
	application, err := app.NewApplicationWithMode(configPath, *nogui)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}
	
	// Force headless backend only when explicitly requested with -nogui
	if *nogui {
		config := application.GetConfig()
		config.Video.Backend = "headless"
		fmt.Println("🖥️  Headless mode requested")
	}
	defer func() {
		if err := application.Cleanup(); err != nil {
			log.Printf("Application cleanup error: %v", err)
		}
	}()

	// Apply debug settings
	if *debug {
		config := application.GetConfig()
		config.UpdateDebug(true, true, true)
		application.ApplyDebugSettings()
		fmt.Println("🐛 Debug mode enabled")
	}

	// Load ROM if specified
	if *romFile != "" {
		fmt.Printf("📁 Loading ROM: %s\n", *romFile)
		if err := application.LoadROM(*romFile); err != nil {
			log.Fatalf("Failed to load ROM: %v", err)
		}
		fmt.Println("✅ ROM loaded successfully")
		
		// Re-apply debug settings after ROM load (PPU might be recreated)
		if *debug {
			application.ApplyDebugSettings()
		}
	}

	if *nogui {
		// Run in headless mode (for testing or automation)
		fmt.Println("Running in headless mode...")
		if *romFile == "" {
			log.Fatal("ROM file required for headless mode")
		}
		runHeadlessMode(application)
	} else {
		// Run full GUI application
		fmt.Println("🖥️  Starting GUI mode...")
		if err := runGUIMode(application); err != nil {
			log.Fatalf("GUI mode failed: %v", err)
		}
	}

	fmt.Println("👋 Emulator shutting down...")
}

// runGUIMode runs the full GUI application
func runGUIMode(application *app.Application) error {
	fmt.Println("🚀 Initializing GUI application...")

	// Display startup information
	config := application.GetConfig()
	windowWidth, windowHeight := config.GetWindowResolution()
	fmt.Printf("   Window: %dx%d (Scale: %dx)\n", windowWidth, windowHeight, config.Window.Scale)
	fmt.Printf("   Audio: %s (%d Hz, %.0f%% volume)\n",
		enabledString(config.Audio.Enabled),
		config.Audio.SampleRate,
		config.Audio.Volume*100)
	fmt.Printf("   Video: %s, %s, VSync: %s\n",
		config.Video.Filter,
		config.Video.AspectRatio,
		enabledString(config.Video.VSync))

	// Start the application
	fmt.Println("🎯 Starting main application loop...")
	if err := application.Run(); err != nil {
		return fmt.Errorf("application run failed: %v", err)
	}

	// Display shutdown statistics
	fmt.Printf("📊 Session Statistics:\n")
	fmt.Printf("   Frames rendered: %d\n", application.GetFrameCount())
	fmt.Printf("   Session time: %v\n", application.GetUptime())
	fmt.Printf("   Average FPS: %.1f\n", application.GetFPS())

	return nil
}

// runHeadlessMode runs the emulator without GUI (for testing/automation)
func runHeadlessMode(application *app.Application) {
	fmt.Println("Running emulator in headless mode...")
	fmt.Println("実行中: 120フレーム（約2秒）でフレームバッファをダンプします")

	// ヘッドレスモードで実際にエミュレーションを実行
	bus := application.GetBus()
	if bus == nil {
		fmt.Println("❌ バスが初期化されていません")
		return
	}

	// 120フレーム実行（約2秒間）
	targetFrames := 120
	for frame := 0; frame < targetFrames; frame++ {
		// 1フレーム分のサイクル実行
		for cycles := 0; cycles < 29780; cycles++ {
			bus.Step()
		}

		// 特定フレームでフレームバッファを出力
		if frame == 30 || frame == 60 || frame == 119 {
			fmt.Printf("📸 フレーム %d のスクリーンショット作成中...\n", frame+1)
			saveFrameBufferAsPPM(bus.PPU.GetFrameBuffer(), fmt.Sprintf("frame_%03d.ppm", frame+1))
			analyzeFrameBuffer(bus.PPU.GetFrameBuffer(), frame+1)
		}

		// 進捗表示
		if frame%30 == 29 {
			fmt.Printf("⏱️  %d/%d フレーム完了\n", frame+1, targetFrames)
		}
	}

	fmt.Println("✅ ヘッドレスモード完了")
	fmt.Println("📁 生成されたファイル:")
	fmt.Println("   - frame_031.ppm (フレーム31のスクリーンショット)")
	fmt.Println("   - frame_061.ppm (フレーム61のスクリーンショット)")
	fmt.Println("   - frame_120.ppm (フレーム120のスクリーンショット)")
	fmt.Println("💡 PPMファイルは画像ビューアで開くか、ImageMagick等で変換できます")
}

// saveFrameBufferAsPPM saves the frame buffer as a PPM image file
func saveFrameBufferAsPPM(frameBuffer [256 * 240]uint32, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("❌ ファイル作成エラー %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	// PPM header
	fmt.Fprintf(file, "P3\n256 240\n255\n")

	// RGB data
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			pixel := frameBuffer[y*256+x]
			r := (pixel >> 16) & 0xFF
			g := (pixel >> 8) & 0xFF
			b := pixel & 0xFF
			fmt.Fprintf(file, "%d %d %d ", r, g, b)
		}
		fmt.Fprintf(file, "\n")
	}

	fmt.Printf("✅ %s 保存完了\n", filename)
}

// analyzeFrameBuffer analyzes the frame buffer content
func analyzeFrameBuffer(frameBuffer [256 * 240]uint32, frame int) {
	colorCounts := make(map[uint32]int)
	for _, pixel := range frameBuffer {
		colorCounts[pixel]++
	}

	nonBlackPixels := 0
	for color, count := range colorCounts {
		if color != 0x000000 {
			nonBlackPixels += count
		}
	}

	fmt.Printf("   フレーム %d: %d個の異なる色, %d個の非黒ピクセル (%.1f%%)\n",
		frame, len(colorCounts), nonBlackPixels,
		float64(nonBlackPixels)/float64(256*240)*100)

	// 主要な色を表示
	if len(colorCounts) > 1 {
		fmt.Printf("   主要色: ")
		count := 0
		for color, pixels := range colorCounts {
			if count >= 3 {
				break
			}
			percentage := float64(pixels) / float64(256*240) * 100
			fmt.Printf("0x%06X(%.1f%%) ", color, percentage)
			count++
		}
		fmt.Println()
	}
}

// setupGracefulShutdown sets up signal handling for graceful shutdown
func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n🛑 Interrupt received, shutting down gracefully...")
		os.Exit(0)
	}()
}

// enabledString returns "enabled" or "disabled" based on boolean value
func enabledString(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func printVersion() {
	version.PrintBuildInfo()
}

func printUsage() {
	fmt.Println("gones - Go NES Emulator")
	fmt.Println()
	fmt.Println("DESCRIPTION:")
	fmt.Println("  A modern NES (Nintendo Entertainment System) emulator written in Go.")
	fmt.Println("  Features cycle-accurate emulation, SDL2 graphics and audio, save states,")
	fmt.Println("  and a user-friendly interface.")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  gones [options]                    # Start GUI mode without ROM")
	fmt.Println("  gones -rom <file> [options]        # Start with ROM loaded")
	fmt.Println("  gones -nogui -rom <file> [options] # Run headless mode")
	fmt.Println()
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  gones                              # Start GUI, load ROM from menu")
	fmt.Println("  gones -rom game.nes                # Start with ROM loaded")
	fmt.Println("  gones -rom game.nes -debug         # Start with debug info enabled")
	fmt.Println("  gones -config custom.json          # Use custom configuration")
	fmt.Println("  gones -nogui -rom test.nes         # Run headless for testing")
	fmt.Println()
	fmt.Println("CONTROLS (Default):")
	fmt.Println("  Player 1:")
	fmt.Println("    Arrow Keys / WASD - D-Pad")
	fmt.Println("    J / Z             - A Button")
	fmt.Println("    K / X             - B Button")
	fmt.Println("    Enter             - Start")
	fmt.Println("    Space             - Select")
	fmt.Println()
	fmt.Println("  Special Keys:")
	fmt.Println("    Escape (2x)       - Quit (double-tap within 3 seconds)")
	fmt.Println("    F1-F10            - Save States")
	fmt.Println("    Shift+F1-F10      - Load States")
	fmt.Println("    F11               - Toggle Fullscreen")
	fmt.Println("    F12               - Screenshot")
	fmt.Println()
	fmt.Println("CONFIGURATION:")
	fmt.Println("  Config file: ./config/gones.json")
	fmt.Println("  ROMs:        ./roms/")
	fmt.Println("  Save States: ./states/")
	fmt.Println("  Screenshots: ./screenshots/")
	fmt.Println()
	fmt.Println("SUPPORTED FORMATS:")
	fmt.Println("  - iNES (.nes)")
	fmt.Println("  - NES 2.0")
	fmt.Println("  - NROM (Mapper 0)")
	fmt.Println()
	fmt.Println("For more information, visit the project documentation.")
}
