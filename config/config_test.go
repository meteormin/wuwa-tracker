package config

import (
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

	wantCORSOrigins := []string{DefaultCORSLocalhost, DefaultCORSLoopback, DefaultCORSIPv6}
	if !slices.Equal(cfg.CORSOrigins, wantCORSOrigins) {
		t.Fatalf("CORSOrigins = %v, want %v", cfg.CORSOrigins, wantCORSOrigins)
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
