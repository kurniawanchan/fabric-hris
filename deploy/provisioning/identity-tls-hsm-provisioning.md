# Identity/TLS/HSM provisioning runbook — DEP-2

Backlog item: **DEP-2** (`agent-suite/06-roadmap/implementation-backlog.md`). Design authority:
ADR-0005 (identity/role mapping), ADR-0009 (superseded by ADR-0019, Domain A carried forward
unchanged), ADR-0019 (four-key-domain amendment), ADR-0021 (`employeeKey_i` independent-random),
`agent-suite/00-architecture/solution/fabric-network-design.md` §3 (Identity & CAs), SEC T2/T14
(`agent-suite/08-security/security-architecture.md`). Depends on **NET-2** (Fabric CA per org,
enrollment-runbook.md — the file-based identity path this runbook operationalizes for a
deployment) and **DEP-1** (the Helm chart whose K8s Secrets carry the material this runbook
provisions).

**Standing scope decision (binding).** This document is a GENERIC ARTIFACT — a runbook to review
and execute against a real target cluster/CA at deploy time. Nothing in this repo has applied it
against a real cloud cluster or a real HSM; the live Fabric network already running in this
workspace (3 orgs, `tenant01`'s channel) was provisioned by NET-1 (cryptogen) and NET-2 (Fabric
CA), both already executed and documented — this runbook does not redo that work, it generalizes
it into a repeatable, reviewable procedure for a **new** deployment or a **new** tenant.

**Domain-separation guardrail (standing, ADR-0009/ADR-0019/ADR-0021 — restated from
`enrollment-runbook.md`).** Every identity this runbook provisions is **Domain A** material
(Fabric MSP signing keys + TLS keys). Nothing here generates, references, or derives
`employeeKey_i`, `KEY_EMPLOYEE`, per-record salts, or any other Domain B/B′/C secret — see
`key-domain-separation-checklist.md` in this same directory for the operational enforcement of
that boundary across all four domains.

---

## 1. Two identity paths coexist in this workspace — know which one you are provisioning

| Path | Where it's used today | Cert issuer | NodeOU config | Revocation |
|---|---|---|---|---|
| **cryptogen (file-based, throwaway)** | `fabric-network/network/crypto-config/crypto-config.yaml` (NET-1) — the actual identities the live 3-org network's peers/orderers boot from | A local, non-revocable throwaway CA generated per `cryptogen generate` invocation | Auto-generated `msp/config.yaml` per identity — no manual step | **None** — cryptogen certs cannot be revoked; "rotation" is wholesale regeneration, never a CRL |
| **Fabric CA (file-based, standard)** | `fabric-network/network/fabric-ca/` (NET-2) — `ca.org1`, `ca.tenant01`, `ca.org3`, one test peer + one test HRIS identity per org enrolled and verified against the real `hyperledger/fabric-ca:1.5` binary | A real Fabric CA server per org, register+enroll workflow | **NOT auto-generated** — must be hand-authored per identity (see `cert-rotation-runbook.md` for the full defect history) | CRL via `fabric-ca-client gencrl`, channel MSP config update |

