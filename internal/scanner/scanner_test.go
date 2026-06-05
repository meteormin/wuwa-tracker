package scanner

import (
	"strings"
	"testing"
)

func TestFindURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		trackingURL string
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "Valid Client.log format",
			input:       `LogHttp: Display: HTTP URL: https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record?serverId=123&playerId=456&recordId=789`,
			trackingURL: "https://aki-gm-resources-oversea.aki-game.net",
			expectedURL: "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record?serverId=123&playerId=456&recordId=789",
			expectErr:   false,
		},
		{
			name:        "Valid debug.log format",
			input:       `"#url": "https://aki-gm-resources.aki-game.com/aki/gacha/index.html#/record?foo=bar"`,
			trackingURL: "https://aki-gm-resources.aki-game.com",
			expectedURL: "https://aki-gm-resources.aki-game.com/aki/gacha/index.html#/record?foo=bar",
			expectErr:   false,
		},
		{
			name:        "Multiple matches, returns last",
			input:       "https://aki-gm-resources.aki-game.net/aki/gacha/index.html#/record?id=1\nhttps://aki-gm-resources.aki-game.net/aki/gacha/index.html#/record?id=2",
			trackingURL: "https://aki-gm-resources.aki-game.net",
			expectedURL: "https://aki-gm-resources.aki-game.net/aki/gacha/index.html#/record?id=2",
			expectErr:   false,
		},
		{
			name:        "No match",
			input:       `Just some random log line`,
			trackingURL: "https://aki-gm-resources-oversea.aki-game.net",
			expectedURL: "",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			url, err := FindURL(r, tt.trackingURL)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil, url: %s", url)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if url != tt.expectedURL {
					t.Errorf("Expected URL %q, got %q", tt.expectedURL, url)
				}
			}
		})
	}
}
