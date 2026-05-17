package report

import (
	"encoding/json"
	"os"

	"github.com/meteormin/wuwa-tracker/internal/tracker"
)

// JSONExporter 는 통계 데이터를 JSON 포맷으로 저장합니다.
type JSONExporter struct{}

// Export 는 stats 맵을 pretty-print된 JSON 형태로 변환하여 outputPath에 저장합니다.
func (e *JSONExporter) Export(stats map[int]tracker.Stats, outputPath string) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0o644)
}
