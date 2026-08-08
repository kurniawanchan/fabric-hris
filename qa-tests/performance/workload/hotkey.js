'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');
const crypto = require('crypto');
const { timed } = require('./latency-recorder');

const SECTIONS = ['PERSONAL', 'EMPLOYMENT', 'EDUCATION', 'ADDITIONAL', 'PAYROLL'];

/**
 * QA-6 hot-key contention workload.
 *
 * Unlike workload/write.js (QA-4's PT-1, which deliberately uses a FRESH
 * employeeID per transaction specifically to AVOID MVCC contention), this
 * workload deliberately INDUCES it: within a single round, the first
 * `activeWorkers` (0-indexed workerIndex < activeWorkers) all submit
 * RecordProfileSection for the EXACT SAME (employeeID, profileSection) pair,
 * with prevHash="" -- every one of them independently reads "no head exists
 * yet" at endorsement time (the key genuinely does not exist before the
 * round runs), so every proposal endorses successfully. Only the first write
 * to actually commit wins; Fabric's own MVCC validation should reject the
 * other (activeWorkers - 1) at COMMIT time, because their simulated read-set
 * (the key's absence) is now stale once the winner's write lands. This is
 * different from -- and must not be confused with -- the chaincode's own
 * ErrStaleChainReference check (record_profile_section.go), which is an
 * APPLICATION-level rejection made against already-read state at
 * ENDORSEMENT time and does not apply here (every worker's simulation sees
 * head == nil, so that branch is never taken; see QA6-RESULTS.md).
 *
 * Caliper's worker/round model fixes a single `workers.number` for an
 * entire benchmark config (there is no per-round worker-count override --
 * verified by inspection of round-orchestrator.js/worker-orchestrator.js).
 * To still get distinct, controlled concurrency levels (N=2/5/10) out of one
 * fixed-size worker pool (10, the largest N under test), every round hands
 * all 10 workers exactly one transaction each (txNumber: 10 => txPerWorker
 * === 1, deterministically, no clamp-to-1 edge case): the first
 * `activeWorkers` of the 10 fire the real same-key race (recorded under
 * `${roundLabel}-hotkey.worker<N>.jsonl`); the remaining (10 - activeWorkers)
 * fire a harmless, uncontended control write to their OWN fresh employeeID
 * (recorded under `${roundLabel}-filler.worker<N>.jsonl`, and tagged
 * `sha256:qa6-filler-...` in dataHash) purely so Caliper's fixed-transaction
 * -count loop (`stats.getTotalSubmittedTx() < number`, caliper-worker.js)
 * terminates for every worker process -- a worker whose submitTransaction()
 * never calls sendRequests() never increments that counter and the round
 * hangs forever. Filler transactions never touch the hot key and are
 * excluded from the QA-6 contention analysis.
 */
class HotkeyWorkload extends WorkloadModuleBase {
    constructor() {
        super();
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);
        this.tenantId = roundArguments.tenantId || 'tenant01';
        this.roundLabel = roundArguments.roundLabel || `round${roundIndex}`;
        this.employeeId = roundArguments.employeeId;
        this.profileSection = roundArguments.profileSection || 'PERSONAL';
        this.activeWorkers = roundArguments.activeWorkers;

        if (!this.employeeId) {
            throw new Error('hotkey.js requires roundArguments.employeeId (fixed, unique, never-before-used per round) -- every racing worker must target the SAME ledger key.');
        }
        if (typeof this.activeWorkers !== 'number' || this.activeWorkers < 1) {
            throw new Error('hotkey.js requires roundArguments.activeWorkers (a positive integer, the N in this round\'s N-way race).');
        }
        this.isHot = this.workerIndex < this.activeWorkers;
    }

    async submitTransaction() {
        if (this.isHot) {
            // The real race: EVERY hot worker targets the identical
            // (employeeId, profileSection) key with prevHash="" -- each
            // believes it is performing a genuine first write. dataHash and
            // updatedBy are worker-salted ONLY so the committed winner can be
            // identified afterwards from the ledger; salting them does not
            // change the key being raced for.
            const dataHash = `sha256:qa6hotkey-w${this.workerIndex}-` + crypto.randomBytes(16).toString('hex');
            const updatedBy = `benchactor-hotkey-w${this.workerIndex}-` + crypto.randomBytes(8).toString('hex');

            const args = {
                contractId: 'employeeprofilerecord',
                contractFunction: 'RecordProfileSection',
                contractArguments: [
                    this.tenantId,
                    this.employeeId,
                    this.profileSection,
                    dataHash,
                    '', // prevHash: every hot worker believes this is a genuine first-write
                    updatedBy,
                    '[]',
                    'JCS-RFC8785-v1',
                    'SHA-256',
                    new Date().toISOString()
                ],
                invokerIdentity: 'admin',
                invokerMspId: 'Org1MSP',
                targetOrganizations: ['Org1MSP', 'OrgClient-tenant01MSP'],
                readOnly: false
            };

            await timed(`${this.roundLabel}-hotkey`, this.workerIndex, () => this.sutAdapter.sendRequests(args));
        } else {
            // Filler: an uncontended, fresh-key control write (same pattern as
            // workload/write.js) so this worker's fixed-count loop terminates.
            // Deliberately excluded from the QA-6 contention analysis.
            const fillerEmployeeId = `qa6-filler-w${this.workerIndex}-` + crypto.randomBytes(20).toString('hex');
            const dataHash = 'sha256:qa6filler-' + crypto.randomBytes(16).toString('hex');
            const updatedBy = 'benchactor-filler-' + crypto.randomBytes(8).toString('hex');
            const section = SECTIONS[this.workerIndex % SECTIONS.length];

            const args = {
                contractId: 'employeeprofilerecord',
                contractFunction: 'RecordProfileSection',
                contractArguments: [
                    this.tenantId,
                    fillerEmployeeId,
                    section,
                    dataHash,
                    '',
                    updatedBy,
                    '[]',
                    'JCS-RFC8785-v1',
                    'SHA-256',
                    new Date().toISOString()
                ],
                invokerIdentity: 'admin',
                invokerMspId: 'Org1MSP',
                targetOrganizations: ['Org1MSP', 'OrgClient-tenant01MSP'],
                readOnly: false
            };

            await timed(`${this.roundLabel}-filler`, this.workerIndex, () => this.sutAdapter.sendRequests(args));
        }
    }
}

function createWorkloadModule() {
    return new HotkeyWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
