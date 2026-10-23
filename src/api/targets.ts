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
  createTarget,
  getTarget,
  getAllTargets,
  updateTarget,
  deleteTarget,
} from '../db/queries.js';
import { scheduler } from '../watcher/scheduler.js';

export const targetsRouter = express.Router();

// GET /api/targets - List all targets
targetsRouter.get('/', (_req, res) => {
  try {
    const targets = getAllTargets();
    res.json(targets);
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// GET /api/targets/:id - Get a specific target
targetsRouter.get('/:id', (req, res) => {
  try {
    const id = parseInt(req.params.id);
    const target = getTarget(id);
    if (!target) {
      res.status(404).json({ error: 'Target not found' });
      return;
    }
    res.json(target);
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// POST /api/targets - Create a new target
targetsRouter.post('/', (req, res) => {
  try {
    const { name, url, adapter_type, adapter_config, check_interval, enabled } = req.body;

    if (!name || !url || !adapter_type || !adapter_config) {
      res.status(400).json({
        error: 'Missing required fields: name, url, adapter_type, adapter_config',
      });
      return;
    }

    const target = {
      name,
      url,
      adapter_type,
      adapter_config: typeof adapter_config === 'string' ? adapter_config : JSON.stringify(adapter_config),
      check_interval: check_interval || 3600,
      enabled: enabled !== undefined ? (enabled ? 1 : 0) : 1,
    };

    const id = createTarget(target);
    const createdTarget = getTarget(id);

    // Schedule the new target
    if (createdTarget) {
      scheduler.scheduleTarget(createdTarget);
    }

    res.status(201).json(createdTarget);
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// PUT /api/targets/:id - Update a target
targetsRouter.put('/:id', (req, res) => {
  try {
    const id = parseInt(req.params.id);
    const updates: any = {};

    if (req.body.name !== undefined) updates.name = req.body.name;
    if (req.body.url !== undefined) updates.url = req.body.url;
    if (req.body.adapter_type !== undefined) updates.adapter_type = req.body.adapter_type;
    if (req.body.adapter_config !== undefined) {
      updates.adapter_config = typeof req.body.adapter_config === 'string'
        ? req.body.adapter_config
        : JSON.stringify(req.body.adapter_config);
    }
    if (req.body.check_interval !== undefined) updates.check_interval = req.body.check_interval;
    if (req.body.enabled !== undefined) updates.enabled = req.body.enabled ? 1 : 0;

    updateTarget(id, updates);

    // Reschedule if necessary
    const target = getTarget(id);
    if (target) {
      scheduler.scheduleTarget(target);
    }

    const updatedTarget = getTarget(id);
    if (updatedTarget) {
      res.json(updatedTarget);
    } else {
      res.status(404).json({ error: 'Target not found' });
    }
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// DELETE /api/targets/:id - Delete a target
targetsRouter.delete('/:id', (req, res) => {
  try {
    const id = parseInt(req.params.id);
    deleteTarget(id);
    scheduler.unscheduleTarget(id);
    res.status(204).send();
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// POST /api/targets/:id/enable - Enable a target
targetsRouter.post('/:id/enable', (req, res) => {
  try {
    const id = parseInt(req.params.id);
    updateTarget(id, { enabled: 1 });
    scheduler.startTarget(id);
    res.json(getTarget(id));
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});

// POST /api/targets/:id/disable - Disable a target
targetsRouter.post('/:id/disable', (req, res) => {
  try {
    const id = parseInt(req.params.id);
    updateTarget(id, { enabled: 0 });
    scheduler.stopTarget(id);
    res.json(getTarget(id));
  } catch (error) {
    res.status(500).json({ error: (error as Error).message });
  }
});
