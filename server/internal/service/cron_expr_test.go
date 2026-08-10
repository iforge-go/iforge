package service

import (
	"testing"
	"time"
)

func TestParseCron(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
		check   func(*CronExpr) bool
	}{
		{
			name: "every minute",
			expr: "* * * * *",
			check: func(c *CronExpr) bool {
				return len(c.Minute) == 60 && len(c.Hour) == 24 &&
					len(c.DayOfMonth) == 31 && len(c.Month) == 12 && len(c.DayOfWeek) == 7
			},
		},
		{
			name: "every 15 minutes",
			expr: "*/15 * * * *",
			check: func(c *CronExpr) bool {
				return len(c.Minute) == 4 && c.Minute[0] == 0 && c.Minute[1] == 15 &&
					c.Minute[2] == 30 && c.Minute[3] == 45
			},
		},
		{
			name: "specific time",
			expr: "30 9 * * *",
			check: func(c *CronExpr) bool {
				return len(c.Minute) == 1 && c.Minute[0] == 30 &&
					len(c.Hour) == 1 && c.Hour[0] == 9
			},
		},
		{
			name: "range",
			expr: "0 9-17 * * *",
			check: func(c *CronExpr) bool {
				return len(c.Hour) == 9 && c.Hour[0] == 9 && c.Hour[8] == 17
			},
		},
		{
			name: "list",
			expr: "0,15,30,45 * * * *",
			check: func(c *CronExpr) bool {
				return len(c.Minute) == 4 && c.Minute[0] == 0 && c.Minute[1] == 15 &&
					c.Minute[2] == 30 && c.Minute[3] == 45
			},
		},
		{
			name: "range with step",
			expr: "0 9-17/2 * * *",
			check: func(c *CronExpr) bool {
				return len(c.Hour) == 5 && c.Hour[0] == 9 && c.Hour[1] == 11 &&
					c.Hour[2] == 13 && c.Hour[3] == 15 && c.Hour[4] == 17
			},
		},
		{
			name: "day of week",
			expr: "0 9 * * 1-5",
			check: func(c *CronExpr) bool {
				return len(c.DayOfWeek) == 5 && c.DayOfWeek[0] == 1 && c.DayOfWeek[4] == 5
			},
		},
		{
			name: "complex expression",
			expr: "0,30 9-17 * * 1-5",
			check: func(c *CronExpr) bool {
				return len(c.Minute) == 2 && len(c.Hour) == 9 && len(c.DayOfWeek) == 5
			},
		},
		{
			name:    "too few fields",
			expr:    "* * * *",
			wantErr: true,
		},
		{
			name:    "too many fields",
			expr:    "* * * * * *",
			wantErr: true,
		},
		{
			name:    "invalid minute",
			expr:    "60 * * * *",
			wantErr: true,
		},
		{
			name:    "invalid hour",
			expr:    "* 24 * * *",
			wantErr: true,
		},
		{
			name:    "invalid day",
			expr:    "* * 32 * *",
			wantErr: true,
		},
		{
			name:    "invalid month",
			expr:    "* * * 13 *",
			wantErr: true,
		},
		{
			name:    "invalid day of week",
			expr:    "* * * * 7",
			wantErr: true,
		},
		{
			name:    "invalid range",
			expr:    "* 17-9 * * *",
			wantErr: true,
		},
		{
			name:    "invalid step",
			expr:    "*/0 * * * *",
			wantErr: true,
		},
		{
			name:    "invalid value",
			expr:    "abc * * * *",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cron, err := ParseCron(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCron(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(cron) {
				t.Errorf("ParseCron(%q) check failed", tt.expr)
			}
		})
	}
}

func TestCronExpr_NextTime(t *testing.T) {
	// Base time: 2024-01-15 10:30:00 (Monday)
	base := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		expr     string
		expected time.Time
	}{
		{
			name:     "every minute",
			expr:     "* * * * *",
			expected: time.Date(2024, 1, 15, 10, 31, 0, 0, time.UTC),
		},
		{
			name:     "every hour at minute 0",
			expr:     "0 * * * *",
			expected: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
		},
		{
			name:     "specific time tomorrow",
			expr:     "0 9 * * *",
			expected: time.Date(2024, 1, 16, 9, 0, 0, 0, time.UTC),
		},
		{
			name:     "every 15 minutes",
			expr:     "*/15 * * * *",
			expected: time.Date(2024, 1, 15, 10, 45, 0, 0, time.UTC),
		},
		{
			name:     "weekday at 9am",
			expr:     "0 9 * * 1-5",
			expected: time.Date(2024, 1, 16, 9, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cron, err := ParseCron(tt.expr)
			if err != nil {
				t.Fatalf("ParseCron(%q) error = %v", tt.expr, err)
			}

			next := cron.NextTime(base)
			if !next.Equal(tt.expected) {
				t.Errorf("NextTime() = %v, want %v", next, tt.expected)
			}
		})
	}
}

func TestCronExpr_match(t *testing.T) {
	tests := []struct {
		name  string
		expr  string
		time  time.Time
		match bool
	}{
		{
			name:  "exact match",
			expr:  "30 10 15 1 *",
			time:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			match: true,
		},
		{
			name:  "minute mismatch",
			expr:  "31 10 15 1 *",
			time:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			match: false,
		},
		{
			name:  "hour mismatch",
			expr:  "30 11 15 1 *",
			time:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			match: false,
		},
		{
			name:  "day mismatch",
			expr:  "30 10 16 1 *",
			time:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			match: false,
		},
		{
			name:  "month mismatch",
			expr:  "30 10 15 2 *",
			time:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			match: false,
		},
		{
			name:  "day of week match (Monday=1)",
			expr:  "30 10 * * 1",
			time:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), // Monday
			match: true,
		},
		{
			name:  "day of week mismatch",
			expr:  "30 10 * * 5",
			time:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), // Monday
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cron, err := ParseCron(tt.expr)
			if err != nil {
				t.Fatalf("ParseCron(%q) error = %v", tt.expr, err)
			}

			result := cron.match(tt.time)
			if result != tt.match {
				t.Errorf("match(%v) = %v, want %v", tt.time, result, tt.match)
			}
		})
	}
}

func BenchmarkParseCron(b *testing.B) {
	exprs := []string{
		"* * * * *",
		"*/15 * * * *",
		"0 9-17 * * 1-5",
		"0,30 9-17 * * 1-5",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, expr := range exprs {
			_, _ = ParseCron(expr)
		}
	}
}
