package report

import "testing"

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{
			name:  "json",
			input: "json",
			want:  FormatJSON,
		},
		{
			name:  "csv with uppercase and spaces",
			input: " CSV ",
			want:  FormatCSV,
		},
		{
			name:  "html",
			input: "html",
			want:  FormatHTML,
		},
		{
			name:    "unsupported",
			input:   "pdf",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParseFormat returned nil error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFormat returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseFormat = %q, want %q", got, tt.want)
			}
		})
	}
}
