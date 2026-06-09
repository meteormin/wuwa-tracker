package scanner

import (
	"strings"
	"testing"
)

const benchmarkResourceURL = "https://aki-gm-resources-oversea.aki-game.net"

func BenchmarkFindURL(b *testing.B) {
	url := benchmarkResourceURL + "/aki/gacha/index.html#/record?serverId=server-1&playerId=player-1&recordId=record-1"

	cases := []struct {
		name string
		log  string
	}{
		{
			name: "single-line",
			log:  "LogHttp: Display: HTTP URL: " + url,
		},
		{
			name: "large-log-last-line",
			log:  benchmarkLog(10000, url),
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tc.log)))
			b.ResetTimer()

			for range b.N {
				got, err := FindURL(strings.NewReader(tc.log), benchmarkResourceURL)
				if err != nil {
					b.Fatalf("FindURL returned error: %v", err)
				}
				if got != url {
					b.Fatalf("FindURL = %q, want %q", got, url)
				}
			}
		})
	}
}

func BenchmarkURLRegexZeroAllocCandidate(b *testing.B) {
	url := benchmarkResourceURL + "/aki/gacha/index.html#/record?serverId=server-1&playerId=player-1&recordId=record-1"
	line := "LogHttp: Display: HTTP URL: " + url
	urlRegex, err := newURLRegex(benchmarkResourceURL)
	if err != nil {
		b.Fatalf("newURLRegex returned error: %v", err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(line)))
	b.ResetTimer()

	for range b.N {
		got := urlRegex.FindString(line)
		if got != url {
			b.Fatalf("FindString = %q, want %q", got, url)
		}
	}
}

func benchmarkLog(lines int, url string) string {
	var builder strings.Builder
	builder.Grow(lines*64 + len(url))
	for i := 0; i < lines-1; i++ {
		builder.WriteString("LogTemp: Display: unrelated client log line\n")
	}
	builder.WriteString("LogHttp: Display: HTTP URL: ")
	builder.WriteString(url)
	return builder.String()
}
