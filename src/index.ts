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
import { createApp, startServer } from './api/index.js';
import { getDatabase, closeDatabase } from './db/connection.js';
import { getEnabledTargets } from './db/queries.js';
import { scheduler } from './watcher/scheduler.js';

async function main(): Promise<void> {
  try {
    console.log('Starting PatchPulse...');

    // Initialize database
    console.log('Initializing database...');
    getDatabase();

    // Load enabled targets and schedule them
    console.log('Loading targets...');
    const targets = getEnabledTargets();
    console.log(`Found ${targets.length} enabled targets`);

    for (const target of targets) {
      scheduler.scheduleTarget(target);
      console.log(`Scheduled target: ${target.name} (${target.url})`);
    }

    // Create and start Express server
    console.log('Starting web server...');
    const app = createApp();
    startServer(app);

    // Graceful shutdown
    process.on('SIGINT', shutdown);
    process.on('SIGTERM', shutdown);

    console.log('PatchPulse is running');
  } catch (error) {
    console.error('Failed to start PatchPulse:', error);
    process.exit(1);
  }
}

async function shutdown(): Promise<void> {
  console.log('\nShutting down PatchPulse...');

  // Stop all scheduled tasks
  scheduler.unscheduleAll();

  // Close database connection
  closeDatabase();

  console.log('PatchPulse stopped');
  process.exit(0);
}

main();
