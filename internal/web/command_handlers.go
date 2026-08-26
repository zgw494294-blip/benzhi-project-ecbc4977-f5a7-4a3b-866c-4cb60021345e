package web

import (
	"bladeready/internal/application"
	"context"
	"net/http"
)

func (s *Server) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var q application.CreateTaskRequest
	if err := decode(r, &q); err != nil {
		writeError(w, err)
		return
	}
	q.CreatedBy = actor(r, q.CreatedBy)
	q.IdempotencyKey = idem(r, q.IdempotencyKey)
	b, err := s.app.CreateTask(context.WithoutCancel(r.Context()), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}
func (s *Server) SetZonesHandler(w http.ResponseWriter, r *http.Request) {
	var q application.SetZonesRequest
	if err := decode(r, &q); err != nil {
		writeError(w, err)
		return
	}
	q.Actor = actor(r, q.Actor)
	q.IdempotencyKey = idem(r, q.IdempotencyKey)
	b, err := s.app.SetZones(context.WithoutCancel(r.Context()), r.PathValue("id"), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, b)
}
func (s *Server) AddObservationHandler(w http.ResponseWriter, r *http.Request) {
	var q application.AddObservationRequest
	if err := decode(r, &q); err != nil {
		writeError(w, err)
		return
	}
	q.Actor = actor(r, q.Actor)
	q.IdempotencyKey = idem(r, q.IdempotencyKey)
	b, err := s.app.AddObservation(context.WithoutCancel(r.Context()), r.PathValue("id"), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}
func (s *Server) AssessHandler(w http.ResponseWriter, r *http.Request) {
	var q application.AssessRequest
	if err := decode(r, &q); err != nil {
		writeError(w, err)
		return
	}
	q.Actor = actor(r, q.Actor)
	q.IdempotencyKey = idem(r, q.IdempotencyKey)
	b, err := s.app.Assess(context.WithoutCancel(r.Context()), r.PathValue("id"), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, b)
}
func (s *Server) RepairPlanHandler(w http.ResponseWriter, r *http.Request) {
	var q application.FreezeRepairPlanRequest
	if err := decode(r, &q); err != nil {
		writeError(w, err)
		return
	}
	q.Actor = actor(r, q.Actor)
	q.IdempotencyKey = idem(r, q.IdempotencyKey)
	b, err := s.app.FreezeRepairPlan(context.WithoutCancel(r.Context()), r.PathValue("id"), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, b)
}
func (s *Server) RetestsHandler(w http.ResponseWriter, r *http.Request) {
	var q application.AddRetestsRequest
	if err := decode(r, &q); err != nil {
		writeError(w, err)
		return
	}
	q.Actor = actor(r, q.Actor)
	q.IdempotencyKey = idem(r, q.IdempotencyKey)
	b, err := s.app.AddRetests(context.WithoutCancel(r.Context()), r.PathValue("id"), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}
func (s *Server) CloseDeviationHandler(w http.ResponseWriter, r *http.Request) {
	var q application.CloseDeviationRequest
	if err := decode(r, &q); err != nil {
		writeError(w, err)
		return
	}
	q.Actor = actor(r, q.Actor)
	q.IdempotencyKey = idem(r, q.IdempotencyKey)
	b, err := s.app.CloseDeviation(context.WithoutCancel(r.Context()), r.PathValue("id"), r.PathValue("deviationID"), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, b)
}
func (s *Server) PrepareReviewHandler(w http.ResponseWriter, r *http.Request) {
	var q application.PrepareReviewRequest
	if err := decode(r, &q); err != nil {
		writeError(w, err)
		return
	}
	q.Actor = actor(r, q.Actor)
	q.IdempotencyKey = idem(r, q.IdempotencyKey)
	b, err := s.app.PrepareReview(context.WithoutCancel(r.Context()), r.PathValue("id"), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, b)
}
func (s *Server) ReleaseHandler(w http.ResponseWriter, r *http.Request) {
	var q application.ReleaseRequest
	if err := decode(r, &q); err != nil {
		writeError(w, err)
		return
	}
	q.Reviewer = actor(r, q.Reviewer)
	q.IdempotencyKey = idem(r, q.IdempotencyKey)
	b, err := s.app.Release(context.WithoutCancel(r.Context()), r.PathValue("id"), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, b)
}
