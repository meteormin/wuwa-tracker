package tracker

import (
	"reflect"
	"testing"
)

func TestCalculateStats(t *testing.T) {
	// API는 최신순(내림차순)으로 반환하므로, 배열 앞쪽에 있는 항목이 더 최근 항목입니다.
	records := []Record{
		{Name: "음림", ResourceID: 1502, QualityLevel: 5, Time: "2024-06-10 10:00:00"},
		{Name: "3성무기C", ResourceID: 21010013, QualityLevel: 3, Time: "2024-06-09 10:00:00"},
		{Name: "모르테피", ResourceID: 1401, QualityLevel: 4, Time: "2024-06-08 10:00:00"},
		{Name: "3성무기B", ResourceID: 21010013, QualityLevel: 3, Time: "2024-06-07 10:00:00"},
		{Name: "앙코", ResourceID: 1105, QualityLevel: 5, Time: "2024-06-06 10:00:00"},
		{Name: "3성무기A", ResourceID: 21010013, QualityLevel: 3, Time: "2024-06-05 10:00:00"},
	}

	// 기염은 2번째(인덱스 5, 4번)이므로 1번째 3성무기A 다음 -> 스택 2에서 등장. (과거부터 계산)
	// 과거 순서:
	// 3성무기A (pity5=1, pity4=1)
	// 기염 (pity5=2, pity4=2) -> 5성 천장 초기화 (pity5=0, pity4=2)
	// 3성무기B (pity5=1, pity4=3)
	// 모르테피 (pity5=2, pity4=4) -> 4성 천장 초기화 (pity5=2, pity4=0)
	// 3성무기C (pity5=3, pity4=1)
	// 음림 (pity5=4, pity4=2) -> 5성 천장 초기화 (pity5=0, pity4=2)

	stats := CalculateStats(1, records)

	if stats.TotalPulls != 6 {
		t.Errorf("Expected TotalPulls 6, got %d", stats.TotalPulls)
	}

	if stats.CurrentPity5 != 0 {
		t.Errorf("Expected CurrentPity5 0, got %d", stats.CurrentPity5)
	}

	if stats.CurrentPity4 != 2 {
		t.Errorf("Expected CurrentPity4 2, got %d", stats.CurrentPity4)
	}

	if len(stats.FiveStars) != 2 {
		t.Errorf("Expected 2 FiveStars, got %d", len(stats.FiveStars))
	}

	// FiveStars는 최신순이 되도록 뒤집었으므로 음림이 첫번째여야 함
	expectedFiveStars := []FiveStarRecord{
		{Name: "음림", Time: "2024-06-10 10:00:00", Pity: 4, IsPickUp: true},
		{Name: "앙코", Time: "2024-06-06 10:00:00", Pity: 2, IsPickUp: false},
	}

	if !reflect.DeepEqual(stats.FiveStars, expectedFiveStars) {
		t.Errorf("Expected FiveStars %+v, got %+v", expectedFiveStars, stats.FiveStars)
	}
}
