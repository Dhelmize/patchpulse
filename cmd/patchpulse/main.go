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

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"patchpulse/internal/api"
	"patchpulse/internal/config"
	"patchpulse/internal/db"
	"patchpulse/internal/watcher"
)

func main() {
	cfg := config.Load()

	// Initialize SQLite DB
	if _, err := db.Init(cfg.DBDir); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "list":
			cmdList()
			return
		case "scan":
			if len(args) < 2 {
				fmt.Println("Usage: patchpulse scan <target_id>")
				os.Exit(1)
			}
			id, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				log.Fatalf("Invalid target ID: %v", err)
			}
			cmdScan(id, cfg)
			return
		case "serve":
			runServer(cfg, false)
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	// Default mode: Desktop app (starts server + opens browser)
	runServer(cfg, true)
}

func printHelp() {
	fmt.Println(`PatchPulse - Agentic Wiki & Repository Auto-Updater

Usage:
  patchpulse             Start in desktop mode (launches background server & opens browser)
  patchpulse serve       Start in headless server mode (for Docker / daemon service)
  patchpulse list        List all configured targets
  patchpulse scan <id>   Trigger an immediate scan for target with given ID
  patchpulse help        Display this help message`)
}

func cmdList() {
	targets, err := db.GetAllTargets()
	if err != nil {
		log.Fatalf("Failed to retrieve targets: %v", err)
	}

	if len(targets) == 0 {
		fmt.Println("No targets configured.")
		return
	}

	fmt.Printf("%-4s  %-30s  %-15s  %-15s  %-10s  %-8s\n", "ID", "NAME", "SOURCE", "ADAPTER", "INTERVAL", "STATUS")
	fmt.Println(string(make([]byte, 90)))
	for _, t := range targets {
		status := "Active"
		if t.Enabled == 0 {
			status = "Paused"
		}
		fmt.Printf("%-4d  %-30s  %-15s  %-15s  %-10s  %-8s\n",
			t.ID, truncate(t.Name, 30), t.SourceType, t.AdapterType, fmt.Sprintf("%ds", t.CheckInterval), status)
	}
}

func cmdScan(targetID int64, cfg *config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Printf("Scanning target #%d...\n", targetID)
	changed, err := watcher.CheckTarget(ctx, targetID, cfg)
	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	if changed {
		fmt.Printf("Patch detected for target #%d! Agent workflow triggered.\n", targetID)
	} else {
		fmt.Printf("No changes detected for target #%d.\n", targetID)
	}
}

func runServer(cfg *config.Config, openBrowser bool) {
	sched := watcher.InitScheduler(cfg)
	if err := sched.Start(); err != nil {
		log.Printf("Scheduler start warning: %v", err)
	}
	defer sched.Stop()

	server := api.NewServer(cfg)
	addr := net.JoinHostPort(cfg.Host, cfg.Port)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: server.Handler(),
	}

	go func() {
		log.Printf("PatchPulse server running on http://%s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	if openBrowser {
		dashboardURL := fmt.Sprintf("http://localhost:%s", cfg.Port)
		go func() {
			time.Sleep(300 * time.Millisecond)
			log.Printf("Opening native browser at %s", dashboardURL)
			openURL(dashboardURL)
		}()
	}

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down PatchPulse...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = httpServer.Shutdown(shutdownCtx)
	log.Println("PatchPulse stopped successfully")
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

