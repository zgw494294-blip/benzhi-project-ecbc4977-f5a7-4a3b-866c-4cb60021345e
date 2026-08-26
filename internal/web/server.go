package web

import (
	"bladeready/internal/application"
	"bladeready/internal/domain"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed assets/*
var assets embed.FS

type Server struct {
	app *application.Service
	mux *http.ServeMux
}

func New(app *application.Service) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) Handler() http.Handler { return requestLog(recoverer(s.mux)) }

func (s *Server) routes() {
	static, _ := fs.Sub(assets, "assets")
	s.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(static))))
	s.mux.HandleFunc("GET /", s.WorkbenchHandler)
	s.mux.HandleFunc("GET /api/health", s.HealthHandler)
	s.mux.HandleFunc("GET /api/tasks", s.ListTasksHandler)
	s.mux.HandleFunc("POST /api/tasks", s.CreateTaskHandler)
	s.mux.HandleFunc("GET /api/tasks/{id}", s.GetTaskHandler)
	s.mux.HandleFunc("GET /api/tasks/{id}/risk", s.RiskHandler)
	s.mux.HandleFunc("POST /api/tasks/{id}/zones", s.SetZonesHandler)
	s.mux.HandleFunc("POST /api/tasks/{id}/observations", s.AddObservationHandler)
	s.mux.HandleFunc("POST /api/tasks/{id}/assess", s.AssessHandler)
	s.mux.HandleFunc("POST /api/tasks/{id}/repair-plan", s.RepairPlanHandler)
	s.mux.HandleFunc("POST /api/tasks/{id}/retests", s.RetestsHandler)
	s.mux.HandleFunc("POST /api/tasks/{id}/deviations/{deviationID}", s.CloseDeviationHandler)
	s.mux.HandleFunc("POST /api/tasks/{id}/review", s.PrepareReviewHandler)
	s.mux.HandleFunc("POST /api/tasks/{id}/release", s.ReleaseHandler)
	s.mux.HandleFunc("GET /api/tasks/{id}/audit", s.AuditHandler)
	s.mux.HandleFunc("GET /api/tasks/{id}/credential", s.CredentialHandler)
}

func decode(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("请求 JSON 无效: %w", err)
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "validation_error"
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrVersionConflict):
		status, code = http.StatusConflict, "version_conflict"
	case errors.Is(err, domain.ErrInvalidTransition), errors.Is(err, domain.ErrFrozen), errors.Is(err, domain.ErrCredentialIssued):
		status, code = http.StatusConflict, "state_conflict"
	}
	writeJSON(w, status, map[string]any{"error": code, "message": err.Error()})
}
func actor(r *http.Request, fallback string) string {
	v := strings.TrimSpace(r.Header.Get("X-Operator"))
	if v == "" {
		return fallback
	}
	return v
}
func idem(r *http.Request, body string) string {
	v := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if v != "" {
		return v
	}
	return body
}
