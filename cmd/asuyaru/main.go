package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/melank/asuyaru/internal/jev"
	"github.com/melank/asuyaru/internal/server"
	"github.com/melank/asuyaru/internal/store"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fs := flag.NewFlagSet("asuyaru", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	addr := fs.String("addr", "127.0.0.1:8080", "待ち受けアドレス")
	dbPath := fs.String("db", "", "SQLite のパス")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if err := requireLoopback(*addr); err != nil {
		log.Error("待ち受けを拒否した", "err", err)
		return 1
	}
	path := *dbPath
	if path == "" {
		var err error
		path, err = defaultDBPath()
		if err != nil {
			log.Error("保存先を決められない", "err", err)
			return 1
		}
	}
	db, err := store.Open(path)
	if err != nil {
		log.Error("SQLite を開けない", "err", err)
		return 1
	}
	defer db.Close()
	if err := store.Migrate(context.Background(), db); err != nil {
		log.Error("マイグレーションに失敗した", "err", err)
		return 1
	}
	pages, err := server.ParseTemplates()
	if err != nil {
		log.Error("テンプレートを解析できない", "err", err)
		return 1
	}
	handler, err := server.New(log, pages, jev.NewFromEnv())
	if err != nil {
		log.Error("ハンドラを組み立てられない", "err", err)
		return 1
	}
	srv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Error("待ち受けを開始できない", "err", err)
		return 1
	}
	fmt.Fprintln(os.Stdout, listenURL(ln.Addr()))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("待ち受けを開始できない", "err", err)
			return 1
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("停止に失敗した", "err", err)
			return 1
		}
	}
	return 0
}

func listenURL(addr net.Addr) string {
	return "http://" + addr.String() + "/"
}

func requireLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("アドレスを読む: %w", err)
	}
	switch host {
	case "127.0.0.1", "::1", "localhost":
		return nil
	default:
		return fmt.Errorf("待ち受けは 127.0.0.1 だけである: %s", host)
	}
}

func defaultDBPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("設定ディレクトリを得る: %w", err)
	}
	return filepath.Join(dir, "asuyaru", "asuyaru.db"), nil
}
