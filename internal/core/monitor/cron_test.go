package monitor

import (
	"testing"
	"time"
)

func TestParseCron_ValidExpressions(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"every minute", "* * * * *"},
		{"every 5 minutes", "*/5 * * * *"},
		{"specific minute", "30 * * * *"},
		{"specific time", "30 8 * * *"},
		{"specific day", "0 0 1 * *"},
		{"specific month", "0 0 1 6 *"},
		{"specific weekday", "0 0 * * 1"},
		{"range minutes", "1-30 * * * *"},
		{"list minutes", "0,15,30,45 * * * *"},
		{"complex", "*/10 9-17 * * 1-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCron(tt.expr)
			if err != nil {
				t.Fatalf("ParseCron(%q) returned error: %v", tt.expr, err)
			}
		})
	}
}

func TestParseCron_InvalidExpressions(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"empty", ""},
		{"too few fields", "* * *"},
		{"too many fields", "* * * * * *"},
		{"invalid minute", "60 * * * *"},
		{"invalid hour", "* 25 * * *"},
		{"invalid day", "* * 32 * *"},
		{"invalid month", "* * * 13 *"},
		{"invalid weekday", "* * * * 8"},
		{"invalid character", "* * * * a"},
		{"negative number", "-1 * * * *"},
		{"invalid range", "5-2 * * * *"},
		{"invalid step", "*/0 * * * *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCron(tt.expr)
			if err == nil {
				t.Fatalf("ParseCron(%q) expected error, got nil", tt.expr)
			}
		})
	}
}

func TestCronSchedule_Next(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		from     time.Time
		expected time.Time
	}{
		{
			name:     "every minute advances to next minute",
			expr:     "* * * * *",
			from:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 1, 1, 12, 1, 0, 0, time.UTC),
		},
		{
			name:     "every 5 minutes from 12:00",
			expr:     "*/5 * * * *",
			from:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 1, 1, 12, 5, 0, 0, time.UTC),
		},
		{
			name:     "every 5 minutes from 12:03",
			expr:     "*/5 * * * *",
			from:     time.Date(2025, 1, 1, 12, 3, 0, 0, time.UTC),
			expected: time.Date(2025, 1, 1, 12, 5, 0, 0, time.UTC),
		},
		{
			name:     "specific time rolls to next day",
			expr:     "30 8 * * *",
			from:     time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 1, 2, 8, 30, 0, 0, time.UTC),
		},
		{
			name:     "specific time same day",
			expr:     "30 8 * * *",
			from:     time.Date(2025, 1, 1, 7, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 1, 1, 8, 30, 0, 0, time.UTC),
		},
		{
			name:     "first day of month rolls to next month",
			expr:     "0 0 1 * *",
			from:     time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "specific weekday (Monday)",
			expr:     "0 9 * * 1",
			from:     time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC), // Monday 10:00
			expected: time.Date(2025, 1, 13, 9, 0, 0, 0, time.UTC), // next Monday 9:00
		},
		{
			name:     "list of minutes",
			expr:     "0,30 * * * *",
			from:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 1, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			name:     "range of hours",
			expr:     "0 9-17 * * *",
			from:     time.Date(2025, 1, 1, 18, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sched, err := ParseCron(tt.expr)
			if err != nil {
				t.Fatalf("ParseCron(%q) returned error: %v", tt.expr, err)
			}

			next := sched.Next(tt.from)
			if !next.Equal(tt.expected) {
				t.Errorf("Next(%v) = %v, want %v", tt.from, next, tt.expected)
			}
		})
	}
}

func TestCronSchedule_Next_AlwaysAfterFrom(t *testing.T) {
	sched, err := ParseCron("*/5 * * * *")
	if err != nil {
		t.Fatal(err)
	}

	from := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	next := sched.Next(from)

	if !next.After(from) {
		t.Errorf("Next(%v) = %v, expected time after from", from, next)
	}
}
