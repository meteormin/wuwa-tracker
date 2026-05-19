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
		GachaType:     gachaType.ID,
		GachaName:     gachaType.Name,
		TotalPulls:    len(records),
		BaseRate:      gachaType.BaseRate,
		ExpectedPulls: gachaType.ExpectedPulls,
		FiveStars:     []types.FiveStarRecord{},
		Records:       []types.Record{},
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

	// 운 점수(Luck Score) 및 관련 통계 지표를 Go 백엔드 측에서 직접 계산합니다.
	fiveStarCount := len(stats.FiveStars)
	if fiveStarCount > 0 {
		stats.HasFiveStar = true
		sumPity := 0
		for _, fs := range stats.FiveStars {
			sumPity += fs.Pity
		}
		stats.AvgPulls = float64(sumPity) / float64(fiveStarCount)
		if stats.TotalPulls > 0 {
			stats.ActualRate = (float64(fiveStarCount) / float64(stats.TotalPulls)) * 100.0
		}
		if stats.AvgPulls > 0 {
			// 운 점수 계산: 픽업 캐릭터 획득 주기(PickUp Cycle) 기반으로 산출
			// 픽업 캐릭터를 획득하기까지 소모된 총 누적 풀(Pulls) 수와 단일 5성 기대치(ExpectedPulls)를 비교
			expectedTotal := 0
			actualTotal := 0
			currentCyclePulls := 0

			for _, fs := range stats.FiveStars {
				currentCyclePulls += fs.Pity
				if !gachaType.HasOffBannerDrop {
					// 상시 또는 무기 가챠처럼 픽뚫이 없는 경우, 모든 5성 획득이 개별 주기 완성
					expectedTotal += gachaType.ExpectedPulls
					actualTotal += currentCyclePulls
					currentCyclePulls = 0
				} else {
					if fs.IsPickUp {
						// 한정 공명자 배너에서 픽업 캐릭터를 뽑은 경우 주기 완성
						expectedTotal += gachaType.ExpectedPulls
						actualTotal += currentCyclePulls
						currentCyclePulls = 0
					}
				}
			}

			// 아직 픽업을 뽑지 못하고 상시 5성(픽뚫) 상태에서 끝난 진행 중인 주기 반영
			if currentCyclePulls > 0 {
				expectedTotal += gachaType.ExpectedPulls
				actualTotal += currentCyclePulls
			}

			if actualTotal > 0 {
				stats.LuckScore = (float64(expectedTotal) / float64(actualTotal)) * 100
			} else {
				stats.LuckScore = 0.0
			}
		}
	} else {
		stats.HasFiveStar = false
		stats.AvgPulls = 0
		stats.ActualRate = 0
		stats.LuckScore = 0
	}

	// 최신 획득 내역이 배열의 앞쪽에 오도록 FiveStars 를 뒤집습니다.
	reverseFiveStars(stats.FiveStars)

	return stats
}

func reverseFiveStars(s []types.FiveStarRecord) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
