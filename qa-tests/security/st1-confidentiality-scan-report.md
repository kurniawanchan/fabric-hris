# ST-1 — Full-Ledger Confidentiality Scan (P0)

Date of scan: **2026-08-07**. Scope: `ST-1`, the single highest-priority test in
`agent-suite/06-roadmap/test-strategy.md` (§0/§1) — prove **zero bytes of PII, and no reversible
derivative of PII, exist anywhere on-chain**: world state, transaction arguments, and chaincode
event payloads, swept across every provisioned tenant channel.

**Verdict: FAIL** — on the identifier-**shape** check, with zero actual PII found. The distinction
is the whole substance of this report and is developed in §3.

---

## 0. Provenance of this document — read this first

This file was compiled **after the fact, on 2026-08-07**, from the primary records of the live
`ST-1` run, each cited inline below. It is not the tool's own output, and no scan was re-run to
produce it. `fabric-network/tools/pilscan/main.go`'s own `writeReport` doc comment specifies this
exact path and states the report is "written by hand from this run's real output, not templated" —
the scan was executed and recorded in four places, but this write-up step was missed at the time.

Every figure below is traceable to one of:

| Source | What it is authoritative for |
|---|---|
| `agent-suite/06-roadmap/implementation-backlog.md` (`QA-5` row, `ST-1` bullet) | Verdict, finding count, root cause, corroboration method |
| `agent-suite/11-execution/grounding-gaps.md` (`G-31`) | Per-channel scan volumes, required shapes, candidate remediations |
| `qa-tests/evaluation/QA7-dsrm-evaluation-p1-p4.md` §5 | Combined scan volumes, the compliance-matrix bearing |
| `agent-suite/06-roadmap/test-strategy.md` (`ST-1` row) | Scan surface as ratified, control mapping |
| `fabric-network/tools/pilscan/main.go` | Method, safety constraints, check design |

Nothing here is asserted from design intent. Where two records differ, §2.1 says so explicitly
rather than picking one silently.

---

## 1. Method — why this is not a query against the API under test

`ST-1` is executed by `fabric-network/tools/pilscan`, a standalone Go tool built for it.

The naive implementation — call `GetProfileHistory` and inspect the returned JSON — would only
prove *the query API does not return PII*, re-using the exact code path under test. That is
circular, and it is not what this tool does. `pilscan` instead fetches **every raw block** from the
live orderer via the real `peer channel fetch` binary and parses the raw protobuf itself, in Go,
with `github.com/hyperledger/fabric-protos-go-apiv2` — the same wire bytes an orderer actually
persisted, not a reconstruction via a chaincode `Evaluate` call.

Two deliberately overlapping checks run per block:

1. **Structured extraction.** Walk `Envelope → Payload → ChannelHeader` (to select
   `ENDORSER_TRANSACTION` envelopes) `→ Transaction → TransactionAction → ChaincodeActionPayload`,
   then both (a) `ChaincodeProposalPayload → ChaincodeInvocationSpec.Input.Args` — the transaction
   arguments **as submitted**, not just resulting state — and (b)
   `ChaincodeEndorsedAction.ProposalResponsePayload → ChaincodeAction → Results`
   (`rwset.TxReadWriteSet → kvrwset.KVRWSet → KVWrite`, the actual world-state writes) and
   `Events` (the marshaled `ChaincodeEvent` from `record_profile_section.go`'s own
   `SetEvent("RecordProfileSection", …)`). Every extracted string is checked against a fixture
   dictionary, and known fields (`EmployeeID`/`UpdatedBy`/`DataHash`/`IpfsCIDs`) additionally
   against their required structural shape.
2. **Raw-byte backstop.** Every fetched block's *undecoded* byte stream is also scanned as a plain
   string for every fixture-dictionary entry. This is independent of check 1's parsing
   correctness: even if the protobuf walk had a bug and missed a nested structure, a literal PII
   string anywhere in the block's bytes would still be caught — Fabric neither compresses nor
   encrypts block contents at rest. This is what makes "zero bytes" a claim about the **bytes**,
   not merely about the fields the tool knew to look at.

