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
import TurndownService from 'turndown';

const turndownService = new TurndownService({
  headingStyle: 'atx',
  codeBlockStyle: 'fenced',
  emDelimiter: '*',
  strongDelimiter: '**',
  linkStyle: 'inlined',
});

// Custom rules for better conversion
turndownService.addRule('strikethrough', {
  filter: ['del', 's', 'strike'],
  replacement: (content) => `~~${content}~~`,
});

turndownService.addRule('listItem', {
  filter: 'li',
  replacement: (content, node) => {
    const parent = node.parentNode;
    if (parent?.nodeName === 'OL') {
      return `${content}\n`;
    }
    return `${content}\n`;
  },
});

turndownService.addRule('table', {
  filter: 'table',
  replacement: (content) => {
    return `\n\n${content}\n\n`;
  },
});

export function htmlToMarkdown(html: string): string {
  const markdown = turndownService.turndown(html);
  // Clean up excessive blank lines
  return markdown.replace(/\n{3,}/g, '\n\n').trim();
}
