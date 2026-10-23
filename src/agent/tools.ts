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
export interface ToolCall {
  name: string;
  arguments: Record<string, any>;
}

export interface ToolDefinition {
  name: string;
  description: string;
  parameters: {
    type: string;
    properties: Record<string, any>;
    required?: string[];
  };
}

export const AVAILABLE_TOOLS: ToolDefinition[] = [
  {
    name: 'read_file',
    description: 'Read the content of a file from the target repository or wiki',
    parameters: {
      type: 'object',
      properties: {
        path: {
          type: 'string',
          description: 'The file path to read (e.g., "docs/game-stats.md" or "data/characters.json")',
        },
      },
      required: ['path'],
    },
  },
  {
    name: 'search_repo',
    description: 'Search for files in the repository that match a query or pattern',
    parameters: {
      type: 'object',
      properties: {
        query: {
          type: 'string',
          description: 'Search query to find relevant files (e.g., "character stats" or "damage values")',
        },
        file_pattern: {
          type: 'string',
          description: 'Optional file pattern to filter (e.g., "*.md" or "*.json")',
        },
      },
      required: ['query'],
    },
  },
  {
    name: 'patch_file',
    description: 'Update a file with new content based on the patch notes',
    parameters: {
      type: 'object',
      properties: {
        path: {
          type: 'string',
          description: 'The file path to update',
        },
        content: {
          type: 'string',
          description: 'The new content for the file',
        },
        commit_message: {
          type: 'string',
          description: 'A descriptive commit message explaining the changes',
        },
      },
      required: ['path', 'content', 'commit_message'],
    },
  },
  {
    name: 'create_pull_request',
    description: 'Create a pull request with all the committed changes',
    parameters: {
      type: 'object',
      properties: {
        title: {
          type: 'string',
          description: 'The title of the pull request',
        },
        body: {
          type: 'string',
          description: 'The body/description of the pull request',
        },
      },
      required: ['title', 'body'],
    },
  },
];

export function getToolSchema(): ToolDefinition[] {
  return AVAILABLE_TOOLS;
}
