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
import { getTarget, updateTarget, createPatchHistory, getPatchHistory, PatchHistory, createAgentLog } from '../db/queries.js';
import { fetchWithRetry } from './fetcher.js';
import { extractMainContent, cleanHtml } from './html-parser.js';
import { htmlToMarkdown } from './markdown-converter.js';
import { parseRSSFeed, isRSSFeed } from './rss-parser.js';
import { computeSHA256 } from './hash-computer.js';
import { createAdapter } from '../adapters/index.js';
import { Agent } from '../agent/agent.js';

export interface DetectionResult {
  changed: boolean;
  patchHistory?: PatchHistory;
  error?: string;
}

export async function diffDetector(targetId: number): Promise<DetectionResult> {
  try {
    const target = getTarget(targetId);
    if (!target) {
      return { changed: false, error: 'Target not found' };
    }

    // Fetch content
    let content: string;
    if (isRSSFeed(target.url)) {
      const items = await parseRSSFeed(target.url);
      if (items.length === 0) {
        return { changed: false, error: 'No items in RSS feed' };
      }
      // Use the most recent item's content
      content = items[0].content || items[0].contentSnippet || items[0].title;
    } else {
      const html = await fetchWithRetry(target.url);
      const mainContent = extractMainContent(html);
      content = cleanHtml(mainContent);
    }

    // Compute hash
    const newHash = computeSHA256(content);

    // Check if changed
    if (target.last_hash === newHash) {
      // Update last_check timestamp even if no change
      updateTarget(target.id!, { last_check: Math.floor(Date.now() / 1000) });
      return { changed: false };
    }

    // Convert to markdown
    const markdown = htmlToMarkdown(content);

    // Create patch history entry
    const patchHistory: Omit<PatchHistory, 'id' | 'detected_at'> = {
      target_id: target.id!,
      content_hash: newHash,
      raw_content: content,
      markdown_content: markdown,
    };

    const patchId = createPatchHistory(patchHistory);

    // Update target
    updateTarget(target.id!, {
      last_hash: newHash,
      last_check: Math.floor(Date.now() / 1000),
    });

    const createdPatch = getPatchHistory(patchId);

    return {
      changed: true,
      patchHistory: createdPatch,
    };
  } catch (error) {
    return {
      changed: false,
      error: (error as Error).message,
    };
  }
}

export async function checkTarget(targetId: number): Promise<void> {
  const result = await diffDetector(targetId);

  if (result.error) {
    console.error(`Error checking target ${targetId}:`, result.error);
    createAgentLog({
      target_id: targetId,
      action_type: 'check_target',
      status: 'error',
      result: result.error,
    });
  } else if (result.changed) {
    console.log(`Patch detected for target ${targetId}:`, result.patchHistory?.id);

    // Trigger agent processing
    await processPatchWithAgent(targetId, result.patchHistory!);
  } else {
    console.log(`No changes detected for target ${targetId}`);
  }
}

async function processPatchWithAgent(targetId: number, patchHistory: PatchHistory): Promise<void> {
  try {
    const target = getTarget(targetId);
    if (!target) {
      throw new Error('Target not found');
    }

    // Parse adapter config
    const adapterConfig = typeof target.adapter_config === 'string'
      ? JSON.parse(target.adapter_config)
      : target.adapter_config;

    // Create adapter instance
    const adapter = createAdapter(target.adapter_type, adapterConfig);
    await adapter.initialize();

    // Create agent instance
    const agent = new Agent(adapter);

    // Log agent start
    createAgentLog({
      target_id: targetId,
      patch_history_id: patchHistory.id,
      action_type: 'agent_start',
      status: 'success',
      result: 'Agent processing started',
    });

    // Process the patch with the agent
    await agent.processPatch(patchHistory.markdown_content, JSON.stringify({
      name: target.name,
      url: target.url,
      adapter_type: target.adapter_type,
    }));

    // Log agent completion
    createAgentLog({
      target_id: targetId,
      patch_history_id: patchHistory.id,
      action_type: 'agent_complete',
      status: 'success',
      result: 'Agent processing completed',
    });

    console.log(`Agent processing completed for target ${targetId}`);
  } catch (error) {
    console.error(`Agent processing failed for target ${targetId}:`, error);
    createAgentLog({
      target_id: targetId,
      patch_history_id: patchHistory.id,
      action_type: 'agent_error',
      status: 'error',
      result: (error as Error).message,
    });
  }
}
