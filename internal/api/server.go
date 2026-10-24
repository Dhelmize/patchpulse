/**
 * Copyright (C) 2024 PatchPulse Contributors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"patchpulse"
	"patchpulse/internal/config"
	"patchpulse/internal/db"
	"patchpulse/internal/watcher"
)

type Server struct {
	cfg *config.Config
	mux *http.ServeMux
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		mux: http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	// Static embedded files
	subFS, err := fs.Sub(patchpulse.PublicFS, "public")
	if err == nil {
		fileServer := http.FileServer(http.FS(subFS))
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	// Health check
	s.mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Global Settings routes
	s.mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			dbSettings, err := db.GetAllSettings()
			if err != nil {
				jsonError(w, http.StatusInternalServerError, err.Error())
				return
			}
			resolved := s.cfg.Resolve(dbSettings)
			response := map[string]string{
				"LLM_PROVIDER":        resolved.LLMProvider,
				"LLM_API_KEY":         resolved.LLMAPIKey,
				"LLM_MODEL":           resolved.LLMModel,
				"LLM_BASE_URL":        resolved.LLMBaseURL,
				"GITHUB_TOKEN":        resolved.GitHubToken,
				"DISCORD_BOT_TOKEN":   resolved.DiscordBotToken,
				"DISCORD_WEBHOOK_URL": resolved.DiscordWebhookURL,
			}
			jsonResponse(w, http.StatusOK, response)

		case http.MethodPost:
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonError(w, http.StatusBadRequest, "invalid json payload")
				return
			}

			if err := db.SetSettings(req); err != nil {
				jsonError(w, http.StatusInternalServerError, err.Error())
				return
			}

			s.cfg = s.cfg.Resolve(req)

			jsonResponse(w, http.StatusOK, map[string]string{
				"status":  "success",
				"message": "Settings saved successfully",
			})

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// Targets collection
	s.mux.HandleFunc("/api/targets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			targets, err := db.GetAllTargets()
			if err != nil {
				jsonError(w, http.StatusInternalServerError, err.Error())
				return
			}
			jsonResponse(w, http.StatusOK, targets)

		case http.MethodPost:
			var req struct {
				Name          string      `json:"name"`
				URL           string      `json:"url"`
				SourceType    string      `json:"source_type"`
				SourceConfig  interface{} `json:"source_config"`
				AdapterType   string      `json:"adapter_type"`
				AdapterConfig interface{} `json:"adapter_config"`
				CheckInterval int         `json:"check_interval"`
				Enabled       *int        `json:"enabled"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonError(w, http.StatusBadRequest, "invalid request body")
				return
			}

			if req.Name == "" || (req.URL == "" && req.SourceConfig == nil) {
				jsonError(w, http.StatusBadRequest, "name and url/source_config are required")
				return
			}

			srcType := req.SourceType
			if srcType == "" {
				srcType = "web"
			}

			srcConfigStr := "{}"
			if req.SourceConfig != nil {
				if s, ok := req.SourceConfig.(string); ok && s != "" {
					srcConfigStr = s
				} else {
					b, _ := json.Marshal(req.SourceConfig)
					srcConfigStr = string(b)
				}
			} else if req.URL != "" {
				srcConfigStr = fmt.Sprintf(`{"url":%q}`, req.URL)
			}

			adpType := req.AdapterType
			if adpType == "" {
				adpType = "github-pr"
			}

			adpConfigStr := "{}"
			if req.AdapterConfig != nil {
				if s, ok := req.AdapterConfig.(string); ok && s != "" {
					adpConfigStr = s
				} else {
					b, _ := json.Marshal(req.AdapterConfig)
					adpConfigStr = string(b)
				}
			}

			interval := req.CheckInterval
			if interval <= 0 {
				interval = 3600
			}

			enabled := 1
			if req.Enabled != nil {
				enabled = *req.Enabled
			}

			target := &db.Target{
				Name:          req.Name,
				URL:           req.URL,
				SourceType:    srcType,
				SourceConfig:  srcConfigStr,
				AdapterType:   adpType,
				AdapterConfig: adpConfigStr,
				CheckInterval: interval,
				Enabled:       enabled,
			}

			id, err := db.CreateTarget(target)
			if err != nil {
				jsonError(w, http.StatusInternalServerError, err.Error())
				return
			}

			created, err := db.GetTarget(id)
			if err == nil && watcher.GlobalScheduler != nil {
				watcher.GlobalScheduler.ScheduleTarget(*created)
			}

			jsonResponse(w, http.StatusCreated, created)

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// Target item & actions
	s.mux.HandleFunc("/api/targets/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/targets/")
		parts := strings.Split(path, "/")

		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid target ID")
			return
		}

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				t, err := db.GetTarget(id)
				if err != nil {
					jsonError(w, http.StatusNotFound, "target not found")
					return
				}
				jsonResponse(w, http.StatusOK, t)

			case http.MethodPut:
				var updates map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
					jsonError(w, http.StatusBadRequest, "invalid json")
					return
				}

				if err := db.UpdateTarget(id, updates); err != nil {
					jsonError(w, http.StatusInternalServerError, err.Error())
					return
				}

				updated, err := db.GetTarget(id)
				if err == nil && watcher.GlobalScheduler != nil {
					watcher.GlobalScheduler.ScheduleTarget(*updated)
				}
				jsonResponse(w, http.StatusOK, updated)

			case http.MethodDelete:
				if err := db.DeleteTarget(id); err != nil {
					jsonError(w, http.StatusInternalServerError, err.Error())
					return
				}
				if watcher.GlobalScheduler != nil {
					watcher.GlobalScheduler.UnscheduleTarget(id)
				}
				w.WriteHeader(http.StatusNoContent)

			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// Subroutes: /enable, /disable
		action := parts[1]
		if r.Method == http.MethodPost {
			switch action {
			case "enable":
				_ = db.UpdateTarget(id, map[string]interface{}{"enabled": 1})
				t, err := db.GetTarget(id)
				if err != nil {
					jsonError(w, http.StatusNotFound, "target not found")
					return
				}
				if watcher.GlobalScheduler != nil {
					watcher.GlobalScheduler.ScheduleTarget(*t)
				}
				jsonResponse(w, http.StatusOK, t)

			case "disable":
				_ = db.UpdateTarget(id, map[string]interface{}{"enabled": 0})
				t, err := db.GetTarget(id)
				if err != nil {
					jsonError(w, http.StatusNotFound, "target not found")
					return
				}
				if watcher.GlobalScheduler != nil {
					watcher.GlobalScheduler.UnscheduleTarget(id)
				}
				jsonResponse(w, http.StatusOK, t)

			default:
				http.NotFound(w, r)
			}
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// Trigger immediate scan
	s.mux.HandleFunc("/api/triggers/scan/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/triggers/scan/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid target ID")
			return
		}

		changed, err := watcher.CheckTarget(r.Context(), id, s.cfg)
		if err != nil {
			jsonResponse(w, http.StatusOK, map[string]interface{}{
				"target_id": id,
				"changed":   false,
				"error":     err.Error(),
			})
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"target_id": id,
			"changed":   changed,
		})
	})

	// Patch history
	s.mux.HandleFunc("/api/logs/patches", func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
				limit = l
			}
		}
		patches, err := db.GetRecentPatchHistory(limit)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, patches)
	})

	s.mux.HandleFunc("/api/logs/patches/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/api/logs/patches/")
		targetID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid target ID")
			return
		}
		patches, err := db.GetPatchHistoryByTarget(targetID, 10)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, patches)
	})

	// Agent logs
	s.mux.HandleFunc("/api/logs/agent", func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
				limit = l
			}
		}
		logs, err := db.GetRecentAgentLogs(limit)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, logs)
	})

	s.mux.HandleFunc("/api/logs/agent/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/api/logs/agent/")
		targetID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid target ID")
			return
		}
		logs, err := db.GetAgentLogsByTarget(targetID, 20)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, logs)
	})
}

func jsonResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, statusCode int, message string) {
	jsonResponse(w, statusCode, map[string]string{"error": message})
}

