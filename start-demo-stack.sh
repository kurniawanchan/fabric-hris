#!/usr/bin/env bash
#
# Demo-stack startup script.
#
# Brings the local Hyperledger Fabric network + integration-bridge +
# the HRIS app's queue worker back up after a laptop restart. Everything here
# was started manually during development (not systemd/supervisor services),
# so a reboot kills all of it -- and /tmp is cleared on macOS reboot, which
# wipes the bridge's local keystore directory, so that must be recreated
# before the bridge starts or every write will error on "fetching
# employeeKey_i: ... no such file or directory".
#
# Usage: ./start-demo-stack.sh
# Requires HRIS_APP_WORKTREE to point at your local HRIS app worktree
# (the one with the HF-integration branch checked out) and
# BRIDGE_KEYSTORE_ENCRYPTION_KEY_HEX to be set in your shell to your own
# dev-only key -- neither is filled in here on purpose, see step 4/5 below.
#
# Does NOT touch tenant02 (deliberately abandoned, do not provision/rejoin
# it) and does NOT regenerate any crypto material -- if the network's TLS
# trust is broken (x509 "certificate signed by unknown authority" in
# `docker logs peer0.org1`), that needs the full recovery procedure in
# docs/QUICKSTART.md §1, not this script.

set -euo pipefail

FABRIC_HRIS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BRIDGE_DIR="$FABRIC_HRIS_DIR/integration-bridge"
BRIDGE_PORT="8082"
KEYSTORE_DIR="/tmp/integrationbridge-keystore"
BRIDGE_LOG="/tmp/integrationbridge.log"

HRIS_APP_WORKTREE="${HRIS_APP_WORKTREE:?set HRIS_APP_WORKTREE to your local HRIS app worktree path}"
APP_WORKSPACE_CONTAINER="app-workspace-1"
QUEUE_LOG="/tmp/queue-listen.log"

FABRIC_CONTAINERS=(
  peer0.org1 peer1.org1 peer0.org3 peer0.tenant01
  orderer0.org1 orderer1.org1 orderer2.org1
  fabric-tools-net
  employeeprofilerecord-ccaas
)

echo "== 1/5: starting Fabric network containers (tenant01/org1/org3 only -- tenant02 untouched) =="
for c in "${FABRIC_CONTAINERS[@]}"; do
  if docker ps --format '{{.Names}}' | grep -qx "$c"; then
    echo "  $c already running"
  else
    echo "  starting $c ..."
    docker start "$c" > /dev/null
  fi
done

echo "== 2/5: waiting for peer0.org1 to report a clean TLS handshake (no x509 errors) =="
sleep 5
if docker logs peer0.org1 --since 20s 2>&1 | grep -qi "certificate signed by unknown authority"; then
  echo "  !! peer0.org1 is logging TLS/x509 errors -- the network's crypto trust is broken."
  echo "  !! Do NOT continue: this needs the full recovery procedure (regenerate crypto +"
  echo "  !! channel block + rejoin + redeploy chaincode), not this script. See"
  echo "  !! docs/QUICKSTART.md section 1 in fabric-hris, or ask Claude to redo the"
  echo "  !! network-recovery steps from this project's prior session history."
  exit 1
fi
echo "  looks clean"

echo "== 3/5: recreating the bridge keystore dir (wiped by macOS on reboot: /tmp is not persistent) =="
mkdir -p "$KEYSTORE_DIR"
echo "  $KEYSTORE_DIR ready"

echo "== 4/5: starting integration-bridge on :$BRIDGE_PORT (background, log at $BRIDGE_LOG) =="
if lsof -nP -iTCP:"$BRIDGE_PORT" -sTCP:LISTEN > /dev/null 2>&1; then
  echo "  something is already listening on :$BRIDGE_PORT -- assuming the bridge is already up, skipping."
