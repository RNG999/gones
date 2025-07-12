package integration

import (
	"testing"
	"time"
	"gones/internal/ppu"
	"gones/internal/memory"
)

// BackgroundDebugIntegration defines comprehensive background debugging integration interface
type BackgroundDebugIntegration interface {
	// Combined debugging interfaces
	// ppu.BackgroundDebugger  // Commented out for now as interface is in test file
	// ppu.BackgroundDebugConsole
	// ppu.BackgroundRealTimeDebugger
	
	// Integration-specific methods
	StartComprehensiveDebugging() error
	StopComprehensiveDebugging() error
	GetDebugReport() *ComprehensiveDebugReport
	SaveDebugSession(filename string) error
	LoadDebugSession(filename string) error
	
	// Cross-component analysis
	AnalyzeBackgroundRenderingPipeline() *PipelineAnalysis
	DetectBackgroundRenderingIssues() []RenderingIssue
	GetBackgroundOptimizationSuggestions() []OptimizationSuggestion
	
	// Debug session management
	CreateDebugSnapshot() *DebugSnapshot
	CompareDebugSnapshots(snapshot1, snapshot2 *DebugSnapshot) *SnapshotComparison
	GetDebugSessionHistory() []DebugSessionInfo
}

// Comprehensive debug data structures

type ComprehensiveDebugReport struct {
	SessionID           string
	StartTime           time.Time
	EndTime             time.Time
	Duration            time.Duration
	
	// Summary statistics
	TotalFramesAnalyzed     int
	TotalScanlines          int
	TotalTilesFetched       int
	TotalPixelsRendered     int
	TotalMemoryAccesses     int
	
	// Performance summary
	OverallPerformance      PerformanceSummary
	BottleneckAnalysis      BottleneckAnalysis
	TimingAnalysis          TimingAnalysisSummary
	
	// Background-specific analysis
	NametableUsage          NametableUsageSummary
	TileUsageStatistics     TileUsageStatistics
	ScrollBehaviorAnalysis  ScrollBehaviorAnalysis
	PatternTableAnalysis    PatternTableAnalysisSummary
	
	// Issue detection
	DetectedIssues          []RenderingIssue
	PerformanceAlerts       []ppu.PerformanceAlert
	TimingViolations        []TimingViolation
	
	// Optimization recommendations
	OptimizationSuggestions []OptimizationSuggestion
	
	// Debug console output
	ConsoleOutput           []string
	ErrorMessages           []string
	WarningMessages         []string
	
	// Raw debug data
	FrameAnalysisData       []ppu.FrameAnalysisData
	ScanlineAnalysisData    []ppu.ScanlineAnalysis
	MemoryAccessEvents      []ppu.MemoryAccessEvent
	PixelTraceResults       []ppu.PixelTraceResult
}

type PerformanceSummary struct {
	AverageFrameTime        time.Duration
	MinFrameTime            time.Duration
	MaxFrameTime            time.Duration
	FrameTimeVariance       float64
	
	AverageTileFetchTime    time.Duration
	TileFetchEfficiency     float64
	
	AveragePixelRenderTime  time.Duration
	PixelRenderEfficiency   float64
	
	MemoryAccessEfficiency  float64
	CacheHitRate           float64
	
	OverallScore           float64 // 0-100
	PerformanceGrade       string  // "A", "B", "C", "D", "F"
}

type BottleneckAnalysis struct {
	PrimaryBottleneck       string
	SecondaryBottleneck     string
	BottleneckSeverity      float64
	BottleneckImpact        string
	RecommendedActions      []string
	
	// Component-specific bottlenecks
	TileFetchBottlenecks    []ComponentBottleneck
	PixelRenderBottlenecks  []ComponentBottleneck
	MemoryBottlenecks       []ComponentBottleneck
	ScrollBottlenecks       []ComponentBottleneck
}

type ComponentBottleneck struct {
	Component       string
	Description     string
	Severity        float64
	FrequencyPct    float64
	Impact          string
	Recommendation  string
}

type TimingAnalysisSummary struct {
	OptimalScanlines        int
	SuboptimalScanlines     int
	TimingAccuracy          float64
	
	CommonTimingIssues      map[string]int
	SlowestOperations       []SlowOperation
	FastestOperations       []FastOperation
	
	TimingVariability       float64
	ConsistencyScore        float64
}

type SlowOperation struct {
	Operation       string
	AverageTime     time.Duration
	MaxTime         time.Duration
	Frequency       int
	Impact          string
}

type FastOperation struct {
	Operation       string
	AverageTime     time.Duration
	MinTime         time.Duration
	Frequency       int
	Efficiency      float64
}

type NametableUsageSummary struct {
	ActiveNametables        []int
	TotalTilesUsed          int
	UniqueTilesUsed         int
	TileReuseRate           float64
	
	NametableHotspots       []NametableHotspot
	AttributeUsagePattern   AttributeUsagePattern
	
	ScrollPatterns          []ScrollPattern
	NametableSwitching      NametableSwitchingAnalysis
}

type NametableHotspot struct {
	NametableIndex  int
	TileX, TileY    int
	AccessCount     int
	AccessFrequency float64
	LastAccessTime  time.Time
}

