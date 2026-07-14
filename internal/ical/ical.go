// Package ical собирает минимальный валидный iCalendar (RFC 5545 subset) без зависимостей.
//
// Реализовано ровно то, что нужно для СПб-календаря: VEVENT c DATE / DATE-TIME,
// UTC-формат, CRLF, line-folding до 75 октетов, escape по RFC 5.3.2 (Backslash, Comma, Semicolon, Newline).
package ical

import (
	"fmt"
	"strings"
	"time"
)

// Event — минимальное событие iCalendar.
type Event struct {
	UID         string
	Start, End  time.Time // локально Europe/Moscow; внутри пишется как UTC DATE-TIME
	AllDay      bool
	Summary     string
	Description string
	Location    string
}

// Write — основной entrypoint: собирает полный VCALENDAR для списка событий.
func Write(events []Event, calendarName string) string {
	var b strings.Builder

	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//spb-fireworks-ics//RU//ICS\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")
	fold(&b, "X-WR-CALNAME:"+calendarName)
	fold(&b, "X-WR-TIMEZONE:Europe/Moscow")

	stamp := dt(time.Now().UTC())
	for _, e := range events {
		writeEvent(&b, e, stamp)
	}
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}

func writeEvent(b *strings.Builder, e Event, stamp string) {
	start, end := timesFor(e)
	b.WriteString("BEGIN:VEVENT\r\n")
	fold(b, "UID:"+e.UID)
	fold(b, "DTSTAMP:"+stamp)
	fold(b, "DTSTART:"+start)
	fold(b, "DTEND:"+end)
	fold(b, "SUMMARY:"+e.Summary)
	if e.Description != "" {
		fold(b, "DESCRIPTION:"+e.Description)
	}
	if e.Location != "" {
		fold(b, "LOCATION:"+e.Location)
	}
	b.WriteString("CATEGORIES:Праздники СПб,Салюты\r\n")
	b.WriteString("STATUS:CONFIRMED\r\n")
	b.WriteString("TRANSP:OPAQUE\r\n")
	b.WriteString("END:VEVENT\r\n")
}

// timesFor формирует DTSTART/DTEND либо как DATE-TIME (UTC), либо как DATE (all-day).
func timesFor(e Event) (string, string) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	if e.AllDay {
		d0 := time.Date(e.Start.Year(), e.Start.Month(), e.Start.Day(), 0, 0, 0, 0, loc)
		return formatDate(d0), formatDate(d0.AddDate(0, 0, 1))
	}
	return formatDateTime(e.Start), formatDateTime(e.End)
}

func formatDate(t time.Time) string {
	return t.Format("20060102")
}

func formatDateTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// dt — формат DATE-TIME без корректировки локали (для DTSTAMP пишем UTC явно).
func dt(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// fold — line folding (RFC 5545 §3.1): максимум 75 октетов, продолжение с пробела.
func fold(b *strings.Builder, line string) {
	const max = 75
	const sep = "\r\n "
	escaped := escape(line)
	body := []byte(escaped)
	if len(body) <= max {
		b.Write(body)
		b.WriteString("\r\n")
		return
	}
	b.Write(body[:max])
	rest := body[max:]
	for len(rest) > 0 {
		n := max - 1 // место для CRLF + пробел (2 октета)
		if n > len(rest) {
			n = len(rest)
		}
		b.WriteString(sep)
		b.Write(rest[:n])
		rest = rest[n:]
	}
	b.WriteString("\r\n")
}

// escape по RFC 5.3.2 / 3.3.11 (TEXT).
func escape(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`,`, `\,`,
		`;`, `\;`,
		"\n", `\n`,
	)
	return r.Replace(s)
}

// SanityCheck — минимальная локальная проверка результата Write: парность BEGIN/END и CRLF.
// Возвращает первую ошибку либо пустую строку.
func SanityCheck(s string) string {
	if strings.Count(s, "BEGIN:VCALENDAR") != 1 || strings.Count(s, "END:VCALENDAR") != 1 {
		return "VCALENDAR счётчик != 1"
	}
	if strings.Count(s, "BEGIN:VEVENT") != strings.Count(s, "END:VEVENT") {
		return "VEVENT парность нарушена"
	}
	if !strings.Contains(s, "\r\n") {
		return "отсутствует CRLF"
	}
	if strings.ContainsAny(s, "\u2028\u2029") {
		return "встречаются Unicode line separators"
	}
	_ = fmt.Sprintf // keep import if needed later
	return ""
}
