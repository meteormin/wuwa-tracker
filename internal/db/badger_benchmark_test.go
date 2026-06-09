package db

import (
	"fmt"
	"testing"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

func BenchmarkBadgerRepositorySaveGachaRecords(b *testing.B) {
	records := benchmarkDBRecords(500, 0)

	b.Run("new-player-500", func(b *testing.B) {
		repository := openBenchmarkBadgerRepository(b)
		b.ReportAllocs()
		b.ResetTimer()

		for i := range b.N {
			playerID := fmt.Sprintf("player-%d", i)
			if err := repository.SaveGachaRecords(playerID, "character", records); err != nil {
				b.Fatalf("SaveGachaRecords returned error: %v", err)
			}
		}
	})

	b.Run("merge-overlap-500", func(b *testing.B) {
		repository := openBenchmarkBadgerRepository(b)
		existing := benchmarkDBRecords(500, 100)
		incoming := append(benchmarkDBRecords(100, 0), existing[:400]...)
		b.ReportAllocs()
		b.ResetTimer()

		for i := range b.N {
			playerID := fmt.Sprintf("merge-player-%d", i)

			b.StopTimer()
			if err := repository.SaveGachaRecords(playerID, "character", existing); err != nil {
				b.Fatalf("SaveGachaRecords setup returned error: %v", err)
			}
			b.StartTimer()

			if err := repository.SaveGachaRecords(playerID, "character", incoming); err != nil {
				b.Fatalf("SaveGachaRecords returned error: %v", err)
			}
		}
	})
}

func openBenchmarkBadgerRepository(b *testing.B) *BadgerRepository {
	b.Helper()

	core, err := OpenBadger(b.TempDir())
	if err != nil {
		b.Fatalf("OpenBadger returned error: %v", err)
	}
	repository, err := NewBadgerRepository(core)
	if err != nil {
		_ = core.Close()
		b.Fatalf("NewBadgerRepository returned error: %v", err)
	}
	b.Cleanup(func() {
		if err := repository.Close(); err != nil {
			b.Fatalf("Close returned error: %v", err)
		}
	})
	return repository
}

func benchmarkDBRecords(count, offset int) []types.Record {
	records := make([]types.Record, count)
	for i := range count {
		index := i + offset
		records[i] = types.Record{
			CardPoolType: "character",
			ResourceID:   100000 + index%100,
			QualityLevel: benchmarkQualityLevel(index),
			ResourceType: "Weapon",
			Name:         fmt.Sprintf("Item-%d", index%100),
			Count:        1,
			Time:         fmt.Sprintf("2026-05-%02d 12:%02d:%02d", 28-index%28, index%60, index%60),
		}
	}
	return records
}

func benchmarkQualityLevel(index int) int {
	switch {
	case index%80 == 0:
		return 5
	case index%10 == 0:
		return 4
	default:
		return 3
	}
}
