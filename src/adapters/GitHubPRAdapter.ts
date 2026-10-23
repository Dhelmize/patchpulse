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
import { Octokit } from '@octokit/rest';
import { BaseAdapter, AdapterConfig } from './BaseAdapter.js';
import { getConfig } from '../config/env.js';
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

interface GitHubConfig extends AdapterConfig {
  owner: string;
  repo: string;
  branch?: string;
  workingDir?: string;
}

export class GitHubPRAdapter extends BaseAdapter {
  private octokit: Octokit;
  private owner: string;
  private repo: string;
  private branch: string;
  private workingDir: string;
  private currentBranch: string;
  private stagedFiles: Set<string> = new Set();

  constructor(config: GitHubConfig) {
    super(config);
    this.owner = config.owner;
    this.repo = config.repo;
    this.branch = config.branch || getConfig().GITHUB_DEFAULT_BRANCH;
    this.workingDir = config.workingDir || path.join(process.cwd(), 'repos', `${this.owner}-${this.repo}`);

    const token = config.token || getConfig().GITHUB_TOKEN;
    if (!token) {
      throw new Error('GitHub token is required for GitHubPRAdapter');
    }

    this.octokit = new Octokit({ auth: token });
    this.currentBranch = `patchpulse-${Date.now()}`;
  }

  async initialize(): Promise<void> {
    // Create working directory if it doesn't exist
    if (!fs.existsSync(this.workingDir)) {
      fs.mkdirSync(this.workingDir, { recursive: true });
    }

    // Clone or fetch the repository
    const repoUrl = `https://github.com/${this.owner}/${this.repo}.git`;

    if (fs.existsSync(path.join(this.workingDir, '.git'))) {
      // Repository exists, fetch latest
      execSync('git fetch origin', { cwd: this.workingDir });
      execSync(`git checkout ${this.branch}`, { cwd: this.workingDir });
      execSync(`git pull origin ${this.branch}`, { cwd: this.workingDir });
    } else {
      // Clone the repository
      execSync(`git clone ${repoUrl} .`, { cwd: this.workingDir });
    }

    // Create a new branch for changes
    this.currentBranch = `patchpulse-${Date.now()}`;
    execSync(`git checkout -b ${this.currentBranch}`, { cwd: this.workingDir });
  }

  async readFile(filePath: string): Promise<string> {
    const fullPath = path.join(this.workingDir, filePath);
    if (!fs.existsSync(fullPath)) {
      throw new Error(`File not found: ${filePath}`);
    }
    return fs.readFileSync(fullPath, 'utf-8');
  }

  async searchFiles(query: string, filePattern?: string): Promise<string[]> {
    // Use git grep to search for files
    try {
      const pattern = filePattern || '*';
      const grepCmd = `git grep -l "${query}" -- ${pattern}`;
      const result = execSync(grepCmd, { cwd: this.workingDir, encoding: 'utf-8' });
      return result.split('\n').filter(f => f.length > 0);
    } catch (error) {
      // If grep fails, try find
      try {
        const findCmd = filePattern
          ? `find . -name "${filePattern}" -type f`
          : `find . -type f`;
        const result = execSync(findCmd, { cwd: this.workingDir, encoding: 'utf-8' });
        const files = result.split('\n').filter(f => f.length > 0);
        // Filter files that contain the query
        const matchingFiles: string[] = [];
        for (const file of files) {
          try {
            const content = fs.readFileSync(path.join(this.workingDir, file), 'utf-8');
            if (content.toLowerCase().includes(query.toLowerCase())) {
              matchingFiles.push(file);
            }
          } catch {
            // Skip files that can't be read
          }
        }
        return matchingFiles;
      } catch {
        return [];
      }
    }
  }

  async updateFile(filePath: string, content: string, _commitMessage: string): Promise<void> {
    const fullPath = path.join(this.workingDir, filePath);
    const dir = path.dirname(fullPath);

    // Create directory if it doesn't exist
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    // Write the file
    fs.writeFileSync(fullPath, content, 'utf-8');

    // Stage the file
    execSync(`git add "${filePath}"`, { cwd: this.workingDir });
    this.stagedFiles.add(filePath);
  }

  async commitChanges(message: string): Promise<string> {
    if (this.stagedFiles.size === 0) {
      throw new Error('No files staged for commit');
    }

    // Commit the changes
    execSync(`git commit -m "${message}"`, { cwd: this.workingDir });

    // Push the branch
    execSync(`git push -u origin ${this.currentBranch}`, { cwd: this.workingDir });

    // Create a pull request
    const pr = await this.octokit.rest.pulls.create({
      owner: this.owner,
      repo: this.repo,
      title: message,
      head: this.currentBranch,
      base: this.branch,
      body: `Automated update by PatchPulse\n\n${message}`,
    });

    return pr.data.html_url;
  }

  getDiffUrl(): string {
    return `https://github.com/${this.owner}/${this.repo}/compare/${this.branch}...${this.currentBranch}`;
  }

  getType(): string {
    return 'github-pr';
  }
}
