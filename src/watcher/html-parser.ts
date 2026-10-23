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
import * as cheerio from 'cheerio';

export interface ParseOptions {
  selector?: string;
  removeSelectors?: string[];
  keepLinks?: boolean;
}

export function extractMainContent(html: string, options: ParseOptions = {}): string {
  const $ = cheerio.load(html);

  // Remove unwanted elements
  const defaultRemoveSelectors = [
    'script',
    'style',
    'nav',
    'header',
    'footer',
    'aside',
    '.advertisement',
    '.ads',
    '.sidebar',
    '.menu',
    'iframe',
    'noscript',
  ];

  const removeSelectors = [...defaultRemoveSelectors, ...(options.removeSelectors || [])];
  removeSelectors.forEach(selector => {
    $(selector).remove();
  });

  // Try to find main content using common selectors
  const contentSelectors = [
    options.selector,
    'main',
    'article',
    '[role="main"]',
    '.content',
    '.post-content',
    '.entry-content',
    '#content',
    '#main',
    '.patch-notes',
    '.changelog',
  ].filter(Boolean) as string[];

  let $content: any = $;
  for (const selector of contentSelectors) {
    const found = $(selector);
    if (found.length > 0) {
      $content = found.first();
      break;
    }
  }

  // Remove any remaining script/style tags that might have been nested
  $content.find('script, style').remove();

  return $content.html() || $.html();
}

export function cleanHtml(html: string): string {
  const $ = cheerio.load(html);

  // Remove empty elements
  $('*:empty').not('br, img, input, hr').remove();

  // Normalize whitespace
  $('*').each((_: any, el: any) => {
    const $el = $(el);
    const text = $el.text();
    if (text.trim() === '') {
      $el.remove();
    }
  });

  return $.html();
}
