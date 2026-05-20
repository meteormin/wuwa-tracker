package report

import (
	"encoding/json"
	"os"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

// JSONExporter 는 통계 데이터를 JSON 포맷으로 저장합니다.
type JSONExporter struct{}

// Export 는 ReportData를 pretty-print된 JSON 형태로 변환하여 outputPath에 저장합니다.
func (e *JSONExporter) Export(data types.ReportData, outputPath string) error {
	res, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, res, 0o644)
}
