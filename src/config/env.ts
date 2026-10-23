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
import dotenv from 'dotenv';
import { z } from 'zod';

// Load environment variables
dotenv.config();

const envSchema = z.object({
  // Server configuration
  PORT: z.string().default('3000'),
  HOST: z.string().default('0.0.0.0'),
  DB_DIR: z.string().optional(),

  // LLM configuration
  LLM_PROVIDER: z.enum(['gemini', 'openai', 'openai-compatible']).default('openai'),
  LLM_API_KEY: z.string().min(1, 'LLM_API_KEY is required'),
  LLM_MODEL: z.string().default('gpt-4'),
  LLM_BASE_URL: z.string().optional(), // For OpenAI-compatible endpoints

  // GitHub configuration (optional)
  GITHUB_TOKEN: z.string().optional(),
  GITHUB_DEFAULT_BRANCH: z.string().default('main'),

  // MediaWiki configuration (optional)
  MEDIAWIKI_API_URL: z.string().optional(),
  MEDIAWIKI_USERNAME: z.string().optional(),
  MEDIAWIKI_PASSWORD: z.string().optional(),
  MEDIAWIKI_API_TOKEN: z.string().optional(),

  // Discord configuration (optional)
  DISCORD_WEBHOOK_URL: z.string().optional(),

  // Scraping configuration
  USER_AGENT: z.string().default('PatchPulse/1.0'),
  REQUEST_TIMEOUT: z.string().default('30000'),
  MAX_RETRIES: z.string().default('3'),
});

export type Env = z.infer<typeof envSchema>;

function validateEnv(): Env {
  try {
    return envSchema.parse(process.env);
  } catch (error) {
    if (error instanceof z.ZodError) {
      const missingVars = error.errors
        .filter(e => e.code === 'invalid_type')
        .map(e => e.path.join('.'))
        .join(', ');
      throw new Error(`Missing or invalid environment variables: ${missingVars}`);
    }
    throw error;
  }
}

export const config = validateEnv();

export function getConfig(): Env {
  return config;
}
