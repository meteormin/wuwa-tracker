package report

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/types"
	"github.com/meteormin/wuwa-tracker/locales"
	"github.com/meteormin/wuwa-tracker/templates"
)

// HTMLExporter 는 통계 데이터를 Tailwind CSS가 적용된 HTML 포맷으로 저장합니다.
type HTMLExporter struct {
	cfg  *config.Config
	lang string
}

type htmlContext struct {
	PlayerID            string
	Lang                string
	Stats               []types.Stats
	LuckScoreThresholds []types.LuckScoreThreshold
}

// Export 는 stats 맵을 HTML 템플릿에 주입하여 w에 보고서를 씁니다.
func (e *HTMLExporter) Export(w io.Writer, data types.ReportData) error {
	resolvedLang, translations, err := locales.LoadUITranslationsWithFallback(e.lang)
	if err != nil {
		return err
	}

	translate := func(key string, args ...any) string {
		text, ok := translations[key]
		if !ok {
			text = key
		}
		for i := 0; i+1 < len(args); i += 2 {
			name, ok := args[i].(string)
			if !ok {
				continue
			}
			text = strings.ReplaceAll(text, "{"+name+"}", fmt.Sprint(args[i+1]))
		}
		return text
	}

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"t": translate,
	}).ParseFS(templates.HTML, "html/report.tmpl")
	if err != nil {
		return err
	}

	ctx := htmlContext{
		PlayerID:            data.PlayerID,
		Lang:                resolvedLang,
		Stats:               data.Stats,
		LuckScoreThresholds: e.cfg.LuckScoreThresholds,
	}

	return tmpl.ExecuteTemplate(w, "report.tmpl", ctx)
}
