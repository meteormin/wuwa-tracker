package report

import (
	"encoding/json"
	"os"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

// JSONExporter 는 통계 데이터를 JSON 포맷으로 저장합니다.
type JSONExporter struct{}

// Export 는 stats 맵을 pretty-print된 JSON 형태로 변환하여 outputPath에 저장합니다.
func (e *JSONExporter) Export(stats []types.Stats, outputPath string) error {
	type FlatRecord struct {
		GachaType    int    `json:"gachaType"`
		GachaName    string `json:"gachaName"`
		CardPoolType string `json:"cardPoolType"`
		ResourceID   int    `json:"resourceId"`
		QualityLevel int    `json:"qualityLevel"`
		ResourceType string `json:"resourceType"`
		Name         string `json:"name"`
		Count        int    `json:"count"`
		Time         string `json:"time"`
	}

	var flatRecords []FlatRecord
	for _, stat := range stats {
		for _, rec := range stat.Records {
			flatRecords = append(flatRecords, FlatRecord{
				GachaType:    stat.GachaType,
				GachaName:    stat.GachaName,
				CardPoolType: rec.CardPoolType,
				ResourceID:   rec.ResourceID,
				QualityLevel: rec.QualityLevel,
				ResourceType: rec.ResourceType,
				Name:         rec.Name,
				Count:        rec.Count,
				Time:         rec.Time,
			})
		}
	}

	data, err := json.MarshalIndent(flatRecords, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0o644)
}
