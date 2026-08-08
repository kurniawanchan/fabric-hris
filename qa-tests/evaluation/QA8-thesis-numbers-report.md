# QA-8 — Real Caliper Numbers for Thesis Table 4.3 / Abstract / BAB V §5.1

Backlog item `QA-8` (`implementation-backlog.md`). Per the user's explicit instruction
(2026-08-07): this is a **separate findings report only** — it does not edit the thesis `.docx`.
You decide how and where to fold these numbers into the actual document.

Source: `qa-tests/performance/RESULTS.md` (`QA-4`) and `qa-tests/performance/QA6-RESULTS.md`
(`QA-6`), both independently re-verified live, twice each, this build phase. Every number below is
a real Caliper measurement — none is estimated, rounded for presentation, or adjusted.

## 1. What the thesis currently prints (to be replaced)

| Metric | Thesis placeholder (Table 4.3 / Abstract / BAB V §5.1) |
|---|---|
| Write throughput | 850.4 TPS |
| Read throughput | 1,950.5 TPS |
| Latency | 2.85s |

`prd.md` §3.1 states plainly these were never a real measurement — this report is what replaces
them.

## 2. Real write throughput/latency — full breakdown by load point

| Requested load | Total submitted | Succeeded | Failed | Success rate | Achieved successful throughput | p50 latency | p95 latency | p99 latency | Min | Max |
|---|---|---|---|---|---|---|---|---|---|---|
| 200 TPS | 2,910 | 500 | 2,410 | 17.2% | ~36 TPS | 62.86s | 69.49s | 70.18s | 49.52s | 71.79s |
| **500 TPS** | 7,562 | 506 | 7,056 | 6.7% | ~208 TPS | **21.08s** | 33.63s | 34.11s | 14.37s | 35.34s |
| 1000 TPS | 6,976 | 531 | 6,445 | 7.6% | ~126 TPS | 52.02s | 54.51s | 54.69s | 33.02s | 54.94s |
| 2000 TPS | 10,323 | 507 | 9,816 | 4.9% | ~129 TPS | 53.09s | 62.57s | 63.23s | 40.84s | 63.69s |

**If Table 4.3 needs a single headline write-latency figure**, the 500 TPS row is the correct one
to cite — it is the exact load point P2's own success bar (`prd.md` §3, "<3 detik pada beban 500
TPS") is defined against. **The honest headline number is 21.08 seconds, not 2.85 — a ~7× miss
against the 3-second bar**, not a pass restated with a bigger number.

Almost all failures across all four rows are `DEADLINE_EXCEEDED` (Caliper/Fabric-Gateway commit
timeout), not application-level rejections — i.e., the network could not complete endorsement+
ordering+commit within Caliper's timeout window for the large majority of submitted transactions
at every load point tested.

## 3. Real read (verify) throughput/latency

| Requested load | Total | Succeeded | Failed | Success rate | Achieved throughput | p50 | p95 | p99 | Min | Max |
|---|---|---|---|---|---|---|---|---|---|---|
| 100 TPS | 1,000 | 1,000 | 0 | **100%** | 100.3 TPS | 7.6ms | 135.6ms | 946.4ms | 2.9ms | 1,048.1ms |

This is the number that plays the role of the thesis's "1,950.5 TPS read" placeholder — the real
figure is much lower in absolute TPS terms but the property that actually matters (sub-second read
latency, FR-12) is **met at p50/p95** and only marginally missed at the p99 tail (one read out of
1,000 exceeded 1 second).

## 4. Bulk-write comparison (batched vs. one-by-one, supplementary — `PT-5`)

| Pattern | Succeeded | Failed | Round wall-clock |
|---|---|---|---|
| Batched (concurrent) | 3/3 | 0 | 3.93s |
| Sequential (one-by-one blocking) | 6/6 | 0 | 14.10s |

Supports PRD NFR-10's own recommendation (batch, don't block one-by-one) with a real, if
small-scale, measured comparison.

## 5. Supplementary: MVCC contention on the same head key (`QA-6`, `DATA` §5.2)

| Concurrent writers (N) | Successes | `MVCC_READ_CONFLICT` rejections | Conflict rate |
|---|---|---|---|
| 2 | 1 | 1 | 50% |
| 5 | 1 | 4 | 80% |
| 10 | 1 | 9 | 90% |

All 14/14 rejections confirmed as genuine Fabric-level `MVCC_READ_CONFLICT` (gRPC status code 11),
independently cross-checked against both endorsing peers' own ledger validation logs. This is a
mechanical `(N-1)/N` property of "one winner per synchronized burst," not a general background
contention rate — do not extrapolate it to normal traffic without saying so.

## 6. The environment caveat — must accompany every number above if cited

Every measurement above was taken on **one laptop simultaneously hosting the entire 3-org/4-peer/
3-orderer Fabric network, the CCaaS chaincode server, and a 2-node IPFS+cluster pair** — confirmed
via an independent, non-Caliper raw `peer chaincode invoke` CLI call that took 20.3 seconds for a
single **unloaded** write, and via `docker stats` showing `peer0.org1` at 128% CPU before any
Caliper load was applied. **State this plainly wherever these numbers appear.** The FAIL verdict
against P2's <3s@500TPS bar is real and should not be minimized; the environment attribution is
also real and should not be omitted, since it changes what a reader should conclude the FAIL
means (a resource-constrained test environment, not a demonstrated flaw in the endorsement-policy
or chaincode design).

## 7. P2 verdict for BAB V / Table 4.3's own pass/fail line

**FAIL**, recorded against the measured figure (21.08s p50 @ 500 TPS), not the placeholder one —
per this item's own DoD.
