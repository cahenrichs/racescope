package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseOptions(t *testing.T) {
	t.Parallel()

	got, err := parseOptions([]string{"--season", "2024", "--meeting", "1242"}, &bytes.Buffer{}, 2026)
	if err != nil {
		t.Fatalf("parseOptions() error = %v", err)
	}
	if got.season != 2024 || got.meetingKey != 1242 {
		t.Fatalf("parseOptions() = %+v, want season 2024 and meeting 1242", got)
	}
}

func TestParseOptionsRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing season", args: []string{"--meeting", "1242"}, want: "season must be between 2023 and 2026"},
		{name: "unsupported season", args: []string{"--season", "2022", "--meeting", "1242"}, want: "season must be between 2023 and 2026"},
		{name: "future season", args: []string{"--season", "2027", "--meeting", "1242"}, want: "season must be between 2023 and 2026"},
		{name: "missing meeting", args: []string{"--season", "2024"}, want: "meeting must be a positive OpenF1 meeting key"},
		{name: "invalid meeting", args: []string{"--season", "2024", "--meeting", "-1"}, want: "meeting must be a positive OpenF1 meeting key"},
		{name: "positional argument", args: []string{"--season", "2024", "--meeting", "1242", "extra"}, want: "unexpected positional arguments"},
		{name: "unknown option", args: []string{"--season", "2024", "--meeting", "1242", "--all"}, want: "flag provided but not defined"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseOptions(test.args, &bytes.Buffer{}, 2026)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("parseOptions() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}
