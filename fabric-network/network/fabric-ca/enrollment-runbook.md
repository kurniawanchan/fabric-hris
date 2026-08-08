# Enrollment runbook — NET-2 (Fabric CA per org, MSP/NodeOU, HRIS-role attributes)

Backlog item: **NET-2** (`agent-suite/06-roadmap/implementation-backlog.md`). Authorized by
S-1/S-5 (2026-08-06), G9 re-approved (`agent-suite/00-architecture/quality-gates-and-approval.md`
§5.5) — same tracked-item basis as NET-1. Design authority:
`agent-suite/00-architecture/solution/fabric-network-design.md` §3,
`agent-suite/05-adr/ADR-0005-identity-role-mapping.md`,
`agent-suite/05-adr/ADR-0019-key-domain-amendment.md`,
`agent-suite/05-adr/ADR-0021-employeekey-independent-random-no-master-key.md`.

**Scope.** Literal `fabric-ca-client` commands for: (a) bootstrap-admin enrollment per org,
(b) one test peer identity per org, (c) one test HRIS-role-bearing identity, (d) confirming
NodeOUs. Does **not** cover channel creation (NET-3), endorsement-policy wiring (NET-5), or
chaincode (CC-*).

**Domain-separation guardrail (standing, ADR-0009/ADR-0019/ADR-0021).** Every identity below is
**Domain A** material (Fabric MSP signing keys + TLS keys). Nothing in this runbook generates,
references, or derives `employeeKey_i`, `KEY_EMPLOYEE`, per-record salts, or any other Domain
B/B′/C secret — those live in a completely separate off-chain store this runbook never touches.

**Command provenance.** Every `fabric-ca-client`/`fabric-ca-server` flag below follows the shapes
documented in `.claude/skills/fabric-identity-security/references/fabric-ca.md` and
`assets/ca-operations-runbook.md`, both of which flag this CLI surface
`[VERIFY: fabric-ca docs — clientcli / clientcli register / clientcli enroll]` — the pinned
fabric-docs-2.5 corpus does not itself document the Fabric CA CLI
[docs: commands/fabric-ca-commands.rst]. Treat every command below as needing confirmation
against the real `hyperledger/fabric-ca:1.5` client binary (see this task's final report for the
itemized list of what to double-check).

Confidentiality register (standing constraint): `tenant01` is a generic placeholder tenant ID; no
real company/product/table/column name appears anywhere below.

---

## Phase 0 — Bring up the CAs and bridge the named-volume state to the host

