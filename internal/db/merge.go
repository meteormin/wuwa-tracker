package db

import (
	"sort"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

// MergeRecords는 새로 가져오거나 업로드한 가챠 기록(newRecords)과 기존 DB의 기록(dbRecords)을 병합합니다.
// 두 슬라이스는 모두 최신순(인덱스 0이 가장 최신)으로 정렬되어 있어야 합니다.
func MergeRecords(dbRecords, newRecords []types.Record) []types.Record {
	if len(dbRecords) == 0 {
		return newRecords
	}
	if len(newRecords) == 0 {
		return dbRecords
	}

	// 1. 시퀀스 매칭을 통한 오버랩 찾기
	// newRecords의 특정 인덱스 k부터 시작하는 접미사(suffix)가 dbRecords의 접두사(prefix)와 일치하는지 확인합니다.
	// 이 방식은 10연차 내에 존재하는 동일한 아이템 중복까지 순서를 보존하며 완벽하게 처리합니다.
	for k := range newRecords {
		suffixLen := len(newRecords) - k
		if suffixLen > len(dbRecords) {
			continue
		}

		match := true
		for i := range suffixLen {
			r1 := newRecords[k+i]
			r2 := dbRecords[i]
			if r1.ResourceID != r2.ResourceID || r1.Time != r2.Time {
				match = false
				break
			}
		}

		if match {
			// 오버랩 지점 찾기, newRecords[0:k]가 DB에 없는 새로운 기록들입니다.
			// 이를 기존 DB 기록의 맨 앞에 붙여 병합합니다.
			merged := make([]types.Record, 0, k+len(dbRecords))
			merged = append(merged, newRecords[0:k]...)
			merged = append(merged, dbRecords...)
			return merged
		}
	}

	// 2. 오버랩이 매칭되지 않는 경우 (6개월 이상의 공백이 있거나 서로 다른 데이터셋인 경우)
	// 시간대 비교를 통해 통째로 앞/뒤에 붙일 수 있는지 판단합니다.
	newestDBTime := dbRecords[0].Time
	oldestNewTime := newRecords[len(newRecords)-1].Time

	if oldestNewTime >= newestDBTime {
		// 새로운 기록이 기존 기록보다 전부 최신인 경우
		merged := make([]types.Record, 0, len(newRecords)+len(dbRecords))
		merged = append(merged, newRecords...)
		merged = append(merged, dbRecords...)
		return merged
	}

	newestNewTime := newRecords[0].Time
	oldestDBTime := dbRecords[len(dbRecords)-1].Time

	if newestNewTime <= oldestDBTime {
		// 새로운 기록이 기존 기록보다 전부 과거인 경우
		merged := make([]types.Record, 0, len(newRecords)+len(dbRecords))
		merged = append(merged, dbRecords...)
		merged = append(merged, newRecords...)
		return merged
	}

	// 3. 최악의 케이스: 시간대가 서로 교차하지만 시퀀스 매칭이 실패한 경우 (일부 데이터 유실 등)
	// 시간 기반 유니온 병합을 수행하여 유실 없이 합칩니다.
	return unionMerge(dbRecords, newRecords)
}

// unionMerge는 시간대별로 기록들을 묶어 중복을 최소화하면서 병합합니다.
func unionMerge(dbRecords, newRecords []types.Record) []types.Record {
	groups := make(map[string][]types.Record)
	allTimesMap := make(map[string]bool)

	for _, r := range dbRecords {
		groups[r.Time] = append(groups[r.Time], r)
		allTimesMap[r.Time] = true
	}

	for _, r := range newRecords {
		if !allTimesMap[r.Time] {
			groups[r.Time] = append(groups[r.Time], r)
			allTimesMap[r.Time] = true
		}
	}

	newTimesMap := make(map[string]bool)
	for _, r := range newRecords {
		newTimesMap[r.Time] = true
	}

	// 양쪽에 모두 존재하는 시간대의 경우 아이템 ID별 최대 빈도수를 보존하여 병합
	for t := range groups {
		if newTimesMap[t] && len(dbRecords) > 0 {
			var dbItems []types.Record
			for _, r := range dbRecords {
				if r.Time == t {
					dbItems = append(dbItems, r)
				}
			}
			var newItems []types.Record
			for _, r := range newRecords {
				if r.Time == t {
					newItems = append(newItems, r)
				}
			}

			if len(dbItems) > 0 && len(newItems) > 0 {
				dbFreq := make(map[int]int)
				dbTemplates := make(map[int]types.Record)
				for _, r := range dbItems {
					dbFreq[r.ResourceID]++
					dbTemplates[r.ResourceID] = r
				}

				newFreq := make(map[int]int)
				newTemplates := make(map[int]types.Record)
				for _, r := range newItems {
					newFreq[r.ResourceID]++
					newTemplates[r.ResourceID] = r
				}

				mergedItems := []types.Record{}
				allResourceIDs := make(map[int]bool)
				for id := range dbFreq {
					allResourceIDs[id] = true
				}
				for id := range newFreq {
					allResourceIDs[id] = true
				}

				for id := range allResourceIDs {
					count := max(newFreq[id], dbFreq[id])

					template := dbTemplates[id]
					if tRecord, ok := newTemplates[id]; ok {
						template = tRecord
					}

					for range count {
						mergedItems = append(mergedItems, template)
					}
				}
				groups[t] = mergedItems
			}
		}
	}

	merged := []types.Record{}
	for _, items := range groups {
		merged = append(merged, items...)
	}

	// 최신순으로 정렬
	sortRecords(merged)

	return merged
}

func sortRecords(s []types.Record) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Time != s[j].Time {
			return s[i].Time > s[j].Time
		}
		if s[i].QualityLevel != s[j].QualityLevel {
			return s[i].QualityLevel > s[j].QualityLevel
		}
		return s[i].ResourceID < s[j].ResourceID
	})
}
