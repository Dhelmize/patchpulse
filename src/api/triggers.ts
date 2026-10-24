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
import express from 'express';
import { getTarget } from '../db/queries.js';
import { diffDetector } from '../watcher/diff-detector.js';

export const triggersRouter = express.Router();

// POST /api/triggers/scan/:id - Trigger immediate scan for a target
triggersRouter.post('/scan/:id', async (req, res) => {
  try {
    const id = parseInt(req.params.id);
    const target = getTarget(id);

    if (!target) {
      res.status(404).json({ error: 'Target not found' });
      return;
    }

    const result = await diffDetector(id);

    res.json({
      target_id: id,
      changed: result.changed,
      patch_history: result.patchHistory,
      error: result.error,
    });
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// POST /api/triggers/test - Test with mock payload
triggersRouter.post('/test', async (req, res) => {
  try {
    const { mock_content, mock_url } = req.body;

    if (!mock_content) {
      res.status(400).json({ error: 'mock_content is required' });
      return;
    }

    // Simulate a detection with mock data
    res.json({
      message: 'Test mode - mock detection',
      mock_url: mock_url || '',
      mock_content_length: mock_content.length,
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});
