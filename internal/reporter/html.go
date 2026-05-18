package report

import (
	"encoding/json"
	"html/template"
	"os"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/types"
	"github.com/meteormin/wuwa-tracker/templates"
)

// HTMLExporter 는 통계 데이터를 Tailwind CSS가 적용된 HTML 포맷으로 저장합니다.
type HTMLExporter struct{}

type htmlContext struct {
	Stats                   []types.Stats
	StatsJSON               string
	LuckScoreThresholds     []types.LuckScoreThreshold
	LuckScoreThresholdsJSON string
}

// Export 는 stats 맵을 HTML 템플릿에 주입하여 보고서 파일을 생성합니다.
func (e *HTMLExporter) Export(stats []types.Stats, outputPath string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	tmpl, err := template.New("report").ParseFS(templates.HTML, "html/report.tmpl")
	if err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	jsonData, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	thresholdsJSON, err := json.Marshal(cfg.LuckScoreThresholds)
	if err != nil {
		return err
	}

	ctx := htmlContext{
		Stats:                   stats,
		StatsJSON:               string(jsonData),
		LuckScoreThresholds:     cfg.LuckScoreThresholds,
		LuckScoreThresholdsJSON: string(thresholdsJSON),
	}

	return tmpl.ExecuteTemplate(f, "report.tmpl", ctx)
}
