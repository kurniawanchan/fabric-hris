# Certificate rotation runbook — DEP-2

Backlog item: **DEP-2**. Design authority: ADR-0009 (superseded by ADR-0019, Domain A's
rotation/revocation-by-CRL rule carried forward unchanged), ADR-0019 (four-key-domain amendment —
Domain A row: "Rotated/revoked via CRL only (D15)"), `fabric-network-design.md` §3/§6, SEC T2
(`security-architecture.md`: "Key theft = impersonation until revocation"). Depends on **NET-2**
(the Fabric CA enrollment path this runbook rotates) and companion file
`identity-tls-hsm-provisioning.md` in this same directory (initial issuance — read that first;
this file covers what happens **after** issuance).

**Standing scope decision (binding).** Generic artifact. No cert in this workspace's live network
was rotated as part of producing this document; every expiry number and defect cited below is
read from this repo's own already-committed CA configs and NET-2's already-executed, verified
runbook — not invented, not run here.

**Scope boundary.** Domain A only (Fabric MSP signing keys + TLS keys). This runbook does not
rotate or touch `KEY_EMPLOYEE`, `employeeKey_i`, `DataHash` salts, or the application AES PII key —
those have no cert/expiry concept at all (they are not X.509 material) and their own
lifecycle/destruction discipline is `key-domain-separation-checklist.md`'s and ADR-0015's, not
this file's.

---

## 1. What actually has an expiry in this workspace, and what does not