type AttributeUsagePattern struct {
	PaletteUsageDistribution map[int]int
	MostUsedPalettes        []int
	UnusedPalettes          []int
	AttributeVariability    float64
}

type ScrollPattern struct {
	PatternType     string // "STATIC", "HORIZONTAL", "VERTICAL", "DIAGONAL", "IRREGULAR"
	Duration        time.Duration
	ScrollSpeed     float64
	Direction       string
	Smoothness      float64
}

type NametableSwitchingAnalysis struct {
	SwitchCount             int
	SwitchFrequency         float64
	SwitchPatterns          []string
	SwitchEfficiency        float64
	UnnecessarySwitches     int
}

type TileUsageStatistics struct {
	TotalTilesInPatternTable    int
	UsedTiles                   int
	UnusedTiles                 int
	TileUsagePercentage         float64
	
	MostUsedTiles              []TileUsageInfo
	LeastUsedTiles             []TileUsageInfo
	TileUsageDistribution      map[uint8]int
	
	PatternTableEfficiency     float64
	WastedTileSpace           int
	OptimizationPotential     float64
}

type TileUsageInfo struct {
	TileID          uint8
	UsageCount      int
	LastUsedTime    time.Time
	UsageFrequency  float64
	Locations       []TileLocation
}

type TileLocation struct {
	NametableIndex  int
	TileX, TileY    int
	FirstSeen       time.Time
	LastSeen        time.Time
}

type ScrollBehaviorAnalysis struct {
	ScrollingActive         bool
	ScrollingSmooth         bool
	ScrollDirection         string
	ScrollSpeed             float64
	ScrollAcceleration      float64
	
	ScrollingPatterns       []ScrollingPattern
	ScrollingEfficiency     float64
	ScrollingIssues         []ScrollingIssue
	
	SubpixelAccuracy        float64
	ScrollingConsistency    float64
}

type ScrollingPattern struct {
	Type            string // "LINEAR", "ACCELERATED", "DECELERATED", "OSCILLATING", "RANDOM"
	Duration        time.Duration
	StartPosition   ScrollPosition
	EndPosition     ScrollPosition
	Velocity        float64
	Smoothness      float64
}

type ScrollPosition struct {
	X, Y    int
	FineX   int
	FineY   int
}

type ScrollingIssue struct {
	IssueType       string // "JERKY_MOTION", "INCONSISTENT_SPEED", "DIRECTION_CHANGE", "EXCESSIVE_UPDATES"
	Severity        float64
	Frequency       int
	Description     string
	Recommendation  string
}

type PatternTableAnalysisSummary struct {
	PatternTable0Usage      PatternTableUsage
	PatternTable1Usage      PatternTableUsage
	PatternTableSwitching   PatternTableSwitchingInfo
	PatternDataEfficiency   float64
	DuplicateTileAnalysis   DuplicateTileInfo
}

type PatternTableUsage struct {
	TableIndex          int
	TilesUsed           int
	TilesTotal          int
	UsagePercentage     float64
	MostAccessedTiles   []uint8
	AccessFrequency     float64
}

type PatternTableSwitchingInfo struct {
	SwitchCount         int
	SwitchFrequency     float64
	UnnecessarySwitches int
	SwitchEfficiency    float64
}

type DuplicateTileInfo struct {
	DuplicateCount      int
	DuplicateTiles      []DuplicateTileGroup
	WastedSpace         int
	DeduplicationSavings float64
}

type DuplicateTileGroup struct {
	TileData        [8][8]uint8
	TileIDs         []uint8
	Locations       []TileLocation
	WastedBytes     int
}

type PipelineAnalysis struct {
	PipelineStages          []PipelineStageAnalysis
	OverallPipelineHealth   float64
	PipelineBottlenecks     []PipelineBottleneck
	PipelineOptimizations   []PipelineOptimization
	
	DataFlowAnalysis        DataFlowAnalysis
	TimingAnalysis          PipelineTimingAnalysis
	EfficiencyAnalysis      PipelineEfficiencyAnalysis
}

type PipelineStageAnalysis struct {
	StageName           string
	StageHealth         float64
	AverageLatency      time.Duration
	Throughput          float64
	ErrorRate           float64
	BottleneckRisk      float64
	Recommendations     []string
}

type PipelineBottleneck struct {
	StageName           string
	BottleneckType      string
	Severity            float64
	Impact              string
	RootCause           string
	SuggestedFix        string
}

type PipelineOptimization struct {
	OptimizationType    string
	TargetStage         string
	ExpectedImprovement float64
	ImplementationCost  string
	Priority            string
	Description         string
}

type DataFlowAnalysis struct {
	TileFetchFlow       DataFlowMetrics
	AttributeFetchFlow  DataFlowMetrics
	PatternFetchFlow    DataFlowMetrics
	PixelOutputFlow     DataFlowMetrics
	
	FlowConsistency     float64
	FlowEfficiency      float64
	FlowBottlenecks     []DataFlowBottleneck
}

type DataFlowMetrics struct {
	FlowName            string
	DataThroughput      float64
	AverageLatency      time.Duration
	ErrorRate           float64
	BackpressureEvents  int
	FlowHealth          float64
}

type DataFlowBottleneck struct {
	FlowStage           string
	BottleneckCause     string
	DataBackup          int
	ResolutionTime      time.Duration
}

