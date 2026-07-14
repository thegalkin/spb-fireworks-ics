package fetch

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func resetCache(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll(".cache"); err != nil {
		t.Fatal(err)
	}
}

func TestFetchBasic(t *testing.T) {
	resetCache(t)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(200)
		io.WriteString(w, "BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n")
	}))
	defer srv.Close()

	res, err := Fetch(context.Background(), ClientOptions{
		URL:                   srv.URL,
		CacheKey:              "k1",
		HTTPClient:            &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.NotModified || res.FromCache {
		t.Error("первый запрос не должен быть из кэша")
	}
	if !bytes.HasPrefix(res.Body, []byte("BEGIN:VCALENDAR")) {
		t.Errorf("unexpected body: %q", res.Body)
	}
	if res.SHA256 == "" {
		t.Error("empty SHA256")
	}
}

func TestFetch304(t *testing.T) {
	resetCache(t)
	if err := os.MkdirAll(".cache", 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(".cache", "k1.ics"), []byte("BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n"), 0o644)
	os.WriteFile(filepath.Join(".cache", "k1.etag"), []byte(`"abc"`+"\n"), 0o644)

	calls := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("If-None-Match") != `"abc"` {
			t.Errorf("expected If-None-Match, got %q", r.Header.Get("If-None-Match"))
		}
		w.WriteHeader(304)
	}))
	defer srv.Close()

	res, err := Fetch(context.Background(), ClientOptions{
		URL:                   srv.URL,
		CacheKey:              "k1",
		HTTPClient:            &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.NotModified || !res.FromCache {
		t.Errorf("ожидали 304/FromCache, получили %+v", res)
	}
	if calls != 1 {
		t.Errorf("ожидали 1 вызов, получили %d", calls)
	}
}

func TestRejectsNonHTTPS(t *testing.T) {
	_, err := Fetch(context.Background(), ClientOptions{
		URL:        "http://insecure.example.com/cal.ics",
		CacheKey:   "k",
		HTTPClient: &http.Client{},
	})
	if err == nil {
		t.Error("должны отвергать http://")
	}
}

func TestSaveArtifactIfChanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.ics")

	changed, err := SaveArtifactIfChanged(path, []byte("A"))
	if err != nil || !changed {
		t.Fatalf("1: changed=%v err=%v", changed, err)
	}
	changed, err = SaveArtifactIfChanged(path, []byte("A"))
	if err != nil || changed {
		t.Fatalf("2: ожидали changed=false, получили changed=%v err=%v", changed, err)
	}
	changed, err = SaveArtifactIfChanged(path, []byte("B"))
	if err != nil || !changed {
		t.Fatalf("3: ожидали changed=true, получили changed=%v err=%v", changed, err)
	}
}
