package report

import (
	"encoding/json"
	"io"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

// JSONExporter 는 통계 데이터를 JSON 포맷으로 저장합니다.
type JSONExporter struct{}

// Export 는 ReportData를 pretty-print된 JSON 형태로 변환하여 w에 저장합니다.
func (e *JSONExporter) Export(w io.Writer, data types.ReportData) error {
	res, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(res)
	return err
}