type PipelineTimingAnalysis struct {
	OverallTiming       float64
	StageTimings        map[string]time.Duration
	TimingVariability   float64
	TimingPredictability float64
	CriticalPath        []string
	TimingMargin        time.Duration
}

type PipelineEfficiencyAnalysis struct {
	OverallEfficiency   float64
	StageEfficiencies   map[string]float64
	ResourceUtilization float64
	WastedCycles        int
	OptimizationPotential float64
}

type RenderingIssue struct {
	IssueID             string
	IssueType           RenderingIssueType
	Severity            IssueSeverity
	Component           string
	Description         string
	DetectedAt          time.Time
	
	// Location information
	ScanlineNumber      int
	CycleNumber         int
	FrameNumber         uint64
	
	// Impact assessment
	PerformanceImpact   float64
	VisualImpact        float64
	CorrectnessThreat   float64
	
	// Resolution
	SuggestedFix        string
	AutoFixAvailable    bool
	FixComplexity       string
	
	// Related data
	RelatedData         map[string]interface{}
	ErrorContext        map[string]string
}

type RenderingIssueType int

const (
	IssueTypeTiming RenderingIssueType = iota
	IssueTypeMemory
	IssueTypeLogic
	IssueTypePerformance
	IssueTypeCorrectness
	IssueTypeCompatibility
)

type IssueSeverity int

