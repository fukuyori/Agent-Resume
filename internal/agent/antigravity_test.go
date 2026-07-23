package agent

import (
	"testing"
)

func TestParseAntigravityTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "RFC3339 standard",
			input:   "2026-07-22T11:40:58Z",
			wantErr: false,
		},
		{
			name:    "RFC3339 with timezone offset",
			input:   "2026-07-22T20:41:30+09:00",
			wantErr: false,
		},
		{
			name:    "RFC3339Nano with UTC subseconds",
			input:   "2026-07-22T11:40:58.1088749Z",
			wantErr: false,
		},
		{
			name:    "RFC3339Nano with timezone offset and subseconds",
			input:   "2026-07-22T20:41:30.6875769+09:00",
			wantErr: false,
		},
		{
			name:    "Invalid time string",
			input:   "invalid-time",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAntigravityTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAntigravityTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.IsZero() {
				t.Errorf("parseAntigravityTime() got zero time for valid input %s", tt.input)
			}
		})
	}
}
