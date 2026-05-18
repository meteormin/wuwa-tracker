package report

import (
	"html/template"
	"os"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/types"
	"github.com/meteormin/wuwa-tracker/templates"
)

// HTMLExporter 는 통계 데이터를 Tailwind CSS가 적용된 HTML 포맷으로 저장합니다.
type HTMLExporter struct{}

type htmlContext struct {
	Stats               []types.Stats
	LuckScoreThresholds []types.LuckScoreThreshold
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

	ctx := htmlContext{
		Stats:               stats,
		LuckScoreThresholds: cfg.LuckScoreThresholds,
	}

	return tmpl.ExecuteTemplate(f, "report.tmpl", ctx)
}