const (
	SeverityInfo IssueSeverity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

type TimingViolation struct {
	ViolationType       string
	ExpectedTiming      time.Duration
	ActualTiming        time.Duration
	Deviation           time.Duration
	ToleranceExceeded   float64
	
	Location            ViolationLocation
	Impact              string
	Recommendation      string
}

type ViolationLocation struct {
	Component       string
	Scanline        int
	Cycle           int
	Frame           uint64
}

type OptimizationSuggestion struct {
	SuggestionID        string
	Category            string
	Priority            OptimizationPriority
	Title               string
	Description         string
	
	// Implementation details
	ImplementationSteps []string
	EstimatedEffort     string
	RequiredChanges     []string
	
	// Expected benefits
	PerformanceGain     float64
	MemoryReduction     float64
	TimingImprovement   time.Duration
	CodeComplexity      string
	
	// Risk assessment
	ImplementationRisk  float64
	BreakingChanges     bool
	TestingRequired     []string
	
	// Supporting data
	Metrics             map[string]float64
	Examples            []string
	References          []string
}

type OptimizationPriority int

const (
	PriorityLow OptimizationPriority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

type DebugSnapshot struct {
	SnapshotID          string
	Timestamp           time.Time
	
	// PPU state
	PPUState            PPUStateSnapshot
	BackgroundState     BackgroundStateSnapshot
	RenderingState      RenderingStateSnapshot
	
	// Memory state
	NametableData       [4][30][32]uint8
	AttributeData       [4][8][8]uint8
	PatternTableData    [2][256][8][8]uint8
	PaletteData         [32]uint8
	
	// Performance metrics
	PerformanceSnapshot PerformanceSnapshot
	
	// Debug configuration
	DebugConfig         DebugConfiguration
}

type PPUStateSnapshot struct {
	Control         uint8
	Mask            uint8
	Status          uint8
	VRAMAddress     uint16
	TempAddress     uint16
	FineX           uint8
	WriteToggle     bool
	Scanline        int
	Cycle           int
	OddFrame        bool
}

type BackgroundStateSnapshot struct {
	BackgroundEnabled   bool
	PatternTableSelect  int
	NametableSelect     int
	
	CurrentTileID       uint8
	NextTileID          uint8
	AttributeByte       uint8
	PatternLow          uint8
	PatternHigh         uint8
	
	ShiftRegisterState  ppu.ShiftRegisterState
	ScrollState         ppu.ScrollDebugInfo
}

type RenderingStateSnapshot struct {
	IsRendering         bool
	TilesFetched        int
	PixelsRendered      int
	MemoryAccesses      int
	
	CurrentFetchPhase   string
	FetchQueue          []string
	
	RenderingMetrics    ppu.BackgroundRenderingMetrics
}

type PerformanceSnapshot struct {
	FrameTime           time.Duration
	TileFetchTime       time.Duration
	PixelRenderTime     time.Duration
	MemoryAccessTime    time.Duration
	
	CacheHitRate        float64
	MemoryBandwidth     float64
	CPUUtilization      float64
	
	PerformanceScore    float64
}

type DebugConfiguration struct {
	LoggingEnabled      bool
	VerbosityLevel      int
	MonitoringEnabled   bool
	TracingEnabled      bool
	
	FilterSettings      ppu.DebugFilter
	TracingRegion       ppu.PixelRegion
	
	SessionSettings     map[string]interface{}
}

type SnapshotComparison struct {
	Snapshot1ID         string
	Snapshot2ID         string
	ComparisonTime      time.Time
	
	// State differences
	PPUStateDifferences     []StateDifference
	BackgroundDifferences   []StateDifference
	MemoryDifferences       []MemoryDifference
	
	// Performance differences
	PerformanceDelta        PerformanceDelta
	
	// Summary
	TotalDifferences        int
	SignificantChanges      []string
	PerformanceImprovement  bool
	RegressionDetected      bool
	
	// Recommendations
	Recommendations         []string
}

type StateDifference struct {
	Field           string
	OldValue        interface{}
	NewValue        interface{}
	ChangeType      string
	Significance    float64
}

type MemoryDifference struct {
	Address         uint16
	Component       string
	OldValue        uint8
	NewValue        uint8
	ChangeContext   string
}

type PerformanceDelta struct {
	FrameTimeDelta      time.Duration
	TileFetchDelta      time.Duration
	PixelRenderDelta    time.Duration
	MemoryAccessDelta   time.Duration
	
	PerformanceScoreDelta   float64
	OverallImprovement      bool
}

type DebugSessionInfo struct {
	SessionID           string
	StartTime           time.Time
	EndTime             time.Time
	Duration            time.Duration
	FramesAnalyzed      int
	IssuesDetected      int
	PerformanceScore    float64
	ConfigUsed          DebugConfiguration
}

// Test suite for comprehensive background debugging integration

func TestBackgroundDebugIntegrationInterface(t *testing.T) {
	t.Run("BackgroundDebugIntegration interface should be implemented", func(t *testing.T) {
		ppu := ppu.New()
		
		// This test will fail initially as the interface is not implemented
		debugIntegration, ok := interface{}(ppu).(BackgroundDebugIntegration)
		if !ok {
			t.Fatal("PPU should implement BackgroundDebugIntegration interface for comprehensive background debugging")
		}
		
		if debugIntegration == nil {
			t.Fatal("BackgroundDebugIntegration interface should not be nil")
		}
	})
}

func TestComprehensiveDebuggingSession(t *testing.T) {
	cart := newTestCartridgeWithCompletePatternData()
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	ppuInstance := ppu.New()
	ppuInstance.SetMemory(ppuMem)
	
	debugIntegration := interface{}(ppuInstance).(BackgroundDebugIntegration)
	setupCompleteTestScene(ppuMem)
	
	t.Run("Start and stop comprehensive debugging", func(t *testing.T) {
		err := debugIntegration.StartComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StartComprehensiveDebugging should not return error: %v", err)
		}
		
		// Simulate complex background rendering scenario
		simulateComplexRenderingScenario(ppuInstance)
		
		err = debugIntegration.StopComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StopComprehensiveDebugging should not return error: %v", err)
		}
	})
	
	t.Run("Get comprehensive debug report", func(t *testing.T) {
		err := debugIntegration.StartComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StartComprehensiveDebugging failed: %v", err)
		}
		
		// Generate significant debug activity
		simulateComplexRenderingScenario(ppuInstance)
		
		report := debugIntegration.GetDebugReport()
		if report == nil {
			t.Fatal("GetDebugReport should return valid report")
		}
		
		// Verify report structure
		if report.SessionID == "" {
			t.Error("Report should have session ID")
		}
		
		if report.StartTime.IsZero() {
			t.Error("Report should have start time")
		}
		
		if report.TotalFramesAnalyzed < 0 {
			t.Error("Total frames analyzed should be non-negative")
		}
		
		if report.OverallPerformance.OverallScore < 0 || report.OverallPerformance.OverallScore > 100 {
			t.Errorf("Overall performance score should be 0-100, got %f", report.OverallPerformance.OverallScore)
		}
		
		validGrades := []string{"A", "B", "C", "D", "F"}
		validGrade := false
		for _, grade := range validGrades {
			if report.OverallPerformance.PerformanceGrade == grade {
				validGrade = true
				break
			}
		}
		if !validGrade {
			t.Errorf("Performance grade should be valid, got %s", report.OverallPerformance.PerformanceGrade)
		}
		
		// Verify analysis sections
		if report.NametableUsage.TotalTilesUsed < 0 {
			t.Error("Total tiles used should be non-negative")
		}
		
		if report.TileUsageStatistics.TileUsagePercentage < 0 || report.TileUsageStatistics.TileUsagePercentage > 100 {
			t.Errorf("Tile usage percentage should be 0-100, got %f", report.TileUsageStatistics.TileUsagePercentage)
		}
		
		if report.ScrollBehaviorAnalysis.ScrollingEfficiency < 0 || report.ScrollBehaviorAnalysis.ScrollingEfficiency > 1 {
			t.Errorf("Scrolling efficiency should be 0-1, got %f", report.ScrollBehaviorAnalysis.ScrollingEfficiency)
		}
		
		// Verify optimization suggestions
		if report.OptimizationSuggestions == nil {
			t.Error("Optimization suggestions should not be nil")
		}
		
		for _, suggestion := range report.OptimizationSuggestions {
			if suggestion.SuggestionID == "" {
				t.Error("Optimization suggestion should have ID")
			}
			
			if suggestion.Title == "" {
				t.Error("Optimization suggestion should have title")
			}
			
			if suggestion.Description == "" {
				t.Error("Optimization suggestion should have description")
			}
			
			if suggestion.PerformanceGain < 0 {
				t.Error("Performance gain should be non-negative")
			}
		}
		
		debugIntegration.StopComprehensiveDebugging()
	})
}