else
  cd "$BRIDGE_DIR"
  BRIDGE_HTTP_ADDR=":$BRIDGE_PORT" \
  BRIDGE_PEER_ENDPOINT=localhost:7051 \
  BRIDGE_TLS_SERVER_NAME=peer0.org1 \
  BRIDGE_TLS_CA_CERT_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem \
  BRIDGE_CLIENT_TLS_CERT_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.crt \
  BRIDGE_CLIENT_TLS_KEY_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.key \
  BRIDGE_MSP_ID=Org1MSP \
  BRIDGE_SIGN_CERT_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/signcerts/Admin@org1-cert.pem \
  BRIDGE_SIGN_KEY_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/keystore/priv_sk \
  BRIDGE_CHANNEL_NAME=tenant-tenant01 \
  BRIDGE_CHAINCODE_NAME=employeeprofilerecord \
  BRIDGE_TENANT_ID=tenant01 \
  BRIDGE_IPFS_PRIMARY_API=127.0.0.1:5001 \
  BRIDGE_IPFS_REPLICA_API=127.0.0.1:5002 \
  BRIDGE_API_KEY="${BRIDGE_API_KEY:?set BRIDGE_API_KEY to your own dev-only value}" \
  BRIDGE_COMPANY_ID="${BRIDGE_COMPANY_ID:?set BRIDGE_COMPANY_ID to your local test tenant's company id}" \
  BRIDGE_KEYSTORE_DIR="$KEYSTORE_DIR" \
  BRIDGE_KEYSTORE_ENCRYPTION_KEY_HEX="${BRIDGE_KEYSTORE_ENCRYPTION_KEY_HEX:?set BRIDGE_KEYSTORE_ENCRYPTION_KEY_HEX to your own dev-only 32-byte hex key -- never commit a real one}" \
  nohup go run ./cmd/integrationbridge > "$BRIDGE_LOG" 2>&1 &
  disown
  sleep 6
  if grep -q "ready" "$BRIDGE_LOG"; then
    echo "  bridge is up: $(grep ready "$BRIDGE_LOG" | tail -1)"
  else
    echo "  !! bridge did not report ready -- check $BRIDGE_LOG"
    cat "$BRIDGE_LOG"
    exit 1
  fi
fi

echo "== 5/5: starting the HRIS app's queue worker (inside $APP_WORKSPACE_CONTAINER) =="
if ! docker ps --format '{{.Names}}' | grep -qx "$APP_WORKSPACE_CONTAINER"; then
  echo "  !! $APP_WORKSPACE_CONTAINER is not running -- start your local docker stack first (cd to your"
  echo "  !! laradock dir and 'docker compose up -d nginx php-fpm workspace redis mysql'),"
  echo "  !! then re-run this script."
  exit 1
fi

EXISTING_WORKERS=$(docker exec "$APP_WORKSPACE_CONTAINER" sh -c "pgrep -f 'yii queue/listen' | wc -l" | tr -d '[:space:]')
if [ "$EXISTING_WORKERS" != "0" ]; then
  echo "  found $EXISTING_WORKERS existing worker process(es) -- killing to avoid duplicates"
  docker exec "$APP_WORKSPACE_CONTAINER" pkill -9 -f "yii queue" || true
  # Wait for them to actually be gone rather than a fixed sleep -- pkill
  # returning doesn't guarantee the process table has caught up yet, and a
  # too-short fixed sleep here is exactly what let a stale worker overlap
  # with the freshly-started one on a previous run of this script.
  for _ in $(seq 1 10); do
    REMAINING=$(docker exec "$APP_WORKSPACE_CONTAINER" sh -c "pgrep -f 'yii queue/listen' | wc -l" | tr -d '[:space:]')
    [ "$REMAINING" = "0" ] && break
    sleep 1
  done
fi

docker exec -d "$APP_WORKSPACE_CONTAINER" sh -c "cd $HRIS_APP_WORKTREE && exec php yii queue/listen --verbose > $QUEUE_LOG 2>&1"
sleep 2
WORKER_COUNT=$(docker exec "$APP_WORKSPACE_CONTAINER" sh -c "pgrep -f 'yii queue/listen' | wc -l" | tr -d '[:space:]')
if [ "$WORKER_COUNT" = "1" ]; then
  echo "  exactly one worker running -- good"
else
  echo "  !! expected exactly 1 worker, found $WORKER_COUNT -- check manually:"
  echo "  !! docker exec $APP_WORKSPACE_CONTAINER ps aux | grep 'yii queue'"
fi

echo ""
echo "== done =="
echo "Quick sanity check -- try this from anywhere:"
echo "  curl -s 'http://localhost:$BRIDGE_PORT/v1/profile-sections/history?employeeInternalID=YOUR_TEST_EMPLOYEE_ID&companyId=YOUR_TEST_COMPANY_ID' \\"
echo "    -H \"X-Api-Key: \$BRIDGE_API_KEY\" -H \"X-Company-ID: \$BRIDGE_COMPANY_ID\""
echo ""
echo "If the HRIS app's .env FABRIC_BRIDGE_SERVICE_URL still points to"
echo "http://host.docker.internal:$BRIDGE_PORT, no further config changes are needed."
