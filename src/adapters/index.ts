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
import { BaseAdapter } from './BaseAdapter.js';
import { GitHubPRAdapter } from './GitHubPRAdapter.js';
import { MediaWikiAdapter } from './MediaWikiAdapter.js';
import { DiscordWebhookAdapter } from './DiscordWebhookAdapter.js';

export type { AdapterConfig } from './BaseAdapter.js';

export function createAdapter(type: string, config: any): BaseAdapter {
  switch (type) {
    case 'github-pr':
      return new GitHubPRAdapter(config);
    case 'mediawiki':
      return new MediaWikiAdapter(config);
    case 'discord-webhook':
      return new DiscordWebhookAdapter(config);
    default:
      throw new Error(`Unknown adapter type: ${type}`);
  }
}

export { BaseAdapter, GitHubPRAdapter, MediaWikiAdapter, DiscordWebhookAdapter };
