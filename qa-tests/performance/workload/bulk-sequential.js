'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');
const crypto = require('crypto');

const SECTIONS = ['PERSONAL', 'EMPLOYMENT', 'EDUCATION', 'ADDITIONAL', 'PAYROLL'];

/**
 * PT-5 (one-by-one blocking leg): the naive comparison point for
 * workload/bulk-batched.js -- same total record count per Caliper
 * "transaction" tick (`batchSize`), same argument shape, but each
 * RecordProfileSection call is awaited (blocked on) before the next one is
 * issued, exactly the "one-by-one blocking calls" pattern PRD NFR-10 asks to
 * be compared against.
 */
class BulkSequentialWorkload extends WorkloadModuleBase {
    constructor() {
        super();
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);
        this.tenantId = roundArguments.tenantId || 'tenant01';
        this.batchSize = roundArguments.batchSize || 20;
        this.counter = 0;
    }

    _buildArgs() {
        this.counter += 1;
        const employeeId = `bench-bulkseq-w${this.workerIndex}-${this.counter}-${crypto.randomBytes(16).toString('hex')}`;
        const section = SECTIONS[this.counter % SECTIONS.length];
        const dataHash = 'sha256:benchbulk' + crypto.randomBytes(24).toString('hex');
        const updatedBy = 'benchbulkactor' + crypto.randomBytes(12).toString('hex');

        return {
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
        };
    }

    async submitTransaction() {
        // One-by-one, BLOCKING: await each submit fully before issuing the next.
        for (let i = 0; i < this.batchSize; i++) {
            await this.sutAdapter.sendRequests(this._buildArgs());
        }
    }
}

function createWorkloadModule() {
    return new BulkSequentialWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
