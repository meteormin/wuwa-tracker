package tracker

import (
	"github.com/meteormin/wuwa-tracker/internal/types"
)

type StatsCalulator struct {
	standardFiveStarResources types.StandardFiveStarResources
}

func NewStatsCalculator(standardFiveStarResources types.StandardFiveStarResources) *StatsCalulator {
	return &StatsCalulator{
		standardFiveStarResources: standardFiveStarResources,
	}
}

// CalculateStats 는 배열의 Record 들을 바탕으로 통계 지표를 계산합니다.
// 입력 배열은 API에서 가져온 기본 순서(최신순)라고 가정합니다.
func (sc *StatsCalulator) CalculateStats(records []types.Record, gachaType types.GachaType) types.Stats {
	stats := types.Stats{
		GachaType:  gachaType.ID,
		GachaName:  gachaType.Name,
		TotalPulls: len(records),
		FiveStars:  []types.FiveStarRecord{},
		Records:    []types.Record{},
	}

	pity5 := 0
	pity4 := 0

	// API 응답은 최신순(내림차순)이므로 과거부터 스택을 쌓기 위해 뒤집어서 순회합니다.
	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]

		// 가챠 획득 스택 쌓기
		pity5++
		pity4++

		// 가챠 기록 저장
		stats.Records = append(stats.Records, rec)

		switch rec.QualityLevel {
		case 5:
			isPickUp := true
			// 한정 캐릭터 배너(1)인 경우에만 픽뚫을 판별합니다.
			if gachaType.HasOffBannerDrop {
				isPickUp = !sc.standardFiveStarResources.Contains(rec.ResourceID)
			}
			stats.FiveStars = append(stats.FiveStars, types.FiveStarRecord{
				Name:     rec.Name,
				Time:     rec.Time,
				Pity:     pity5,
				IsPickUp: isPickUp,
			})
			pity5 = 0
		case 4:
			pity4 = 0
		}
	}

	stats.CurrentPity5 = pity5
	stats.CurrentPity4 = pity4

	// 최신 획득 내역이 배열의 앞쪽에 오도록 FiveStars 를 뒤집습니다.
	reverseFiveStars(stats.FiveStars)

	return stats
}

func reverseFiveStars(s []types.FiveStarRecord) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
