'use strict';

const fs = require('fs');
const path = require('path');

// Hyperledger Caliper 0.7.1's default report/report-builder only computes
// max/min/avg latency -- no percentiles (verified: no percentile/p95/p99
// logic anywhere in @hyperledger/caliper-core's report code). PT-2 requires
// p50/p95/p99, so this module independently records the wall-clock latency
// of every submitted transaction to a per-worker/per-round JSON-lines file
// under ../results/, which scripts/compute-percentiles.js later aggregates.
const RESULTS_DIR = path.join(__dirname, '..', 'results', 'raw-latencies');

function ensureDir() {
    fs.mkdirSync(RESULTS_DIR, { recursive: true });
}

/**
 * @param {string} roundLabel a stable label identifying the round (e.g. "pt1-write-500tps")
 * @param {number} workerIndex
 * @returns {string} path to this worker's JSON-lines file for this round
 */
function fileFor(roundLabel, workerIndex) {
    ensureDir();
    return path.join(RESULTS_DIR, `${roundLabel}.worker${workerIndex}.jsonl`);
}

/**
 * Time an async request function and append {latencyMs, success, ts} to this
 * worker/round's results file. Re-throws whatever the wrapped function throws.
 *
 * @param {string} roundLabel
 * @param {number} workerIndex
 * @param {() => Promise<*>} fn the async operation to time (e.g. sutAdapter.sendRequests)
 * @returns {Promise<*>} whatever fn resolves to
 */
async function timed(roundLabel, workerIndex, fn) {
    const start = process.hrtime.bigint();
    let success = true;
    let result;
    try {
        result = await fn();
        // Caliper's sendRequests doesn't throw on chaincode-level tx failure for
        // a single request, it returns a TxStatus with SetStatusFail() -- check it.
        if (result && typeof result.IsCommitted === 'function' && !result.IsCommitted()) {
            success = false;
        }
        return result;
    } catch (err) {
        success = false;
        throw err;
    } finally {
        const end = process.hrtime.bigint();
        const latencyMs = Number(end - start) / 1e6;
        fs.appendFileSync(fileFor(roundLabel, workerIndex), JSON.stringify({ ts: Date.now(), latencyMs, success }) + '\n');
    }
}

module.exports = { timed, fileFor, RESULTS_DIR };
