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
export interface AdapterConfig {
  [key: string]: any;
}

export abstract class BaseAdapter {
  protected config: AdapterConfig;

  constructor(config: AdapterConfig) {
    this.config = config;
  }

  abstract initialize(): Promise<void>;

  abstract readFile(path: string): Promise<string>;

  abstract searchFiles(query: string, filePattern?: string): Promise<string[]>;

  abstract updateFile(path: string, content: string, commitMessage: string): Promise<void>;

  abstract commitChanges(message: string): Promise<string>;

  abstract getDiffUrl(): string;

  abstract getType(): string;
}
