// Empirical probe: does peer0.org1:7051 accept a TLS connection WITHOUT a client cert?
// (CORE_PEER_TLS_CLIENTAUTHREQUIRED=true is set on every peer per network-docker-compose.yaml)
const grpc = require('@grpc/grpc-js');
const fs = require('fs');

const base = '/Users/chan/www/hyperledger/fabric-hris/fabric-network/network/crypto-config/peerOrganizations/org1';
const rootCert = fs.readFileSync(`${base}/tlsca/tlsca.org1-cert.pem`);
const clientCert = fs.readFileSync(`${base}/users/Admin@org1/tls/client.crt`);
const clientKey = fs.readFileSync(`${base}/users/Admin@org1/tls/client.key`);

function tryConnect(label, creds) {
    return new Promise((resolve) => {
        const client = new grpc.Client('localhost:7051', creds, {
            'grpc.ssl_target_name_override': 'peer0.org1',
        });
        const deadline = new Date(Date.now() + 5000);
        client.waitForReady(deadline, (err) => {
            if (err) {
                console.log(`[${label}] FAILED: ${err.message}`);
            } else {
                console.log(`[${label}] SUCCESS: channel ready`);
            }
            client.close();
            resolve();
        });
    });
}

(async () => {
    // 1. server-TLS-only (root cert, no client cert) -- what Caliper's PeerGateway.js does
    await tryConnect('no-client-cert (Caliper PeerGateway behavior)', grpc.credentials.createSsl(rootCert));

    // 2. mutual TLS with client cert supplied
    await tryConnect('with-client-cert (mTLS)', grpc.credentials.createSsl(rootCert, clientKey, clientCert));
})();
