package main

import (
	"bladeready/internal/application"
	"bladeready/internal/store"
	"bladeready/internal/web"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("服务退出", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := parseConfig()
	if err != nil {
		return err
	}
	if cfg.selfcheck {
		dir, err := os.MkdirTemp("", "bladeready-selfcheck-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
		cfg.db = filepath.Join(dir, "selfcheck.db")
	}
	repo, err := store.Open(cfg.db)
	if err != nil {
		return err
	}
	defer repo.Close()
	app := application.New(repo)
	handler := web.New(app).Handler()
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", cfg.addr, err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("风叶巡检复机放行台已启动", "addr", listener.Addr().String())
		if e := server.Serve(listener); e != nil && !errors.Is(e, http.ErrServerClosed) {
			errCh <- e
		}
	}()
	if cfg.selfcheck {
		base := "http://" + listener.Addr().String()
		if err = executeSelfcheck(base); err != nil {
			_ = server.Close()
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = server.Shutdown(ctx); err != nil {
			return err
		}
		fmt.Println("selfcheck passed: 建案、边界冻结、观测、评估、维修计划、复测、安全复核和放行凭证均已验证")
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
	case err = <-errCh:
		return err
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