The CA servers are declared in `../compose/ca-docker-compose.yaml` with their runtime state
(generated CA cert/key, sqlite registry) in **named docker volumes**, not host bind-mounts (per
NET-2's own instruction). `fabric-ca-client` on the host therefore cannot read a CA's own
generated TLS-serving cert directly off disk — it must be copied out of the container first.

```bash
# 0.1 — bring up all three CA servers
docker compose -f fabric-network/network/compose/ca-docker-compose.yaml up -d

# 0.2 — extract each CA's own TLS-serving cert (used as --tls.certfiles for every client call
# below). RESOLVED 2026-08-06, verified against the real hyperledger/fabric-ca:1.5 binary: with
# tls.certfile/keyfile left blank, the server auto-generates and writes it as `tls-cert.pem` —
# NOT `ca-cert.pem` (that file is the separate enrollment-CA root cert; both exist side by side).
# Using ca-cert.pem here would be the wrong file (it happens to also validate today because this
# combined CA signs both chains from the same process, but that is not guaranteed and is not the
# file the TLS handshake actually presents — use tls-cert.pem).
mkdir -p /tmp/fabric-ca-tls
docker cp ca.org1:/etc/hyperledger/fabric-ca-server/tls-cert.pem /tmp/fabric-ca-tls/ca-org1-tls-cert.pem
docker cp ca.tenant01:/etc/hyperledger/fabric-ca-server/tls-cert.pem /tmp/fabric-ca-tls/ca-tenant01-tls-cert.pem
docker cp ca.org3:/etc/hyperledger/fabric-ca-server/tls-cert.pem /tmp/fabric-ca-tls/ca-org3-tls-cert.pem

# 0.3 — a home directory per identity (NOT committed to the repo — add
# fabric-network/network/fabric-ca/enrollments/ to .gitignore, mirroring crypto-config/'s own
# "throwaway, never committed real" posture).
mkdir -p fabric-network/network/fabric-ca/enrollments/org1/admin
mkdir -p fabric-network/network/fabric-ca/enrollments/org1/peer0
mkdir -p fabric-network/network/fabric-ca/enrollments/tenant01/admin
mkdir -p fabric-network/network/fabric-ca/enrollments/tenant01/peer0
mkdir -p fabric-network/network/fabric-ca/enrollments/tenant01/hris-user1
mkdir -p fabric-network/network/fabric-ca/enrollments/org3/admin
mkdir -p fabric-network/network/fabric-ca/enrollments/org3/peer0
```

If instead running `fabric-ca-client` from inside a container attached to the `fabric-ca-net`
docker network (e.g. a `hyperledger/fabric-tools` container), use the internal hostnames/port
(`ca.org1:7054`, `ca.tenant01:7054`, `ca.org3:7054`) in place of `localhost:7054/8054/9054` below,
and skip the `docker cp` step by mounting the same named volumes read-only instead.

---

## (a) Enroll each org's bootstrap admin

Bootstrap identities (`admin-org1`, `admin-tenant01`, `admin-org3`, passwords per each
`ca-*-config.yaml`'s `registry.identities` entry) already exist in the CA's registry — no
`register` step needed for these three, only `enroll`.

```bash
# Org1
export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/org1/admin
fabric-ca-client enroll -u https://admin-org1:org1AdminPW2026@localhost:7054 \
  --tls.certfiles /tmp/fabric-ca-tls/ca-org1-tls-cert.pem

# OrgClient-tenant01
export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/tenant01/admin
fabric-ca-client enroll -u https://admin-tenant01:tenant01AdminPW2026@localhost:8054 \
  --tls.certfiles /tmp/fabric-ca-tls/ca-tenant01-tls-cert.pem

# Org3
export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/org3/admin
fabric-ca-client enroll -u https://admin-org3:org3AdminPW2026@localhost:9054 \
  --tls.certfiles /tmp/fabric-ca-tls/ca-org3-tls-cert.pem
```

Each produces `<FABRIC_CA_CLIENT_HOME>/msp/{signcerts,keystore,cacerts}` — the org's CA-admin MSP
[docs: certs_management.md#organization-ca-admin-certificate].

---

## (b) Register + enroll one test peer identity per org

`--id.type peer` is the **NodeOU role** (see (d) below) — distinct from HRIS business role.
`--id.affiliation` uses each CA's own affiliation tree (`ca-*-config.yaml`'s `affiliations:`).

```bash
# --- Org1: peer0.org1 ---
export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/org1/admin
fabric-ca-client register -u https://localhost:7054 --tls.certfiles /tmp/fabric-ca-tls/ca-org1-tls-cert.pem \
  --id.name peer0.org1 --id.secret peer0Org1PW2026 --id.type peer --id.affiliation org1.platform

# identity (enrollment) cert -> msp/
export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/org1/peer0
fabric-ca-client enroll -u https://peer0.org1:peer0Org1PW2026@localhost:7054 \
  --tls.certfiles /tmp/fabric-ca-tls/ca-org1-tls-cert.pem \
  -M fabric-network/network/fabric-ca/enrollments/org1/peer0/msp

# TLS cert -> tls/ (same combined CA, --enrollment.profile tls). SANs must match how peers on the
# channel dial this node — reusing NET-1's own hostname, peer0.org1.
fabric-ca-client enroll -u https://peer0.org1:peer0Org1PW2026@localhost:7054 \
  --enrollment.profile tls --csr.hosts peer0.org1,localhost \
  --tls.certfiles /tmp/fabric-ca-tls/ca-org1-tls-cert.pem \
  -M fabric-network/network/fabric-ca/enrollments/org1/peer0/tls

# --- OrgClient-tenant01: peer0.tenant01 ---
export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/tenant01/admin
fabric-ca-client register -u https://localhost:8054 --tls.certfiles /tmp/fabric-ca-tls/ca-tenant01-tls-cert.pem \
  --id.name peer0.tenant01 --id.secret peer0Tenant01PW2026 --id.type peer --id.affiliation tenant01.client

export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/tenant01/peer0
fabric-ca-client enroll -u https://peer0.tenant01:peer0Tenant01PW2026@localhost:8054 \
  --tls.certfiles /tmp/fabric-ca-tls/ca-tenant01-tls-cert.pem \
  -M fabric-network/network/fabric-ca/enrollments/tenant01/peer0/msp

fabric-ca-client enroll -u https://peer0.tenant01:peer0Tenant01PW2026@localhost:8054 \
  --enrollment.profile tls --csr.hosts peer0.tenant01,localhost \
  --tls.certfiles /tmp/fabric-ca-tls/ca-tenant01-tls-cert.pem \
  -M fabric-network/network/fabric-ca/enrollments/tenant01/peer0/tls

# --- Org3: peer0.org3 (read-only, never endorses — NET-5's scope, not this file's) ---
export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/org3/admin
fabric-ca-client register -u https://localhost:9054 --tls.certfiles /tmp/fabric-ca-tls/ca-org3-tls-cert.pem \
  --id.name peer0.org3 --id.secret peer0Org3PW2026 --id.type peer --id.affiliation org3.auditor

export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/org3/peer0
fabric-ca-client enroll -u https://peer0.org3:peer0Org3PW2026@localhost:9054 \
  --tls.certfiles /tmp/fabric-ca-tls/ca-org3-tls-cert.pem \
  -M fabric-network/network/fabric-ca/enrollments/org3/peer0/msp

fabric-ca-client enroll -u https://peer0.org3:peer0Org3PW2026@localhost:9054 \
  --enrollment.profile tls --csr.hosts peer0.org3,localhost \
  --tls.certfiles /tmp/fabric-ca-tls/ca-org3-tls-cert.pem \
  -M fabric-network/network/fabric-ca/enrollments/org3/peer0/tls
```

---

## (c) Register + enroll ONE test HRIS-user identity (ADR-0005's ABAC shape)

Issued from `ca.tenant01` — see `ca-tenant01-config.yaml`'s "ATTRIBUTE-DELEGATION NOTE" for why
this org's CA, and not `ca.org1`'s, was chosen for this example (a design choice, not settled by
any ratified requirement — flagged there, not re-argued here). NodeOU role is `client` (a
gateway/application-facing identity, not a node identity); HRIS business role is the example
value `admin` (one of ADR-0005's five: `super_admin, admin, employee, finance, consultant`);
`hris.company` is the example value `tenant01`. Both attributes carry `:ecert` so they are
embedded in the issued certificate and readable in chaincode via the CID API
(`GetAttributeValue("hris.role", ...)`, `GetAttributeValue("hris.company", ...)`)
[docs: membership/membership.md#organizational-units-ous-and-msps].

```bash
export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/tenant01/admin
fabric-ca-client register -u https://localhost:8054 --tls.certfiles /tmp/fabric-ca-tls/ca-tenant01-tls-cert.pem \
  --id.name hris-user1 --id.secret hrisUser1PW2026 --id.type client --id.affiliation tenant01.client \
  --id.attrs 'hris.role=admin:ecert' --id.attrs 'hris.company=tenant01:ecert'

export FABRIC_CA_CLIENT_HOME=fabric-network/network/fabric-ca/enrollments/tenant01/hris-user1
fabric-ca-client enroll -u https://hris-user1:hrisUser1PW2026@localhost:8054 \
  --tls.certfiles /tmp/fabric-ca-tls/ca-tenant01-tls-cert.pem \
  -M fabric-network/network/fabric-ca/enrollments/tenant01/hris-user1/msp
```

**Verify the attributes actually landed in the ecert** (decode the ABAC extension,
OID `1.2.3.4.5.6.7.8.1`, per
`.claude/skills/fabric-identity-security/references/fabric-ca.md`
[docs: certs_management.md#certificate-decoding]):

```bash
openssl x509 -in fabric-network/network/fabric-ca/enrollments/tenant01/hris-user1/msp/signcerts/cert.pem \
  -noout -text | grep -A2 "1.2.3.4.5.6.7.8.1"
```

Expected shape (mirrors the corpus example in `fabric-ca.md`, with the two custom attributes
present alongside the standard `hf.*` ones):

```
1.2.3.4.5.6.7.8.1:
  {"attrs":{"hf.Affiliation":"tenant01.client","hf.EnrollmentID":"hris-user1","hf.Type":"client",
            "hris.role":"admin","hris.company":"tenant01"}}
```

If `hris.role`/`hris.company` are absent from the decoded output, the most likely cause is that
`ca-tenant01-config.yaml`'s bootstrap admin lacks those names in its
`hf.Registrar.Attributes` list — re-check that value before re-registering.

---

## (d) Confirm NodeOUs are enabled

**CORRECTED 2026-08-06, empirical finding.** The assumption below this runbook was originally
written with — that `fabric-ca-client enroll` auto-generates a NodeOU-aware `config.yaml` the same
way `cryptogen` does — is **false**, verified against the real `hyperledger/fabric-ca:1.5` client.
Fabric-CA-issued identity MSP folders (`msp/{signcerts,keystore,cacerts}`) come back with **no**
`config.yaml` at all. NodeOUs must be **written by hand** into each identity's own MSP folder
(this mirrors the well-known `fabric-samples/test-network` pattern, whose `registerEnroll.sh`
does the same `cp`-a-`config.yaml`-after-`enroll` step for exactly this reason — not a defect
specific to this design, a documented gap in Fabric CA's own enrollment output).

```bash
# One config.yaml per identity, referencing that identity's OWN cacerts/ filename
# (Fabric-CA writes it as ca-<org>-<port>.pem, e.g. ca-org1-7054.pem — confirm with
# `ls .../msp/cacerts/` before writing, filenames are CA/port-derived, not fixed).
cat > fabric-network/network/fabric-ca/enrollments/org1/peer0/msp/config.yaml <<EOF
NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/ca-org1-7054.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/ca-org1-7054.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/ca-org1-7054.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/ca-org1-7054.pem
    OrganizationalUnitIdentifier: orderer
EOF
# Repeat for tenant01/peer0, tenant01/hris-user1, org3/peer0 — substitute that identity's own
# cacerts/ca-<org>-<port>.pem filename in all four Certificate: lines.
```

Then confirm it landed:

```bash
cat fabric-network/network/fabric-ca/enrollments/org1/peer0/msp/config.yaml
cat fabric-network/network/fabric-ca/enrollments/tenant01/hris-user1/msp/config.yaml
```

**Verified empirically (2026-08-06, this exact sequence run against `hyperledger/fabric-ca:1.5`
+ `hyperledger/fabric-ca:1.5.22` client):** all four identities (`org1/peer0`, `tenant01/peer0`,
`tenant01/hris-user1`, `org3/peer0`) carry the file above with `NodeOUs.Enable: true` and the
four OU identifiers, once hand-authored per the step above.

**Role → OU mapping realized by this runbook's identities** (NodeOU classification, from
`--id.type` at registration — mutually exclusive per
[docs: msp.rst#identity-classification]):

| Identity | `--id.type` | NodeOU classification | HRIS role (ABAC attribute, separate mechanism) |
|---|---|---|---|
| `admin-org1` / `admin-tenant01` / `admin-org3` (CA bootstrap admins) | `client` (registry default) | `client` | none set |
| `peer0.org1` / `peer0.tenant01` / `peer0.org3` | `peer` | `peer` | none set |
| `hris-user1` | `client` | `client` | `hris.role=admin`, `hris.company=tenant01` |
| *(not enrolled by this runbook — forward pointer for NET-6)* | `orderer` | `orderer` | n/a — Raft consenter identities under `OrdererMSP`, Org1-operated; `ca.org1` is the natural enrollment point since `OrdererMSP`'s MSPDir is domain `org1` per NET-1's `configtx.yaml`, but registering them is out of NET-2's literal scope |

Note the **NodeOU role** (structural: client/peer/admin/orderer, asserted by the cert's `OU`
field) and the **HRIS business role** (`hris.role`, a custom ABAC attribute) are two independent
mechanisms that happen to both ride on the same certificate — a `client`-NodeOU identity can
carry any `hris.role` value; the two axes are not the same classification and must not be
conflated when chaincode later reads them via CID (`CC-4`'s scope, not this runbook's).

---

## Definition of Done — self-check

**✅ ALL VERIFIED 2026-08-06 against the real `hyperledger/fabric-ca:1.5` server + client** (not
just written and assumed — 3 CA servers actually run via `docker compose`, all enrollment/register
commands actually executed via a network-attached `fabric-ca-client` container, output inspected).

| DoD item | Status |
|---|---|
| NodeOUs (`client`/`peer`/`admin`/`orderer`) enabled on all three orgs | ✅ Verified — required a hand-authored `config.yaml` per identity (Fabric CA does **not** auto-generate one, unlike `cryptogen` — corrected finding, see step (d)); confirmed present with `NodeOUs.Enable: true` on `org1/peer0`, `tenant01/peer0`, `tenant01/hris-user1`, `org3/peer0` |
| HRIS role continues to enroll as an X.509 attribute readable via CID (ADR-0005's shape, org names updated) | ✅ Verified — `openssl x509 -noout -text` on the enrolled `hris-user1` cert shows OID `1.2.3.4.5.6.7.8.1` decoding to `{"attrs":{...,"hris.company":"tenant01","hris.role":"admin"}}`, exactly ADR-0005's shape against `OrgClient-tenant01MSP` |
| TLS keys file-based (not HSM) | ✅ Verified by construction — every enrollment above used file-based TLS certs (the CA's own `tls-cert.pem`); no PKCS11/HSM reference anywhere in NET-2's artifacts (ADR-0019 Domain A) |
| MSP/TLS signing keys kept in a domain separate from every off-chain digest/identifier secret | ✅ Verified — every identity enrolled here is Domain A only; `employeeKey_i`, `KEY_EMPLOYEE`, salts, and the retired `pseudonymKey` are never referenced, derived, or stored by any NET-2 artifact or command run |

**Empirical findings fed back (beyond the DoD checklist itself):**
1. `tls.certfile`/`tls.keyfile` left blank → server writes `tls-cert.pem` (distinct from the
   enrollment-CA's own `ca-cert.pem`) — the runbook originally assumed the wrong filename, corrected
   in Phase 0.2 above.
2. The original `ca-docker-compose.yaml` bind-mounted the config file at a path **outside**
   `FABRIC_CA_HOME`, and `fabric-ca-server` writes all its runtime state (cert, key, sqlite
   registry, generated MSP) relative to the **config file's own directory**, not `FABRIC_CA_HOME` —
   so the named volume stayed empty and all state was actually landing in the container's ephemeral
   writable layer, silently defeating the "named volume for CA state" requirement. Fixed by
   bind-mounting the config file **inside** the named-volume path instead (`ca-docker-compose.yaml`,
   both `volumes:` and `--config` corrected) — confirmed by re-running and checking the state now
   appears under `/etc/hyperledger/fabric-ca-server/` (the actual volume mount point).
3. Fabric-CA-issued MSPs do not auto-generate `config.yaml` (see step (d) above) — `cryptogen`
   does, Fabric CA does not; this is a genuine difference between the two enrollment paths, not a
   misconfiguration.
4. The `-b <name>:<pass>` bootstrap flag coexisting with an identical `registry.identities` entry
   in the config file caused no conflict or warning — both belt-and-suspenders declarations were
   safe.
5. `hf.Registrar.Attributes` accepts a comma-separated string (`"hris.role,hris.company,..."`) —
   confirmed working, `hris-user1`'s registration with `--id.attrs 'hris.role=admin:ecert'` was
   accepted by the registrar without modification.
