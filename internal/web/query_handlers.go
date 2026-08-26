package web

import (
	"bladeready/internal/domain"
	"bladeready/internal/store"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) WorkbenchHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	raw, err := assets.ReadFile("assets/index.html")
	if err != nil {
		http.Error(w, "页面资源不可用", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(raw)
}
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextTimeout(r, 2*time.Second)
	defer cancel()
	if err := s.app.Repository().Check(ctx); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"status": "ok", "database": "ok", "time": time.Now().UTC()})
}
func (s *Server) ListTasksHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := store.TaskListFilter{Status: domain.TaskStatus(q.Get("status")), WindFarm: q.Get("wind_farm")}
	if raw := q.Get("from"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			u := t.UTC()
			filter.From = &u
		} else {
			writeError(w, domain.ValidationError{Field: "from", Message: "时间格式必须为 RFC3339"})
			return
		}
	}
	if raw := q.Get("to"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			u := t.UTC()
			filter.To = &u
		} else {
			writeError(w, domain.ValidationError{Field: "to", Message: "时间格式必须为 RFC3339"})
			return
		}
	}
	items, err := s.app.ListFiltered(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"tasks": items})
}
func (s *Server) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	b, err := s.app.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if b.Assessment != nil && (r.URL.Query().Get("level") != "" || r.URL.Query().Get("risk_level") != "" || r.URL.Query().Get("blocked") != "") {
		level := r.URL.Query().Get("level")
		if level == "" {
			level = r.URL.Query().Get("risk_level")
		}
		blockedRaw := r.URL.Query().Get("blocked")
		var blocked *bool
		if blockedRaw != "" {
			v, e := strconv.ParseBool(blockedRaw)
			if e != nil {
				writeError(w, domain.ValidationError{Field: "blocked", Message: "blocked 必须是 true 或 false"})
				return
			}
			blocked = &v
		}
		filtered := b.Assessment.Results[:0]
		for _, result := range b.Assessment.Results {
			if level != "" && result.Level != level {
				continue
			}
			if blocked != nil && result.Blocked != *blocked {
				continue
			}
			filtered = append(filtered, result)
		}
		copyAssessment := *b.Assessment
		copyAssessment.Results = filtered
		b.Assessment = &copyAssessment
	}
	writeJSON(w, 200, b)
}

func (s *Server) RiskHandler(w http.ResponseWriter, r *http.Request) {
	b, err := s.app.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if b.Assessment == nil {
		writeError(w, domain.ValidationError{Field: "assessment", Message: "尚未生成风险快照"})
		return
	}
	level := r.URL.Query().Get("level")
	if level == "" {
		level = r.URL.Query().Get("risk_level")
	}
	blockedRaw := r.URL.Query().Get("blocked")
	var blocked *bool
	if blockedRaw != "" {
		v, e := strconv.ParseBool(blockedRaw)
		if e != nil {
			writeError(w, domain.ValidationError{Field: "blocked", Message: "blocked 必须是 true 或 false"})
			return
		}
		blocked = &v
	}
	results := make([]domain.RiskResult, 0, len(b.Assessment.Results))
	for _, result := range b.Assessment.Results {
		if level != "" && result.Level != level {
			continue
		}
		if blocked != nil && result.Blocked != *blocked {
			continue
		}
		results = append(results, result)
	}
	writeJSON(w, http.StatusOK, map[string]any{"rule_version": b.Assessment.RuleVersion, "results": results, "zone_summaries": b.Assessment.ZoneSummaries})
}
func (s *Server) AuditHandler(w http.ResponseWriter, r *http.Request) {
	events, err := s.app.Audit(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"events": events})
}
func (s *Server) CredentialHandler(w http.ResponseWriter, r *http.Request) {
	b, err := s.app.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if b.Credential == nil {
		writeError(w, fmtError("尚未签发放行凭证"))
		return
	}
	writeJSON(w, 200, b.Credential)
}