func TestPipelineAnalysisAndIssueDetection(t *testing.T) {
	cart := newTestCartridgeWithCompletePatternData()
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	ppuInstance := ppu.New()
	ppuInstance.SetMemory(ppuMem)
	
	debugIntegration := interface{}(ppuInstance).(BackgroundDebugIntegration)
	setupCompleteTestScene(ppuMem)
	
	t.Run("Analyze background rendering pipeline", func(t *testing.T) {
		err := debugIntegration.StartComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StartComprehensiveDebugging failed: %v", err)
		}
		
		// Simulate rendering to generate pipeline data
		simulateComplexRenderingScenario(ppuInstance)
		
		pipelineAnalysis := debugIntegration.AnalyzeBackgroundRenderingPipeline()
		if pipelineAnalysis == nil {
			t.Fatal("AnalyzeBackgroundRenderingPipeline should return valid analysis")
		}
		
		if pipelineAnalysis.OverallPipelineHealth < 0 || pipelineAnalysis.OverallPipelineHealth > 1 {
			t.Errorf("Overall pipeline health should be 0-1, got %f", pipelineAnalysis.OverallPipelineHealth)
		}
		
		if pipelineAnalysis.PipelineStages == nil {
			t.Error("Pipeline stages should not be nil")
		}
		
		if len(pipelineAnalysis.PipelineStages) == 0 {
			t.Error("Should have pipeline stages")
		}
		
		// Verify pipeline stages
		for _, stage := range pipelineAnalysis.PipelineStages {
			if stage.StageName == "" {
				t.Error("Pipeline stage should have name")
			}
			
			if stage.StageHealth < 0 || stage.StageHealth > 1 {
				t.Errorf("Stage health should be 0-1, got %f", stage.StageHealth)
			}
			
			if stage.AverageLatency < 0 {
				t.Error("Average latency should be non-negative")
			}
			
			if stage.Throughput < 0 {
				t.Error("Throughput should be non-negative")
			}
			
			if stage.ErrorRate < 0 || stage.ErrorRate > 1 {
				t.Errorf("Error rate should be 0-1, got %f", stage.ErrorRate)
			}
		}
		
		// Verify data flow analysis
		if pipelineAnalysis.DataFlowAnalysis.FlowConsistency < 0 || pipelineAnalysis.DataFlowAnalysis.FlowConsistency > 1 {
			t.Errorf("Flow consistency should be 0-1, got %f", pipelineAnalysis.DataFlowAnalysis.FlowConsistency)
		}
		
		if pipelineAnalysis.DataFlowAnalysis.FlowEfficiency < 0 || pipelineAnalysis.DataFlowAnalysis.FlowEfficiency > 1 {
			t.Errorf("Flow efficiency should be 0-1, got %f", pipelineAnalysis.DataFlowAnalysis.FlowEfficiency)
		}
		
		// Verify timing analysis
		if pipelineAnalysis.TimingAnalysis.OverallTiming < 0 || pipelineAnalysis.TimingAnalysis.OverallTiming > 1 {
			t.Errorf("Overall timing should be 0-1, got %f", pipelineAnalysis.TimingAnalysis.OverallTiming)
		}
		
		if pipelineAnalysis.TimingAnalysis.TimingVariability < 0 {
			t.Error("Timing variability should be non-negative")
		}
		
		debugIntegration.StopComprehensiveDebugging()
	})
	
	t.Run("Detect background rendering issues", func(t *testing.T) {
		err := debugIntegration.StartComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StartComprehensiveDebugging failed: %v", err)
		}
		
		// Simulate problematic rendering conditions
		simulateProblematicRenderingConditions(ppuInstance)
		
		issues := debugIntegration.DetectBackgroundRenderingIssues()
		if issues == nil {
			t.Fatal("DetectBackgroundRenderingIssues should return issue slice")
		}
		
		// Verify issue structure
		for _, issue := range issues {
			if issue.IssueID == "" {
				t.Error("Issue should have ID")
			}
			
			if issue.Description == "" {
				t.Error("Issue should have description")
			}
			
			if issue.DetectedAt.IsZero() {
				t.Error("Issue should have detection time")
			}
			
			if issue.PerformanceImpact < 0 || issue.PerformanceImpact > 1 {
				t.Errorf("Performance impact should be 0-1, got %f", issue.PerformanceImpact)
			}
			
			if issue.VisualImpact < 0 || issue.VisualImpact > 1 {
				t.Errorf("Visual impact should be 0-1, got %f", issue.VisualImpact)
			}
			
			if issue.CorrectnessThreat < 0 || issue.CorrectnessThreat > 1 {
				t.Errorf("Correctness threat should be 0-1, got %f", issue.CorrectnessThreat)
			}
			
			if issue.SuggestedFix == "" {
				t.Error("Issue should have suggested fix")
			}
			
			// Verify issue type is valid
			validTypes := []RenderingIssueType{
				IssueTypeTiming,
				IssueTypeMemory,
				IssueTypeLogic,
				IssueTypePerformance,
				IssueTypeCorrectness,
				IssueTypeCompatibility,
			}
			validType := false
			for _, issueType := range validTypes {
				if issue.IssueType == issueType {
					validType = true
					break
				}
			}
			if !validType {
				t.Errorf("Issue type should be valid, got %d", int(issue.IssueType))
			}
			
			// Verify severity is valid
			validSeverities := []IssueSeverity{
				SeverityInfo,
				SeverityLow,
				SeverityMedium,
				SeverityHigh,
				SeverityCritical,
			}
			validSeverity := false
			for _, severity := range validSeverities {
				if issue.Severity == severity {
					validSeverity = true
					break
				}
			}
			if !validSeverity {
				t.Errorf("Issue severity should be valid, got %d", int(issue.Severity))
			}
		}
		
		debugIntegration.StopComprehensiveDebugging()
	})
	
	t.Run("Get optimization suggestions", func(t *testing.T) {
		err := debugIntegration.StartComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StartComprehensiveDebugging failed: %v", err)
		}
		
		// Generate data for optimization analysis
		simulateSuboptimalRenderingScenario(ppuInstance)
		
		suggestions := debugIntegration.GetBackgroundOptimizationSuggestions()
		if suggestions == nil {
			t.Fatal("GetBackgroundOptimizationSuggestions should return suggestion slice")
		}
		
		// Verify suggestions if any exist
		for _, suggestion := range suggestions {
			if suggestion.SuggestionID == "" {
				t.Error("Suggestion should have ID")
			}
			
			if suggestion.Title == "" {
				t.Error("Suggestion should have title")
			}
			
			if suggestion.Description == "" {
				t.Error("Suggestion should have description")
			}
			
			if suggestion.PerformanceGain < 0 {
				t.Error("Performance gain should be non-negative")
			}
			
			if suggestion.ImplementationRisk < 0 || suggestion.ImplementationRisk > 1 {
				t.Errorf("Implementation risk should be 0-1, got %f", suggestion.ImplementationRisk)
			}
			
			// Verify priority is valid
			validPriorities := []OptimizationPriority{
				PriorityLow,
				PriorityMedium,
				PriorityHigh,
				PriorityCritical,
			}
			validPriority := false
			for _, priority := range validPriorities {
				if suggestion.Priority == priority {
					validPriority = true
					break
				}
			}
			if !validPriority {
				t.Errorf("Suggestion priority should be valid, got %d", int(suggestion.Priority))
			}
			
			if suggestion.ImplementationSteps == nil {
				t.Error("Implementation steps should not be nil")
			}
			
			if suggestion.Metrics == nil {
				t.Error("Metrics should not be nil")
			}
		}
		
		debugIntegration.StopComprehensiveDebugging()
	})
}

