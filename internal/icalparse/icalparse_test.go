package icalparse

import (
	"strings"
	"testing"
	"time"
)

const sampleICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//test//
BEGIN:VEVENT
UID:abc-1@test
SUMMARY:Выходной
DTSTART;VALUE=DATE:20260101
DTEND;VALUE=DATE:20260109
LOCATION:Russia
END:VEVENT
BEGIN:VEVENT
UID:abc-2@test
SUMMARY:Длинная
 строка
DTSTART:20260509T190000Z
DTEND:20260509T191500Z
END:VEVENT
END:VCALENDAR
`

func TestParseBasic(t *testing.T) {
	evts, err := Parse(strings.NewReader(sampleICS))
	if err != nil {
		t.Fatal(err)
	}
	if len(evts) != 2 {
		t.Fatalf("ожидали 2 события, получили %d", len(evts))
	}
	a := evts[0]
	if a.UID != "abc-1@test" || a.Summary != "Выходной" || a.Location != "Russia" {
		t.Errorf("evt0 wrong: %+v", a)
	}
	if !a.AllDay {
		t.Error("evt0 должен быть all-day")
	}
	if a.Start.Format("20060102") != "20260101" {
		t.Errorf("evt0 date: %s", a.Start.Format("20060102"))
	}

	b := evts[1]
	if b.AllDay {
		t.Error("evt1 не all-day")
	}
	// RFC 5545 §3.1: leading space of continuation line is part of the fold marker, dropped.
	if b.Summary != "Длиннаястрока" {
		t.Errorf("evt1 folding: %q", b.Summary)
	}
	if !b.Start.Equal(time.Date(2026, 5, 9, 19, 0, 0, 0, time.UTC)) {
		t.Errorf("evt1 DTSTART: %v", b.Start)
	}
}

func TestParseRejectsUnbalanced(t *testing.T) {
	bad := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nEND:VCALENDAR\n"
	if _, err := Parse(strings.NewReader(bad)); err == nil {
		t.Error("должны ловить незакрытый VEVENT")
	}
}

func TestParseRejectsEventOutsideCalendar(t *testing.T) {
	bad := "BEGIN:VEVENT\nEND:VEVENT\nEND:VCALENDAR\n"
	if _, err := Parse(strings.NewReader(bad)); err == nil {
		t.Error("должны ловить VEVENT вне VCALENDAR")
	}
}
