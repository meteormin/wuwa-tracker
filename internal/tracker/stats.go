package tracker

import "slices"

// StandardFiveStarResourceIDs 는 상시 5성 캐릭터들의 resourceId 목록입니다.
// 앙코 : 1203
// 카카루 : 1301
// 감심 : 1405
// 능양 : 1104
// 벨리나 : 1503
var StandardFiveStarResourceIDs = []int{1203, 1301, 1405, 1104, 1503}

func isStandardFiveStar(resourceId int) bool {
	return slices.Contains(StandardFiveStarResourceIDs, resourceId)
}

// CalculateStats 는 배열의 Record 들을 바탕으로 통계 지표를 계산합니다.
// 입력 배열은 API에서 가져온 기본 순서(최신순)라고 가정합니다.
func CalculateStats(gachaType int, records []Record) Stats {
	stats := Stats{
		GachaType:  gachaType,
		TotalPulls: len(records),
		FiveStars:  []FiveStarRecord{},
	}

	pity5 := 0
	pity4 := 0

	// API 응답은 최신순(내림차순)이므로 과거부터 스택을 쌓기 위해 뒤집어서 순회합니다.
	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]
		pity5++
		pity4++

		switch rec.QualityLevel {
		case 5:
			isPickUp := true
			// 한정 캐릭터 배너(1)인 경우에만 픽뚫을 판별합니다.
			if gachaType == 1 {
				isPickUp = !isStandardFiveStar(rec.ResourceID)
			}
			stats.FiveStars = append(stats.FiveStars, FiveStarRecord{
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

func reverseFiveStars(s []FiveStarRecord) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
