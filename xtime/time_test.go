package xtime

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // modifies global locale state
func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("successful init", func(t *testing.T) {
		err := Init(Config{Language: "zh-CN", Timezone: "UTC"})
		require.NoError(t, err)
		// Test that GetLocation returns a valid location
		loc := GetLocation()
		assert.NotNil(t, loc)
		assert.Equal(t, "UTC", loc.String())
	})

	t.Run("invalid timezone", func(t *testing.T) {
		t.Parallel()

		err := Init(Config{Language: "en", Timezone: "Invalid/Timezone"})
		assert.Error(t, err)
	})

	t.Run("default values", func(t *testing.T) {
		t.Parallel()

		err := Init(Config{})
		require.NoError(t, err)
		// Test that GetLocation returns a valid location
		loc := GetLocation()
		assert.NotNil(t, loc)
		assert.Equal(t, "UTC", loc.String())
	})
}

func TestTime(t *testing.T) {
	t.Parallel()

	err := InitSimple("en")
	require.NoError(t, err)

	tests := []struct {
		name      string
		timestamp int64
		want      time.Time
	}{
		{
			name:      "zero timestamp",
			timestamp: 0,
			want:      time.Time{},
		},
		{
			name:      "valid timestamp",
			timestamp: 1577836800, // 2020-01-01 00:00:00 UTC
			want:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Time(tt.timestamp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormat(t *testing.T) {
	t.Parallel()

	t.Run("with init", func(t *testing.T) {
		t.Parallel()

		err := Init(Config{Language: "en", Timezone: "UTC"})
		require.NoError(t, err)

		testTime := time.Date(2020, 3, 15, 10, 30, 0, 0, time.UTC)
		result := Format(testTime)
		// The format should include the actual date/time, not the template
		assert.Contains(t, result, "2020-03-15")
		assert.Contains(t, result, "10:30:00")
	})

	t.Run("without init", func(t *testing.T) {
		t.Parallel()

		// Reset global variables to test fallback
		location.Store(time.UTC)
		testTime := time.Date(2020, 3, 15, 10, 30, 0, 0, time.UTC)
		result := Format(testTime)
		assert.Contains(t, result, "2020-03-15")
		assert.Contains(t, result, "10:30:00")
	})
}

func TestNextDailyTime(t *testing.T) {
	t.Parallel()

	err := InitSimple("en")
	require.NoError(t, err)

	tests := []struct {
		name     string
		now      time.Time
		delay    time.Duration
		expected time.Time
	}{
		{
			name:     "normal case - next day",
			now:      time.Date(2020, 3, 15, 10, 0, 0, 0, time.UTC),
			delay:    2 * time.Hour,
			expected: time.Date(2020, 3, 16, 2, 0, 0, 0, time.UTC),
		},
		{
			name:     "delay crosses midnight",
			now:      time.Date(2020, 3, 15, 23, 0, 0, 0, time.UTC),
			delay:    3 * time.Hour,
			expected: time.Date(2020, 3, 16, 3, 0, 0, 0, time.UTC),
		},
		{
			name:     "zero delay",
			now:      time.Date(2020, 3, 15, 10, 0, 0, 0, time.UTC),
			delay:    0,
			expected: time.Date(2020, 3, 16, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next := NextDailyTime(tt.now, tt.delay)
			assert.Equal(t, tt.expected, next)
		})
	}
}

func TestNextWeeklyTime(t *testing.T) {
	t.Parallel()

	err := InitSimple("en")
	require.NoError(t, err)

	tests := []struct {
		name     string
		now      time.Time
		delay    time.Duration
		expected time.Time
	}{
		{
			name:     "from Sunday",
			now:      time.Date(2020, 3, 15, 10, 0, 0, 0, time.UTC), // Sunday
			delay:    3 * time.Hour,
			expected: time.Date(2020, 3, 16, 3, 0, 0, 0, time.UTC), // Next Monday + 3h (same week)
		},
		{
			name:     "from Monday",
			now:      time.Date(2020, 3, 16, 10, 0, 0, 0, time.UTC), // Monday
			delay:    2 * time.Hour,
			expected: time.Date(2020, 3, 23, 2, 0, 0, 0, time.UTC), // Next Monday + 2h (next week)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next := NextWeeklyTime(tt.now, tt.delay)
			assert.Equal(t, tt.expected, next)
		})
	}
}

func TestNextMonthlyTime(t *testing.T) {
	t.Parallel()

	err := InitSimple("en")
	require.NoError(t, err)

	tests := []struct {
		name     string
		now      time.Time
		delay    time.Duration
		expected time.Time
	}{
		{
			name:     "normal case - next month",
			now:      time.Date(2020, 3, 15, 10, 0, 0, 0, time.UTC),
			delay:    3 * time.Hour,
			expected: time.Date(2020, 4, 1, 3, 0, 0, 0, time.UTC),
		},
		{
			name:     "year end",
			now:      time.Date(2020, 12, 31, 22, 0, 0, 0, time.UTC),
			delay:    4 * time.Hour,
			expected: time.Date(2021, 1, 1, 4, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next := NextMonthlyTime(tt.now, tt.delay)
			assert.Equal(t, tt.expected, next)
		})
	}
}

func TestStartOfDay(t *testing.T) {
	t.Parallel()

	err := InitSimple("en")
	require.NoError(t, err)

	input := time.Date(2024, 3, 15, 10, 30, 45, 123456789, time.UTC)
	expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	result := StartOfDay(input)
	assert.Equal(t, expected, result)
}

func TestStartOfWeek(t *testing.T) {
	t.Parallel()

	err := InitSimple("en")
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "Sunday",
			input:    time.Date(2020, 3, 15, 10, 30, 45, 0, time.UTC), // Sunday
			expected: time.Date(2020, 3, 9, 0, 0, 0, 0, time.UTC),     // Previous Monday
		},
		{
			name:     "Monday",
			input:    time.Date(2020, 3, 16, 10, 30, 45, 0, time.UTC), // Monday
			expected: time.Date(2020, 3, 16, 0, 0, 0, 0, time.UTC),    // Same Monday
		},
		{
			name:     "Saturday",
			input:    time.Date(2020, 3, 21, 10, 30, 45, 0, time.UTC), // Saturday
			expected: time.Date(2020, 3, 16, 0, 0, 0, 0, time.UTC),    // Monday of same week
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := StartOfWeek(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStartOfMonth(t *testing.T) {
	t.Parallel()

	t.Run("with carbon", func(t *testing.T) {
		t.Parallel()

		err := InitSimple("en")
		require.NoError(t, err)

		input := time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC)
		expected := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

		result := StartOfMonth(input)
		assert.Equal(t, expected, result)
	})

	t.Run("fallback without carbon", func(t *testing.T) {
		t.Parallel()

		// Reset global variable to test fallback
		location.Store(time.UTC)

		input := time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC)
		expected := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

		result := StartOfMonth(input)
		assert.Equal(t, expected, result)
	})
}

func TestInTimezone(t *testing.T) {
	t.Parallel()

	t.Run("valid timezone", func(t *testing.T) {
		t.Parallel()

		input := time.Date(2020, 3, 15, 10, 0, 0, 0, time.UTC)
		result, err := InTimezone(input, "Asia/Shanghai")
		require.NoError(t, err)

		// Should be 8 hours ahead
		expected := time.Date(2020, 3, 15, 18, 0, 0, 0, time.FixedZone("CST", 8*3600))
		assert.Equal(t, expected.Hour(), result.Hour())
		assert.True(t, result.Equal(expected))
	})

	t.Run("invalid timezone", func(t *testing.T) {
		t.Parallel()

		input := time.Date(2020, 3, 15, 10, 0, 0, 0, time.UTC)
		_, err := InTimezone(input, "Invalid/Timezone")
		assert.Error(t, err)
	})
}

// modifies global locale state
//
//nolint:paralleltest
func TestGetLocation(t *testing.T) {
	t.Run("with initialized location", func(t *testing.T) {
		err := Init(Config{Language: "en", Timezone: "Asia/Shanghai"})
		require.NoError(t, err)

		loc := GetLocation()
		assert.Equal(t, "Asia/Shanghai", loc.String())
	})

	t.Run("without initialized location", func(t *testing.T) {
		// Reset global variable
		location.Store(time.UTC)

		loc := GetLocation()
		assert.Equal(t, time.UTC, loc)
	})
}

// TestLeapYearAndMonthDays tests leap year and different month lengths
func TestLeapYearAndMonthDays(t *testing.T) {
	t.Parallel()

	err := InitSimple("en")
	require.NoError(t, err)

	// Test leap year February (29 days)
	t.Run("leap year February", func(t *testing.T) {
		t.Parallel()

		// 2020 is a leap year
		leapFeb := time.Date(2020, 2, 15, 10, 30, 0, 0, time.UTC)

		// Test StartOfMonth
		startOfMonth := StartOfMonth(leapFeb)
		expected := time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, startOfMonth)

		// Test NextMonthlyTime from leap February
		nextMonth := NextMonthlyTime(leapFeb, 2*time.Hour)
		expectedNext := time.Date(2020, 3, 1, 2, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedNext, nextMonth)

		// Test from February 29th
		feb29 := time.Date(2020, 2, 29, 15, 0, 0, 0, time.UTC)
		nextFromFeb29 := NextMonthlyTime(feb29, 3*time.Hour)
		expectedFromFeb29 := time.Date(2020, 3, 1, 3, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedFromFeb29, nextFromFeb29)
	})

	// Test non-leap year February (28 days)
	t.Run("non-leap year February", func(t *testing.T) {
		t.Parallel()

		// 2021 is not a leap year
		nonLeapFeb := time.Date(2021, 2, 15, 10, 30, 0, 0, time.UTC)

		// Test StartOfMonth
		startOfMonth := StartOfMonth(nonLeapFeb)
		expected := time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, startOfMonth)

		// Test NextMonthlyTime from non-leap February
		nextMonth := NextMonthlyTime(nonLeapFeb, 2*time.Hour)
		expectedNext := time.Date(2021, 3, 1, 2, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedNext, nextMonth)

		// Test from February 28th
		feb28 := time.Date(2021, 2, 28, 15, 0, 0, 0, time.UTC)
		nextFromFeb28 := NextMonthlyTime(feb28, 3*time.Hour)
		expectedFromFeb28 := time.Date(2021, 3, 1, 3, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedFromFeb28, nextFromFeb28)
	})

	// Test 31-day months (big months)
	t.Run("31-day months", func(t *testing.T) {
		t.Parallel()

		bigMonths := []time.Month{
			time.January, time.March, time.May, time.July,
			time.August, time.October, time.December,
		}

		for _, month := range bigMonths {
			t.Run(month.String(), func(t *testing.T) {
				t.Parallel()

				// Test from 31st day
				day31 := time.Date(2021, month, 31, 14, 30, 0, 0, time.UTC)

				// Test StartOfMonth
				startOfMonth := StartOfMonth(day31)
				expected := time.Date(2021, month, 1, 0, 0, 0, 0, time.UTC)
				assert.Equal(t, expected, startOfMonth)

				// Test NextMonthlyTime
				nextMonth := NextMonthlyTime(day31, 4*time.Hour)

				var expectedNext time.Time

				if month == time.December {
					// December to January of next year
					expectedNext = time.Date(2022, 1, 1, 4, 0, 0, 0, time.UTC)
				} else {
					expectedNext = time.Date(2021, month+1, 1, 4, 0, 0, 0, time.UTC)
				}

				assert.Equal(t, expectedNext, nextMonth)
			})
		}
	})

	// Test 30-day months (small months)
	t.Run("30-day months", func(t *testing.T) {
		t.Parallel()

		smallMonths := []time.Month{
			time.April, time.June, time.September, time.November,
		}

		for _, month := range smallMonths {
			t.Run(month.String(), func(t *testing.T) {
				t.Parallel()

				// Test from 30th day
				day30 := time.Date(2021, month, 30, 14, 30, 0, 0, time.UTC)

				// Test StartOfMonth
				startOfMonth := StartOfMonth(day30)
				expected := time.Date(2021, month, 1, 0, 0, 0, 0, time.UTC)
				assert.Equal(t, expected, startOfMonth)

				// Test NextMonthlyTime
				nextMonth := NextMonthlyTime(day30, 4*time.Hour)
				expectedNext := time.Date(2021, month+1, 1, 4, 0, 0, 0, time.UTC)
				assert.Equal(t, expectedNext, nextMonth)
			})
		}
	})
}

// TestLeapYearDetection tests various leap year scenarios
func TestLeapYearDetection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		year        int
		isLeap      bool
		description string
	}{
		{2000, true, "divisible by 400"},
		{1900, false, "divisible by 100 but not 400"},
		{2004, true, "divisible by 4 but not 100"},
		{2001, false, "not divisible by 4"},
		{2020, true, "recent leap year"},
		{2021, false, "recent non-leap year"},
		{2024, true, "future leap year"},
		{1600, true, "old leap year divisible by 400"},
		{1700, false, "old non-leap year divisible by 100"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("year_%d_%s", tc.year, tc.description), func(t *testing.T) {
			t.Parallel()

			// Test February in the given year
			feb15 := time.Date(tc.year, 2, 15, 12, 0, 0, 0, time.UTC)
			nextMonth := NextMonthlyTime(feb15, 0)

			// Should always be March 1st regardless of leap year
			expectedNext := time.Date(tc.year, 3, 1, 0, 0, 0, 0, time.UTC)
			assert.Equal(t, expectedNext, nextMonth)

			// Test the actual leap year behavior by checking February days
			if tc.isLeap {
				// Leap year: February 29th should be valid
				feb29 := time.Date(tc.year, 2, 29, 0, 0, 0, 0, time.UTC)
				assert.Equal(t, 29, feb29.Day(), "February 29th should be valid in leap year %d", tc.year)
			} else {
				// Non-leap year: February 29th should roll over to March 1st
				feb29Attempt := time.Date(tc.year, 2, 29, 0, 0, 0, 0, time.UTC)
				assert.Equal(t, time.March, feb29Attempt.Month(), "February 29th should roll over to March in non-leap year %d", tc.year)
				assert.Equal(t, 1, feb29Attempt.Day(), "February 29th should roll over to March 1st in non-leap year %d", tc.year)
			}
		})
	}
}

