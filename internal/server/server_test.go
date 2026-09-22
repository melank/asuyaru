package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/melank/asuyaru/internal/jev"
)

func TestHome(t *testing.T) {
	t.Parallel()
	pages, err := ParseTemplates()
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h, err := New(log, pages, jev.NewFromEnv())
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "明日やる") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if rec.Header().Get("Set-Cookie") == "" {
		t.Fatal("CSRF の Cookie が無い")
	}

	css := httptest.NewRecorder()
	h.ServeHTTP(css, httptest.NewRequest(http.MethodGet, "/static/app.css", nil))
	if css.Code != http.StatusOK {
		t.Fatalf("css status = %d", css.Code)
	}
	if !strings.Contains(css.Body.String(), "Hiragino Sans") {
		t.Fatalf("css = %s", css.Body.String())
	}
}

func TestPostWithoutToken(t *testing.T) {
	t.Parallel()
	pages, err := ParseTemplates()
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h, err := New(log, pages, &jev.Client{})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("csrf=nope"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}
