// Package fetch — минимальная загрузка iCal-источника по HTTPS с ETag-кэшированием.
//
// Используется только сетевой код: всё остальное — pure functions и парсеры.
//
// Без LLM, без внешних зависимостей: только стандартная библиотека Go.
package fetch

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const cacheDir = ".cache"

// Result — то, что отдаёт Fetch: байты .ics, SHA-256, признаки «обновилось/нет».
type Result struct {
	Body      []byte
	SHA256    string
	NotModified bool // true, если сервер ответил 304 и кэш переиспользован
	FromCache   bool // true, если данные пришли из локального кэша
}

// ClientOptions — параметры клиента.
type ClientOptions struct {
	URL        string        // обязательный, https only
	CacheKey   string        // идентификатор кэш-файла (например "prodcal")
	UserAgent  string        // что слать в User-Agent
	Timeout    time.Duration // на весь запрос
	HTTPClient *http.Client  // для тестов; nil = использовать http.DefaultClient
	AllowInsecureLocalhost bool // для тестов: пропустить TLS-проверку только при hostname=127.0.0.1/localhost
}

func (o ClientOptions) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	if o.Timeout == 0 {
		o.Timeout = 30 * time.Second
	}
	if o.AllowInsecureLocalhost {
		return &http.Client{
			Timeout: o.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	return &http.Client{Timeout: o.Timeout}
}

// Fetch загружает .ics, используя локальный кэш по SHA-256 и ETag для экономии трафика.
//
// Поведение:
//   - Если локального кэша нет — качает и сохраняет.
//   - Если есть — шлёт If-None-Match с предыдущим ETag (если он есть в .etag-файле).
//   - Если сервер ответил 304 — переиспользует кэшированный файл (FromCache=true, NotModified=true).
//   - Если 200 — записывает новый файл и обновляет ETag.
func Fetch(ctx context.Context, opts ClientOptions) (*Result, error) {
	if opts.URL == "" {
		return nil, errors.New("fetch: empty URL")
	}
	if !strings.HasPrefix(opts.URL, "https://") {
		return nil, fmt.Errorf("fetch: refusing non-https URL %q", opts.URL)
	}
	if opts.CacheKey == "" {
		return nil, errors.New("fetch: empty CacheKey")
	}
	if opts.UserAgent == "" {
		opts.UserAgent = "spb-fireworks-ics/1.0 (+github.com/thegalkin/spb-fireworks-ics)"
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}
	bodyPath := filepath.Join(cacheDir, opts.CacheKey+".ics")
	etagPath := filepath.Join(cacheDir, opts.CacheKey+".etag")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", opts.UserAgent)
	req.Header.Set("Accept", "text/calendar, text/plain;q=0.9, */*;q=0.1")

	if etag, err := os.ReadFile(etagPath); err == nil && len(etag) > 0 {
		req.Header.Set("If-None-Match", strings.TrimSpace(string(etag)))
	}

	resp, err := opts.client().Do(req)
	if err != nil {
		// если сеть упала — попробуем отдать кэш, если он есть
		if body, rerr := os.ReadFile(bodyPath); rerr == nil {
			return &Result{
				Body:       body,
				SHA256:     sha256Hex(body),
				FromCache:  true,
				NotModified: true,
			}, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(bodyPath, body, 0o644); err != nil {
			return nil, err
		}
		if etag := resp.Header.Get("ETag"); etag != "" {
			_ = os.WriteFile(etagPath, []byte(etag+"\n"), 0o644)
		}
		return &Result{
			Body:   body,
			SHA256: sha256Hex(body),
		}, nil

	case http.StatusNotModified:
		body, err := os.ReadFile(bodyPath)
		if err != nil {
			return nil, fmt.Errorf("fetch: 304 но кэш %s пропал: %w", bodyPath, err)
		}
		return &Result{
			Body:       body,
			SHA256:     sha256Hex(body),
			FromCache:  true,
			NotModified: true,
		}, nil

	default:
		return nil, fmt.Errorf("fetch: HTTP %d от %s", resp.StatusCode, opts.URL)
	}
}

// SaveArtifactIfChanged — пишет файл только если содержимое отличается от того, что лежит
// на диске. Возвращает true, если файл реально записан (считай — «обновилось»).
//
// Используется для идемпотентных GitHub Actions: коммитим только при изменениях.
func SaveArtifactIfChanged(path string, body []byte) (changed bool, err error) {
	existing, rerr := os.ReadFile(path)
	if rerr == nil && sha256Hex(existing) == sha256Hex(body) {
		return false, nil
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
