package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronExpr represents a parsed cron expression (5 fields: minute hour day-of-month month day-of-week)
//
// Supports:
//   - *           any value
//   - */N         step (e.g., */15 means every 15 minutes)
//   - N           single value
//   - N-M         range
//   - N,M,K       list
//
// Advanced features not supported (L, W, #, ?, @yearly, etc.) to keep implementation simple.
type CronExpr struct {
	Minute     []int // 0-59
	Hour       []int // 0-23
	DayOfMonth []int // 1-31
	Month      []int // 1-12
	DayOfWeek  []int // 0-6 (0=Sunday, consistent with Go time.Weekday())
}

// ParseCron parses a 5-field cron expression
func ParseCron(expr string) (*CronExpr, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(fields))
	}
	minute, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute field: %w", err)
	}
	hour, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour field: %w", err)
	}
	dom, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day-of-month field: %w", err)
	}
	month, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month field: %w", err)
	}
	dow, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("day-of-week field: %w", err)
	}
	return &CronExpr{Minute: minute, Hour: hour, DayOfMonth: dom, Month: month, DayOfWeek: dow}, nil
}

// parseCronField parses a single cron field
func parseCronField(field string, min, max int) ([]int, error) {
	if field == "*" {
		result := make([]int, 0, max-min+1)
		for i := min; i <= max; i++ {
			result = append(result, i)
		}
		return result, nil
	}
	// */N
	if strings.HasPrefix(field, "*/") {
		stepStr := field[2:]
		step, err := strconv.Atoi(stepStr)
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step %q", stepStr)
		}
		var result []int
		for i := min; i <= max; i += step {
			result = append(result, i)
		}
		return result, nil
	}
	// Handle comma-separated (N,M,K or mixed)
	var result []int
	for _, part := range strings.Split(field, ",") {
		// N-M range (can have /N step, e.g., 1-10/2)
		step := 1
		rangePart := part
		if idx := strings.Index(part, "/"); idx > 0 {
			stepStr := part[idx+1:]
			var err error
			step, err = strconv.Atoi(stepStr)
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step %q in %q", stepStr, part)
			}
			rangePart = part[:idx]
		}
		if strings.Contains(rangePart, "-") {
			rangeParts := strings.SplitN(rangePart, "-", 2)
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range %q", rangePart)
			}
			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q", rangeParts[0])
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q", rangeParts[1])
			}
			if start < min || end > max || start > end {
				return nil, fmt.Errorf("range %d-%d out of bounds [%d,%d]", start, end, min, max)
			}
			for i := start; i <= end; i += step {
				result = append(result, i)
			}
		} else {
			v, err := strconv.Atoi(rangePart)
			if err != nil {
				return nil, fmt.Errorf("invalid value %q", rangePart)
			}
			if v < min || v > max {
				return nil, fmt.Errorf("value %d out of bounds [%d,%d]", v, min, max)
			}
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("empty field %q", field)
	}
	return result, nil
}

// NextTime calculates the first time after 'from' that matches the cron expression (minute-level precision)
//
// Iterates up to 366 days; returns zero value if no match found (usually indicates expression error).
// Note: day-of-month and day-of-week use AND (stricter, common behavior in CI/CD tools);
// standard cron uses OR (uses the other when either is *), but CI/CD users typically expect AND.
func (e *CronExpr) NextTime(from time.Time) time.Time {
	// Start from from + 1 minute, check minute by minute
	t := from.Add(1 * time.Minute).Truncate(time.Minute)
	// Use local timezone (consistent with from)
	loc := from.Location()
	maxIterations := 366 * 24 * 60 // up to 1 year
	for i := 0; i < maxIterations; i++ {
		if e.match(t.In(loc)) {
			return t.In(loc)
		}
		t = t.Add(1 * time.Minute)
	}
	return time.Time{}
}

// match checks if a time matches the cron expression
func (e *CronExpr) match(t time.Time) bool {
	if !intSliceContains(e.Minute, t.Minute()) {
		return false
	}
	if !intSliceContains(e.Hour, t.Hour()) {
		return false
	}
	if !intSliceContains(e.Month, int(t.Month())) {
		return false
	}
	if !intSliceContains(e.DayOfMonth, t.Day()) {
		return false
	}
	// Go: Sunday=0, Monday=1, ..., Saturday=6; consistent with cron
	dow := int(t.Weekday())
	if !intSliceContains(e.DayOfWeek, dow) {
		return false
	}
	return true
}

func intSliceContains(slice []int, v int) bool {
	for _, x := range slice {
		if x == v {
			return true
		}
	}
	return false
}
