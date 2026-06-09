package tracker

import (
	"fmt"
	"testing"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

func BenchmarkStatsCalculatorCalc(b *testing.B) {
	calc := NewStatsCalculator([]int{1101, 1102, 1103, 1104, 1105}, types.CostPolicy{
		AstritePerPull: 160,
	})
	gachaType := types.GachaType{
		ID:               1,
		Name:             "Character Event",
		HasOffBannerDrop: true,
		BaseRate:         0.8,
		ExpectedPulls:    80,
	}

	for _, count := range []int{100, 1000, 10000} {
		records := benchmarkStatsRecords(count)
		b.Run(fmt.Sprintf("records-%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				stats := calc.Calc(records, gachaType)
				if stats.TotalPulls != count {
					b.Fatalf("TotalPulls = %d, want %d", stats.TotalPulls, count)
				}
			}
		})
	}
}

func BenchmarkStatsReverseZeroAlloc(b *testing.B) {
	b.Run("records", func(b *testing.B) {
		records := benchmarkStatsRecords(1000)
		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			reverseRecords(records)
		}
	})

	b.Run("five-stars", func(b *testing.B) {
		fiveStars := benchmarkFiveStarRecords(1000)
		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			reverseFiveStars(fiveStars)
		}
	})
}

func benchmarkStatsRecords(count int) []types.Record {
	records := make([]types.Record, count)
	for i := range count {
		qualityLevel := 3
		resourceID := 21010000 + i%20
		name := "Common"

		switch {
		case i%80 == 0:
			qualityLevel = 5
			if i%160 == 0 {
				resourceID = 1101
				name = "Standard"
			} else {
				resourceID = 1500 + i%10
				name = "Limited"
			}
		case i%10 == 0:
			qualityLevel = 4
			resourceID = 1400 + i%10
			name = "FourStar"
		}

		records[i] = types.Record{
			CardPoolType: "character",
			ResourceID:   resourceID,
			QualityLevel: qualityLevel,
			ResourceType: "Weapon",
			Name:         name,
			Count:        1,
			Time:         fmt.Sprintf("2026-05-%02d 12:%02d:%02d", 28-i%28, i%60, i%60),
		}
	}
	return records
}

func benchmarkFiveStarRecords(count int) []types.FiveStarRecord {
	records := make([]types.FiveStarRecord, count)
	for i := range count {
		records[i] = types.FiveStarRecord{
			Name:     "FiveStar",
			Time:     fmt.Sprintf("2026-05-%02d 12:%02d:%02d", 28-i%28, i%60, i%60),
			Pity:     1 + i%80,
			IsPickUp: i%2 == 0,
		}
	}
	return records
}
