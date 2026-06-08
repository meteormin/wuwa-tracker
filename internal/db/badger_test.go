package db

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

func TestNewBadgerRepositoryRejectsMissingCore(t *testing.T) {
	repository, err := NewBadgerRepository(nil)
	if err == nil {
		t.Fatal("NewBadgerRepository returned nil error")
	}
	if repository != nil {
		t.Fatalf("NewBadgerRepository returned repository: %#v", repository)
	}
}

func TestBadgerRepository_GetGachaRecordsMissingKey(t *testing.T) {
	repository := openTestBadgerRepository(t)

	records, err := repository.GetGachaRecords("player-1", "character")
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

func TestBadgerRepository_SaveAndGetGachaRecords(t *testing.T) {
	repository := openTestBadgerRepository(t)
	records := []types.Record{
		testRecord(1001, "character", "Alpha", "2026-05-20 12:00:00"),
		testRecord(1002, "character", "Beta", "2026-05-19 12:00:00"),
	}

	if err := repository.SaveGachaRecords("player-1", "character", records); err != nil {
		t.Fatalf("SaveGachaRecords returned error: %v", err)
	}

	got, err := repository.GetGachaRecords("player-1", "character")
	if err != nil {
		t.Fatalf("GetGachaRecords returned error: %v", err)
	}
	if !reflect.DeepEqual(got, records) {
		t.Fatalf("records mismatch\nexpected: %+v\nactual:   %+v", records, got)
	}
}

func TestBadgerRepository_SaveGachaRecordsMergesExistingRecords(t *testing.T) {
	repository := openTestBadgerRepository(t)

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

	if err := repository.SaveGachaRecords("player-1", "weapon", existing); err != nil {
		t.Fatalf("SaveGachaRecords existing returned error: %v", err)
	}
	if err := repository.SaveGachaRecords("player-1", "weapon", incoming); err != nil {
		t.Fatalf("SaveGachaRecords incoming returned error: %v", err)
	}

	got, err := repository.GetGachaRecords("player-1", "weapon")
	if err != nil {
		t.Fatalf("GetGachaRecords returned error: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("records mismatch\nexpected: %+v\nactual:   %+v", expected, got)
	}
}

func TestBadgerRepository_SaveGachaRecordsNilInputStoresEmptySlice(t *testing.T) {
	repository := openTestBadgerRepository(t)

	if err := repository.SaveGachaRecords("player-1", "character", nil); err != nil {
		t.Fatalf("SaveGachaRecords returned error: %v", err)
	}

	records, err := repository.GetGachaRecords("player-1", "character")
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

func TestBadgerRepository_SaveGachaRecordsUsesMergePolicy(t *testing.T) {
	repository := openTestBadgerRepository(t, WithMergePolicy(overwriteMergePolicy{}))

	existing := []types.Record{testRecord(1001, "character", "Old", "2026-05-19 12:00:00")}
	incoming := []types.Record{testRecord(1002, "character", "New", "2026-05-20 12:00:00")}

	if err := repository.SaveGachaRecords("player-1", "character", existing); err != nil {
		t.Fatalf("SaveGachaRecords existing returned error: %v", err)
	}
	if err := repository.SaveGachaRecords("player-1", "character", incoming); err != nil {
		t.Fatalf("SaveGachaRecords incoming returned error: %v", err)
	}

	got, err := repository.GetGachaRecords("player-1", "character")
	if err != nil {
		t.Fatalf("GetGachaRecords returned error: %v", err)
	}
	if !reflect.DeepEqual(got, incoming) {
		t.Fatalf("records mismatch\nexpected: %+v\nactual:   %+v", incoming, got)
	}
}

func TestBadgerRepository_ListPlayers(t *testing.T) {
	repository := openTestBadgerRepository(t)

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
		if err := repository.SaveGachaRecords(save.playerID, save.cardPoolType, []types.Record{save.record}); err != nil {
			t.Fatalf("SaveGachaRecords(%q, %q) returned error: %v", save.playerID, save.cardPoolType, err)
		}
	}

	players, err := repository.ListPlayers()
	if err != nil {
		t.Fatalf("ListPlayers returned error: %v", err)
	}
	sort.Strings(players)

	expected := []string{"player-1", "player-2"}
	if !reflect.DeepEqual(players, expected) {
		t.Fatalf("players mismatch\nexpected: %+v\nactual:   %+v", expected, players)
	}
}

func TestBadgerRepository_BackupAndMergeFromBackup(t *testing.T) {
	source := openTestBadgerRepository(t)
	target := openTestBadgerRepository(t)

	sourceRecords := []types.Record{
		testRecord(1001, "character", "Alpha", "2026-05-20 12:00:00"),
		testRecord(1002, "character", "Beta", "2026-05-19 12:00:00"),
	}
	targetRecords := []types.Record{
		testRecord(1002, "character", "Beta", "2026-05-19 12:00:00"),
		testRecord(1003, "character", "Gamma", "2026-05-18 12:00:00"),
	}
	weaponRecords := []types.Record{
		testRecord(2001, "weapon", "Blade", "2026-05-17 12:00:00"),
	}

	if err := source.SaveGachaRecords("player-1", "character", sourceRecords); err != nil {
		t.Fatalf("source SaveGachaRecords returned error: %v", err)
	}
	if err := source.SaveGachaRecords("player-2", "weapon", weaponRecords); err != nil {
		t.Fatalf("source SaveGachaRecords weapon returned error: %v", err)
	}
	if err := target.SaveGachaRecords("player-1", "character", targetRecords); err != nil {
		t.Fatalf("target SaveGachaRecords returned error: %v", err)
	}

	var backup bytes.Buffer
	if _, err := source.Backup(&backup); err != nil {
		t.Fatalf("Backup returned error: %v", err)
	}

	result, err := target.MergeFromBackup(bytes.NewReader(backup.Bytes()))
	if err != nil {
		t.Fatalf("MergeFromBackup returned error: %v", err)
	}
	if result.Players != 2 {
		t.Fatalf("expected 2 players, got %d", result.Players)
	}
	if result.Banners != 2 {
		t.Fatalf("expected 2 banners, got %d", result.Banners)
	}
	if result.Records != 3 {
		t.Fatalf("expected 3 imported records, got %d", result.Records)
	}

	gotCharacter, err := target.GetGachaRecords("player-1", "character")
	if err != nil {
		t.Fatalf("GetGachaRecords character returned error: %v", err)
	}
	expectedCharacter := []types.Record{
		testRecord(1001, "character", "Alpha", "2026-05-20 12:00:00"),
		testRecord(1002, "character", "Beta", "2026-05-19 12:00:00"),
		testRecord(1003, "character", "Gamma", "2026-05-18 12:00:00"),
	}
	if !reflect.DeepEqual(gotCharacter, expectedCharacter) {
		t.Fatalf("character records mismatch\nexpected: %+v\nactual:   %+v", expectedCharacter, gotCharacter)
	}

	gotWeapon, err := target.GetGachaRecords("player-2", "weapon")
	if err != nil {
		t.Fatalf("GetGachaRecords weapon returned error: %v", err)
	}
	if !reflect.DeepEqual(gotWeapon, weaponRecords) {
		t.Fatalf("weapon records mismatch\nexpected: %+v\nactual:   %+v", weaponRecords, gotWeapon)
	}
}

func TestBadgerRepository_Stats(t *testing.T) {
	repository := openTestBadgerRepository(t)

	if err := repository.SaveGachaRecords("player-1", "character", []types.Record{
		testRecord(1001, "character", "Alpha", "2026-05-20 12:00:00"),
	}); err != nil {
		t.Fatalf("SaveGachaRecords returned error: %v", err)
	}

	stats, err := repository.Stats()
	if err != nil {
		t.Fatalf("Stats returned error: %v", err)
	}
	if stats.Path == "" {
		t.Fatal("Stats returned empty path")
	}
	if stats.FileCount == 0 {
		t.Fatal("Stats returned no files")
	}
	if stats.ApparentSizeBytes <= 0 {
		t.Fatalf("ApparentSizeBytes = %d, want positive value", stats.ApparentSizeBytes)
	}
	if badgerSize := stats.LSMSizeBytes + stats.VLogSizeBytes; badgerSize > 0 && stats.ApparentSizeBytes != badgerSize {
		t.Fatalf("ApparentSizeBytes = %d, want lsm+vlog %d", stats.ApparentSizeBytes, stats.LSMSizeBytes+stats.VLogSizeBytes)
	}
	if stats.DiskUsageBytes < 0 {
		t.Fatalf("DiskUsageBytes = %d, want non-negative value", stats.DiskUsageBytes)
	}
}

func TestStatsFromPathCountsBadgerFileTypes(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"000001.vlog": "value log",
		"000002.sst":  "table",
		"000003.mem":  "memtable",
		"MANIFEST":    "manifest",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) returned error: %v", name, err)
		}
	}

	stats, err := StatsFromPath(dir)
	if err != nil {
		t.Fatalf("StatsFromPath returned error: %v", err)
	}
	if stats.FileCount != len(files) {
		t.Fatalf("FileCount = %d, want %d", stats.FileCount, len(files))
	}
	if stats.VLogCount != 1 {
		t.Fatalf("VLogCount = %d, want 1", stats.VLogCount)
	}
	if stats.SSTCount != 1 {
		t.Fatalf("SSTCount = %d, want 1", stats.SSTCount)
	}
	if stats.MemTableCount != 1 {
		t.Fatalf("MemTableCount = %d, want 1", stats.MemTableCount)
	}
}

