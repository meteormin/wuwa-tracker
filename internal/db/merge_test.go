package db

import (
	"reflect"
	"testing"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

func TestMergeRecords_Overlap(t *testing.T) {
	dbRecords := []types.Record{
		{ResourceID: 3, Time: "2026-05-18 12:00:00", Name: "C"},
		{ResourceID: 4, Time: "2026-05-17 12:00:00", Name: "D"},
		{ResourceID: 5, Time: "2026-05-16 12:00:00", Name: "E"},
	}

	newRecords := []types.Record{
		{ResourceID: 1, Time: "2026-05-20 12:00:00", Name: "A"},
		{ResourceID: 2, Time: "2026-05-19 12:00:00", Name: "B"},
		{ResourceID: 3, Time: "2026-05-18 12:00:00", Name: "C"},
		{ResourceID: 4, Time: "2026-05-17 12:00:00", Name: "D"},
	}

	expected := []types.Record{
		{ResourceID: 1, Time: "2026-05-20 12:00:00", Name: "A"},
		{ResourceID: 2, Time: "2026-05-19 12:00:00", Name: "B"},
		{ResourceID: 3, Time: "2026-05-18 12:00:00", Name: "C"},
		{ResourceID: 4, Time: "2026-05-17 12:00:00", Name: "D"},
		{ResourceID: 5, Time: "2026-05-16 12:00:00", Name: "E"},
	}

	result := MergeRecords(dbRecords, newRecords)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected overlap merge to match expected, got: %+v", result)
	}
}

func TestMergeRecords_OverlapWithDuplicates(t *testing.T) {
	// 10연차 등으로 같은 시간대에 동일한 아이템 ID가 중복된 경우 시퀀스 일치 여부 테스트
	dbRecords := []types.Record{
		{ResourceID: 100, Time: "2026-05-18 12:00:00", Name: "ItemX"},
		{ResourceID: 100, Time: "2026-05-18 12:00:00", Name: "ItemX"}, // 중복 아이템
		{ResourceID: 200, Time: "2026-05-17 12:00:00", Name: "ItemY"},
	}

	newRecords := []types.Record{
		{ResourceID: 300, Time: "2026-05-19 12:00:00", Name: "ItemZ"},
		{ResourceID: 100, Time: "2026-05-18 12:00:00", Name: "ItemX"},
		{ResourceID: 100, Time: "2026-05-18 12:00:00", Name: "ItemX"},
	}

	expected := []types.Record{
		{ResourceID: 300, Time: "2026-05-19 12:00:00", Name: "ItemZ"},
		{ResourceID: 100, Time: "2026-05-18 12:00:00", Name: "ItemX"},
		{ResourceID: 100, Time: "2026-05-18 12:00:00", Name: "ItemX"},
		{ResourceID: 200, Time: "2026-05-17 12:00:00", Name: "ItemY"},
	}

	result := MergeRecords(dbRecords, newRecords)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected overlap merge with duplicates to match, got: %+v", result)
	}
}

func TestMergeRecords_NoOverlap_Newer(t *testing.T) {
	// 오버랩은 없으나 새로운 레코드가 전적으로 최신인 경우 (기존 최신 날짜보다 더 나중의 과거에서부터 시작)
	dbRecords := []types.Record{
		{ResourceID: 3, Time: "2026-05-10 12:00:00", Name: "C"},
	}

	newRecords := []types.Record{
		{ResourceID: 1, Time: "2026-05-20 12:00:00", Name: "A"},
		{ResourceID: 2, Time: "2026-05-18 12:00:00", Name: "B"},
	}

	expected := []types.Record{
		{ResourceID: 1, Time: "2026-05-20 12:00:00", Name: "A"},
		{ResourceID: 2, Time: "2026-05-18 12:00:00", Name: "B"},
		{ResourceID: 3, Time: "2026-05-10 12:00:00", Name: "C"},
	}

	result := MergeRecords(dbRecords, newRecords)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected no-overlap newer merge to match, got: %+v", result)
	}
}

func TestMergeRecords_NoOverlap_Older(t *testing.T) {
	// 오버랩은 없으나 새로운 레코드가 기존 DB보다 전적으로 오래된 과거 기록인 경우 (예: 백업 데이터 추가)
	dbRecords := []types.Record{
		{ResourceID: 1, Time: "2026-05-20 12:00:00", Name: "A"},
	}

	newRecords := []types.Record{
		{ResourceID: 2, Time: "2026-05-10 12:00:00", Name: "B"},
		{ResourceID: 3, Time: "2026-05-08 12:00:00", Name: "C"},
	}

	expected := []types.Record{
		{ResourceID: 1, Time: "2026-05-20 12:00:00", Name: "A"},
		{ResourceID: 2, Time: "2026-05-10 12:00:00", Name: "B"},
		{ResourceID: 3, Time: "2026-05-08 12:00:00", Name: "C"},
	}

	result := MergeRecords(dbRecords, newRecords)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected no-overlap older merge to match, got: %+v", result)
	}
}

func TestMergeRecords_Empty(t *testing.T) {
	records := []types.Record{
		{ResourceID: 1, Time: "2026-05-20 12:00:00", Name: "A"},
	}

	if !reflect.DeepEqual(MergeRecords(nil, records), records) {
		t.Errorf("Nil dbRecords should return newRecords")
	}

	if !reflect.DeepEqual(MergeRecords(records, nil), records) {
		t.Errorf("Nil newRecords should return dbRecords")
	}
}
