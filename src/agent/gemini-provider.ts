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
import { GoogleGenAI } from '@google/genai';
import { ToolDefinition, ToolCall } from './tools.js';

export class GeminiProvider {
  private client: GoogleGenAI;
  private model: string;

  constructor(apiKey: string, model: string = 'gemini-1.5-pro') {
    this.client = new GoogleGenAI({ apiKey });
    this.model = model;
  }

  async generate(
    systemPrompt: string,
    userPrompt: string,
    _tools?: ToolDefinition[]
  ): Promise<{ content: string; toolCalls?: ToolCall[] }> {
    const response = await this.client.models.generateContent({
      model: this.model,
      contents: userPrompt,
      config: {
        systemInstruction: systemPrompt,
        temperature: 0.7,
        topK: 40,
        topP: 0.95,
        maxOutputTokens: 8192,
      },
    });

    const text = response.text || '';

    // Parse tool calls if present
    const toolCalls = this.extractToolCalls(text);

    return {
      content: text,
      toolCalls,
    };
  }

  private extractToolCalls(_text: string): ToolCall[] {
    // Gemini doesn't have native tool calling in the same way as OpenAI
    // We'll rely on the parser to extract tool calls from the text
    return [];
  }
}
