package backups

import (
	"github.com/ocelot-cloud/shared/assert"
	"testing"
	"time"
)

func TestIsMaintenanceDue(t *testing.T) {
	assert.False(t, isMaintenanceCycleDue(false, false))
	assert.True(t, isMaintenanceCycleDue(false, true))
	assert.False(t, isMaintenanceCycleDue(true, false))
	assert.False(t, isMaintenanceCycleDue(true, true))
}

func TestWasLastMaintenanceCycleExecutedToday(t *testing.T) {
	// Same day, same year
	{
		now := time.Date(2025, 4, 17, 10, 0, 0, 0, time.UTC)
		last := time.Date(2025, 4, 17, 1, 0, 0, 0, time.UTC)
		assert.True(t, wasLastMaintenanceCycleExecutedToday(now, last))
	}

	// Different day, same year
	{
		now := time.Date(2025, 4, 17, 1, 0, 0, 0, time.UTC)
		last := time.Date(2025, 4, 16, 20, 0, 0, 0, time.UTC)
		assert.False(t, wasLastMaintenanceCycleExecutedToday(now, last))
	}

	// Different day, different year
	{
		now := time.Date(2024, 12, 31, 23, 30, 0, 0, time.UTC)
		last := time.Date(2025, 1, 1, 0, 30, 0, 0, time.UTC)
		assert.False(t, wasLastMaintenanceCycleExecutedToday(now, last))
	}

	// Same day, different year
	{
		now := time.Date(2024, 1, 1, 0, 30, 0, 0, time.UTC)
		last := time.Date(2025, 1, 1, 1, 30, 0, 0, time.UTC)
		assert.False(t, wasLastMaintenanceCycleExecutedToday(now, last))
	}
}

func TestIsWithinTimeRangeOfBeingExecutedToday(t *testing.T) {
	preferredHour := 4

	tests := []struct {
		name     string
		nowUTC   time.Time
		expected bool
	}{
		{
			name:     "Before window",
			nowUTC:   time.Date(2025, 4, 17, 1, 59, 59, 0, time.UTC),
			expected: false,
		},
		{
			name:     "At window start",
			nowUTC:   time.Date(2025, 4, 17, 4, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Inside window",
			nowUTC:   time.Date(2025, 4, 17, 4, 30, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "At window end",
			nowUTC:   time.Date(2025, 4, 17, 5, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "After window",
			nowUTC:   time.Date(2025, 4, 17, 5, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWithinTimeRangeOfBeingExecutedToday(tt.nowUTC, preferredHour)
			assert.Equal(t, tt.expected, result)
		})
	}
}
