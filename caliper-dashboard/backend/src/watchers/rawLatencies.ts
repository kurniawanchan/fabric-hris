import { openSync, closeSync, readSync, statSync, readdirSync } from "node:fs";
import path from "node:path";
import { config } from "../config.js";
import { setIsLive } from "../state.js";
import { broadcast, captureContext } from "../stream.js";

interface LatencyRecord {
  ts: number;
  latencyMs: number;
  success: boolean;
}

const TICK_MS = 1000;

// Per-file byte offset already consumed -- lets us read only newly-appended
// bytes on each tick rather than re-reading whole files (AD-3: file-watching
// only, no spawn; this is the tailing half of that).
const offsets = new Map<string, number>();

/** jsonlLastActivity backs the heartbeat's sources.jsonl timestamp (AD-6). */
let jsonlLastActivity: string | null = null;
export function getJsonlLastActivity(): string | null {
  return jsonlLastActivity;
}

let everSeenData = false;

// FR2: "the running count of successful vs. failed transactions... for the
// in-progress round" -- cumulative totals, not a per-tick sample. Simplification
// noted: these accumulate for the life of the backend process; there is no
// multi-run reset within one process lifetime, which is correct for this
// product's single-demo-session use (Story 1.1's UJ-1).
let totalSuccessCount = 0;
let totalFailureCount = 0;

function readNewLines(filePath: string, fromOffset: number): { lines: string[]; newOffset: number } {
  const size = statSync(filePath).size;
  // A file smaller than our recorded offset was truncated or recreated
  // (e.g. a new run reusing the same round-label file name) -- resume from
  // the start rather than silently seeing "nothing new" forever.
  const effectiveOffset = size < fromOffset ? 0 : fromOffset;
  if (size <= effectiveOffset) {
    return { lines: [], newOffset: effectiveOffset };
  }
  fromOffset = effectiveOffset;
  const length = size - fromOffset;
  const buffer = Buffer.alloc(length);
  const fd = openSync(filePath, "r");
  try {
    readSync(fd, buffer, 0, length, fromOffset);
  } finally {
    closeSync(fd);
  }
  const text = buffer.toString("utf8");
  // The trailing chunk may be a partial line if Caliper is mid-write; only
  // advance the offset past the last COMPLETE line so a half-written JSON
  // object is picked up whole on the next tick, never parsed truncated.
  const lastNewline = text.lastIndexOf("\n");
  if (lastNewline === -1) {
    return { lines: [], newOffset: fromOffset };
  }
  const complete = text.slice(0, lastNewline);
  const lines = complete.split("\n").filter((l) => l.length > 0);
  return { lines, newOffset: fromOffset + lastNewline + 1 };
}

function tick(): void {
  let files: string[];
  try {
    files = readdirSync(config.rawLatenciesDir).filter((f) => f.endsWith(".jsonl"));
  } catch {
    // NFR3: directory not existing yet (no run has ever started) degrades
    // silently -- never crashes the backend or the rest of the app.
    return;
  }

  const records: LatencyRecord[] = [];

  for (const file of files) {
    const filePath = path.join(config.rawLatenciesDir, file);
    try {
      const prevOffset = offsets.get(file) ?? 0;
      const { lines, newOffset } = readNewLines(filePath, prevOffset);
      offsets.set(file, newOffset);

      for (const line of lines) {
        try {
          const rec = JSON.parse(line) as LatencyRecord;
          if (typeof rec.latencyMs === "number" && typeof rec.success === "boolean") {
            records.push(rec);
          }
        } catch {
          // NFR3: one malformed line is skipped, never fails the whole tick.
        }
      }
    } catch {
      // NFR3: this one file failing (deleted mid-read, permissions, etc.)
      // never blocks the other files' data from being read this tick.
      continue;
    }
  }

  if (records.length === 0) {
    return;
  }

  jsonlLastActivity = new Date().toISOString();

  // AD-5: isLive flips on the first successfully PARSED data record, never
  // on mere file-existence/open.
  if (!everSeenData) {
    everSeenData = true;
    setIsLive(true);
  }

  const tickSuccessCount = records.filter((r) => r.success).length;
  const tickFailureCount = records.length - tickSuccessCount;
  totalSuccessCount += tickSuccessCount;
  totalFailureCount += tickFailureCount;

  const avgLatency = records.reduce((sum, r) => sum + r.latencyMs, 0) / records.length;
  const throughput = records.length / (TICK_MS / 1000);

  broadcast({
    type: "metric-update",
    context: captureContext(),
    throughput,
    latency: avgLatency,
    successCount: totalSuccessCount,
    failureCount: totalFailureCount,
  });
}

let intervalHandle: NodeJS.Timeout | undefined;

/**
 * Baselines every currently-existing file's size as its starting offset, so
 * content already on disk from a PAST run is never mistaken for live data
 * the moment this process boots. Only bytes appended after this call counts
 * as live -- without this, historical raw-latencies files (real runs with
 * thousands of records) would flip isLive true and report a live benchmark
 * that was never actually running.
 */
function baselineExistingFiles(): void {
  let files: string[];
  try {
    files = readdirSync(config.rawLatenciesDir).filter((f) => f.endsWith(".jsonl"));
  } catch {
    return;
  }
  for (const file of files) {
    try {
      const size = statSync(path.join(config.rawLatenciesDir, file)).size;
      offsets.set(file, size);
    } catch {
      // File disappeared between readdir and stat -- skip, next tick's own
      // readdir will simply not see it.
    }
  }
}

export function startRawLatenciesWatcher(): void {
  if (intervalHandle) return;
  baselineExistingFiles();
  intervalHandle = setInterval(tick, TICK_MS);
}

export function stopRawLatenciesWatcher(): void {
  if (intervalHandle) clearInterval(intervalHandle);
  intervalHandle = undefined;
}
