package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.ServerHost != DefaultServerHost {
		t.Fatalf("ServerHost = %q, want %q", cfg.ServerHost, DefaultServerHost)
	}
	if cfg.ServerPort != DefaultServerPort {
		t.Fatalf("ServerPort = %q, want %q", cfg.ServerPort, DefaultServerPort)
	}
	if cfg.DBPath != DefaultDBPath {
		t.Fatalf("DBPath = %q, want %q", cfg.DBPath, DefaultDBPath)
	}
	if cfg.ReportFormat != DefaultReportFormat {
		t.Fatalf("ReportFormat = %q, want %q", cfg.ReportFormat, DefaultReportFormat)
	}
	if cfg.ReportOutput != DefaultReportOutput {
		t.Fatalf("ReportOutput = %q, want %q", cfg.ReportOutput, DefaultReportOutput)
	}
	if cfg.Language != DefaultLanguage {
		t.Fatalf("Language = %q, want %q", cfg.Language, DefaultLanguage)
	}
	if cfg.HTTPTimeout != DefaultHTTPTimeout {
		t.Fatalf("HTTPTimeout = %s, want %s", cfg.HTTPTimeout, DefaultHTTPTimeout)
	}
	if cfg.DBGCEnabled != DefaultDBGCEnabled {
		t.Fatalf("DBGCEnabled = %t, want %t", cfg.DBGCEnabled, DefaultDBGCEnabled)
	}
	if cfg.DBGCInterval != DefaultDBGCInterval {
		t.Fatalf("DBGCInterval = %s, want %s", cfg.DBGCInterval, DefaultDBGCInterval)
	}
	if cfg.DBGCDiscardRatio != DefaultDBGCDiscard {
		t.Fatalf("DBGCDiscardRatio = %f, want %f", cfg.DBGCDiscardRatio, DefaultDBGCDiscard)
	}
	if cfg.CostPolicy.AstritePerPull != DefaultAstritePerPull {
		t.Fatalf("CostPolicy.AstritePerPull = %d, want %d", cfg.CostPolicy.AstritePerPull, DefaultAstritePerPull)
	}
	if cfg.TrackingURL != DefaultTrackingURL {
		t.Fatalf("TrackingURL = %q, want %q", cfg.TrackingURL, DefaultTrackingURL)
	}
	if cfg.ResourcesURL != DefaultResourcesURL {
		t.Fatalf("ResourcesURL = %q, want %q", cfg.ResourcesURL, DefaultResourcesURL)
	}
	if !slices.Equal(cfg.StandardFiveStarResources, DefaultStandardFiveStarResources) {
		t.Fatalf("StandardFiveStarResources = %v, want %v", cfg.StandardFiveStarResources, DefaultStandardFiveStarResources)
	}
	if !reflect.DeepEqual(cfg.GachaTypes.Items, DefaultGachaTypes) {
		t.Fatalf("GachaTypes.Items = %+v, want %+v", cfg.GachaTypes.Items, DefaultGachaTypes)
	}
	if !reflect.DeepEqual(cfg.LuckScoreThresholds, DefaultLuckScoreThresholds) {
		t.Fatalf("LuckScoreThresholds = %+v, want %+v", cfg.LuckScoreThresholds, DefaultLuckScoreThresholds)
	}
	if !slices.Equal(cfg.ScanLogPaths, DefaultScanLogPaths) {
		t.Fatalf("ScanLogPaths = %v, want %v", cfg.ScanLogPaths, DefaultScanLogPaths)
	}

	wantCORSOrigins := []string{DefaultCORSLocalhost, DefaultCORSLoopback, DefaultCORSIPv6}
	if !slices.Equal(cfg.CORSOrigins, wantCORSOrigins) {
		t.Fatalf("CORSOrigins = %v, want %v", cfg.CORSOrigins, wantCORSOrigins)
	}
}

func TestDefaultDBPathUsesHomeDir(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		t.Skipf("user home directory unavailable: %v", err)
	}

	want := filepath.Join(homeDir, DefaultAppDirName, DefaultDBDirName)
	if DefaultDBPath != want {
		t.Fatalf("DefaultDBPath = %q, want %q", DefaultDBPath, want)
	}
	if cfg := Default(); cfg.DBPath != want {
		t.Fatalf("Default().DBPath = %q, want %q", cfg.DBPath, want)
	}
}

func TestDefaultConfigDoesNotShareSliceBackingArrays(t *testing.T) {
	cfg := Default()

	cfg.StandardFiveStarResources[0] = 9999
	cfg.GachaTypes.Items[0].Key = "changed"
	cfg.LuckScoreThresholds[0].State = "changed"
	cfg.ScanLogPaths[0] = "changed"
	cfg.CORSOrigins[0] = "changed"

	next := Default()
	if next.StandardFiveStarResources[0] == 9999 {
		t.Fatal("StandardFiveStarResources shares backing array")
	}
	if next.GachaTypes.Items[0].Key == "changed" {
		t.Fatal("GachaTypes shares backing array")
	}
	if next.LuckScoreThresholds[0].State == "changed" {
		t.Fatal("LuckScoreThresholds shares backing array")
	}
	if next.ScanLogPaths[0] == "changed" {
		t.Fatal("ScanLogPaths shares backing array")
	}
	if next.CORSOrigins[0] == "changed" {
		t.Fatal("CORSOrigins shares backing array")
	}
}

func TestNewDefaultConfigAppliesEnv(t *testing.T) {
	t.Setenv(EnvVarHost, "0.0.0.0")
	t.Setenv(EnvVarPort, "8080")
	t.Setenv(EnvVarDBPath, "tmp/db")
	t.Setenv(EnvVarCORSOrigins, "http://localhost:5173, http://example.com,")

	cfg := Default()

	if cfg.ServerHost != "0.0.0.0" {
		t.Fatalf("ServerHost = %q, want %q", cfg.ServerHost, "0.0.0.0")
	}
	if cfg.ServerPort != "8080" {
		t.Fatalf("ServerPort = %q, want %q", cfg.ServerPort, "8080")
	}
	if cfg.DBPath != "tmp/db" {
		t.Fatalf("DBPath = %q, want %q", cfg.DBPath, "tmp/db")
	}

	wantCORSOrigins := []string{"http://localhost:5173", "http://example.com"}
	if !slices.Equal(cfg.CORSOrigins, wantCORSOrigins) {
		t.Fatalf("CORSOrigins = %v, want %v", cfg.CORSOrigins, wantCORSOrigins)
	}
}