The fixture dictionary is the exact set of synthetic-but-realistic strings this build phase's own
tests used as operational-DB content, extracted by grepping the real test sources
(`writepaths/*.go`, `gateway-client/*.go`, `ipfsclient/*.go`, `keystore/*.go`) — not guessed.

**Scan surface, per `test-strategy.md`'s ratified `ST-1` row:** world state + tx args + chaincode
event payloads + `IPFSCIDs` + all pseudonymous-identifier fields, across every provisioned tenant
channel. Event payloads were added to this surface in response to finding `S-4` at the G8 review
(`agent-suite/09-review/verification/g8-review-report.md`) — they are on-chain and immutable, and
the pre-`S-4` version of this test did not cover them. Private Data Collections are **not** part of
the scan because none exists anywhere in this design (ADR-0015); the surface is wider than the
pre-reconciliation version, not narrower.

### 1.1 Safety constraints the run operated under

`pilscan` is **read-only against the orderer**. It performs no `peer channel join`, no
`osnadmin channel join`, and no chaincode lifecycle action of any kind. It never touches
`peer0.tenant02`, which by design is joined to no channel. `tenant-tenant02` is read via
`peer channel fetch` against the orderer alone, the channel having been confirmed to exist by a
prior read-only `osnadmin channel list`.

