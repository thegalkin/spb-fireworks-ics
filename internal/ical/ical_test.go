package ical

import (
	"strings"
	"testing"
	"time"
)

func TestEscape(t *testing.T) {
	cases := map[string]string{
		"просто текст":   `просто текст`,
		`с \backslash`:   `с \\backslash`,
		"с, запятой":     `с\, запятой`,
		"с; точкой с запятой": `с\; точкой с запятой`,
		"две\nстроки":    `две\nстроки`,
	}
	for in, want := range cases {
		if got := escape(in); got != want {
			t.Errorf("escape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFoldShort(t *testing.T) {
	var b strings.Builder
	fold(&b, "SHORT:hello")
	out := b.String()
	if !strings.HasPrefix(out, "SHORT:hello\r\n") {
		t.Errorf("short fold wrong: %q", out)
	}
}

func TestFoldLong(t *testing.T) {
	var b strings.Builder
	longLine := "X-WR-CALDESC:" + strings.Repeat("А", 200)
	fold(&b, longLine)
	got := b.String()
	if !strings.HasSuffix(got, "\r\n") {
		t.Fatal("no trailing CRLF")
	}
	for i, line := range strings.Split(got, "\r\n") {
		if line == "" {
			continue
		}
		// continuation lines начинаются с пробела — учитываем это при измерении
		effective := line
		if i > 0 && strings.HasPrefix(line, " ") {
			effective = line[1:]
		}
		if len(effective) > 75 {
			t.Errorf("line %d after fold exceeds 75 octets: %d", i, len(effective))
		}
	}
}

func TestSanityCheck(t *testing.T) {
	if msg := SanityCheck(sampleICS()); msg != "" {
		t.Errorf("sanity: %s", msg)
	}
}

func TestWriteBasics(t *testing.T) {
	ev := []Event{{
		UID:     "test-2026@spb",
		Start:   time.Date(2026, 5, 9, 22, 0, 0, 0, mustMSK()),
		End:     time.Date(2026, 5, 9, 22, 20, 0, 0, mustMSK()),
		Summary: "Салют: День Победы",
		Location: "СПб",
	}}
	out := Write(ev, "Тест")
	if !strings.Contains(out, "BEGIN:VCALENDAR") || !strings.Contains(out, "END:VCALENDAR") {
		t.Fatal("missing VCALENDAR wrapper")
	}
	if !strings.Contains(out, "DTSTART:20260509T190000Z") { // 22:00 MSK = 19:00 UTC
		t.Error("DTSTART in UTC wrong")
	}
}

func sampleICS() string {
	return Write([]Event{{
		UID: "x", Start: time.Date(2026, 1, 27, 21, 0, 0, 0, mustMSK()),
		End:  time.Date(2026, 1, 27, 21, 15, 0, 0, mustMSK()),
	}}, "Test")
}

func mustMSK() *time.Location {
	l, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}
	return l
}