func TestDebugSnapshotAndComparison(t *testing.T) {
	cart := newTestCartridgeWithCompletePatternData()
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	ppuInstance := ppu.New()
	ppuInstance.SetMemory(ppuMem)
	
	debugIntegration := interface{}(ppuInstance).(BackgroundDebugIntegration)
	setupCompleteTestScene(ppuMem)
	
	t.Run("Create debug snapshot", func(t *testing.T) {
		err := debugIntegration.StartComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StartComprehensiveDebugging failed: %v", err)
		}
		
		// Set up specific PPU state
		ppuInstance.WriteRegister(0x2000, 0x80) // Enable NMI
		ppuInstance.WriteRegister(0x2001, 0x0A) // Enable background
		ppuInstance.WriteRegister(0x2005, 0x40) // X scroll
		ppuInstance.WriteRegister(0x2005, 0x30) // Y scroll
		
		snapshot := debugIntegration.CreateDebugSnapshot()
		if snapshot == nil {
			t.Fatal("CreateDebugSnapshot should return valid snapshot")
		}
		
		if snapshot.SnapshotID == "" {
			t.Error("Snapshot should have ID")
		}
		
		if snapshot.Timestamp.IsZero() {
			t.Error("Snapshot should have timestamp")
		}
		
		// Verify PPU state capture
		if snapshot.PPUState.Control != 0x80 {
			t.Errorf("Expected control 0x80, got 0x%02X", snapshot.PPUState.Control)
		}
		
		if snapshot.PPUState.Mask != 0x0A {
			t.Errorf("Expected mask 0x0A, got 0x%02X", snapshot.PPUState.Mask)
		}
		
		// Verify background state capture
		if !snapshot.BackgroundState.BackgroundEnabled {
			t.Error("Background should be enabled in snapshot")
		}
		
		// Verify memory data capture
		// Check that nametable data is captured
		nonZeroBytes := 0
		for nt := 0; nt < 4; nt++ {
			for y := 0; y < 30; y++ {
				for x := 0; x < 32; x++ {
					if snapshot.NametableData[nt][y][x] != 0 {
						nonZeroBytes++
					}
				}
			}
		}
		if nonZeroBytes == 0 {
			t.Error("Snapshot should capture nametable data")
		}
		
		debugIntegration.StopComprehensiveDebugging()
	})
	
	t.Run("Compare debug snapshots", func(t *testing.T) {
		err := debugIntegration.StartComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StartComprehensiveDebugging failed: %v", err)
		}
		
		// Create first snapshot
		ppuInstance.WriteRegister(0x2001, 0x0A) // Enable background
		snapshot1 := debugIntegration.CreateDebugSnapshot()
		if snapshot1 == nil {
			t.Fatal("First snapshot should be created")
		}
		
		// Change PPU state
		ppuInstance.WriteRegister(0x2005, 0x20) // Change X scroll
		ppuInstance.WriteRegister(0x2005, 0x40) // Change Y scroll
		ppuInstance.WriteRegister(0x2000, 0x80) // Enable NMI
		
		// Simulate some rendering to change state
		for i := 0; i < 100; i++ {
			ppuInstance.Step()
		}
		
		// Create second snapshot
		snapshot2 := debugIntegration.CreateDebugSnapshot()
		if snapshot2 == nil {
			t.Fatal("Second snapshot should be created")
		}
		
		// Compare snapshots
		comparison := debugIntegration.CompareDebugSnapshots(snapshot1, snapshot2)
		if comparison == nil {
			t.Fatal("CompareDebugSnapshots should return valid comparison")
		}
		
		if comparison.Snapshot1ID != snapshot1.SnapshotID {
			t.Error("Comparison should reference correct snapshot 1 ID")
		}
		
		if comparison.Snapshot2ID != snapshot2.SnapshotID {
			t.Error("Comparison should reference correct snapshot 2 ID")
		}
		
		if comparison.ComparisonTime.IsZero() {
			t.Error("Comparison should have timestamp")
		}
		
		if comparison.TotalDifferences < 0 {
			t.Error("Total differences should be non-negative")
		}
		
		// Should detect scroll changes
		if len(comparison.PPUStateDifferences) == 0 {
			t.Error("Should detect PPU state differences")
		}
		
		// Verify difference structure
		for _, diff := range comparison.PPUStateDifferences {
			if diff.Field == "" {
				t.Error("Difference should have field name")
			}
			
			if diff.ChangeType == "" {
				t.Error("Difference should have change type")
			}
			
			if diff.Significance < 0 || diff.Significance > 1 {
				t.Errorf("Difference significance should be 0-1, got %f", diff.Significance)
			}
		}
		
		if comparison.Recommendations == nil {
			t.Error("Comparison should have recommendations")
		}
		
		debugIntegration.StopComprehensiveDebugging()
	})
}