This matters beyond tidiness: the immediately preceding item (`QA-4`'s `PT-4`) crashed four peer
containers by acting on the `tenant-tenant02` channel, the second occurrence of the `NET-7` defect
class. `ST-1` was built to sweep that channel without repeating it.

---

## 2. Scan volume

| Metric | Value | Source |
|---|---|---|
| Channels swept | 2 — `tenant-tenant01`, `tenant-tenant02` | `pilscan` `channels` |
| Blocks scanned, both channels | **232** | backlog `QA-5`; `QA7` §5 |
| Blocks scanned, `tenant-tenant01` alone | **230** | `G-31` |
| Transaction arguments checked | **36,489** | `G-31`; `QA7` §5 |
| World-state writes checked | **3,628** | `G-31`; `QA7` §5 |
| Chaincode events checked | **3,622** | `G-31`; `QA7` §5 |
| Findings | **31,895** | all four records |

### 2.1 One reconciliation, stated rather than smoothed over

`G-31` attributes the 36,489 / 3,628 / 3,622 counts to `tenant-tenant01` (230 blocks); `QA7` §5
attributes the identical three counts to both channels combined (232 blocks). The two records
agree on every transaction-level figure and differ only on block count.

That is consistent with `tenant-tenant02` contributing its 2 genesis/config blocks and **no
endorser transactions at all** — which is what the design expects of a channel that was created,
never had chaincode committed to it, and was deliberately abandoned after `PT-4`. Recording it here
as the reading that reconciles both records, **not** as a separately measured figure: no record
states `tenant-tenant02`'s per-channel envelope breakdown directly.

---

## 3. Result — the two parts, and why the order matters

### 3.1 Zero actual PII, triple-independently corroborated

**No real personal data was found anywhere on-chain.** Three mutually independent mechanisms each
returned zero hits:

1. the fixture-string dictionary check (structured extraction, §1 check 1);
2. the raw-block pattern backstop over undecoded bytes — email/SSN/name-shaped regexes (§1 check 2);
3. the verifier's own separate, independent pattern search, run during adversarial re-verification.

This is the first real test evidence for the confidentiality claim that `prd.md` §3.2 names as
"the most important open issue in this document" — previously argument-only. It holds.

### 3.2 The failure: 31,895 identifier-shape violations

The same scan found **31,895 findings, every one of them an identifier-shape violation** —
`employeeID`/`updatedBy` values persisted as raw, human-readable test-fixture strings instead of
the 64-hex-character HMAC-SHA256 pseudonyms the design requires. Representative values, from `G-31`:

```
rec3gatewaytest0000...0001
bench-bulkbatch-w0-3-...
```

Required shapes, per `api-contracts.md`'s stated digest-format convention:

| Field | Required shape |
|---|---|
| `employeeID`, `updatedBy` | `^[0-9a-f]{64}$` (HMAC-SHA256 output) |
| `dataHash`, `prevHash` | `^sha256:[0-9a-f]{64}$`, or empty |

**Root cause — a test-hygiene failure that exposed a real design gap.** Several *other* suites
across this build phase (`REC-3`, `REC-7`, `INT-2`, `ST-5`, and `QA-4`'s Caliper benchmark) called
`RecordProfileSection` directly with hand-rolled string literals for testing convenience, bypassing
`writepaths.Hooks.anchor()`'s real pseudonymization pipeline entirely. **The chaincode accepted
every one of them without complaint.**

`requiredStringArgs` in `record_profile_section.go` checks only that
`employeeID`/`updatedBy`/`dataHash`/`prevHash` are **non-empty**. It never checks they are shaped
like the pseudonyms and digests the design depends on.

### 3.3 Why this is a real finding and not merely dirty test data

Nothing at the chaincode layer stops a genuine bug, a future write path, or a misconfigured
integration from submitting a **real, directly-identifying string** as `employeeID`/`updatedBy` and
having it persist permanently and irreversibly on an append-only ledger. The confidentiality
property currently rests on **disciplined caller behaviour**, not on anything the contract
enforces — and this run is the proof that callers do in fact fail to be disciplined, because five
of them already were not.

The 31,895 non-conforming records already committed to the live demo ledger are **permanent**.
Append-only means they cannot be retroactively fixed; any remediation prevents recurrence only.

---

## 4. Disposition

- Tracked as grounding gap **`G-31`** (`agent-suite/11-execution/grounding-gaps.md`).
- Disclosed in the source itself at `fabric-network/chaincode/employeeprofilerecord/chaincode/record_profile_section.go:100`
  (`[FLAGGED GAP — G-31, found at QA-5's ST-1 full-ledger scan, 2026-08-07]`).
- Carried into `qa-tests/evaluation/QA7-dsrm-evaluation-p1-p4.md` §5 as an explicit caveat attached
  to compliance-matrix row 8 — **not** a status downgrade: what was tested passed; what was not
  tested (a caller that fails to pseudonymize) is now a named, disclosed gap rather than an
  unstated assumption.

**Two candidate closing paths, neither taken here**, recorded so the choice stays open and explicit:

1. Add chaincode-level shape validation (`^[0-9a-f]{64}$` on identifiers,
   `^sha256:[0-9a-f]{64}$` or empty on hashes) as defense-in-depth, rejecting malformed values
   before they ever reach `PutState`.
2. Ratify the current caller-trust model as a documented, deliberate design boundary via a
   superseding ADR note — explicitly, not silently.

Either is a ratified-design change (the on-chain schema and its validation are ADR-governed), so
neither was made unilaterally at test time.

---

## 5. Re-running this scan

```sh
cd fabric-network/tools/pilscan
go build -o pilscan . && ./pilscan
```

Needs the live network up (`docs/QUICKSTART.md`). Takes **5–10 minutes** — one `peer channel fetch`
per block — and exits non-zero on FAIL. It prints a `=== MACHINE-READABLE SUMMARY ===` block
(`channels_scanned=… total_blocks=… total_findings=… verdict=…`, then one `FINDING` line per
finding) designed to be quoted verbatim into this report.

**Expect a FAIL verdict with a large finding count.** That is the recorded state of this ledger, for
the reasons in §3.2, and is not itself evidence of a new problem — the 31,895 committed records are
permanent. Treat a re-run as informative only if the **finding count or class breakdown has
meaningfully changed**: a new finding class, or any finding that is not an identifier-shape
violation, would be a genuinely new result and should be investigated as one.
