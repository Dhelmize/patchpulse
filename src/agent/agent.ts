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
import { getConfig } from '../config/env.js';
import { GeminiProvider } from './gemini-provider.js';
import { OpenAIProvider } from './openai-provider.js';
import { getToolSchema, ToolCall } from './tools.js';
import { SYSTEM_PROMPT, getUserPrompt } from './prompts.js';
import { BaseAdapter } from '../adapters/BaseAdapter.js';

export class Agent {
  private provider: GeminiProvider | OpenAIProvider;
  private adapter: BaseAdapter;

  constructor(adapter: BaseAdapter) {
    const config = getConfig();

    if (config.LLM_PROVIDER === 'gemini') {
      this.provider = new GeminiProvider(config.LLM_API_KEY, config.LLM_MODEL);
    } else {
      this.provider = new OpenAIProvider(
        config.LLM_API_KEY,
        config.LLM_MODEL,
        config.LLM_BASE_URL
      );
    }

    this.adapter = adapter;
  }

  async processPatch(patchNotes: string, targetInfo: string): Promise<void> {
    const tools = getToolSchema();
    const systemPrompt = SYSTEM_PROMPT;
    const userPrompt = getUserPrompt(patchNotes, targetInfo);

    let conversationHistory: Array<{ role: string; content: string }> = [
      { role: 'system', content: systemPrompt },
      { role: 'user', content: userPrompt },
    ];

    let maxIterations = 10;
    let iteration = 0;

    while (iteration < maxIterations) {
      iteration++;

      const response = await this.provider.generate(
        systemPrompt,
        this.buildConversationPrompt(conversationHistory),
        tools
      );

      conversationHistory.push({
        role: 'assistant',
        content: response.content,
      });

      // Process tool calls
      if (response.toolCalls && response.toolCalls.length > 0) {
        for (const toolCall of response.toolCalls) {
          const toolResult = await this.executeTool(toolCall);
          conversationHistory.push({
            role: 'user',
            content: `Tool ${toolCall.name} result: ${JSON.stringify(toolResult)}`,
          });
        }
      } else {
        // No more tool calls, agent is done
        break;
      }
    }

    if (iteration >= maxIterations) {
      console.warn('Agent reached maximum iterations without completion');
    }
  }

  private buildConversationPrompt(history: Array<{ role: string; content: string }>): string {
    return history
      .map(msg => `${msg.role}: ${msg.content}`)
      .join('\n\n');
  }

  private async executeTool(toolCall: ToolCall): Promise<any> {
    const { name, arguments: args } = toolCall;

    switch (name) {
      case 'read_file':
        return await this.adapter.readFile(args.path);
      case 'search_repo':
        return await this.adapter.searchFiles(args.query, args.file_pattern);
      case 'patch_file':
        return await this.adapter.updateFile(args.path, args.content, args.commit_message);
      case 'create_pull_request':
        return await this.adapter.commitChanges(args.title);
      default:
        throw new Error(`Unknown tool: ${name}`);
    }
  }
}
