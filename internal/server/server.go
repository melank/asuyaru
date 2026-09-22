// Package server はルーティング、ハンドラ、HTML テンプレートを持つ。
package server

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/melank/asuyaru/internal/jev"
)

//go:embed templates/*.html static/app.css
var files embed.FS

// ParseTemplates は起動時に HTML を一度だけ解析する。
func ParseTemplates() (*template.Template, error) {
	t, err := template.ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("テンプレートを解析する: %w", err)
	}
	return t, nil
}

// New はページを出すハンドラを返す。
func New(log *slog.Logger, pages *template.Template, client *jev.Client) (http.Handler, error) {
	mux := http.NewServeMux()
	static, err := fs.Sub(files, "static")
	if err != nil {
		return nil, fmt.Errorf("CSS を読む: %w", err)
	}
	app := &app{log: log, pages: pages, jev: client}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /{$}", app.home)
	return recoverMiddleware(log, logMiddleware(log, csrfMiddleware(mux))), nil
}

type app struct {
	log   *slog.Logger
	pages *template.Template
	jev   *jev.Client
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.pages.ExecuteTemplate(w, "home.html", nil); err != nil {
		a.log.Error("ページを書く", "err", err)
		http.Error(w, "ページを表示できません。", http.StatusInternalServerError)
	}
}

func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("asuyaru_csrf")
		token := ""
		if err == nil {
			token = cookie.Value
		}
		if token == "" {
			token, err = newToken()
			if err != nil {
				http.Error(w, "ページを表示できません。", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "asuyaru_csrf",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
		if r.Method == http.MethodPost && subtle.ConstantTimeCompare([]byte(r.FormValue("csrf")), []byte(token)) != 1 {
			http.Error(w, "送信を受け付けられません。", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func logMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Info("リクエスト", "method", r.Method, "path", r.URL.Path, "status", sw.code, "ms", time.Since(start).Milliseconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func recoverMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("ハンドラがパニックした", "err", rec)
				http.Error(w, "ページを表示できません。", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
