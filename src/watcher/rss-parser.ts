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
import Parser from 'rss-parser';

const parser = new Parser({
  timeout: 10000,
  customFields: {
    item: ['pubDate', 'content:encoded', 'description'],
  },
});

export interface RSSFeedItem {
  title: string;
  link: string;
  pubDate?: string;
  content?: string;
  contentSnippet?: string;
  description?: string;
}

export async function parseRSSFeed(url: string): Promise<RSSFeedItem[]> {
  try {
    const feed = await parser.parseURL(url);
    return feed.items.map(item => ({
      title: item.title || '',
      link: item.link || '',
      pubDate: item.pubDate,
      content: item['content:encoded'] || item.content,
      contentSnippet: item.contentSnippet || item.description,
    }));
  } catch (error) {
    throw new Error(`Failed to parse RSS feed: ${(error as Error).message}`);
  }
}

export function isRSSFeed(url: string): boolean {
  return url.endsWith('.rss') ||
         url.endsWith('.xml') ||
         url.includes('/feed') ||
         url.includes('/rss');
}
