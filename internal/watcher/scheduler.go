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
	"log"
	"sync"
	"time"

	"patchpulse/internal/config"
	"patchpulse/internal/db"
)

type Scheduler struct {
	cfg     *config.Config
	tasks   map[int64]chan struct{}
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

var GlobalScheduler *Scheduler

func InitScheduler(cfg *config.Config) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	GlobalScheduler = &Scheduler{
		cfg:    cfg,
		tasks:  make(map[int64]chan struct{}),
		ctx:    ctx,
		cancel: cancel,
	}
	return GlobalScheduler
}

func (s *Scheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	targets, err := db.GetEnabledTargets()
	if err != nil {
		return err
	}

	for _, t := range targets {
		s.ScheduleTarget(t)
	}

	log.Printf("[Scheduler] Started watching %d targets", len(targets))
	return nil
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	for id, ch := range s.tasks {
		close(ch)
		delete(s.tasks, id)
	}

	s.cancel()
	s.running = false
	log.Println("[Scheduler] Stopped all target watchers")
}

func (s *Scheduler) ScheduleTarget(target db.Target) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing worker if already registered
	if ch, exists := s.tasks[target.ID]; exists {
		close(ch)
		delete(s.tasks, target.ID)
	}

	if target.Enabled == 0 {
		return
	}

	interval := time.Duration(target.CheckInterval) * time.Second
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}

	stopCh := make(chan struct{})
	s.tasks[target.ID] = stopCh

	go func(t db.Target, stop chan struct{}) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
				_, _ = CheckTarget(ctx, t.ID, s.cfg)
				cancel()
			}
		}
	}(target, stopCh)
}

func (s *Scheduler) UnscheduleTarget(targetID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, exists := s.tasks[targetID]; exists {
		close(ch)
		delete(s.tasks, targetID)
	}
}

