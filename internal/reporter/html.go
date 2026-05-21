package report

import (
	"html/template"
	"io"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/types"
	"github.com/meteormin/wuwa-tracker/templates"
)

// HTMLExporter 는 통계 데이터를 Tailwind CSS가 적용된 HTML 포맷으로 저장합니다.
type HTMLExporter struct {
	cfg *config.Config
}

type htmlContext struct {
	PlayerID            string
	Stats               []types.Stats
	LuckScoreThresholds []types.LuckScoreThreshold
}

// Export 는 stats 맵을 HTML 템플릿에 주입하여 w에 보고서를 씁니다.
func (e *HTMLExporter) Export(w io.Writer, data types.ReportData) error {
	tmpl, err := template.New("report").ParseFS(templates.HTML, "html/report.tmpl")
	if err != nil {
		return err
	}

	ctx := htmlContext{
		PlayerID:            data.PlayerID,
		Stats:               data.Stats,
		LuckScoreThresholds: e.cfg.LuckScoreThresholds,
	}

	return tmpl.ExecuteTemplate(w, "report.tmpl", ctx)
}
