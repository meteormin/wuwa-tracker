package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScanURLMissingPath(t *testing.T) {
	svc := &Service{}

	_, err := svc.ScanURL("  ")
	if !errors.Is(err, ErrMissingScanPath) {
		t.Fatalf("ScanURL error = %v, want %v", err, ErrMissingScanPath)
	}
}

func TestScanURLFromLogFile(t *testing.T) {
	svc := &Service{}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "Client.log")
	wantURL := "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record?player_id=123"

	if err := os.WriteFile(logPath, []byte("before\n"+wantURL+"\nafter\n"), 0o644); err != nil {
		t.Fatalf("failed to write test log: %v", err)
	}

	gotURL, err := svc.ScanURL(logPath)
	if err != nil {
		t.Fatalf("ScanURL returned error: %v", err)
	}
	if gotURL != wantURL {
		t.Fatalf("ScanURL = %q, want %q", gotURL, wantURL)
	}
}
