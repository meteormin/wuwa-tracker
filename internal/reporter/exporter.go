package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
	FormatHTML Format = "html"
)

func ParseFormat(value string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(value))) {
	case FormatJSON:
		return FormatJSON, nil
	case FormatCSV:
		return FormatCSV, nil
	case FormatHTML:
		return FormatHTML, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", value)
	}
}

// Exporter 인터페이스는 다양한 포맷으로 통계 데이터를 출력하는 기능을 정의합니다.
type Exporter interface {
	Export(w io.Writer, data types.ReportData) error
}

// NewExporter 는 주어진 포맷 이름에 따라 적절한 Exporter를 반환합니다.
func NewExporter(cfg *config.Config, format Format, lang ...string) (Exporter, error) {
	reportLang := config.DefaultLanguage
	if len(lang) > 0 && lang[0] != "" {
		reportLang = lang[0]
	}

	switch format {
	case FormatJSON:
		return &JSONExporter{}, nil
	case FormatCSV:
		return &CSVExporter{}, nil
	case FormatHTML:
		return &HTMLExporter{cfg: cfg, lang: reportLang}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