The live network's peers/orderers currently run on the **cryptogen** identities (NET-1); the
**Fabric CA** servers (NET-2) exist alongside them as a proven, verified enrollment path but are
not what the running peers currently boot from — the two paths are parallel and were never merged
`[NET-2 header: "mixing the two would present peers/orderers with identities that
../configtx/configtx.yaml's MSPDir fields... do not recognize"]`. A real production deployment
should standardize on the **Fabric CA path** (revocable, auditable registry, ABAC attribute
issuance per ADR-0005) and treat cryptogen as dev/test-only — this is Fabric's own documented
guidance, not a claim invented here (`crypto-config.yaml`'s own header: "THIS IS NOT THE
PRODUCTION IDENTITY PATH").

## 2. Org1 and Org3 signing-key/TLS provisioning

Org1 and Org3 are **platform-scaffolded, not per-tenant** — provisioned once per network
deployment, not once per tenant. The literal, verified command sequence is
`fabric-network/network/fabric-ca/enrollment-runbook.md` phases 0 and (a)/(b)/(d) — not
reproduced here (no duplication; that runbook is the authoritative, already-executed source).
Concretely, for a **new** deployment (not this workspace's already-provisioned network):

1. Bring up `ca.org1` and `ca.org3` from `ca-docker-compose.yaml` (§Phase 0.1 of
   `enrollment-runbook.md`).
2. Extract each CA's `tls-cert.pem` (not `ca-cert.pem` — enrollment-runbook.md §Phase 0.2's
   corrected finding) for use as `--tls.certfiles` in every client call.
3. Enroll each org's bootstrap admin (§(a)) — the identity registry entries already exist in
   `ca-org1-config.yaml`/`ca-org3-config.yaml`'s `registry.identities`, no `register` step needed
   for these two.
4. Register + enroll one peer identity per org, **both** an MSP enrollment cert and a **separate**
   TLS enrollment (`--enrollment.profile tls`) per node (§(b)) — two distinct certs per node,
   Domain A's own internal split (signing vs. transport).
5. Confirm NodeOUs per §(d) — hand-author `msp/config.yaml` referencing that identity's own
   `cacerts/ca-<org>-<port>.pem` filename (see §3 below and `cert-rotation-runbook.md` for why
   this step cannot be skipped or assumed automatic).
6. Repeat for every orderer node — **not** covered by `enrollment-runbook.md`'s literal scope
   (forward-pointer left there for NET-6); the same register/enroll/NodeOU-confirm sequence
   applies with `--id.type orderer`, enrolled against `ca.org1` (Org1 operates all three Raft
   consenters, ADR-0012 §2 — `OrdererMSP`'s MSPDir is domain `org1`).

Org3's real-world operator is **not fixed by any ratified requirement** (`ca-org3-config.yaml`'s
OWNERSHIP NOTE, gap G-04 residual) — this runbook stands up the technical CA a chosen
auditor-operator would eventually run; it does not resolve who that operator is.

## 3. Per-tenant `OrgClient-<tenantID>` — the repeatable enrollment procedure

**Concrete tool this runbook operationalizes:** `fabric-network/tools/tenantprovision/main.go`
(backlog item NET-7, already run once against a real second tenant, `tenant02`, on the live
network already carrying `tenant01`). This is the actual automation a real deployment would
invoke for tenant onboarding — described below as it is actually implemented, not as an imagined
alternative tool.

### 3.1 What `tenantprovision` actually does (5 steps, `main()`)

| Step | Function | What it provisions |
|---|---|---|
| 1 | `genCryptoMaterial` | Writes a per-tenant cryptogen fragment (`crypto-config-<tenantID>.yaml`, `EnableNodeOUs: true`) and runs `cryptogen generate` against **only** that fragment — the **Domain A identity/TLS material** this runbook is scoped to |
| 2 | `addConfigtxOrgAndProfile` | Inserts the new org's block + a `<TitleTenant>ChannelGenesis` profile into `configtx.yaml` (channel-config scope, NET-3's territory, not identity provisioning — described here only for completeness) |
| 3 | `genChannelBlock` | Runs `configtxgen` for the new tenant's genesis block |
| 4 | `addPeerComposeService` | Inserts + brings up `peer0.<tenantID>` in `network-docker-compose.yaml` |
| 5 | `joinNewChannel` | `osnadmin channel join` on all 3 orderers + `peer channel join` on all 4 peers (NET-5's endorsement-policy wiring is realized here by which peers join, not by a separate step) |

**Invocation:** `cd fabric-network/tools/tenantprovision && go run . <tenantID>` (e.g. `tenant03`) —
the tool hardcodes `networkDir = "../../network"`, so it must be run from its own module
directory, exactly like the `write-path-integration` Go-workspace quirk documented elsewhere in
this repo, though for an unrelated reason (a relative-path constant, not a `go.work` boundary —
`tenantprovision` has its own standalone `go.mod`, not part of `write-path-integration/go.work`).

### 3.2 Identity-provisioning-specific operational notes (Step 1 only — the rest is NET-3/NET-5's scope)

- **Idempotency guard, read before re-running.** `cryptogen` does **not** skip an org whose
  directory already exists — rerunning it regenerates a fresh keypair/CA for that org, which then
  mismatches whatever an already-running peer for that org loaded into memory at boot. This was
  found the hard way (`genCryptoMaterial`'s own comment): "re-running this tool a second time
  silently broke peer0.tenant02's TLS trust — 'certificate signed by unknown authority'". The tool
  guards this by checking `crypto-config/peerOrganizations/<tenantID>` for existence and skipping
  cryptogen entirely if found. **A rotation or re-provisioning of an already-live tenant's identity
  must never be done by re-running this tool** — see `cert-rotation-runbook.md` for the actual
  rotation procedure.
- **Scoped regeneration, verified safe.** `cryptogen` only touches the org(s) named in the
  fragment it is given — verified empirically before this tool was written (a throwaway test org
  generated cleanly with `org1`/`tenant01`/`org3`'s material byte-identical before and after).
  **Never** rerun `cryptogen` with the full shared `crypto-config.yaml` against a live network —
  that regenerates every org's keys and invalidates every already-running peer's identity network-
  wide, not just the new tenant's.

### 3.3 Honest gap: this tool does NOT automate NET-2's Fabric-CA enrollment path for a new tenant

`tenantprovision/main.go`'s own package doc says it "performs NET-2's per-tenant CA enrollment" —
that phrase is **imprecise**. Reading the actual code: Step 1 calls `cryptogen generate`
exclusively; there is no `fabric-ca-client register`/`enroll` call anywhere in this tool, and no
`ca.<tenantID>` CA server is stood up or referenced. What the tool's own comment calls "CA
enrollment" is the cryptogen identity-issuance step (§1's first path), **not** NET-2's separate
Fabric-CA-server-based flow (§1's second path). Do not read this tool as proof that Fabric-CA
enrollment is automated per-tenant — it is not, today.

A real deployment standardizing on the Fabric-CA path (recommended, §1 above) for a **new**
tenant's `OrgClient-<tenantID>` must additionally, manually:

1. Stand up `ca.<tenantID>` — copy `ca-tenant01-config.yaml`'s pattern (own `affiliations:` tree,
   own `registry.identities` bootstrap admin, own `signing.profiles`), mirroring the
   `ca.tenant01` block in `ca-docker-compose.yaml`.
2. Repeat `enrollment-runbook.md` steps (a) (bootstrap admin), (b) (peer identity, MSP **and**
   separate TLS enrollment), and (d) (hand-author `msp/config.yaml` — see §4 below) against the
   new CA.
3. Optionally repeat step (c) (HRIS-role-bearing test identity with `hris.role`/`hris.company`
   ABAC attributes, ADR-0005) if the new tenant needs an example application-facing identity
   provisioned at the same time.

None of this is currently wired into `tenantprovision`; extending it to also drive
`fabric-ca-client` for a new tenant's CA is future engineering work, not claimed as done here.

### 3.4 Ownership handoff — who actually runs this

Per `fabric-network-design.md` §6, each `OrgClient-<tenantID>` peer is, by definition, **operated
by the client itself** — it does not sit inside the platform's own deployment boundary. This
workspace's own `ca-tenant01-config.yaml` OWNERSHIP NOTE states the precedent directly: this
config was "scaffolded by fabric-architect for the prototype's single pilot tenant... a real
per-tenant onboarding flow would hand an equivalent config to the client to run themselves." The
same applies to `tenantprovision`: for this workspace's own dev/thesis network, the platform
operator runs it directly (as was done for `tenant02`); in a real multi-operator deployment, the
platform hands the client (a) the templated `crypto-config-<tenantID>.yaml` fragment / CA config
pattern and (b) this runbook's §3.1–3.3 steps, and the **client** executes them against their own
infrastructure and CA, then submits the resulting channel-join request back to the platform's
Org1 orderer-admin (the privileged operation ADR-0013 requires).

## 4. NodeOU config for Fabric-CA-issued identities — do not assume auto-generation

**Empirical finding, NET-2, 2026-08-06.** Fabric-CA-issued identity MSP folders
(`msp/{signcerts,keystore,cacerts}`) come back with **no** `config.yaml` — NodeOUs must be
hand-authored per identity, referencing that identity's own `cacerts/ca-<org>-<port>.pem`
filename (CA/port-derived, confirm with `ls` before writing, do not assume the name).
`cryptogen`-issued identities do not have this gap — cryptogen auto-generates a NodeOU-aware
`config.yaml`. See `cert-rotation-runbook.md` for the full implication of this asymmetry across
a rotation (not just initial issuance).

## 5. HSM/PKCS11 option for MSP signing keys — a production recommendation, not this workspace's current state

**Be explicit about the gap.** Neither this workspace's cryptogen path nor its Fabric CA path
(NET-2) uses an HSM today — every `ca-*-config.yaml`'s `bccsp` block reads `default: SW` (software
crypto provider), and `ca-tenant01-config.yaml`'s own comment says so plainly: "SW is a
design-stage placeholder, not a production HSM decision. TLS keys stay file-based
unconditionally." This section documents the PKCS11 option a **production** deployment should
adopt for MSP signing keys specifically — it is a recommendation, not a description of anything
already wired up in this repo.

### 5.1 What Fabric documents (`[docs: hsm.md]`) and what carries over from ADR-0009/ADR-0019 unchanged

- HSM applies to **MSP signing keys only** (peer/orderer/CA node identities). **TLS must remain
  file-based regardless** — a documented Fabric constraint (`hsm.md`: "for TLS you must use
  file-based keys"), not a choice this design or ADR-0009/ADR-0019 made independently; both ADRs
  cite `[docs: hsm.md]` directly for this split.
- Fabric communicates with an HSM via **PKCS11** exclusively.
- Prebuilt Fabric Docker images are **not** PKCS11-enabled — a production deployment must build
  its own images with `make docker GO_TAGS=pkcs11`, a build-pipeline decision this artifact set
  does not itself execute (no CI system is invoked per the standing scope decision).

### 5.2 Concrete config fields a real deployment sets (`bccsp` section, `core.yaml`/`orderer.yaml`/CA server config)

```yaml
bccsp:
  default: PKCS11
  pkcs11:
    Library: /etc/hyperledger/fabric/libCryptoki2_64.so   # vendor-supplied PKCS11 .so path — differs per HSM vendor/CloudHSM client, [VERIFY] against the actual production HSM chosen
    Label: fabric-org1-msp                                  # token/slot label — one label can back multiple keys; recommend one label per org's MSP to keep the audit boundary at org granularity
    Pin: ${HSM_PKCS11_PIN}                                  # NEVER a literal value in any file that reaches git or a Helm values.yaml — a secret reference only (see §5.4)
    hash: SHA2
    security: 256
    Immutable: false          # set true only after confirming the target HSM supports PKCS11 object-copy — Immutable:true blocks re-copying key attributes post-generation
    # AltID: <vendor-specific unique string>   # AWS CloudHSM only — assigns the Subject Key Identifier explicitly; omit for other HSM vendors
```

- **Library**: absolute path to the PKCS11 shared object, mounted into the node's container image
  (DEP-1's Helm chart is responsible for the volume mount — this runbook only names the field, it
  does not author DEP-1's chart).
- **Label**: the HSM slot/token label created by the HSM operator ahead of time (`hsm.md`'s "Before
  you begin" step 1) — recommend one label per org's MSP (`fabric-org1-msp`, `fabric-org3-msp`,
  `fabric-<tenantID>-msp`), never one shared label across orgs, so an HSM-side audit log can
  attribute a signing operation to the correct org without cross-referencing Fabric's own logs.
- **PIN**: the single most sensitive field in this block. See §5.4 — never a literal.
- **Key label at the per-node level**: Fabric's own retrieval mechanism does **not** use a
  filename — per `hsm.md`, "the Fabric node will use the subject key identifier of the signing
  certificate in the `signcerts` folder to retrieve the private key from inside the HSM." The
  `keystore/` folder of that node's MSP remains **empty** by design when HSM-backed; an empty
  `keystore/` is the expected, correct state, not a misconfiguration.

### 5.3 Provisioning flow differs by CA choice — both documented, neither is this workspace's default

| | Using Fabric CA (this workspace's NET-2 pattern) | Using your own CA (neither of this workspace's two paths) |
|---|---|---|
| CA's own signing key | Optional — only needed if the CA's own signing cert should also be HSM-protected; skip if only node identities need HSM | HSM key generated by your own CA tooling directly |
| Node private key generation | `fabric-ca-client enroll` with the client's own `bccsp` section pointed at PKCS11 (replacing the default `SW` config) generates the key **inside** the HSM during enroll — `keystore/` stays empty | Your CA generates the key inside the HSM directly, then places only the signing cert in `signcerts/` |
| Node config | `core.yaml`/`orderer.yaml`'s `bccsp` section + `mspConfigPath`/`LocalMSPDir` point at the MSP folder produced above | Same |

Applying this to this workspace's own two paths concretely: switching NET-2's `ca-tenant01-config.yaml`-style enrollment to HSM-backed MSP keys means changing the **client-side** `bccsp` block used at `fabric-ca-client enroll` time (the enrollment-runbook.md §(b) MSP-cert enroll call, not the TLS-cert enroll call — TLS stays file-based) from `SW` to `PKCS11`, while the CA **server's** own `bccsp` block may stay `SW` if only node identities (not the CA's own signing key) need HSM protection.

### 5.4 PIN handling — secret reference only, never a literal

Consistent with this repo's own confidentiality/secrets discipline (crypto material never
committed, `fabric-ca/enrollments/` excluded from version control per NET-2's own instruction):

- The PKCS11 PIN must be injected at container-start time via a **K8s Secret** (DEP-1's chart)
  referenced as an environment variable override (`CORE_PEER_BCCSP_PKCS11_PIN` /
  `ORDERER_GENERAL_BCCSP_PKCS11_PIN` / `FABRIC_CA_SERVER_BCCSP_PKCS11_PIN` — `hsm.md`'s documented
  env-var overrides for peer/orderer/CA respectively) sourced from `secretKeyRef`, or via a
  Vault-style secret-injection sidecar/agent if the target platform has one — **which** mechanism
  DEP-1's chart actually wires is that item's own decision, not fixed here; this runbook fixes
  only the requirement ("never a literal") and the field name.
  Never write the PIN into `values.yaml`, a `ConfigMap`, this runbook, or any file that reaches
  git. If in doubt, treat a PKCS11 PIN with the same handling discipline this repo already applies
  to `pseudonymKey`'s HSM-custody requirement (ADR-0019 §Consequences: "must never leave
  HSM-protected custody and must never be included in any backup that leaves that boundary") — the
  PIN is the credential that opens the equivalent boundary for Domain A.

## Review checklist (design-artifact review, not empirical verification)

Per the standing scope decision, nothing below was run against a real cluster or HSM — this is a
reviewer's checklist for the runbook itself, not a DoD claiming live verification (contrast
`enrollment-runbook.md`'s own DoD table, which **was** run against the real live network).

| Item | Status |
|---|---|
| Both identity paths (cryptogen, Fabric CA) described, neither invented | Reviewed — both cite the actual files in this repo (`crypto-config.yaml`, `enrollment-runbook.md`) |
| Per-tenant procedure describes the real `tenantprovision/main.go`, not an imagined tool | Reviewed — 5 steps enumerated from the actual `main()` function; the tool's own imprecise "CA enrollment" doc comment is called out explicitly as a gap, not repeated uncritically |
| HSM section labeled as an option/recommendation, not this workspace's current state | Reviewed — §5 opens by stating neither path uses HSM today, citing `ca-tenant01-config.yaml`'s own comment |
| PKCS11 fields named concretely (library, slot/label, PIN-via-secret, key label) | Reviewed — §5.2/§5.4 |
| No cross-reference into Domain B/B′/C material anywhere in this file | Reviewed — every generated/stored/read item above is Domain A only; see `key-domain-separation-checklist.md` for the enforcement mechanism |
