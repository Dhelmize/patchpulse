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

package adapter

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type LocalDirConfig struct {
	Directory string `json:"directory"`
}

type LocalDirAdapter struct {
	dir         string
	stagedEdits map[string]string
	diffURL     string
}

func NewLocalDirAdapter(cfg LocalDirConfig) *LocalDirAdapter {
	dir := cfg.Directory
	if dir == "" {
		dir = "."
	}
	return &LocalDirAdapter{
		dir:         dir,
		stagedEdits: make(map[string]string),
	}
}

func (a *LocalDirAdapter) Initialize(ctx context.Context) error {
	abs, err := filepath.Abs(a.dir)
	if err != nil {
		return err
	}
	a.dir = abs
	return os.MkdirAll(a.dir, 0755)
}

func (a *LocalDirAdapter) GetType() string {
	return "local-dir"
}

func (a *LocalDirAdapter) IsNotificationOnly() bool {
	return false
}

func (a *LocalDirAdapter) GetDiffURL() string {
	return a.diffURL
}

func (a *LocalDirAdapter) ReadFile(ctx context.Context, relPath string) (string, error) {
	if content, exists := a.stagedEdits[relPath]; exists {
		return content, nil
	}

	target := filepath.Join(a.dir, filepath.Clean(relPath))
	b, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (a *LocalDirAdapter) SearchRepo(ctx context.Context, query string) ([]string, error) {
	var files []string
	qLower := strings.ToLower(query)

	err := filepath.WalkDir(a.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(a.dir, path)
		if err == nil {
			relNorm := filepath.ToSlash(rel)
			if query == "" || strings.Contains(strings.ToLower(relNorm), qLower) {
				files = append(files, relNorm)
			}
		}
		return nil
	})

	return files, err
}

func (a *LocalDirAdapter) UpdateFile(ctx context.Context, path, content, commitMessage string) error {
	a.stagedEdits[path] = content
	return nil
}

func (a *LocalDirAdapter) CreatePullRequest(ctx context.Context, title, body, branch string) (string, error) {
	if len(a.stagedEdits) == 0 {
		return "", fmt.Errorf("no edits staged to apply")
	}

	for relPath, content := range a.stagedEdits {
		targetPath := filepath.Join(a.dir, filepath.Clean(relPath))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return "", err
		}
	}

	a.diffURL = a.dir
	a.stagedEdits = make(map[string]string)
	return a.diffURL, nil
}

func (a *LocalDirAdapter) SendNotification(ctx context.Context, message, diffURL string) error {
	return nil
}