| Cert class | Issuer | Expiry (from this repo's own config) | Renewable in place? |
|---|---|---|---|
| cryptogen-issued MSP signing certs (Org1, `OrgClient-tenant01`, Org3, `OrdererMSP` — what the **live network's peers/orderers actually boot from**, NET-1) | cryptogen's own throwaway local CA | Fixed at generation time by cryptogen's own default validity; `crypto-config.yaml` exposes **no expiry field** to override this — [VERIFY] the exact cryptogen default against the `hyperledger/fabric-tools:2.5` binary before relying on a specific number | **No** — cryptogen has no renew/re-enroll concept; "rotation" is regenerate-and-redistribute (§3.1) |
| cryptogen-issued TLS certs (same orgs) | Same throwaway CA | Same caveat as above | No — same as above |
| Fabric-CA-issued enrollment (MSP) certs, `signing.default.expiry` | `ca.org1`/`ca.tenant01`/`ca.org3` (own signing CA) | **`8760h` = 1 year** (`ca-tenant01-config.yaml` line 96, `ca-org1-config.yaml` mirrors it per NET-2's own no-duplication convention) | **Yes** — re-enroll the same registered identity for a fresh cert under the same key or a fresh key, no re-registration needed |
| Fabric-CA-issued TLS certs, `signing.profiles.tls.expiry` | Same CA, `tls` enrollment profile | **`8760h` = 1 year** (`ca-tenant01-config.yaml` line 112) | Yes — `--enrollment.profile tls` re-enroll |
| Fabric-CA's own root/signing CA cert, `signing.profiles.ca.expiry` / `csr.ca.expiry` | Self-signed at `fabric-ca-server start` time | **`131400h` ≈ 15 years** (`ca-tenant01-config.yaml` lines 102 and 129) | Not a "renewal" — replacing the CA root is a trust-anchor change, see §4 |
| Fabric-CA CRL | `crl.expiry` | **`24h`** (`ca-tenant01-config.yaml` line 56) — the CRL itself must be regenerated daily regardless of whether any cert was revoked, or Fabric nodes will treat it as stale | N/A — operational cadence, not a cert |

The 1-year enrollment/TLS expiry (Fabric-CA path) is the number this runbook's monitoring
thresholds (§2) are built around — it is a real, already-committed config value, not a guess.
Cryptogen's identities have no equivalent documented number in this repo; treat that as an open
item for whichever deployment actually adopts the cryptogen path in a context where expiry
matters (a real production deployment should be on the Fabric-CA path per
`identity-tls-hsm-provisioning.md` §1's recommendation, precisely to get a known, configurable
expiry and CRL-based revocation).

## 2. Cert-expiry monitoring approach

**This runbook does not author dashboards or alert rules — that is DEP-5's own artifact**
(`implementation-backlog.md`: DEP-5's DoD explicitly includes "cert expiry" alongside anchor-write
failures/latency and per-channel peer/orderer health). What this runbook fixes is (a) what must be
tracked, feeding DEP-5's inventory, and (b) the response procedure once an alert fires (§3–§5).

### 2.1 What to track, per identity

For every MSP signing cert and every TLS cert, across **all three orgs' nodes plus every
provisioned tenant's `OrgClient-<tenantID>` peer** (the same "every provisioned tenant channel"
scope QA-5's `ST-14` uses for isolation testing — cert monitoring must match that same enumeration,
not just the pilot tenant):

- Subject / SKI (to disambiguate multiple certs on the same node — MSP vs. TLS are always two
  separate certs per node, `identity-tls-hsm-provisioning.md` §2 step 4).
- `notAfter` — extract via `openssl x509 -enddate -noout -in <cert path>`. For an HSM-backed MSP
  key (§5 of the companion runbook), the private key is not exportable but the **signing
  certificate** in `signcerts/` still is a normal X.509 file — `notAfter` extraction works
  identically regardless of whether the key behind it lives in an HSM or a file.
- Issuing CA (which of `ca.org1`/`ca.<tenantID>`/`ca.org3` — needed to route the renewal request to
  the right CA and the right operator, since `OrgClient-<tenantID>` orgs are client-operated, not
  platform-operated, per §6 below).

### 2.2 Threshold recommendation — operator judgment, not a ratified requirement

**[ASSUMPTION]**, flagged as such: no PRD/ADR fixes a numeric alert threshold for cert expiry.
The following is a standard operational practice recommendation for DEP-5 to encode, not a
ratified number:

- **30 days before `notAfter`**: warning-severity alert, routed to the owning org's operator
  (Org1/Org3: platform `sre`; `OrgClient-<tenantID>`: that tenant's own operator per §6).
- **7 days before `notAfter`**: high-severity alert.
- **Past `notAfter`**: critical — the node can no longer sign/endorse/connect; this is a live
  incident, not a rotation-scheduling item anymore (a peer with an expired TLS cert fails every
  mTLS handshake; a peer with an expired MSP cert has every endorsement it produces rejected as
  invalid by every other peer's signature verification).

A 1-year Fabric-CA cert with a 30-day lead time gives roughly a monthly cadence of "nothing to do"
checks and one real rotation window per identity per year — sized for the actual `8760h` value in
§1, not an arbitrary round number.

## 3. Rotation procedure — leaf cert renewal under the SAME CA root (the common case)

This is the lightweight path: the identity keeps the same CA trust anchor, only its own
enrollment/TLS cert is replaced.

### 3.1 Cryptogen path — regenerate-and-redistribute, no true "renew"

Cryptogen has no enroll/renew verb. Rotating a cryptogen-issued identity means:

1. Take the affected node **out of service** first (this path cannot rotate a single leaf cert
   in place — the whole org's `cryptogen generate` output for that node is regenerated together).
2. Re-run `cryptogen generate` for **only** that org's fragment (never the shared
   `crypto-config.yaml` against a live network — `tenantprovision/main.go`'s own guard comment
   documents exactly why: doing so "would regenerate every org's keys and invalidate already-
   running peers' identities").
3. Redistribute the new `msp/`/`tls/` folders to the node (file-copy/volume-mount update).
4. Restart the node pointed at the new material.
5. If this identity's TLS root CA cert is embedded in **other** nodes' `tls-cross/` trust bundles
   (see `network-docker-compose.yaml`'s per-peer `tlsca.<org>-cert.pem` mounts — every peer trusts
   every other org's TLS CA statically at the compose level today), those bundles do not need
   updating for a **leaf** rotation under the same cryptogen CA root — the root itself is
   unchanged, only the leaf cert/key pair is new. This is a genuine practical advantage of not
   changing the root: no cross-org file redistribution needed, only the rotated node's own
   restart.

**Caution specific to this path:** because cryptogen has no revocation mechanism, an old,
still-`notAfter`-valid cryptogen cert that is compromised **cannot be revoked** — the only
containment available is: rotate to a fresh keypair as above, and separately update every other
node's trust bundle to **stop** trusting the old CA root entirely (a root-level operation, §4,
heavier than a routine rotation). This is a structural reason to prefer the Fabric-CA path for
anything beyond throwaway dev/test, restated from `identity-tls-hsm-provisioning.md` §1.

### 3.2 Fabric-CA path — re-enroll, no re-registration

The registered identity (`--id.name`, its secret) does not change; only the cert is refreshed:

```bash
# Re-enroll the MSP signing cert under the SAME registered identity — new keypair + cert,
# same CA, same identity name/affiliation/attributes.
export FABRIC_CA_CLIENT_HOME=<identity's existing home dir>
fabric-ca-client enroll -u https://<id>:<secret>@<ca-host>:<port> \
  --tls.certfiles <ca-tls-cert> \
  -M <fresh MSP output dir>

# Repeat for the TLS leg with the SAME --csr.hosts as originally issued (must match how peers
# dial this node — do not drop or add SANs casually mid-rotation).
fabric-ca-client enroll -u https://<id>:<secret>@<ca-host>:<port> \
  --enrollment.profile tls --csr.hosts <same hosts as original issuance> \
  --tls.certfiles <ca-tls-cert> \
  -M <fresh TLS output dir>
```

**Do not reuse the identity's original `-M` output directory in place** — enroll into a fresh
directory, verify it (§3.3), then swap it in atomically (rename/relink), so a failed enroll never
leaves the node's live MSP folder half-written.

### 3.3 The NodeOU gap applies to rotation, not just initial issuance — this is the real defect to account for

**This is NET-2's actual, empirically-found defect, and it recurs on every re-enroll, not just
the first one.** A fresh `fabric-ca-client enroll` output folder is **not** NodeOU-aware by
default — `msp/config.yaml` is simply absent from the tool's own output
(`enrollment-runbook.md` §(d), corrected 2026-08-06 against the real `hyperledger/fabric-ca:1.5`
client: "Fabric-CA-issued identity MSP folders... come back with no config.yaml at all. NodeOUs
must be written by hand"). `cryptogen` auto-generates this file every time (§3.1 above therefore
does **not** have this problem); Fabric-CA never does, on **any** enroll call, including a
rotation re-enroll.

**Concrete rotation step, do not skip:** after §3.2's re-enroll produces a fresh MSP folder, copy
the identity's already-hand-authored `config.yaml` into the new folder before swapping it in:

```bash
cp <identity's existing msp dir>/config.yaml <fresh MSP output dir>/config.yaml
```

This works unchanged **only if the CA root did not also change** — `config.yaml`'s
`Certificate:` fields reference a specific `cacerts/ca-<org>-<port>.pem` filename, and that
filename is stable across a leaf-only rotation under the same CA. If the rotation is instead a
**CA root replacement** (§4), the `cacerts/` filename and the file's own contents both change, and
`config.yaml` must be re-authored against the new filename, not blindly copied. Skipping this
step reproduces the exact defect NET-2 already found and fixed once: a peer whose MSP folder has
no `NodeOUs.Enable: true` config is not classified into the `peer`/`client`/`admin`/`orderer` OU
at all, which breaks every policy (endorsement, channel access) that depends on NodeOU
classification.

### 3.4 HSM-backed MSP key rotation (production option, per `identity-tls-hsm-provisioning.md` §5)

When the MSP signing key lives in an HSM (option, not this workspace's current state):

1. Generate a **new** key inside the HSM under the same (or a new, clearly-labeled) PKCS11
   `Label`/token — the old key is never exported, never leaves the HSM boundary, at any point in
   this procedure.
2. Re-enroll (§3.2) with the client's `bccsp` pointed at PKCS11 so the enroll call binds the new
   cert to the new HSM-resident key (`hsm.md`'s "Using an HSM with a Fabric CA" step 3 — enroll
   generates/stores the private key **inside** the HSM directly, `keystore/` stays empty).
3. Apply §3.3's NodeOU-config-copy step identically — the HSM/file-based distinction only affects
   where the private key lives, not whether `config.yaml` is auto-generated (it still is not).
4. Only after the new cert/key pair is confirmed working does the **old** HSM key/slot become a
   candidate for deletion from the HSM — per key lifecycle policy set by whoever operates the HSM,
   not fixed here.

## 4. CA root / intermediate rotation — the heavy path, requires a channel config update

If the **CA's own signing cert** is replaced (as opposed to a leaf identity's cert under an
unchanged CA root), every downstream identity's `cacerts/` trust reference changes, and — because
each org's MSP definition inside the **channel config** (not just each node's local MSP folder)
embeds that org's root/intermediate CA certs — the channel config itself must be updated via a
channel config-update transaction signed per that channel's Admins policy
(`Admins: { Type: ImplicitMeta, Rule: "MAJORITY Admins" }`, `configtx.yaml`'s own channel-profile
shape, `tenantprovision/main.go`'s inserted profile mirrors it). This is categorically heavier
than §3's leaf rotation:

1. Coordinate the timing — every node under the affected org needs its local MSP `cacerts/`
   updated in the **same maintenance window** the channel config update lands, or nodes running
   the old trust reference will reject transactions signed under the new root as untrusted (and
   vice versa) during the gap.
2. Submit the channel config-update transaction adding the new root/intermediate CA cert to the
   org's MSP definition (and, once every node is migrated and no live cert depends on the old
   root, a follow-up update removing the old one — do not remove the old root before every
   outstanding cert issued under it has itself been rotated, or you revoke every such identity's
   trust simultaneously).
3. For `OrgClient-<tenantID>` orgs specifically: this is an operation on **that tenant's own MSP
   definition** inside **that tenant's own channel** — it does not touch any other tenant's
   channel config, consistent with ADR-0013's channel-per-tenant isolation. It still needs Org1's
   participation to land (Org1 sits on every channel and the `AND` endorsement/lifecycle policies
   name it explicitly), but it is not a network-wide event across every tenant.
4. Re-verify NodeOU config for every identity re-issued under the new root (§3.3 applies again,
   this time with the `cacerts/` filename actually changed — `config.yaml` must be re-authored,
   not copied).

**This is not a routine operation** — it should be triggered only by a scheduled root-rotation
policy (out of scope to fix a cadence for here) or a suspected CA-level compromise, never bundled
casually into a leaf-cert renewal window.

## 5. Emergency revocation (compromise, not routine expiry)

Only meaningful on the Fabric-CA path (§1 — cryptogen has no revocation mechanism at all, §3.1's
caution):

```bash
fabric-ca-client revoke -u https://<ca-host>:<port> --tls.certfiles <ca-tls-cert> \
  --id.name <compromised identity name> --gencrl
```

1. Revoking marks the identity's registration revoked in the CA registry **and** generates a
   fresh CRL (`--gencrl`) — but that CRL only takes effect for nodes that actually load it.
2. The CRL must be distributed into the affected channel's config (each org's MSP definition
   carries a `revocation_list` field) via a config-update transaction — the same class of
   operation as §4, again needing that org's Admins-policy signatures.
3. Until that config-update transaction commits, other peers/orderers continue to accept
   transactions signed by the revoked identity's still-valid-looking cert — **revocation is not
   instantaneous**; treat the gap between `revoke --gencrl` and the config-update committing as
   an active exposure window, and prioritize accordingly (this is exactly SEC T2's stated
   residual: "Key theft = impersonation until revocation").
4. Rotate every **other** identity that shares the compromised identity's private key material
   (there should be none, by construction — Domain A is never shared across identities in this
   design — but confirm, do not assume).

## 6. Who executes rotation — the client-operated boundary applies here too

Restated from `identity-tls-hsm-provisioning.md` §3.4: Org1 and Org3 rotations are executed by
their respective operators (platform `sre` for Org1; Org3's TBD operator, gap G-04). **A
`OrgClient-<tenantID>` org's rotation is executed by that tenant's own operator, against their own
CA/nodes** — the platform does not reach into a client-operated CA or peer to rotate its
certs directly. The platform's role in a tenant's rotation is limited to: (a) surfacing the
expiry alert (§2, DEP-5's dashboard, scoped per-tenant) to the tenant, and (b) co-signing any
channel config-update transaction that needs Org1's Admins-policy signature (§4/§5) once the
tenant has completed their own local re-enrollment.

## 7. Caution carried over from NET-2's own operational finding

`ca-docker-compose.yaml`'s own corrected finding: `fabric-ca-server` writes **all** runtime state
(CA cert/key, sqlite registry, generated MSP) relative to the config file's own directory, not to
`FABRIC_CA_HOME`. If a CA container is ever redeployed/restarted with the config file mounted
outside the named-volume path (the original, since-corrected mistake), a freshly-rotated CA's own
material — including anything just regenerated during a root-rotation event (§4) — lands in the
container's ephemeral writable layer and is silently lost on the next container recreation, with
no error. Before any root-rotation maintenance window, confirm the CA's compose/Helm definition
still bind-mounts its config **inside** the named-volume/PVC path, exactly as `ca-docker-compose.yaml`
was corrected to do.

## Review checklist (design-artifact review, not empirical verification)

| Item | Status |
|---|---|
| Expiry numbers cited from this repo's real, committed CA configs, not invented | Reviewed — `8760h`/`131400h`/`24h` traced to `ca-tenant01-config.yaml` line numbers |
| NET-2's NodeOU-config defect explicitly carried into the rotation procedure, not just the issuance procedure | Reviewed — §3.3, called out as recurring on every re-enroll |
| Leaf-rotation vs. root-rotation distinguished, with the channel-config-update requirement named for the heavier case | Reviewed — §3 vs §4 |
| Per-tenant client-operated boundary applied to rotation, not just initial provisioning | Reviewed — §6 |
| No Domain B/B′/C material referenced anywhere in this file | Reviewed — scope boundary stated up front; this file is X.509/CRL lifecycle only |
