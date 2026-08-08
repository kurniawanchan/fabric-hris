'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');
const crypto = require('crypto');
const { timed } = require('./latency-recorder');

const SECTIONS = ['PERSONAL', 'EMPLOYMENT', 'EDUCATION', 'ADDITIONAL', 'PAYROLL'];

/**
 * PT-1 / PT-2 workload: profile-write throughput and latency.
 *
 * Every submitted transaction uses a FRESH, unique employeeID (random 64-hex-char
 * string) so prevHash="" is always a genuine first-write for that
 * (employeeID, profileSection) pair -- never fighting MVCC/stale-prevHash
 * rejections as a benchmark artifact (per the task's explicit guidance). All
 * argument content is synthetic-only (no real PII-shaped content), matching this
 * repo's own testing convention (confidentiality register: tenant01/tenant02,
 * "Generic Company" style placeholders only).
 */
class WriteWorkload extends WorkloadModuleBase {
    constructor() {
        super();
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);
        this.tenantId = roundArguments.tenantId || 'tenant01';
        this.roundLabel = roundArguments.roundLabel || `round${roundIndex}`;
        this.txCounter = 0;
    }

    _freshEmployeeId() {
        // workerIndex-salted counter + random bytes: guarantees uniqueness across
        // parallel workers within a single benchmark run.
        this.txCounter += 1;
        return `bench-w${this.workerIndex}-${this.txCounter}-${crypto.randomBytes(20).toString('hex')}`;
    }

    async submitTransaction() {
        const employeeId = this._freshEmployeeId();
        const section = SECTIONS[this.txCounter % SECTIONS.length];
        const dataHash = 'sha256:bench' + crypto.randomBytes(28).toString('hex');
        const updatedBy = 'benchactor' + crypto.randomBytes(16).toString('hex');

        const args = {
            contractId: 'employeeprofilerecord',
            contractFunction: 'RecordProfileSection',
            contractArguments: [
                this.tenantId,
                employeeId,
                section,
                dataHash,
                '', // prevHash: always empty -> genuine first-write, no MVCC contention by construction
                updatedBy,
                '[]', // ipfsCIDs, JSON-encoded array string
                'JCS-RFC8785-v1',
                'SHA-256',
                new Date().toISOString()
            ],
            invokerIdentity: 'admin',
            invokerMspId: 'Org1MSP',
            targetOrganizations: ['Org1MSP', 'OrgClient-tenant01MSP'],
            readOnly: false
        };

        await timed(this.roundLabel, this.workerIndex, () => this.sutAdapter.sendRequests(args));
    }
}

function createWorkloadModule() {
    return new WriteWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
