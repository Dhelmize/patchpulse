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
import { ToolCall } from './tools.js';

export interface LLMResponse {
  content: string;
  toolCalls?: ToolCall[];
}

export function parseToolCalls(response: string): ToolCall[] {
  const toolCalls: ToolCall[] = [];

  // Try to parse as JSON array
  try {
    const parsed = JSON.parse(response);
    if (Array.isArray(parsed)) {
      return parsed;
    }
  } catch {
    // Not JSON, continue with other parsing methods
  }

  // Try to extract JSON from markdown code blocks
  const jsonMatch = response.match(/```(?:json)?\s*([\s\S]*?)```/);
  if (jsonMatch) {
    try {
      const parsed = JSON.parse(jsonMatch[1]);
      if (Array.isArray(parsed)) {
        return parsed;
      }
    } catch {
      // Invalid JSON in code block
    }
  }

  // Try to parse individual tool calls in format: tool_name(arg1="value1", arg2="value2")
  const toolRegex = /(\w+)\s*\(([^)]*)\)/g;
  let match;
  while ((match = toolRegex.exec(response)) !== null) {
    const name = match[1];
    const argsStr = match[2];

    const args: Record<string, any> = {};
    const argRegex = /(\w+)=["']([^"']*)["']/g;
    let argMatch;
    while ((argMatch = argRegex.exec(argsStr)) !== null) {
      args[argMatch[1]] = argMatch[2];
    }

    if (Object.keys(args).length > 0) {
      toolCalls.push({ name, arguments: args });
    }
  }

  return toolCalls;
}

export function isToolCallComplete(response: string): boolean {
  // Check if response contains a clear completion indicator
  return response.includes('DONE') ||
         response.includes('COMPLETE') ||
         response.includes('FINISHED') ||
         response.toLowerCase().includes('no further changes needed');
}
