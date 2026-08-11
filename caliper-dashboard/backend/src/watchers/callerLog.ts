import { openSync, closeSync, readSync, statSync, existsSync } from "node:fs";
import { config } from "../config.js";
import { broadcast, captureContext } from "../stream.js";

const TICK_MS = 500;

let offset = 0;

let lastActivity: string | null = null;
export function getLogLastActivity(): string | null {
  return lastActivity;
}

function readNewLines(): string[] {
  const size = statSync(config.callerLogPath).size;
  if (size < offset) {
    // File was truncated/recreated (a fresh run reusing the same path) --
    // resume from the start, same defensive rule as the raw-latencies watcher.
    offset = 0;
  }
  if (size <= offset) return [];

  const length = size - offset;
  const buffer = Buffer.alloc(length);
  const fd = openSync(config.callerLogPath, "r");
  try {
    readSync(fd, buffer, 0, length, offset);
  } finally {
    closeSync(fd);
  }

  const text = buffer.toString("utf8");
  const lastNewline = text.lastIndexOf("\n");
  if (lastNewline === -1) return [];

  offset += lastNewline + 1;
  return text
    .slice(0, lastNewline)
    .split("\n")
    .filter((l) => l.length > 0);
}

function tick(): void {
  if (!existsSync(config.callerLogPath)) {
    // NFR3: no log file yet (Caliper hasn't been started) is not an error --
    // degrades silently, never crashes the backend or the rest of the app.
    // Reset the offset so that whenever the file DOES next appear, it's
    // baselined fresh rather than measured against a stale prior offset.
    offset = 0;
    return;
  }

  let lines: string[];
  try {
    lines = readNewLines();
  } catch {
    return;
  }

  if (lines.length === 0) return;

  lastActivity = new Date().toISOString();
  for (const line of lines) {
    broadcast({
      type: "log-line",
      context: captureContext(),
      line,
      timestamp: lastActivity,
    });
  }
}

let intervalHandle: NodeJS.Timeout | undefined;

/**
 * Baselines the file's existing size (0 if it doesn't exist yet) BEFORE the
 * first tick runs -- exactly like the raw-latencies watcher's
 * baselineExistingFiles(). Doing this lazily on "the first tick where the
 * file happens to exist" is a real race: the operator's redirect can create
 * the file with its first line already written by the time that tick
 * fires, silently swallowing that line into the baseline instead of
 * broadcasting it.
 */
export function startCallerLogWatcher(): void {
  if (intervalHandle) return;
  offset = existsSync(config.callerLogPath) ? statSync(config.callerLogPath).size : 0;
  intervalHandle = setInterval(tick, TICK_MS);
}

export function stopCallerLogWatcher(): void {
  if (intervalHandle) clearInterval(intervalHandle);
  intervalHandle = undefined;
}
