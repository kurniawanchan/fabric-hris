'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');
const crypto = require('crypto');

const SECTIONS = ['PERSONAL', 'EMPLOYMENT', 'EDUCATION', 'ADDITIONAL', 'PAYROLL'];

/**
 * PT-5 (batched leg): simulates a bulk operation (e.g. an annual salary-band
 * adjustment across many employees) as CONCURRENT, non-blocking submits --
 * each Caliper "transaction" tick fires `batchSize` RecordProfileSection calls
 * for DIFFERENT employeeIDs at once (Promise.all), rather than blocking on
 * each one sequentially. Compare its wall-clock time / effective TPS against
 * workload/bulk-sequential.js's one-by-one blocking version, same batchSize,
 * same total record count.
 */
class BulkBatchedWorkload extends WorkloadModuleBase {
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
        const employeeId = `bench-bulkbatch-w${this.workerIndex}-${this.counter}-${crypto.randomBytes(16).toString('hex')}`;
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
        // Fire the whole batch concurrently -- this is what "batched" means
        // here: the client does not block on transaction N before issuing
        // transaction N+1.
        const requests = [];
        for (let i = 0; i < this.batchSize; i++) {
            requests.push(this._buildArgs());
        }
        await this.sutAdapter.sendRequests(requests);
    }
}

function createWorkloadModule() {
    return new BulkBatchedWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
