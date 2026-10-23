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

interface DiscordConfig extends AdapterConfig {
  webhookUrl: string;
}

export class DiscordWebhookAdapter extends BaseAdapter {
  private webhookUrl: string;

  constructor(config: DiscordConfig) {
    super(config);
    this.webhookUrl = config.webhookUrl || getConfig().DISCORD_WEBHOOK_URL || '';

    if (!this.webhookUrl) {
      throw new Error('Discord webhook URL is required');
    }
  }

  async initialize(): Promise<void> {
    // No initialization needed for Discord webhook
  }

  async readFile(_path: string): Promise<string> {
    throw new Error('DiscordWebhookAdapter does not support file reading');
  }

  async searchFiles(_query: string, _filePattern?: string): Promise<string[]> {
    throw new Error('DiscordWebhookAdapter does not support file search');
  }

  async updateFile(_path: string, _content: string, _commitMessage: string): Promise<void> {
    throw new Error('DiscordWebhookAdapter does not support file updates');
  }

  async commitChanges(_message: string): Promise<string> {
    throw new Error('DiscordWebhookAdapter does not support commits');
  }

  async sendNotification(data: {
    title: string;
    description: string;
    url?: string;
    color?: number;
    fields?: Array<{ name: string; value: string; inline?: boolean }>;
  }): Promise<void> {
    const payload = {
      embeds: [
        {
          title: data.title,
          description: data.description,
          url: data.url,
          color: data.color || 0x00ff00,
          fields: data.fields || [],
          timestamp: new Date().toISOString(),
        },
      ],
    };

    const response = await fetch(this.webhookUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      throw new Error(`Discord webhook failed: ${response.statusText}`);
    }
  }

  getDiffUrl(): string {
    return '';
  }

  getType(): string {
    return 'discord-webhook';
  }
}
