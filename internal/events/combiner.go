package events

import (
	"sort"
	"time"

	"github.com/thegalkin/spb-fireworks-ics/internal/ical"
	"github.com/thegalkin/spb-fireworks-ics/internal/icalparse"
)

// Filter переиспользует upstream-события (производственный календарь prodcal.ics):
// оставляет только те, что пересекаются с целевым годом, плюс ограничивает
// категориями, которые нам интересны для подсказки о переносах.
//
// Возвращает только федеральные «Выходной» / «Праздник» / «Предпраздничный день»
// — нам нужно знать, какие дни официально нерабочие. Сам салют-календарь
// составляется из правил (см. SPbFireworks), а не из upstream.
func Filter(upstream []icalparse.Event, year int) []ical.Event {
	out := make([]ical.Event, 0, len(upstream))
	for _, u := range upstream {
		if u.Start.Year() != year {
			continue
		}
		switch category(u.Summary) {
		case "holiday", "preholiday":
			// эти мы покажем в финальном .ics, чтобы пользователь видел переносы
			out = append(out, ical.Event{
				UID:         "ru-" + u.UID + "-2026@spb-fireworks-ics",
				Start:       u.Start,
				End:         u.End,
				AllDay:      u.AllDay,
				Summary:     "Федеральный: " + u.Summary,
				Description: u.Location,
				Location:    u.Location,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// category эвристически классифицирует upstream-событие. Значения берутся из реального prodcal.ics:
//   "Выходной", "Праздник", "Предпраздничный день", "Рабочий день".
func category(summary string) string {
	switch {
	case contains(summary, "Праздник"):
		return "holiday"
	case contains(summary, "Выходной"):
		return "holiday"
	case contains(summary, "Предпраздничный"):
		return "preholiday"
	}
	return "skip"
}

func contains(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// Deduplicate удаляет дубликаты по UID и по (Summary, Start) паре.
// Нужно потому, что SPb-правила могут задать событие, которое upstream тоже даёт
// (например, 12 июня — «День России»).
func Deduplicate(events []ical.Event) []ical.Event {
	byUID := make(map[string]ical.Event, len(events))
	byKey := make(map[string]bool, len(events))
	for _, e := range events {
		if _, ok := byUID[e.UID]; ok {
			continue
		}
		k := e.Summary + "@" + e.Start.UTC().Format(time.RFC3339)
		if byKey[k] {
			continue
		}
		byKey[k] = true
		byUID[e.UID] = e
	}
	out := make([]ical.Event, 0, len(byUID))
	for _, e := range byUID {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// Build — финальная сборка: SPb-правила + фильтрованный upstream.
func Build(year int, upstream []icalparse.Event) []ical.Event {
	spb := SPbFireworks(year)
	federal := Filter(upstream, year)
	combined := append(spb, federal...)
	return Deduplicate(combined)
}
