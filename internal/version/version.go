// Package version provides build information for the gones NES emulator
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"
)

var (
	// These will be set at build time via -ldflags
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
	BuildUser = "unknown"
)

// BuildInfo contains detailed build information
type BuildInfo struct {
	Version    string `json:"version"`
	GitCommit  string `json:"git_commit"`
	BuildTime  string `json:"build_time"`
	BuildUser  string `json:"build_user"`
	GoVersion  string `json:"go_version"`
	Platform   string `json:"platform"`
	Arch       string `json:"arch"`
	CGOEnabled bool   `json:"cgo_enabled"`
}

// GetBuildInfo returns detailed build information
func GetBuildInfo() BuildInfo {
	buildInfo := BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		BuildUser: BuildUser,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	// Try to get additional information from debug.BuildInfo
	if info, ok := debug.ReadBuildInfo(); ok {
		// Look for VCS information
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if GitCommit == "unknown" {
					buildInfo.GitCommit = setting.Value
				}
			case "vcs.time":
				if BuildTime == "unknown" {
					buildInfo.BuildTime = setting.Value
				}
			case "CGO_ENABLED":
				buildInfo.CGOEnabled = setting.Value == "1"
			}
		}
	}

	return buildInfo
}

// GetVersion returns a simple version string
func GetVersion() string {
	if Version == "dev" {
		buildInfo := GetBuildInfo()
		if buildInfo.GitCommit != "unknown" && len(buildInfo.GitCommit) >= 7 {
			return fmt.Sprintf("dev-%s", buildInfo.GitCommit[:7])
		}
	}
	return Version
}

// GetDetailedVersion returns a detailed version string
func GetDetailedVersion() string {
	buildInfo := GetBuildInfo()

	versionStr := fmt.Sprintf("gones version %s", buildInfo.Version)

	if buildInfo.GitCommit != "unknown" {
		if len(buildInfo.GitCommit) >= 7 {
			versionStr += fmt.Sprintf(" (commit %s)", buildInfo.GitCommit[:7])
		} else {
			versionStr += fmt.Sprintf(" (commit %s)", buildInfo.GitCommit)
		}
	}

	if buildInfo.BuildTime != "unknown" {
		if parsedTime, err := time.Parse(time.RFC3339, buildInfo.BuildTime); err == nil {
			versionStr += fmt.Sprintf(" built on %s", parsedTime.Format("2006-01-02 15:04:05"))
		} else {
			versionStr += fmt.Sprintf(" built on %s", buildInfo.BuildTime)
		}
	}

	versionStr += fmt.Sprintf(" with %s for %s/%s", buildInfo.GoVersion, buildInfo.Platform, buildInfo.Arch)

	if buildInfo.BuildUser != "unknown" {
		versionStr += fmt.Sprintf(" by %s", buildInfo.BuildUser)
	}

	return versionStr
}

// PrintBuildInfo prints formatted build information
func PrintBuildInfo() {
	buildInfo := GetBuildInfo()

	fmt.Printf("gones - Go NES Emulator\n")
	fmt.Printf("Version:     %s\n", buildInfo.Version)
	fmt.Printf("Git Commit:  %s\n", buildInfo.GitCommit)
	fmt.Printf("Build Time:  %s\n", buildInfo.BuildTime)
	fmt.Printf("Build User:  %s\n", buildInfo.BuildUser)
	fmt.Printf("Go Version:  %s\n", buildInfo.GoVersion)
	fmt.Printf("Platform:    %s/%s\n", buildInfo.Platform, buildInfo.Arch)
	fmt.Printf("CGO Enabled: %t\n", buildInfo.CGOEnabled)
	fmt.Printf("Repository:  https://github.com/your-org/gones\n")
}
