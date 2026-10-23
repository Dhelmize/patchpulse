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
import {
  getRecentPatchHistory,
  getPatchHistoryByTarget,
  getRecentAgentLogs,
  getAgentLogsByTarget,
  getAgentLogsByPatch,
} from '../db/queries.js';

export const logsRouter = express.Router();

// GET /api/logs/patches - Get recent patch history
logsRouter.get('/patches', (req, res) => {
  try {
    const limit = req.query.limit ? parseInt(req.query.limit as string) : 20;
    const patches = getRecentPatchHistory(limit);
    res.json(patches);
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// GET /api/logs/patches/:targetId - Get patch history for a specific target
logsRouter.get('/patches/:targetId', (req, res) => {
  try {
    const targetId = parseInt(req.params.targetId);
    const limit = req.query.limit ? parseInt(req.query.limit as string) : 10;
    const patches = getPatchHistoryByTarget(targetId, limit);
    res.json(patches);
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// GET /api/logs/agent - Get recent agent logs
logsRouter.get('/agent', (req, res) => {
  try {
    const limit = req.query.limit ? parseInt(req.query.limit as string) : 50;
    const logs = getRecentAgentLogs(limit);
    res.json(logs);
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// GET /api/logs/agent/:targetId - Get agent logs for a specific target
logsRouter.get('/agent/:targetId', (req, res) => {
  try {
    const targetId = parseInt(req.params.targetId);
    const limit = req.query.limit ? parseInt(req.query.limit as string) : 20;
    const logs = getAgentLogsByTarget(targetId, limit);
    res.json(logs);
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// GET /api/logs/agent/patch/:patchId - Get agent logs for a specific patch
logsRouter.get('/agent/patch/:patchId', (req, res) => {
  try {
    const patchId = parseInt(req.params.patchId);
    const logs = getAgentLogsByPatch(patchId);
    res.json(logs);
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});
