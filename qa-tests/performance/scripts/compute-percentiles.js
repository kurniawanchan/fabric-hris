'use strict';

// Aggregates the JSON-lines latency files written by workload/latency-recorder.js
// (one file per worker per round, under ../results/raw-latencies/) into
// p50/p95/p99 per round label. Needed because Caliper 0.7.1's own report only
// computes max/min/avg latency (PT-2 asks for percentiles).
//
// Usage: node scripts/compute-percentiles.js [roundLabelPrefix]
// With no argument, aggregates every round found in results/raw-latencies/.

const fs = require('fs');
const path = require('path');

const RAW_DIR = path.join(__dirname, '..', 'results', 'raw-latencies');

function percentile(sortedArr, p) {
    if (sortedArr.length === 0) {
        return null;
    }
    const idx = Math.min(sortedArr.length - 1, Math.ceil((p / 100) * sortedArr.length) - 1);
    return sortedArr[Math.max(0, idx)];
}

function main() {
    const filterPrefix = process.argv[2];

    if (!fs.existsSync(RAW_DIR)) {
        console.error(`No raw latency directory found at ${RAW_DIR}`);
        process.exit(1);
    }

    const files = fs.readdirSync(RAW_DIR).filter(f => f.endsWith('.jsonl'));
    const byRound = new Map();

    for (const file of files) {
        // filename shape: <roundLabel>.worker<N>.jsonl -- roundLabel itself may
        // contain dots (it doesn't in our configs, but split conservatively).
        const m = file.match(/^(.+)\.worker\d+\.jsonl$/);
        if (!m) {
            continue;
        }
        const roundLabel = m[1];
        if (filterPrefix && !roundLabel.startsWith(filterPrefix)) {
            continue;
        }

        const lines = fs.readFileSync(path.join(RAW_DIR, file), 'utf8').split('\n').filter(Boolean);
        if (!byRound.has(roundLabel)) {
            byRound.set(roundLabel, []);
        }
        const arr = byRound.get(roundLabel);
        for (const line of lines) {
            try {
                const rec = JSON.parse(line);
                arr.push(rec);
            } catch (e) {
                // skip malformed line
            }
        }
    }

    const results = [];
    for (const [roundLabel, records] of byRound.entries()) {
        const succeeded = records.filter(r => r.success);
        const latencies = succeeded.map(r => r.latencyMs).sort((a, b) => a - b);

        results.push({
            roundLabel,
            totalRecorded: records.length,
            succeeded: succeeded.length,
            failed: records.length - succeeded.length,
            p50Ms: percentile(latencies, 50),
            p95Ms: percentile(latencies, 95),
            p99Ms: percentile(latencies, 99),
            minMs: latencies.length ? latencies[0] : null,
            maxMs: latencies.length ? latencies[latencies.length - 1] : null,
            avgMs: latencies.length ? (latencies.reduce((a, b) => a + b, 0) / latencies.length) : null
        });
    }

    results.sort((a, b) => a.roundLabel.localeCompare(b.roundLabel));

    console.log(JSON.stringify(results, null, 2));

    // Also print a human-readable table
    console.log('\n| Round | Recorded | Succeeded | Failed | p50 (ms) | p95 (ms) | p99 (ms) | min (ms) | max (ms) | avg (ms) |');
    console.log('|---|---|---|---|---|---|---|---|---|---|');
    for (const r of results) {
        const f = (v) => v === null ? 'n/a' : v.toFixed(1);
        console.log(`| ${r.roundLabel} | ${r.totalRecorded} | ${r.succeeded} | ${r.failed} | ${f(r.p50Ms)} | ${f(r.p95Ms)} | ${f(r.p99Ms)} | ${f(r.minMs)} | ${f(r.maxMs)} | ${f(r.avgMs)} |`);
    }
}

main();