func TestDebugSessionManagement(t *testing.T) {
	cart := newTestCartridgeWithCompletePatternData()
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	ppuInstance := ppu.New()
	ppuInstance.SetMemory(ppuMem)
	
	debugIntegration := interface{}(ppuInstance).(BackgroundDebugIntegration)
	
	t.Run("Get debug session history", func(t *testing.T) {
		// Run multiple debug sessions
		for i := 0; i < 3; i++ {
			err := debugIntegration.StartComprehensiveDebugging()
			if err != nil {
				t.Fatalf("StartComprehensiveDebugging %d failed: %v", i, err)
			}
			
			// Simulate brief activity
			ppuInstance.WriteRegister(0x2001, 0x0A)
			for j := 0; j < 50; j++ {
				ppuInstance.Step()
			}
			
			err = debugIntegration.StopComprehensiveDebugging()
			if err != nil {
				t.Fatalf("StopComprehensiveDebugging %d failed: %v", i, err)
			}
		}
		
		history := debugIntegration.GetDebugSessionHistory()
		if history == nil {
			t.Fatal("GetDebugSessionHistory should return session slice")
		}
		
		if len(history) < 3 {
			t.Errorf("Should have at least 3 sessions, got %d", len(history))
		}
		
		// Verify session info structure
		for _, session := range history {
			if session.SessionID == "" {
				t.Error("Session should have ID")
			}
			
			if session.StartTime.IsZero() {
				t.Error("Session should have start time")
			}
			
			if session.EndTime.IsZero() {
				t.Error("Session should have end time")
			}
			
			if session.Duration <= 0 {
				t.Error("Session should have positive duration")
			}
			
			if session.FramesAnalyzed < 0 {
				t.Error("Frames analyzed should be non-negative")
			}
			
			if session.IssuesDetected < 0 {
				t.Error("Issues detected should be non-negative")
			}
			
			if session.PerformanceScore < 0 || session.PerformanceScore > 100 {
				t.Errorf("Performance score should be 0-100, got %f", session.PerformanceScore)
			}
		}
	})
	
	t.Run("Save and load debug session", func(t *testing.T) {
		err := debugIntegration.StartComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StartComprehensiveDebugging failed: %v", err)
		}
		
		// Generate some debug data
		simulateComplexRenderingScenario(ppuInstance)
		
		// Save session
		filename := "/tmp/test_debug_session.json"
		err = debugIntegration.SaveDebugSession(filename)
		if err != nil {
			t.Fatalf("SaveDebugSession should not return error: %v", err)
		}
		
		err = debugIntegration.StopComprehensiveDebugging()
		if err != nil {
			t.Fatalf("StopComprehensiveDebugging failed: %v", err)
		}
		
		// Load session (this would typically restore debug state)
		err = debugIntegration.LoadDebugSession(filename)
		if err != nil {
			t.Fatalf("LoadDebugSession should not return error: %v", err)
		}
		
		// Verify session was loaded by checking if we can get a report
		// (This assumes loading restores the session state)
		report := debugIntegration.GetDebugReport()
		if report == nil {
			t.Error("Should be able to get debug report after loading session")
		}
	})
}

// Helper functions for test scenarios

