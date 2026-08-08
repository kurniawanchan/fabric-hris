'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');
const crypto = require('crypto');
const { timed } = require('./latency-recorder');

const SECTIONS = ['PERSONAL', 'EMPLOYMENT', 'EDUCATION', 'ADDITIONAL', 'PAYROLL'];

/**
 * PT-3 workload: verify-read (Evaluate) latency for GetProfileSectionRecord.
 *
 * A separate/lighter round from PT-1/2 as required: no ordering involved, purely
 * a read against peer state (LevelDB/CouchDB via the chaincode), evaluated (not
 * submitted) so it never touches the orderer.
 *
 * IMPORTANT (a past real defect in this codebase): tenantId is the FIRST
 * argument to GetProfileSectionRecord -- omitting it produces "Incorrect number
 * of params. Expected 3, received 2".
 *
 * This round first writes a small, fixed pool of real records (so reads have
 * something genuine to find), then evaluates GetProfileSectionRecord against
 * that pool. Read latency is measured on the evaluate calls only.
 */
class ReadWorkload extends WorkloadModuleBase {
    constructor() {
        super();
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);
        this.tenantId = roundArguments.tenantId || 'tenant01';
        this.roundLabel = roundArguments.roundLabel || `round${roundIndex}`;
        this.poolSize = roundArguments.poolSize || 20;
        this.pool = [];

        // Seed a small pool of genuine records for this worker to read back,
        // so PT-3 measures real read latency against real ledger state, not
        // guaranteed not-found lookups.
        for (let i = 0; i < this.poolSize; i++) {
            const employeeId = `bench-read-w${this.workerIndex}-${i}-${crypto.randomBytes(12).toString('hex')}`;
            const section = SECTIONS[i % SECTIONS.length];
            const dataHash = 'sha256:benchread' + crypto.randomBytes(24).toString('hex');
            const updatedBy = 'benchreadactor' + crypto.randomBytes(12).toString('hex');

            await this.sutAdapter.sendRequests({
                contractId: 'employeeprofilerecord',
                contractFunction: 'RecordProfileSection',
                contractArguments: [
                    this.tenantId, employeeId, section, dataHash, '', updatedBy,
                    '[]', 'JCS-RFC8785-v1', 'SHA-256', new Date().toISOString()
                ],
                invokerIdentity: 'admin',
                invokerMspId: 'Org1MSP',
                targetOrganizations: ['Org1MSP', 'OrgClient-tenant01MSP'],
                readOnly: false
            });

            this.pool.push({ employeeId, section });
        }

        this.readCounter = 0;
    }

    async submitTransaction() {
        const record = this.pool[this.readCounter % this.pool.length];
        this.readCounter += 1;

        const args = {
            contractId: 'employeeprofilerecord',
            contractFunction: 'GetProfileSectionRecord',
            contractArguments: [this.tenantId, record.employeeId, record.section],
            invokerIdentity: 'admin',
            invokerMspId: 'Org1MSP',
            readOnly: true
        };

        await timed(this.roundLabel, this.workerIndex, () => this.sutAdapter.sendRequests(args));
    }
}

function createWorkloadModule() {
    return new ReadWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
