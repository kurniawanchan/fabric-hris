# Inventaris Dokumen — Rekonsiliasi 2 s.d. 6 Agustus 2026

> Snapshot seluruh dokumen yang dibuat atau diubah selama sesi ratifikasi PRD dan rekonsiliasi
> paket `agent-suite/` terhadap tesis (2–6 Agustus 2026). Disusun sebagai lampiran navigasi bagi
> `rencana-rekonsiliasi.md`, bukan pengganti isinya.

---

## 📄 Workspace PRD — `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/`

| Berkas | Isi |
|---|---|
| **`prd.md`** | Dokumen utama — ratifikasi PB-4/PB-5, §5.2 kriptografi final, 37 FR, 11 NFR, matriks UU PDP 12-baris, status: **final** |
| **`errata-tesis.md`** | 14 cacat tesis (E-1…E-14) berjenjang prioritas |
| **`rencana-rekonsiliasi.md`** | 40 langkah rekonsiliasi, 6 gelombang, status terkini |
| **`review-rubric.md`** | Tinjauan kualitas PRD 7 dimensi |
| **`source-thesis-extract.txt`** | Ekstrak teks tesis (sumber ratifikasi) |
| **`inventaris-dokumen-2026-08-06.md`** | Berkas ini |
| `.memlog.md` | Jejak audit kronologis — 61+ entri |

## 🏛️ ADR — `agent-suite/05-adr/` (21 total)

| # | Isi | Status |
|---|---|---|
| 0001–0010 | ADR asli (2026-07-13) | 6 *Superseded*, 4 masih *Accepted* (0002, 0005, 0007, 0008) |
| **0011** | Skema *digest* per-section | *Superseded* sebagian oleh 0021 |
| **0012** | Konsorsium tiga organisasi | Accepted |
| **0013** | Channel-per-tenant | Accepted |
| **0014** | Pencatatan *in-band* | Accepted |
| **0015** | *Crypto-shred* erasure | Accepted |
| **0016** | IPFS Private Cluster | Accepted |
| **0017** | Tier relasional (OQ-2, rekomendasi kerja) | Accepted |
| **0018** | Evaluasi kinerja masuk cakupan | Accepted |
| **0019** | Amandemen domain kunci (4 domain) | *Superseded* sebagian oleh 0021 |
| **0020** | Kontrak chaincode + batas *hashing* | Accepted |
| **0021** | `employeeKey_i` acak independen — menutup T16b | Accepted |

Plus `README.md` (indeks tersinkronkan) dan `_TEMPLATE.adr.md` (kosakata status 3-nilai).

## 🏗️ Desain Solusi — `agent-suite/00-architecture/`

- `solution/data-model.md`, `fabric-network-design.md`, `integration-design.md`, `api-contracts.md` — ditulis ulang total
- `diagrams/network-topology.mmd`, `anchor-write-sequence.mmd`, `integration-flow.mmd` — digambar ulang

## 🔒 Keamanan — `agent-suite/08-security/`

`security-architecture.md`, `control-matrices.md`, `trust-boundaries.mmd` — STRIDE diturunkan ulang (T9/T10/T11/T15/T16 + T17–T21 baru)

## 🗺️ Roadmap — `agent-suite/06-roadmap/`

- **`dsrm-phase-artifact-map.md`** *(baru)* — peta 6 fase DSRM → artefak → teori BAB II
- `implementation-backlog.md` — ditulis ulang, 38 item build
- `test-strategy.md` — ST-1 dst.
- `dsrm-roadmap.md` — tidak diubah

## ⚠️ Risiko & Eksekusi

- `agent-suite/10-risk/risk-register.md` — 2 kelas risiko baru
- `agent-suite/11-execution/knowledge-graph.md`, `grounding-gaps.md` — ditulis ulang; `execution-strategy.md` tidak diubah

## 📚 Pustaka Konteks — `agent-suite/context/` (18 dokumen disentuh)

`ADR.md`, `BLOCKCHAIN-DATA-MODEL.md`, `BLOCKCHAIN-INTEGRATION.md`, `CRYPTOGRAPHY.md`, `DSRM.md`, `FABRIC-CHAINCODE.md`, `FABRIC-CHANNELS.md`, `FABRIC-MSP.md`, `FABRIC-ORDERING.md`, `FABRIC-POLICIES.md`, `FABRIC-PRIVATE-DATA.md`, `GO-ARCHITECTURE.md` (§8 saja), `OWASP-API.md`, `OWASP-ASVS.md`, `PHP-INTEGRATION.md`, `PRIVACY-BY-DESIGN.md`, `SECURITY-BY-DESIGN.md`, `ZERO-KNOWLEDGE-PROOF.md`

## 🛠️ Skill — `agent-suite/skills/`

- `privacy-by-design/SKILL.md` + `assets/dpia-template.md` — tulis ulang terdalam, **siap menutup gap G-25**
- `dsrm-research-assistant/SKILL.md`, `zkp-designer/SKILL.md` — sapuan cepat

## 🤖 Spesifikasi Agen

`01-agents/specs/{architect,backend-engineer,fabric-architect,fabric-engineer,reviewer,security-architect}.md` + `agents/{fabric-architect,fabric-engineer,security-architect}.md` (yang terakhir ini adalah *symlink* ke `.claude/agents/` — jadi registri operasional ikut terbarui otomatis)

## 📋 Tinjauan Gate

`agent-suite/09-review/verification/g8-review-report-r2.md` *(baru)* — jujur menyatakan G8a lulus, G8b belum dimulai; `g8-review-report.md` ditandai *SUPERSEDED* di bagian atas

## 🔑 Berkas Sentral

- **`_bmad-output/planning-artifacts/project-context.md`** — 34 aturan, ditulis ulang penuh
- **`agent-suite/README.md`** — *status board* dipecah (historis vs rekonsiliasi berjalan)

---

## Yang masih menunggu (Gelombang 6 — bukan bagian inventaris ini)

| # | Item | Kenapa belum |
|---|---|---|
| 37 | Edit tesis (`.docx`) | Butuh Chandra langsung — lihat `errata-tesis.md` |
| 39 | Persetujuan ulang G9 | Keputusan sponsor |
| 40 | Jalankan Caliper | Butuh S-1 diputuskan + infrastruktur nyata |

**Total: ~50 berkas dibuat atau diubah pada rentang 2–6 Agustus 2026.**
