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
import cron from 'node-cron';
import { Target } from '../db/queries.js';
import { checkTarget } from './diff-detector.js';

interface ScheduledTask {
  targetId: number;
  task: cron.ScheduledTask;
}

class Scheduler {
  private tasks: Map<number, ScheduledTask> = new Map();

  scheduleTarget(target: Target): void {
    // Remove existing task if any
    this.unscheduleTarget(target.id!);

    // Convert interval (seconds) to cron expression
    // node-cron uses 6 fields: sec min hour day month weekday
    const interval = target.check_interval;
    const cronExpression = this.intervalToCron(interval);

    const task = cron.schedule(cronExpression, async () => {
      try {
        await checkTarget(target.id!);
      } catch (error) {
        console.error(`Error checking target ${target.id}:`, error);
      }
    }, {
      scheduled: target.enabled === 1,
    });

    this.tasks.set(target.id!, { targetId: target.id!, task });
  }

  unscheduleTarget(targetId: number): void {
    const scheduled = this.tasks.get(targetId);
    if (scheduled) {
      scheduled.task.stop();
      this.tasks.delete(targetId);
    }
  }

  unscheduleAll(): void {
    this.tasks.forEach(({ task }) => task.stop());
    this.tasks.clear();
  }

  startTarget(targetId: number): void {
    const scheduled = this.tasks.get(targetId);
    if (scheduled) {
      scheduled.task.start();
    }
  }

  stopTarget(targetId: number): void {
    const scheduled = this.tasks.get(targetId);
    if (scheduled) {
      scheduled.task.stop();
    }
  }

  private intervalToCron(intervalSeconds: number): string {
    // For simplicity, we'll use minute-based scheduling
    // Minimum interval is 1 minute
    const minutes = Math.max(1, Math.floor(intervalSeconds / 60));

    if (minutes < 60) {
      // Every N minutes
      return `*/${minutes} * * * *`;
    } else if (minutes < 1440) {
      // Every N hours
      const hours = Math.floor(minutes / 60);
      return `0 */${hours} * * *`;
    } else {
      // Every N days
      const days = Math.floor(minutes / 1440);
      return `0 0 */${days} * *`;
    }
  }

  getScheduledCount(): number {
    return this.tasks.size;
  }
}

export const scheduler = new Scheduler();
