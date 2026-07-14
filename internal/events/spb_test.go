package events

import (
	"testing"
	"time"

	"github.com/thegalkin/spb-fireworks-ics/internal/icalparse"
)

func TestSPbFireworksContainsAnchors(t *testing.T) {
	evs := SPbFireworks(2026)
	want := []string{
		"День Победы",
		"День города",
		"День России",
		"Алые паруса",
		"День ВМФ",
		"блокады",
		"День защитника Отечества",
		"народного единства",
	}
	have := ""
	for _, e := range evs {
		have += e.Summary + "\n"
	}
	for _, s := range want {
		if !contains(have, s) {
			t.Errorf("отсутствует событие со словом %q", s)
		}
	}
}

func TestSPbIsSortedByTime(t *testing.T) {
	evs := SPbFireworks(2026)
	for i := 1; i < len(evs); i++ {
		if evs[i].Start.Before(evs[i-1].Start) {
			t.Errorf("не отсортировано: %v после %v", evs[i].Start, evs[i-1].Start)
		}
	}
}

func TestFilterDropsOutOfYear(t *testing.T) {
	up := []icalparse.Event{
		{UID: "x1", Summary: "Выходной", Start: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)},
		{UID: "x2", Summary: "Выходной", Start: time.Date(2027, 5, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2027, 5, 2, 0, 0, 0, 0, time.UTC)},
	}
	got := Filter(up, 2026)
	if len(got) != 1 {
		t.Fatalf("ожидали 1, получили %d", len(got))
	}
	if got[0].Summary != "Федеральный: Выходной" {
		t.Errorf("prefix не добавился: %q", got[0].Summary)
	}
}

func TestDeduplicateBySummaryStart(t *testing.T) {
	evs := []testEventLike{
		mkEv("a", "12 июня", time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)),
		mkEv("b", "12 июня", time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)),
		mkEv("c", "13 июня", time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)),
	}
	out := dedupeAny(evs)
	if len(out) != 2 {
		t.Errorf("ожидали 2 после dedup, получили %d", len(out))
	}
}

type testEventLike struct {
	UID, Summary string
	Start        time.Time
}

func mkEv(uid, sum string, t time.Time) testEventLike {
	return testEventLike{uid, sum, t}
}

func dedupeAny(in []testEventLike) []testEventLike {
	seen := map[string]bool{}
	out := make([]testEventLike, 0, len(in))
	for _, x := range in {
		k := x.Summary + "@" + x.Start.UTC().Format(time.RFC3339)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, x)
	}
	return out
}
