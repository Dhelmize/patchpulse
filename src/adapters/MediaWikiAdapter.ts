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
import { BaseAdapter, AdapterConfig } from './BaseAdapter.js';
import { getConfig } from '../config/env.js';

interface MediaWikiConfig extends AdapterConfig {
  apiUrl: string;
  username?: string;
  password?: string;
  apiToken?: string;
}

export class MediaWikiAdapter extends BaseAdapter {
  private apiUrl: string;
  private username?: string;
  private password?: string;
  private apiToken?: string;
  private authToken?: string;
  private stagedEdits: Map<string, { content: string; summary: string }> = new Map();

  constructor(config: MediaWikiConfig) {
    super(config);
    this.apiUrl = config.apiUrl || getConfig().MEDIAWIKI_API_URL || '';
    this.username = config.username || getConfig().MEDIAWIKI_USERNAME;
    this.password = config.password || getConfig().MEDIAWIKI_PASSWORD;
    this.apiToken = config.apiToken || getConfig().MEDIAWIKI_API_TOKEN;

    if (!this.apiUrl) {
      throw new Error('MediaWiki API URL is required');
    }
  }

  async initialize(): Promise<void> {
    // Authenticate if credentials are provided
    if (this.username && this.password) {
      await this.authenticate();
    } else if (this.apiToken) {
      this.authToken = this.apiToken;
    }
  }

  private async authenticate(): Promise<void> {
    try {
      // Get login token
      const tokenResponse = await fetch(
        `${this.apiUrl}?action=query&meta=tokens&type=login&format=json`
      );
      const tokenData = await tokenResponse.json() as any;
      const loginToken = tokenData.query.tokens.logintoken;

      // Login
      const loginResponse = await fetch(this.apiUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          action: 'login',
          lgname: this.username!,
          lgpassword: this.password!,
          lgtoken: loginToken,
          format: 'json',
        }),
      });

      const loginData = await loginResponse.json() as any;
      if (loginData.login?.result !== 'Success') {
        throw new Error('MediaWiki login failed');
      }

      // Get CSRF token for edits
      const csrfResponse = await fetch(
        `${this.apiUrl}?action=query&meta=tokens&format=json`
      );
      const csrfData = await csrfResponse.json() as any;
      this.authToken = csrfData.query.tokens.csrftoken;
    } catch (error) {
      throw new Error(`MediaWiki authentication failed: ${(error as Error).message}`);
    }
  }

  async readFile(pageTitle: string): Promise<string> {
    const url = `${this.apiUrl}?action=query&prop=revisions&rvprop=content&rvslots=main&titles=${encodeURIComponent(pageTitle)}&format=json`;
    const response = await fetch(url);
    const data = await response.json() as any;

    const pages = data.query.pages;
    const pageId = Object.keys(pages)[0];
    const page = pages[pageId];

    if (pageId === '-1') {
      throw new Error(`Page not found: ${pageTitle}`);
    }

    return page.revisions[0].slots.main['*'];
  }

  async searchFiles(query: string, _filePattern?: string): Promise<string[]> {
    const url = `${this.apiUrl}?action=query&list=search&srsearch=${encodeURIComponent(query)}&srlimit=20&format=json`;
    const response = await fetch(url);
    const data = await response.json() as any;

    if (data.error) {
      throw new Error(`Search failed: ${data.error.info}`);
    }

    return data.query.search.map((result: any) => result.title);
  }

  async updateFile(pageTitle: string, content: string, summary: string): Promise<void> {
    // Stage the edit for later submission
    this.stagedEdits.set(pageTitle, { content, summary });
  }

  async commitChanges(message: string): Promise<string> {
    if (this.stagedEdits.size === 0) {
      throw new Error('No edits staged for commit');
    }

    const editUrls: string[] = [];

    for (const [pageTitle, edit] of this.stagedEdits.entries()) {
      const params = new URLSearchParams({
        action: 'edit',
        title: pageTitle,
        text: edit.content,
        summary: `${message}: ${edit.summary}`,
        format: 'json',
      });

      if (this.authToken) {
        params.append('token', this.authToken);
      }

      const response = await fetch(this.apiUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: params,
      });

      const data = await response.json() as any;

      if (data.error) {
        throw new Error(`Edit failed for ${pageTitle}: ${data.error.info}`);
      }

      // Get the diff URL
      const diffUrl = `${this.apiUrl.replace('api.php', 'index.php')}?title=${encodeURIComponent(pageTitle)}&diff=${data.edit.newrevid}&oldid=${data.edit.oldrevid}`;
      editUrls.push(diffUrl);
    }

    this.stagedEdits.clear();

    // Return the first edit URL (or could return all)
    return editUrls[0] || '';
  }

  getDiffUrl(): string {
    if (this.stagedEdits.size === 0) {
      return '';
    }
    const firstPage = this.stagedEdits.keys().next().value;
    if (!firstPage) {
      return '';
    }
    return `${this.apiUrl.replace('api.php', 'index.php')}?title=${encodeURIComponent(firstPage)}`;
  }

  getType(): string {
    return 'mediawiki';
  }
}
