package databases

import (
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
)

func TestDatabaseUptimeSecondsClampsIntervalsAndExcludesDowntime(t *testing.T) {
	periodStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	beforePeriod := periodStart.Add(-24 * time.Hour)
	firstStop := periodStart.Add(48 * time.Hour)
	restart := periodStart.Add(72 * time.Hour)
	afterPeriod := periodEnd.Add(24 * time.Hour)

	intervals := []database.DatabaseUptimeInterval{
		{StartedAt: beforePeriod, EndedAt: &firstStop},
		{StartedAt: restart},
		{StartedAt: periodEnd.Add(time.Hour), EndedAt: &afterPeriod},
	}

	want := int64((48*time.Hour + now.Sub(restart)) / time.Second)
	if got := databaseUptimeSeconds(intervals, periodStart, periodEnd, now); got != want {
		t.Fatalf("databaseUptimeSeconds() = %d, want %d", got, want)
	}
}

func TestDatabaseUptimeSecondsClampsHistoricalIntervalToMonth(t *testing.T) {
	periodStart := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)
	startedBefore := periodStart.AddDate(0, -1, 0)
	endedAfter := periodEnd.AddDate(0, 1, 0)

	intervals := []database.DatabaseUptimeInterval{{StartedAt: startedBefore, EndedAt: &endedAfter}}
	want := int64(periodEnd.Sub(periodStart) / time.Second)
	if got := databaseUptimeSeconds(intervals, periodStart, periodEnd, periodEnd); got != want {
		t.Fatalf("databaseUptimeSeconds() = %d, want %d", got, want)
	}
}
