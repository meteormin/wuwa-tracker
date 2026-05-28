package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

func TestHTMLExporterUsesTranslations(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	exporter, err := NewExporter(cfg, FormatHTML, "en")
	if err != nil {
		t.Fatalf("failed to create exporter: %v", err)
	}

	var buf bytes.Buffer
	err = exporter.Export(&buf, types.ReportData{
		PlayerID: "player-1",
		Stats: []types.Stats{
			{
				GachaType:     1,
				GachaName:     "Character Event",
				TotalPulls:    1,
				CurrentPity5:  1,
				BaseRate:      0.8,
				ExpectedPulls: 80,
				Records: []types.Record{
					{
						Name:         "Sample",
						QualityLevel: 3,
						ResourceType: "Weapon",
						Time:         "2026-05-27 00:00:00",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to export html: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "Tuning Statistics Report") {
		t.Fatalf("expected translated report title in html")
	}
	if !strings.Contains(html, "Player ID: player-1") {
		t.Fatalf("expected translated player id in html")
	}
}
