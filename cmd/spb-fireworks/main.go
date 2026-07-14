// cmd/spb-fireworks собирает .ics-календарь салютов и праздников СПб.
//
// Использование:
//
//	go run ./cmd/spb-fireworks -year=2026 -out=spb-fireworks-2026.ics
//
// Источник федеральных праздников — публичный .ics от nikitastupin/prodcal_ics.
// СПб-события генерируются из правил (см. internal/events/spb.go).
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/thegalkin/spb-fireworks-ics/internal/events"
	"github.com/thegalkin/spb-fireworks-ics/internal/fetch"
	"github.com/thegalkin/spb-fireworks-ics/internal/ical"
	"github.com/thegalkin/spb-fireworks-ics/internal/icalparse"
)

const defaultUpstream = "https://prodcal.nikitastupin.com/prodcal.ics"

func main() {
	var (
		year      = flag.Int("year", 0, "целевой год (по умолчанию — текущий по системным часам)")
		outPath   = flag.String("out", "spb-fireworks.ics", "путь для записи .ics")
		upstream  = flag.String("upstream", defaultUpstream, "URL upstream-календаря")
		skipUp    = flag.Bool("skip-upstream", false, "пропустить загрузку upstream (только правила СПб)")
		writeBoth = flag.Bool("write-both-years", false, "сгенерировать календарь на 2 года (текущий и следующий) в один файл")
	)
	flag.Parse()

	if *year == 0 {
		*year = time.Now().Year()
	}

	ctx := context.Background()
	var upstreamEvents []icalparse.Event

	if !*skipUp {
		res, err := fetch.Fetch(ctx, fetch.ClientOptions{
			URL:      *upstream,
			CacheKey: "prodcal",
		})
		if err != nil {
			log.Printf("WARN: upstream fetch failed: %v", err)
			// продолжаем — upstream опционален: СПб-правила всё равно сработают
		} else {
			log.Printf("upstream: %d bytes, sha256=%s, from_cache=%v, not_modified=%v",
				len(res.Body), res.SHA256[:12], res.FromCache, res.NotModified)
			evts, err := icalparse.Parse(bytes.NewReader(res.Body))
			if err != nil {
				log.Fatalf("parse upstream: %v", err)
			}
			upstreamEvents = evts
		}
	}

	allYears := []int{*year}
	if *writeBoth {
		allYears = append(allYears, *year+1)
	}

	var allOut []ical.Event
	for _, y := range allYears {
		allOut = append(allOut, events.Build(y, upstreamEvents)...)
	}

	out := ical.Write(allOut, fmt.Sprintf("Салюты и праздники СПб %d", *year))
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*outPath, []byte(out), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("OK %s — %d событий\n", *outPath, len(allOut))
}
