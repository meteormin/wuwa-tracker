package report

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

// CSVExporter 는 통계 데이터를 CSV 포맷으로 저장합니다.
type CSVExporter struct{}

// Export 는 stats 맵을 순회하며 배너별 통계와 5성 내역을 평탄화하여 CSV에 씁니다.
func (e *CSVExporter) Export(data types.ReportData, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// 헤더 작성
	_ = writer.Write([]string{"PlayerID", "GachaType", "GachaName", "CardPoolType", "ResourceID", "QualityLevel", "ResourceType", "Name", "Count", "Time"})

	for _, stat := range data.Stats {
		for _, rec := range stat.Records {
			_ = writer.Write([]string{
				data.PlayerID,
				fmt.Sprintf("%d", stat.GachaType),
				stat.GachaName,
				rec.CardPoolType,
				fmt.Sprintf("%d", rec.ResourceID),
				fmt.Sprintf("%d", rec.QualityLevel),
				rec.ResourceType,
				rec.Name,
				fmt.Sprintf("%d", rec.Count),
				rec.Time,
			})
		}
	}

	return nil
}
