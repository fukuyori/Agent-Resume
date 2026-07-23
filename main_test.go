package main

import (
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantLimit   int
		wantVersion bool
		wantHelp    bool
		wantErr     bool
	}{
		{
			name:        "default limit",
			args:        []string{},
			wantLimit:   10,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "positional limit",
			args:        []string{"20"},
			wantLimit:   20,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "flag -n limit",
			args:        []string{"-n", "15"},
			wantLimit:   15,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "flag -n= limit",
			args:        []string{"-n=25"},
			wantLimit:   25,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "flag --limit",
			args:        []string{"--limit", "30"},
			wantLimit:   30,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "flag --limit=",
			args:        []string{"--limit=35"},
			wantLimit:   35,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "version flag -v",
			args:        []string{"-v"},
			wantLimit:   10,
			wantVersion: true,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "version flag --version",
			args:        []string{"--version"},
			wantLimit:   10,
			wantVersion: true,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "help flag -h",
			args:        []string{"-h"},
			wantLimit:   10,
			wantVersion: false,
			wantHelp:    true,
			wantErr:     false,
		},
		{
			name:        "help flag --help",
			args:        []string{"--help"},
			wantLimit:   10,
			wantVersion: false,
			wantHelp:    true,
			wantErr:     false,
		},
		{
			name:    "invalid negative limit",
			args:    []string{"-5"},
			wantErr: true,
		},
		{
			name:    "invalid non-integer positional arg",
			args:    []string{"foo"},
			wantErr: true,
		},
		{
			name:    "missing value for -n",
			args:    []string{"-n"},
			wantErr: true,
		},
		{
			name:    "invalid value for -n",
			args:    []string{"-n", "abc"},
			wantErr: true,
		},
		{
			name:    "zero value for -n",
			args:    []string{"-n", "0"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotVersion, gotHelp, err := parseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if gotLimit != tt.wantLimit {
				t.Errorf("parseArgs() gotLimit = %v, want %v", gotLimit, tt.wantLimit)
			}
			if gotVersion != tt.wantVersion {
				t.Errorf("parseArgs() gotVersion = %v, want %v", gotVersion, tt.wantVersion)
			}
			if gotHelp != tt.wantHelp {
				t.Errorf("parseArgs() gotHelp = %v, want %v", gotHelp, tt.wantHelp)
			}
		})
	}
}
