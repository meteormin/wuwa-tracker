package report

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/meteormin/wuwa-tracker/internal/tracker"
)

// CSVExporter 는 통계 데이터를 CSV 포맷으로 저장합니다.
type CSVExporter struct{}

// Export 는 stats 맵을 순회하며 배너별 통계와 5성 내역을 평탄화하여 CSV에 씁니다.
func (e *CSVExporter) Export(stats map[int]tracker.Stats, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// 헤더 작성
	_ = writer.Write([]string{"GachaType", "TotalPulls", "CurrentPity5", "CurrentPity4", "5Star_Name", "5Star_Time", "5Star_Pity", "5Star_IsPickUp"})

	for gachaType, stat := range stats {
		if len(stat.FiveStars) == 0 {
			_ = writer.Write([]string{
				fmt.Sprintf("%d", gachaType),
				fmt.Sprintf("%d", stat.TotalPulls),
				fmt.Sprintf("%d", stat.CurrentPity5),
				fmt.Sprintf("%d", stat.CurrentPity4),
				"", "", "", "",
			})
			continue
		}

		// 5성 획득 내역이 있는 경우 각각의 로우로 작성
		for _, fs := range stat.FiveStars {
			_ = writer.Write([]string{
				fmt.Sprintf("%d", gachaType),
				fmt.Sprintf("%d", stat.TotalPulls),
				fmt.Sprintf("%d", stat.CurrentPity5),
				fmt.Sprintf("%d", stat.CurrentPity4),
				fs.Name,
				fs.Time,
				fmt.Sprintf("%d", fs.Pity),
				fmt.Sprintf("%t", fs.IsPickUp),
			})
		}
	}

	return nil
}