// TestMonthBoundaries tests edge cases around month boundaries
func TestMonthBoundaries(t *testing.T) {
	t.Parallel()

	err := InitSimple("en")
	require.NoError(t, err)

	// Test transition from January to February in leap year
	t.Run("January to February leap year", func(t *testing.T) {
		t.Parallel()

		jan31 := time.Date(2020, 1, 31, 23, 59, 59, 0, time.UTC)
		nextMonth := NextMonthlyTime(jan31, 1*time.Hour)
		expected := time.Date(2020, 2, 1, 1, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, nextMonth)
	})

	// Test transition from January to February in non-leap year
	t.Run("January to February non-leap year", func(t *testing.T) {
		t.Parallel()

		jan31 := time.Date(2021, 1, 31, 23, 59, 59, 0, time.UTC)
		nextMonth := NextMonthlyTime(jan31, 1*time.Hour)
		expected := time.Date(2021, 2, 1, 1, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, nextMonth)
	})

	// Test year boundary (December to January)
	t.Run("December to January", func(t *testing.T) {
		t.Parallel()

		dec31 := time.Date(2021, 12, 31, 23, 0, 0, 0, time.UTC)
		nextMonth := NextMonthlyTime(dec31, 2*time.Hour)
		expected := time.Date(2022, 1, 1, 2, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, nextMonth)
	})

	// Test March to April (31 to 30 days)
	t.Run("March to April", func(t *testing.T) {
		t.Parallel()

		mar31 := time.Date(2021, 3, 31, 15, 30, 0, 0, time.UTC)
		nextMonth := NextMonthlyTime(mar31, 3*time.Hour)
		expected := time.Date(2021, 4, 1, 3, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, nextMonth)
	})

	// Test April to May (30 to 31 days)
	t.Run("April to May", func(t *testing.T) {
		t.Parallel()

		apr30 := time.Date(2021, 4, 30, 20, 15, 0, 0, time.UTC)
		nextMonth := NextMonthlyTime(apr30, 5*time.Hour)
		expected := time.Date(2021, 5, 1, 5, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, nextMonth)
	})
}
