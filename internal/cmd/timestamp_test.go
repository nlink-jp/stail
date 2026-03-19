package cmd

import (
	"testing"
)

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "Slack ts integer seconds",
			input: "1700000000.000000",
			want:  "1700000000.000000",
		},
		{
			name:  "Slack ts with sub-second precision",
			input: "1700000000.123456",
			want:  "1700000000.123456",
		},
		{
			name:  "Slack ts without fractional part",
			input: "1700000000",
			want:  "1700000000",
		},
		{
			name:  "RFC3339 UTC",
			input: "2023-11-15T01:46:40Z",
			want:  "1700012800.000000",
		},
		{
			name:  "RFC3339 with offset (same instant as UTC case)",
			input: "2023-11-15T10:46:40+09:00",
			want:  "1700012800.000000",
		},
		{
			name:    "invalid format",
			input:   "not-a-timestamp",
			wantErr: true,
		},
		{
			name:    "small float (not a Unix ts)",
			input:   "123.456",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTimestamp(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseTimestamp(%q): expected error, got %q", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseTimestamp(%q): unexpected error: %v", tc.input, err)
				return
			}
			if got != tc.want {
				t.Errorf("parseTimestamp(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
