// Package icalparse — узкий, заточенный под задачу парсер iCalendar (RFC 5545 subset).
//
// Поддерживается ровно то, что встречается в prodcal.ics и нашем writer:
//   - раскручивание LINE-FOLDING (RFC 3.1);
//   - BEGIN/END блоков (только VCALENDAR и VEVENT);
//   - простые property «KEY» и «KEY;PARAM=val»: SUMMARY, DTSTART, DTEND, LOCATION, UID.
//
// НЕ поддерживается (и не встретится в наших источниках): recurrence (RRULE), exceptions,
// alarms, time zones. Только самое нужное: «список событий на год».
package icalparse

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Event — минимальное представление.
type Event struct {
	UID         string
	Summary     string
	Location    string
	Start, End  time.Time // для all-day = 00:00 UTC
	AllDay      bool
}

// Parse читает iCal-поток и возвращает список событий.
// Любая незакрытая скобка / непарный BEGIN считается ошибкой.
func Parse(r io.Reader) ([]Event, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<16), 1<<20)

	var lines []string
	for sc.Scan() {
		line := sc.Text()
		// Согласно RFC 5545 line endings = CRLF; Scanner отрезает оба варианта, ок.
		// Folding (RFC 3.1): продолжение начинается с SPACE или HTAB — объединяем с предыдущей строкой.
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && len(lines) > 0 {
			lines[len(lines)-1] += line[1:]
			continue
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	var (
		out   []Event
		cur   *Event
		inEv  bool
		inCal bool
	)
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		switch {
		case line == "BEGIN:VCALENDAR":
			if inCal {
				return nil, errors.New("icalparse: nested VCALENDAR")
			}
			inCal = true
		case line == "END:VCALENDAR":
			if !inCal {
				return nil, errors.New("icalparse: END:VCALENDAR without BEGIN")
			}
			if inEv {
				return nil, errors.New("icalparse: END:VCALENDAR при незакрытом VEVENT")
			}
			flush()
			inCal = false
		case line == "BEGIN:VEVENT":
			if !inCal {
				return nil, errors.New("icalparse: VEVENT вне VCALENDAR")
			}
			if inEv {
				return nil, errors.New("icalparse: nested VEVENT")
			}
			cur = &Event{}
			inEv = true
		case line == "END:VEVENT":
			if !inEv {
				return nil, errors.New("icalparse: END:VEVENT without BEGIN")
			}
			flush()
			inEv = false
		case inEv:
			k, v, ok := splitProp(line)
			if !ok {
				continue
			}
			applyEvent(cur, k, v)
		}
	}
	return out, nil
}

// splitProp разделяет «SUMMARY:foo» или «DTSTART;VALUE=DATE:20260509» на (key, value).
func splitProp(line string) (key, val string, ok bool) {
	i := strings.IndexByte(line, ':')
	if i < 0 {
		return "", "", false
	}
	left := line[:i]
	val = line[i+1:]
	if j := strings.IndexByte(left, ';'); j >= 0 {
		key = left[:j]
	} else {
		key = left
	}
	return key, val, true
}

// applyEvent кладёт распарсенное свойство в Event. Неизвестные — игнорируются.
func applyEvent(e *Event, key, val string) {
	switch key {
	case "UID":
		e.UID = unescape(val)
	case "SUMMARY":
		e.Summary = unescape(val)
	case "LOCATION":
		e.Location = unescape(val)
	case "DTSTART":
		t, allDay, err := parseDateOrDateTime(val)
		if err == nil {
			e.Start = t
			e.AllDay = allDay
		}
	case "DTEND":
		t, _, err := parseDateOrDateTime(val)
		if err == nil {
			e.End = t
		}
	}
}

// parseDateOrDateTime понимает DATE (20260127) и DATE-TIME (20260127T210000Z).
func parseDateOrDateTime(s string) (time.Time, bool, error) {
	if len(s) == 8 { // DATE
		t, err := time.Parse("20060102", s)
		if err != nil {
			return time.Time{}, false, fmt.Errorf("icalparse: bad DATE %q: %w", s, err)
		}
		return t, true, nil
	}
	// DATE-TIME без TZ = floating; мы их не поддерживаем, считаем ошибкой.
	if !strings.HasSuffix(s, "Z") {
		return time.Time{}, false, fmt.Errorf("icalparse: unsupported DT format %q", s)
	}
	t, err := time.Parse("20060102T150405Z", s)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("icalparse: bad DATE-TIME %q: %w", s, err)
	}
	return t, false, nil
}

// unescape — обратное преобразование escape из RFC 5.3.2.
func unescape(s string) string {
	if !strings.ContainsAny(s, `\,`) {
		return s
	}
	r := strings.NewReplacer(
		`\n`, "\n",
		`\,`, ",",
		`\;`, ";",
		`\\`, `\`,
	)
	return r.Replace(s)
}
