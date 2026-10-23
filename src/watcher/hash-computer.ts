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
import crypto from 'crypto';

export function computeSHA256(content: string): string {
  return crypto.createHash('sha256').update(content, 'utf8').digest('hex');
}

export function computeHash(content: string, algorithm: 'sha256' | 'md5' = 'sha256'): string {
  if (algorithm === 'md5') {
    return crypto.createHash('md5').update(content, 'utf8').digest('hex');
  }
  return computeSHA256(content);
}