func newTestCartridgeWithCompletePatternData() *TestCartridgeIntegration {
	cart := &TestCartridgeIntegration{
		chrRom: make([]uint8, 0x2000), // 8KB CHR ROM
	}
	
	// Fill with varied pattern data
	for i := 0; i < len(cart.chrRom); i++ {
		// Create more interesting patterns
		tileIndex := i / 16
		byteInTile := i % 16
		if byteInTile < 8 {
			// Low byte plane
			cart.chrRom[i] = uint8((tileIndex * 3 + byteInTile) % 256)
		} else {
			// High byte plane
			cart.chrRom[i] = uint8((tileIndex * 5 + byteInTile) % 256)
		}
	}
	
	return cart
}

func setupCompleteTestScene(ppuMem *memory.PPUMemory) {
	// Fill multiple nametables with varied data
	for nt := 0; nt < 4; nt++ {
		baseAddr := uint16(0x2000 + nt*0x400)
		
		// Fill nametable with pattern
		for y := 0; y < 30; y++ {
			for x := 0; x < 32; x++ {
				addr := baseAddr + uint16(y*32+x)
				tileID := uint8((x + y + nt*10) % 256)
				ppuMem.Write(addr, tileID)
			}
		}
		
		// Fill attribute table
		attrBaseAddr := baseAddr + 0x3C0
		for i := 0; i < 64; i++ {
			addr := attrBaseAddr + uint16(i)
			attrValue := uint8((i + nt) % 4)
			ppuMem.Write(addr, attrValue)
		}
	}
	
	// Set up comprehensive palette
	for i := uint16(0); i < 32; i++ {
		ppuMem.Write(0x3F00+i, uint8((i*7)%64))
	}
}

func simulateComplexRenderingScenario(ppuInstance *ppu.PPU) {
	// Enable background rendering
	ppuInstance.WriteRegister(0x2001, 0x0A)
	ppuInstance.WriteRegister(0x2000, 0x80)
	
	// Simulate scrolling
	for scroll := 0; scroll < 100; scroll += 10 {
		ppuInstance.WriteRegister(0x2005, uint8(scroll))   // X scroll
		ppuInstance.WriteRegister(0x2005, uint8(scroll/2)) // Y scroll
		
		// Render several scanlines
		for scanline := 0; scanline < 10; scanline++ {
			for cycle := 0; cycle < 256; cycle++ {
				ppuInstance.Step()
			}
		}
	}
	
	// Simulate pattern table switching
	ppuInstance.WriteRegister(0x2000, 0x90) // Switch pattern table
	
	// More rendering
	for i := 0; i < 1000; i++ {
		ppuInstance.Step()
	}
}

func simulateProblematicRenderingConditions(ppuInstance *ppu.PPU) {
	// Rapid register changes that might cause issues
	for i := 0; i < 50; i++ {
		ppuInstance.WriteRegister(0x2000, uint8(i%2*0x80)) // Toggle NMI
		ppuInstance.WriteRegister(0x2001, uint8((i%2)*0x0A)) // Toggle background
		ppuInstance.WriteRegister(0x2005, uint8(i*3))        // Rapid scroll changes
		ppuInstance.WriteRegister(0x2005, uint8(i*5))
		
		// Step a few cycles
		for j := 0; j < 10; j++ {
			ppuInstance.Step()
		}
	}
	
	// Simulate mid-frame register changes
	for scanline := 0; scanline < 240; scanline++ {
		for cycle := 0; cycle < 256; cycle++ {
			if cycle == 128 { // Mid-scanline change
				ppuInstance.WriteRegister(0x2005, uint8(scanline))
			}
			ppuInstance.Step()
		}
	}
}

func simulateSuboptimalRenderingScenario(ppuInstance *ppu.PPU) {
	// Inefficient scrolling patterns
	ppuInstance.WriteRegister(0x2001, 0x0A)
	
	// Oscillating scroll that wastes cycles
	for i := 0; i < 200; i++ {
		scrollX := uint8((i % 20) * 8)
		scrollY := uint8((i % 15) * 5)
		
		ppuInstance.WriteRegister(0x2005, scrollX)
		ppuInstance.WriteRegister(0x2005, scrollY)
		
		// Minimal rendering to show inefficiency
		for j := 0; j < 5; j++ {
			ppuInstance.Step()
		}
	}
	
	// Unnecessary pattern table switches
	for i := 0; i < 100; i++ {
		control := uint8(0x80)
		if i%3 == 0 {
			control |= 0x10 // Pattern table 1
		}
		ppuInstance.WriteRegister(0x2000, control)
		
		for j := 0; j < 10; j++ {
			ppuInstance.Step()
		}
	}
}

type TestCartridgeIntegration struct {
	chrRom []uint8
}

func (c *TestCartridgeIntegration) ReadCHR(address uint16) uint8 {
	if int(address) < len(c.chrRom) {
		return c.chrRom[address]
	}
	return 0
}

func (c *TestCartridgeIntegration) WriteCHR(address uint16, value uint8) {
	if int(address) < len(c.chrRom) {
		c.chrRom[address] = value
	}
}

func (c *TestCartridgeIntegration) ReadPRG(address uint16) uint8 { return 0 }
func (c *TestCartridgeIntegration) WritePRG(address uint16, value uint8) {}