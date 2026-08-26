package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				slog.Error("request panic", "error", v)
				writeJSON(w, 500, map[string]string{"error": "internal_error", "message": "服务内部错误"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func contextTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}
func fmtError(message string) error { return fmt.Errorf("%s", message) }
