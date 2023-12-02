package sql

import (
	"fmt"
	"testing"
	"time"
)

func TestSqlDate(t *testing.T) {
	locationNewYork, _ := time.LoadLocation("America/New_York")
	locationTokyo, _ := time.LoadLocation("Asia/Tokyo")

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Standard date (local)",
			input:    time.Date(2023, 3, 14, 0, 0, 0, 0, time.Local),
			expected: "'2023-03-14'",
		},
		{
			name:     "Leap year date UTC",
			input:    time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
			expected: "'2020-02-29T00:00:00Z'",
		},
		{
			name:     "Date with time zone New York",
			input:    time.Date(2023, 3, 14, 0, 0, 0, 0, locationNewYork),
			expected: "'2023-03-14T00:00:00-04:00'",
		},
		{
			name:     "Date with time zone Tokyo",
			input:    time.Date(2023, 3, 14, 0, 0, 0, 0, locationTokyo),
			expected: "'2023-03-14T00:00:00+09:00'",
		},
		{
			name:  "Unix epoch",
			input: time.Unix(0, 0),
			expected: func() string {
				// compute unix time 1970-01-01 in time.Local
				t := time.Unix(0, 0)
				return "'" + t.Format("2006-01-02") + "'"
			}(),
		},
		{
			name:     "Date before Unix epoch",
			input:    time.Date(1969, 12, 31, 0, 0, 0, 0, time.UTC),
			expected: "'1969-12-31T00:00:00Z'",
		},
		// Add more test cases if needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sqlDate(tt.input)
			if err != nil {
				t.Errorf("sqlDate(%v) returned an error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("sqlDate(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSqlDateTime(t *testing.T) {
	locationNewYork, _ := time.LoadLocation("America/New_York")
	locationTokyo, _ := time.LoadLocation("Asia/Tokyo")

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Standard datetime (local)",
			input:    time.Date(2023, 3, 14, 21, 36, 45, 0, time.Local),
			expected: "'2023-03-14T21:36:45'",
		},
		{
			name:     "Standard datetime UTC",
			input:    time.Date(2023, 3, 14, 21, 36, 45, 0, time.UTC),
			expected: "'2023-03-14T21:36:45Z'",
		},
		{
			name:     "Datetime with nanoseconds UTC",
			input:    time.Date(2023, 3, 14, 21, 36, 45, 123456789, time.UTC),
			expected: "'2023-03-14T21:36:45Z'",
		},
		{
			name:     "Datetime with time zone New York",
			input:    time.Date(2023, 3, 14, 21, 36, 45, 0, locationNewYork),
			expected: fmt.Sprintf("'%s'", time.Date(2023, 3, 14, 21, 36, 45, 0, locationNewYork).Format(time.RFC3339)),
		},
		{
			name:     "Datetime with time zone Tokyo",
			input:    time.Date(2023, 3, 14, 21, 36, 45, 0, locationTokyo),
			expected: fmt.Sprintf("'%s'", time.Date(2023, 3, 14, 21, 36, 45, 0, locationTokyo).Format(time.RFC3339)),
		},
		{
			name:     "Unix epoch with time",
			input:    time.Unix(0, 0).Add(8*time.Hour + 30*time.Minute),
			expected: "'1970-01-01T03:30:00'",
		},
		{
			name:     "Datetime before Unix epoch",
			input:    time.Date(1969, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: "'1969-12-31T23:59:59Z'",
		},
		// Add more test cases if needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sqlDateTime(tt.input)
			if err != nil {
				t.Errorf("sqlDateTime(%v) returned an error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("sqlDateTime(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
