package db

import (
	"testing"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

func BenchmarkMergeRecordsZeroAllocCandidates(b *testing.B) {
	existing := benchmarkDBRecords(500, 0)
	incoming := benchmarkDBRecords(500, 500)

	cases := []struct {
		name     string
		existing []types.Record
		incoming []types.Record
	}{
		{
			name:     "empty-existing",
			existing: nil,
			incoming: incoming,
		},
		{
			name:     "empty-incoming",
			existing: existing,
			incoming: nil,
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				merged := MergeRecords(tc.existing, tc.incoming)
				if len(merged) == 0 {
					b.Fatal("MergeRecords returned empty records")
				}
			}
		})
	}
}
