package plan

import (
	"testing"
	"time"
)

func TestFormatDateUsesLocal(t *testing.T) {
	t.Parallel()
	utc := time.Date(2026, 9, 22, 15, 30, 0, 0, time.UTC)
	if got, want := FormatDate(utc), utc.In(time.Local).Format("2006-01-02"); got != want {
		t.Fatalf("FormatDate = %s, want %s", got, want)
	}
	got := ClockFrom(utc)
	local := utc.In(time.Local)
	if got.Hour != local.Hour() || got.Minute != local.Minute() {
		t.Fatalf("ClockFrom = %02d:%02d, want %02d:%02d", got.Hour, got.Minute, local.Hour(), local.Minute())
	}
}
