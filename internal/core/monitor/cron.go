package monitor

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronSchedule represents a parsed cron expression with minute, hour,
// day-of-month, month, and day-of-week fields.
type CronSchedule struct {
	Minutes    []bool // 0-59
	Hours      []bool // 0-23
	DaysOfMonth []bool // 1-31 (index 0 unused)
	Months     []bool // 1-12 (index 0 unused)
	DaysOfWeek []bool // 0-6 (Sunday=0)
}

// ParseCron parses a standard 5-field cron expression.
// Fields: minute hour day-of-month month day-of-week
// Supports: *, */N, N, N-M, N,M,O
func ParseCron(expr string) (CronSchedule, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return CronSchedule{}, fmt.Errorf("empty cron expression")
	}

	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return CronSchedule{}, fmt.Errorf("cron expression must have 5 fields, got %d", len(fields))
	}

	minutes, err := parseField(fields[0], 0, 59)
	if err != nil {
		return CronSchedule{}, fmt.Errorf("invalid minute field: %w", err)
	}

	hours, err := parseField(fields[1], 0, 23)
	if err != nil {
		return CronSchedule{}, fmt.Errorf("invalid hour field: %w", err)
	}

	days, err := parseField(fields[2], 1, 31)
	if err != nil {
		return CronSchedule{}, fmt.Errorf("invalid day-of-month field: %w", err)
	}

	months, err := parseField(fields[3], 1, 12)
	if err != nil {
		return CronSchedule{}, fmt.Errorf("invalid month field: %w", err)
	}

	weekdays, err := parseField(fields[4], 0, 6)
	if err != nil {
		return CronSchedule{}, fmt.Errorf("invalid day-of-week field: %w", err)
	}

	sched := CronSchedule{
		Minutes:     make([]bool, 60),
		Hours:       make([]bool, 24),
		DaysOfMonth: make([]bool, 32), // index 0 unused
		Months:      make([]bool, 13), // index 0 unused
		DaysOfWeek:  make([]bool, 7),
	}

	for _, v := range minutes {
		sched.Minutes[v] = true
	}
	for _, v := range hours {
		sched.Hours[v] = true
	}
	for _, v := range days {
		sched.DaysOfMonth[v] = true
	}
	for _, v := range months {
		sched.Months[v] = true
	}
	for _, v := range weekdays {
		sched.DaysOfWeek[v] = true
	}

	return sched, nil
}

// Next returns the next time after from that matches the cron schedule.
func (c CronSchedule) Next(from time.Time) time.Time {
	// Start from the next minute boundary.
	t := from.Truncate(time.Minute).Add(time.Minute)

	// Search up to ~4 years to find a match (handles leap years, etc.)
	limit := t.Add(4 * 365 * 24 * time.Hour)

	for t.Before(limit) {
		// Check month
		if !c.Months[t.Month()] {
			// Advance to next month
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}

		// Check day of month
		if !c.DaysOfMonth[t.Day()] {
			// Advance to next day
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}

		// Check day of week
		if !c.DaysOfWeek[t.Weekday()] {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}

		// Check hour
		if !c.Hours[t.Hour()] {
			// Advance to next hour
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			continue
		}

		// Check minute
		if !c.Minutes[t.Minute()] {
			t = t.Add(time.Minute)
			continue
		}

		return t
	}

	// Should not happen with valid expressions
	return time.Time{}
}

// parseField parses a single cron field and returns the set of matching values.
func parseField(field string, min, max int) ([]int, error) {
	var result []int

	parts := strings.Split(field, ",")
	for _, part := range parts {
		vals, err := parsePart(part, min, max)
		if err != nil {
			return nil, err
		}
		result = append(result, vals...)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("field produced no values")
	}

	return result, nil
}

// parsePart parses a single part of a cron field (e.g. "*/5", "1-30", "5").
func parsePart(part string, min, max int) ([]int, error) {
	// Handle step: */N or N-M/N
	var step int
	if idx := strings.Index(part, "/"); idx >= 0 {
		stepStr := part[idx+1:]
		var err error
		step, err = strconv.Atoi(stepStr)
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step %q", stepStr)
		}
		part = part[:idx]
	}

	var rangeMin, rangeMax int

	if part == "*" {
		rangeMin = min
		rangeMax = max
	} else if idx := strings.Index(part, "-"); idx >= 0 {
		var err error
		rangeMin, err = strconv.Atoi(part[:idx])
		if err != nil {
			return nil, fmt.Errorf("invalid range start %q", part[:idx])
		}
		rangeMax, err = strconv.Atoi(part[idx+1:])
		if err != nil {
			return nil, fmt.Errorf("invalid range end %q", part[idx+1:])
		}
		if rangeMin > rangeMax {
			return nil, fmt.Errorf("invalid range %d-%d", rangeMin, rangeMax)
		}
	} else {
		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q", part)
		}
		if val < min || val > max {
			return nil, fmt.Errorf("value %d out of range [%d, %d]", val, min, max)
		}
		if step == 0 {
			return []int{val}, nil
		}
		rangeMin = val
		rangeMax = max
	}

	if rangeMin < min || rangeMax > max {
		return nil, fmt.Errorf("range %d-%d out of bounds [%d, %d]", rangeMin, rangeMax, min, max)
	}

	if step == 0 {
		step = 1
	}

	var result []int
	for i := rangeMin; i <= rangeMax; i += step {
		result = append(result, i)
	}

	return result, nil
}