func TestStatsFromPathRejectsFilePath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "badger-file")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := StatsFromPath(filePath); err == nil {
		t.Fatal("StatsFromPath returned nil error for file path")
	}
}

func TestBadgerRepository_RunValueLogGCNoRewrite(t *testing.T) {
	repository := openTestBadgerRepository(t)

	if err := repository.RunValueLogGC(0.5); err != nil {
		t.Fatalf("RunValueLogGC returned error: %v", err)
	}
}

func TestBadgerRepository_RunValueLogGCRejectsInvalidDiscardRatio(t *testing.T) {
	repository := openTestBadgerRepository(t)

	if err := repository.RunValueLogGC(1); err == nil {
		t.Fatal("RunValueLogGC returned nil error for invalid discard ratio")
	}
}

func TestBadgerRepository_StartValueLogGCStop(t *testing.T) {
	repository := openTestBadgerRepository(t)

	if err := repository.StartValueLogGC(t.Context(), ValueLogGCOptions{
		Interval:     time.Hour,
		DiscardRatio: 0.5,
	}, nil); err != nil {
		t.Fatalf("StartValueLogGC returned error: %v", err)
	}

	repository.StopValueLogGC()
	repository.StopValueLogGC()
}

type overwriteMergePolicy struct{}

func (overwriteMergePolicy) Merge(existing, incoming []types.Record) []types.Record {
	return incoming
}

func openTestBadgerRepository(t *testing.T, opts ...BadgerRepositoryOption) *BadgerRepository {
	t.Helper()

	core, err := OpenBadger(t.TempDir())
	if err != nil {
		t.Fatalf("OpenBadger returned error: %v", err)
	}
	repository, err := NewBadgerRepository(core, opts...)
	if err != nil {
		_ = core.Close()
		t.Fatalf("NewBadgerRepository returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})

	return repository
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
