import { readFileSync } from "node:fs";
import path from "node:path";
import { config } from "./config.js";

export interface HistoryRun {
  id: string;
  source: string;
  section: string;
  headers: string[];
  rows: Record<string, string>[];
}

/** Strips GFM inline emphasis/code markers so cell text displays cleanly. */
function cleanCell(cell: string): string {
  return cell
    .trim()
    .replace(/\*\*(.+?)\*\*/g, "$1")
    .replace(/`([^`]+)`/g, "$1");
}

function splitRow(line: string): string[] {
  // GFM table rows are pipe-delimited; leading/trailing pipes are optional.
  const trimmed = line.trim().replace(/^\|/, "").replace(/\|$/, "");
  return trimmed.split("|").map(cleanCell);
}

const SEPARATOR_ROW = /^\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?$/;

/**
 * FR6/AD-4: extracts every GFM table in a Markdown file into structured
 * JSON, tagged with the nearest preceding heading -- never returns raw
 * Markdown text. `RESULTS.md`/`QA6-RESULTS.md` are dense narrative
 * documents with genuinely different measurement shapes per section (write
 * throughput vs. read latency vs. MVCC contention have different columns),
 * so this deliberately does not force one fixed run schema onto them --
 * each table becomes its own run entry with its own real headers.
 */
export function extractTables(filePath: string): HistoryRun[] {
  const text = readFileSync(filePath, "utf8");
  const lines = text.split("\n");
  const sourceName = path.basename(filePath);
  const runs: HistoryRun[] = [];

  let currentHeading = "";
  let tableCounter = 0;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i] ?? "";
    const headingMatch = /^#{1,6}\s+(.+)$/.exec(line);
    if (headingMatch?.[1]) {
      currentHeading = cleanCell(headingMatch[1]);
      continue;
    }

    const isTableStart =
      line.trim().startsWith("|") &&
      lines[i + 1] !== undefined &&
      SEPARATOR_ROW.test((lines[i + 1] ?? "").trim());

    if (!isTableStart) continue;

    const headers = splitRow(line);
    i += 2; // skip header + separator rows

    const rows: Record<string, string>[] = [];
    while (i < lines.length && (lines[i] ?? "").trim().startsWith("|")) {
      const cells = splitRow(lines[i] ?? "");
      const row: Record<string, string> = {};
      headers.forEach((h, idx) => {
        row[h] = cells[idx] ?? "";
      });
      rows.push(row);
      i++;
    }
    i--; // outer loop's own i++ accounts for the line we stopped on

    tableCounter++;
    runs.push({
      id: `${sourceName}#${tableCounter}`,
      source: sourceName,
      section: currentHeading,
      headers,
      rows,
    });
  }

  return runs;
}

export function loadHistory(): HistoryRun[] {
  return [...extractTables(config.resultsMdPath), ...extractTables(config.qa6ResultsMdPath)];
}
