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
import cors from 'cors';
import bodyParser from 'body-parser';
import { getConfig } from '../config/env.js';
import { targetsRouter } from './targets.js';
import { triggersRouter } from './triggers.js';
import { logsRouter } from './logs.js';

export function createApp(): express.Express {
  const app = express();

  // Middleware
  app.use(cors());
  app.use(bodyParser.json());
  app.use(bodyParser.urlencoded({ extended: true }));

  // Serve static files from public directory
  app.use(express.static('public'));

  // API routes
  app.use('/api/targets', targetsRouter);
  app.use('/api/triggers', triggersRouter);
  app.use('/api/logs', logsRouter);

  // Health check endpoint
  app.get('/api/health', (_req, res) => {
    res.json({ status: 'ok', timestamp: new Date().toISOString() });
  });

  // Root redirect to dashboard
  app.get('/', (_req, res) => {
    res.sendFile('index.html', { root: 'public' });
  });

  return app;
}

export function startServer(app: express.Express): void {
  const config = getConfig();
  const port = parseInt(config.PORT);
  const host = config.HOST;

  app.listen(port, host, () => {
    console.log(`PatchPulse server running on http://${host}:${port}`);
  });
}
