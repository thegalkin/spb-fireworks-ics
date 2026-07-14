package calendar

import (
	"testing"
	"time"
)

func TestLastSaturdayOfJune(t *testing.T) {
	cases := []struct {
		year    int
		wantMD  string
	}{
		{2026, "2026-06-27"},
		{2027, "2027-06-26"},
		{2025, "2025-06-28"},
	}
	for _, c := range cases {
		got := LastSaturdayOfJune(c.year).Format("2006-01-02")
		if got != c.wantMD {
			t.Errorf("LastSaturdayOfJune(%d) = %s, want %s", c.year, got, c.wantMD)
		}
	}
}

func TestLastSundayOfJuly(t *testing.T) {
	cases := []struct {
		year   int
		wantMD string
	}{
		{2026, "2026-07-26"},
		{2027, "2027-07-25"},
		{2025, "2025-07-27"},
	}
	for _, c := range cases {
		got := LastSundayOfJuly(c.year).Format("2006-01-02")
		if got != c.wantMD {
			t.Errorf("LastSundayOfJuly(%d) = %s, want %s", c.year, got, c.wantMD)
		}
	}
}

func TestWeekdayStability(t *testing.T) {
	// Сама проверка: получаемая дата действительно — тот день недели, который заявлен.
	for y := 2024; y <= 2030; y++ {
		d := LastSaturdayOfJune(y)
		if d.Weekday() != time.Saturday {
			t.Errorf("%d-06: %s is not Saturday", y, d.Format("2006-01-02"))
		}
		d = LastSundayOfJuly(y)
		if d.Weekday() != time.Sunday {
			t.Errorf("%d-07: %s is not Sunday", y, d.Format("2006-01-02"))
		}
	}
}
