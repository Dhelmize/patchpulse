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

package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"patchpulse/internal/adapter"
	"patchpulse/internal/agent"
	"patchpulse/internal/config"
	"patchpulse/internal/db"
	"patchpulse/internal/source"
)

func CheckTarget(ctx context.Context, targetID int64, cfg *config.Config) (bool, error) {
	// Dynamically resolve configuration from database settings first, falling back to OS env vars
	if dbSettings, err := db.GetAllSettings(); err == nil && len(dbSettings) > 0 {
		cfg = cfg.Resolve(dbSettings)
	}

	target, err := db.GetTarget(targetID)
	if err != nil {
		return false, fmt.Errorf("target not found: %w", err)
	}

	src, err := source.NewSource(target.SourceType, target.SourceConfig, target.URL, cfg)
	if err != nil {
		errStr := err.Error()
		_, _ = db.CreateAgentLog(&db.AgentLog{
			TargetID:   &targetID,
			ActionType: "source_initialization",
			Result:     &errStr,
			Status:     "error",
		})
		return false, err
	}

	res, err := src.Fetch(ctx)
	if err != nil {
		errStr := err.Error()
		_, _ = db.CreateAgentLog(&db.AgentLog{
			TargetID:   &targetID,
			ActionType: "fetch_source",
			Result:     &errStr,
			Status:     "error",
		})
		return false, fmt.Errorf("failed to fetch from source: %w", err)
	}

	now := time.Now().Unix()

	if target.LastHash != nil && *target.LastHash == res.Hash {
		// No changes detected, update last_check
		_ = db.UpdateTarget(target.ID, map[string]interface{}{
			"last_check": now,
		})
		return false, nil
	}

	// Change detected!
	patchID, err := db.CreatePatchHistory(&db.PatchHistory{
		TargetID:        target.ID,
		ContentHash:     res.Hash,
		RawContent:      res.RawContent,
		MarkdownContent: res.MarkdownContent,
	})
	if err != nil {
		return false, fmt.Errorf("failed to save patch history: %w", err)
	}

	_ = db.UpdateTarget(target.ID, map[string]interface{}{
		"last_hash":  res.Hash,
		"last_check": now,
	})

	log.Printf("[Watcher] Patch detected for target #%d (%s)! Saved patch #%d", target.ID, target.Name, patchID)

	// Run agent processing
	go func() {
		procCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		adp, err := adapter.NewAdapter(target.AdapterType, target.AdapterConfig, cfg)
		if err != nil {
			log.Printf("[Agent] Failed to create adapter for target #%d: %v", target.ID, err)
			return
		}

		if err := adp.Initialize(procCtx); err != nil {
			log.Printf("[Agent] Failed to initialize adapter: %v", err)
			return
		}

		ag, err := agent.NewAgent(adp, cfg)
		if err != nil {
			log.Printf("[Agent] Failed to create agent: %v", err)
			return
		}

		targetInfo, _ := json.Marshal(map[string]interface{}{
			"id":           target.ID,
			"name":         target.Name,
			"source_type":  target.SourceType,
			"adapter_type": target.AdapterType,
		})

		startMsg := "Patch processing initiated"
		_, _ = db.CreateAgentLog(&db.AgentLog{
			TargetID:       &target.ID,
			PatchHistoryID: &patchID,
			ActionType:     "agent_start",
			Result:         &startMsg,
			Status:         "success",
		})

		if err := ag.ProcessPatch(procCtx, target.ID, patchID, res.MarkdownContent, string(targetInfo)); err != nil {
			log.Printf("[Agent] Error processing patch #%d: %v", patchID, err)
			errMsg := err.Error()
			_, _ = db.CreateAgentLog(&db.AgentLog{
				TargetID:       &target.ID,
				PatchHistoryID: &patchID,
				ActionType:     "agent_error",
				Result:         &errMsg,
				Status:         "error",
			})
			return
		}

		completeMsg := "Patch processing completed"
		_, _ = db.CreateAgentLog(&db.AgentLog{
			TargetID:       &target.ID,
			PatchHistoryID: &patchID,
			ActionType:     "agent_complete",
			Result:         &completeMsg,
			Status:         "success",
		})
	}()

	return true, nil
}

