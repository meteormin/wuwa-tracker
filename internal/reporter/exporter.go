package report

import (
	"fmt"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
	FormatHTML Format = "html"
)

// Exporter 인터페이스는 다양한 포맷으로 통계 데이터를 출력하는 기능을 정의합니다.
type Exporter interface {
	Export(data types.ReportData, outputPath string) error
}

// NewExporter 는 주어진 포맷 이름에 따라 적절한 Exporter를 반환합니다.
func NewExporter(cfg *config.Config, format Format) (Exporter, error) {
	switch format {
	case FormatJSON:
		return &JSONExporter{}, nil
	case FormatCSV:
		return &CSVExporter{}, nil
	case FormatHTML:
		return &HTMLExporter{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
