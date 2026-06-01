package db

import (
	"reflect"
	"sort"
	"testing"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

func TestBadgerDB_GetGachaRecordsMissingKey(t *testing.T) {
	database := openTestBadgerDB(t)

	records, err := database.GetGachaRecords("player-1", "character")
	if err != nil {
		t.Fatalf("GetGachaRecords returned error: %v", err)
	}
	if records == nil {
		t.Fatal("GetGachaRecords returned nil records")
	}
	if len(records) != 0 {
		t.Fatalf("expected empty records, got %+v", records)
	}
}

func TestBadgerDB_SaveAndGetGachaRecords(t *testing.T) {
	database := openTestBadgerDB(t)
	records := []types.Record{
		testRecord(1001, "character", "Alpha", "2026-05-20 12:00:00"),
		testRecord(1002, "character", "Beta", "2026-05-19 12:00:00"),
	}

	if err := database.SaveGachaRecords("player-1", "character", records); err != nil {
		t.Fatalf("SaveGachaRecords returned error: %v", err)
	}

	got, err := database.GetGachaRecords("player-1", "character")
	if err != nil {
		t.Fatalf("GetGachaRecords returned error: %v", err)
	}
	if !reflect.DeepEqual(got, records) {
		t.Fatalf("records mismatch\nexpected: %+v\nactual:   %+v", records, got)
	}
}

func TestBadgerDB_SaveGachaRecordsMergesExistingRecords(t *testing.T) {
	database := openTestBadgerDB(t)

	existing := []types.Record{
		testRecord(1003, "weapon", "Gamma", "2026-05-18 12:00:00"),
		testRecord(1004, "weapon", "Delta", "2026-05-17 12:00:00"),
	}
	incoming := []types.Record{
		testRecord(1001, "weapon", "Alpha", "2026-05-20 12:00:00"),
		testRecord(1002, "weapon", "Beta", "2026-05-19 12:00:00"),
		testRecord(1003, "weapon", "Gamma", "2026-05-18 12:00:00"),
	}
	expected := []types.Record{
		testRecord(1001, "weapon", "Alpha", "2026-05-20 12:00:00"),
		testRecord(1002, "weapon", "Beta", "2026-05-19 12:00:00"),
		testRecord(1003, "weapon", "Gamma", "2026-05-18 12:00:00"),
		testRecord(1004, "weapon", "Delta", "2026-05-17 12:00:00"),
	}

	if err := database.SaveGachaRecords("player-1", "weapon", existing); err != nil {
		t.Fatalf("SaveGachaRecords existing returned error: %v", err)
	}
	if err := database.SaveGachaRecords("player-1", "weapon", incoming); err != nil {
		t.Fatalf("SaveGachaRecords incoming returned error: %v", err)
	}

	got, err := database.GetGachaRecords("player-1", "weapon")
	if err != nil {
		t.Fatalf("GetGachaRecords returned error: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("records mismatch\nexpected: %+v\nactual:   %+v", expected, got)
	}
}

func TestBadgerDB_SaveGachaRecordsNilInputStoresEmptySlice(t *testing.T) {
	database := openTestBadgerDB(t)

	if err := database.SaveGachaRecords("player-1", "character", nil); err != nil {
		t.Fatalf("SaveGachaRecords returned error: %v", err)
	}

	records, err := database.GetGachaRecords("player-1", "character")
	if err != nil {
		t.Fatalf("GetGachaRecords returned error: %v", err)
	}
	if records == nil {
		t.Fatal("GetGachaRecords returned nil records")
	}
	if len(records) != 0 {
		t.Fatalf("expected empty records, got %+v", records)
	}
}

func TestBadgerDB_ListPlayers(t *testing.T) {
	database := openTestBadgerDB(t)

	saves := []struct {
		playerID     string
		cardPoolType string
		record       types.Record
	}{
		{"player-2", "character", testRecord(2001, "character", "Charlie", "2026-05-20 12:00:00")},
		{"player-1", "weapon", testRecord(1001, "weapon", "Alpha", "2026-05-19 12:00:00")},
		{"player-1", "character", testRecord(1002, "character", "Beta", "2026-05-18 12:00:00")},
	}

	for _, save := range saves {
		if err := database.SaveGachaRecords(save.playerID, save.cardPoolType, []types.Record{save.record}); err != nil {
			t.Fatalf("SaveGachaRecords(%q, %q) returned error: %v", save.playerID, save.cardPoolType, err)
		}
	}

	players, err := database.ListPlayers()
	if err != nil {
		t.Fatalf("ListPlayers returned error: %v", err)
	}
	sort.Strings(players)

	expected := []string{"player-1", "player-2"}
	if !reflect.DeepEqual(players, expected) {
		t.Fatalf("players mismatch\nexpected: %+v\nactual:   %+v", expected, players)
	}
}

func openTestBadgerDB(t *testing.T) *BadgerDB {
	t.Helper()

	database, err := NewBadgerDB(t.TempDir())
	if err != nil {
		t.Fatalf("NewBadgerDB returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})

	return database
}

func testRecord(resourceID int, cardPoolType, name, time string) types.Record {
	return types.Record{
		CardPoolType: cardPoolType,
		ResourceID:   resourceID,
		QualityLevel: 4,
		ResourceType: "Weapon",
		Name:         name,
		Count:        1,
		Time:         time,
	}
}
