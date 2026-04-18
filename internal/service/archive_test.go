package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Modificator/readlater-wip/internal/store"
)

func TestDownloadImages_DedupeAndFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a.png":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("image-a"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	repo := store.NewMemoryRepository()
	svc := NewArchiveService(repo, NewRuleService(repo), NewGiteaService(repo), t.TempDir())

	assetsDir := t.TempDir()
	html := `<img src="/a.png"><img data-src="/a.png"><img src="/missing.png">`
	mapping, stats := svc.downloadImages(html, ts.URL+"/page", assetsDir)

	if stats.Downloaded != 1 {
		t.Fatalf("expected downloaded=1, got %d", stats.Downloaded)
	}
	if stats.Failed != 1 {
		t.Fatalf("expected failed=1, got %d", stats.Failed)
	}
	if mapping["/a.png"] == "" {
		t.Fatalf("expected /a.png to be mapped")
	}
	if !strings.HasPrefix(mapping["/a.png"], "assets/") {
		t.Fatalf("expected mapped path in assets/, got %s", mapping["/a.png"])
	}
	if _, err := os.Stat(assetsDir + "/" + strings.TrimPrefix(mapping["/a.png"], "assets/")); err != nil {
		t.Fatalf("expected downloaded file to exist: %v", err)
	}
}
