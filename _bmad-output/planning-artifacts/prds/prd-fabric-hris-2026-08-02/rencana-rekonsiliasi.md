---
title: 'Rencana Rekonsiliasi — Pemecahan Gate G8a/G8b + 40 Perubahan Berurutan'
status: draft
created: '2026-08-03'
project: fabric-hris
sumber: 'Sapuan dampak otomatis 7 area / 9 agen, 2 Agustus 2026'
dokumen_induk: 'prd.md'
---

# Rencana Rekonsiliasi Paket `agent-suite`

| | |
|---|---|
| **Pemicu** | Ratifikasi "tesis yang menang" (2 Agustus 2026) + adopsi §5.2 (*salt* + HMAC) |
| **Cakupan** | **315 item terdampak**, **127 blocking**, **40 perubahan berurutan**, **10 ADR baru** |
| **Dokumen induk** | [`prd.md`](prd.md) · [`errata-tesis.md`](errata-tesis.md) |

> **Apa yang sebenarnya menahan build sekarang.** Bukan lagi PB-1/PB-2/PB-3. Yang menahan adalah
> **pembatalan gate G8/G9** dan **kebuntuan tata kelola** — keduanya hanya dapat dipecah sponsor,
> bukan diselesaikan secara teknis.

---

## 1. Kebuntuan Tata Kelola

### 1.1. Bentuk kebuntuannya

```
Kriteria sukses P1, P2, P4 menuntut prototipe berjalan + harness Caliper
        ↓
Kode prototipe harus ada
        ↓
Aturan "0 baris kode prototipe" adalah syarat kelulusan G8
        ↓
G8 re-run GAGAL checklist-nya sendiri
        ↓
G9 tidak tercapai  →  build tidak berwenang
        ↓
    (kembali ke awal — tidak ada jalan keluar)
```

Sumber aturannya: `00-architecture/quality-gates-and-approval.md` §6 menjadikan "0 baris kode
prototipe aplikasi/chaincode" butir *checklist* kelulusan G8; §5.2 menjadikan pelanggarannya temuan
CONFIRMED yang memblokir G9.

### 1.2. Cakupan penegakan — lebih luas dari yang dilaporkan sapuan

Sapuan menyebut "dua belas titik penegakan". **Verifikasi saya menemukan lebih banyak:**

| Pola pencarian | Kemunculan | Berkas |
|---|---|---|
| `"0 lines"` / `"no prototype code"` / `"0 baris"` | **24** | **17** |
| `"before G9"` / `"pre-G9"` / `"design-only"` | **68** | — |

Ke-17 berkas itu mencakup: 3 spesifikasi agen (`architect`, `security-architect`, `backend-engineer`),
3 definisi agen aktif (`fabric-engineer`, `fabric-architect`, `security-architect`), 2 diagram
(`review-approval-loop.mmd`, `orchestration-flow.mmd`), 3 ADR (**0005**, **0008**, **0009**),
`execution-strategy.md`, `knowledge-graph.md`, `orchestration-and-collaboration.md`,
`implementation-backlog.md`, `quality-gates-and-approval.md`, dan `g8-review-report.md`.

> ⚠️ **Konsekuensi praktis:** aturan ini tertanam di **definisi agen yang aktif**. Selama belum
> dipensiunkan, agen mana pun yang di-*dispatch* untuk menulis kode akan **menolak** — sesuai
> instruksinya sendiri. Menandai risiko R-05 "*spent*" di *risk register* **tidak cukup**; itu
> meninggalkan kriteria yang tetap ditegakkan di 17 tempat.

### 1.3. Usulan pemecahan: G8 → G8a + G8b

| | **G8a — Sapuan Konsistensi Dokumen** | **G8b — Evaluasi Terukur Artefak** |
|---|---|---|
| **Pertanyaan** | Apakah paket desain konsisten secara internal setelah rekonsiliasi? | Apakah artefak yang dibangun memenuhi P1–P4? |
| **Kode prototipe** | **0 baris** — aturan lama tetap berlaku di sini | **Diharapkan dan diwajibkan** |
| **Masukan** | Gelombang 1–5 rencana ini | Prototipe berjalan + *harness* Caliper |
| **Keluaran** | `g8a-review-report.md` | `g8b-review-report.md` + angka terukur |
| **Metode DSRM** | *Descriptive* + *Analytical* | *Testing* + *Experimental* |
| **Verifikasi** | *Linter* referensi silang (**harus ditulis dulu — belum ada**) | Uji SC-A…SC-F + isolasi + Caliper |

**Aturan pensiun:** aturan "0 baris kode" **dipertahankan** sebagai kriteria G8a, dan **dipensiunkan
secara eksplisit** untuk G8b di seluruh 17 berkas — dengan penanda `[G8a-only]`, bukan dihapus, agar
jejak keputusannya tetap terbaca.

---

## 2. Keputusan Sponsor — ✅ **SEMUA DIPUTUSKAN 6 Agustus 2026, satu sesi**

Semula lima keputusan. OQ-1 sudah diputuskan 2 Agustus; **S-1…S-5 diputuskan bersama 6 Agustus**.
Pencatatan resmi: `quality-gates-and-approval.md` §5.5.

| # | Keputusan | Diputuskan | Konsekuensi |
|---|---|---|---|
| **S-1** | **Pemecahan G8a/G8b** + pensiun aturan kode untuk G8b | ✅ **Terima §1.3 apa adanya** | **Build G8b terbuka** — per item, mengikuti *Definition of Done* |
| **S-2** | **OQ-2** — basis data otoritatif | ✅ **Pertahankan basis data yang ada** (ADR-0017 dikonfirmasi) | Tidak ada migrasi |
| **S-3** | **OQ-6** — klaim rekonstruksi nilai asli | ✅ **Persempit klaim (opsi C+A) dikonfirmasi** | UJ-1/FR-17 sudah konsisten dengan ini sejak 3 Agustus |
| **S-4** | **PB-2 / G-23** — larut atau dipertahankan? | ✅ **Dilarutkan** | `SEC` T9/T10 pensiun; `GAPS` G-19/G-21/G-23 ditutup |
| **S-5** | **Persetujuan ulang G9** | ✅ **Disetujui** terhadap desain 6 Agustus | Build resmi berwenang |

**Yang TIDAK ikut diputuskan hari ini, dan tetap terbuka atas dasar teknisnya sendiri:** **PB-1**
(batas kepercayaan header *gateway*) dan **PB-3** (pin kanonikalisasi JCS) — keduanya *item-scoped*,
bukan pemblokir seluruh build.

---

## 3. Empat Puluh Perubahan dalam Enam Gelombang

Diurutkan agar setiap perubahan mendarat di tanah yang stabil. Merekonsiliasi dokumen desain sebelum
ADR yang mengaturnya adalah urutan yang salah.

### Gelombang 0 — Mekanika (harus lebih dulu, semuanya kecil)

| # | Artefak | Tindakan | Effort |
|---|---|---|---|
| # | Artefak | Tindakan | Effort | Status |
|---|---|---|---|---|
| **1** | Sesi sponsor | Putuskan S-1…S-5, catat di PRD §12 | S | ⏳ Terbuka |
| **2** | `05-adr/_TEMPLATE.adr.md`, `05-adr/README.md`, `context/ADR.md` | **Tambah status ketiga `Superseded by ADR-NNNN`** + field `Supersedes`/`Superseded-by`/`Ratifying-authority`. Template semula berbunyi *"Status is BINARY … Nothing else."* — **verifikasi mengonfirmasi ini** | S | ✅ **Selesai** 3 Agt 2026 |
| **3** | `fabric-skill-suite/docs/AUTHORING-CONTRACT.md` §4 | Perbaiki path korpus mati `fabric-talenta` → `fabric-hris`; longgarkan aturan `[VERIFY]` untuk topik di luar korpus (IPFS, UU PDP, PostgreSQL) | S | ✅ **Selesai** 3 Agt 2026 — §8 klausa "English throughout" **sengaja tidak disentuh**: itu tata tertib `fabric-skill-suite` yang lebih luas (skill Fabric umum, bukan artefak spesifik HRIS/tesis), sehingga keputusan bahasa Indonesia untuk *DSRM-step artifacts* tidak seharusnya menjalar ke sana |
| **4** | `.claude/skills/fabric-*` (8 symlink) | Diarahkan ke `../../fabric-skill-suite/skills/`; kedelapan skill terverifikasi termuat | — | ✅ **Selesai** 2 Agt 2026 |
| **5** | `.claude/settings.local.json` | Tambah izin Bash: `ipfs`, `ipfs-cluster-ctl`, `psql`, `npx`. Kosongkan `disabledMcpjsonServers` (kontradiksi `terraform` per sapuan — **belum diverifikasi ulang independen** apakah `terraform` benar-benar *wired* di `04-mcp/mcp-architecture.md`) | S | ✅ **Selesai** 3 Agt 2026 |

> **#2 adalah kebuntuan mekanis.** Tanpa itu, **tidak satu pun** dari 10 ADR baru dapat dicatat.
>
> **Gelombang 0 selesai 3 Agustus 2026**, kecuali #1 (sesi sponsor — milik Chandra).

### Gelombang 1 — Pengaman (mencegah build desain mati saat rekonsiliasi berjalan)

| # | Artefak | Tindakan | Effort | Status |
|---|---|---|---|---|
| **6** | `project-context.md` (blok otoritas saja) + `context/README.md` | Sisipkan blok otoritas: presedensi tesis, path sumber otoritatif, tanggal ratifikasi, kelas sitasi `[thesis: BAB …]`, dan **STOP-LIST** komponen yang sudah mati | S | ✅ **Selesai** 6 Agt 2026 |
| **7** | `implementation-backlog.md` — AS-3, AS-5, AS-6, INT-1, INT-2, CC-5, NET-3-PDC | Tandai `BLOCKED-SUPERSEDED` **di tempat**. Jangan hapus dulu, jangan tarik *story point* | S | ✅ **Selesai** 6 Agt 2026 — ketujuh item ditandai dengan alasan satu-baris |
| **8** | `11-execution/sources/README.md` | Pensiunkan HLF-6/Confluence sebagai otoritas penutup; arahkan ke `source-thesis-extract.txt` + `prd.md` | S | ✅ **Selesai** 6 Agt 2026 |

> **Gelombang 1 selesai 6 Agustus 2026.** Pemicu: sesi perencanaan integrasi nyata dengan
> `talenta-core` (modul Employee) sebagai lanjutan/eksekusi paket ini — bukan inisiatif terpisah.
>
> #7 adalah **perlindungan termurah** terhadap risiko membangun desain mati selama jendela
> rekonsiliasi. Ketujuh item itulah yang akan membangun *commitment builder* ber-*salt* per-event,
> konsumer Kafka, klien *read-back*, dan koleksi PDC — semuanya sudah tidak berlaku.

### Gelombang 2 — Tulang Belakang Persyaratan

| # | Artefak | Tindakan | Effort | Status |
|---|---|---|---|---|
| **9** | `quality-gates-and-approval.md` §5.1, §5.2, §6 | **Eksekusi keputusan S-1.** Pecah G8a/G8b; pensiunkan aturan kode di **17 berkas** (bukan 12) | M | ⏳ Terbuka — milik S-1 |
| **10** | `11-execution/grounding-gaps.md` (24 baris + header + log) | Tulis ulang sebagai satu edit. Mode **THESIS-AUTHORITATIVE**. Tutup G-01 & G-05 dengan **transkripsi verbatim** PRD §3 dan §4 — jangan diparafrase. Isi log prosedur penutupan yang kini berbunyi *"(none yet)"* | L | ✅ **Selesai** 6 Agt 2026 (`dsrm-researcher`) — G-01/G-05 ditutup verbatim; G-08 dibuka kembali; G-19/21/23 ditandai menunggu S-4; 5 gap baru (G-25…G-29) |
| **11** | `11-execution/knowledge-graph.md` — Layer A–F, peta, rantai 1–5 | Bangun ulang tulang konsistensi. Tambahkan aktor yang **hilang padahal seluruh objektif melayaninya**: *pemverifikasi independen milik klien enterprise* | L | ⏳ **Belum dikerjakan** — tidak ada agen yang ditugaskan; masih membaca fakta lama |
| **12** | `09-review/review-workflow.md` §4 + skrip *linter* baru | **Tulis *linter*-nya, atau hapus kata "linter" dan "machine-checked assertion".** Verifikasi saya: **tidak ada skrip apa pun** di seluruh tree — itulah sebabnya 8 symlink menggantung lolos dari G3 dan G8 | M | ⏳ **Belum dikerjakan** |

### Gelombang 3 — Sepuluh ADR ✅ SELESAI 6 Agustus 2026

| # | ADR | Isi | Menggantikan | Status |
|---|---|---|---|---|
| **13** | **ADR-0011** | Skema *digest* — per section, `SHA-256(salt ‖ JCS)`, hierarki `pseudonymKey`→`employeeKey_i` (§5.2 PRD) | ADR-0001 | ✅ `fabric-engineer` |
| **14** | **ADR-0012** | Tiga org: platform (2 peer + 3 Raft), klien enterprise, auditor *read-only*. **Risiko *ordering* semua-di-Org1 dinyatakan eksplisit**, tidak disembunyikan | ADR-0003 | ✅ `fabric-architect` |
| **15** | **ADR-0013** | Satu channel per tenant; kunci *world state* melepas prefiks `companyId` | ADR-0004 | ✅ `fabric-architect` |
| **16** | **ADR-0020** | Kontrak 4 fungsi chaincode + batas *hashing*: digest off-chain, plaintext tidak pernah jadi argumen, verifikasi *evaluate-only* (protokol B′ PRD §7) | *baru* | ✅ `fabric-engineer` — dikonfirmasi **paling berbahaya**, sekarang tertutup |
| **17** | **ADR-0014** | Pencatatan *in-band*; Kafka = 0, *anchor-service* = 0. Digroundkan ke 5 jalur tulis nyata `talenta-core` (internal, tidak ditulis literal) | ADR-0006 | ✅ `fabric-engineer` |
| **18** | **ADR-0015** + **ADR-0016** + **ADR-0017** + **ADR-0019** | Erasure (crypto-shred); IPFS Private Cluster; *tier* relasional (S-2, rekomendasi kerja); **empat** domain kunci dengan adjudikasi *custody*/*destroy-ability* — **menemukan T16b** (lihat sintesis) | ADR-0010, ADR-0009 | ✅ `security-architect` (0015/0019) + `fabric-architect` (0016/0017) |
| **19** | **ADR-0018** | Kinerja masuk cakupan; reuse *harness* Caliper `fabric-performance` (perbaiki `SUT_BIND` 2.2→2.5); `hotKeyFraction` sudah ada = kontensi MVCC adalah konfigurasi bukan kode baru | *baru* | ✅ Ditulis langsung — **terlewat dari *dispatch* awal**, ditutup manual |
| **20** | `05-adr/README.md` | Sinkronisasi indeks terpusat: 20 baris, status *Superseded-by* utuh, peta gap→ADR diperluas | — | ✅ **Selesai** — disinkronkan terpusat setelah keempat agen kembali |

### Gelombang 4 — Dokumen Desain (sebagian besar SELESAI)

| # | Artefak | Tindakan | Effort | Status |
|---|---|---|---|---|
| **21** | `implementation-backlog.md` §1 + `dsrm-roadmap.md` | Tulis ulang gate *pre-build* penuh | M | ⏳ **Sebagian** — 7 item ditandai BLOCKED-SUPERSEDED (Gelombang 1 #7), belum ditulis ulang penuh dengan NET-6/7/8 baru |
| **22** | `solution/data-model.md` (12 bagian) | Tulis ulang total → `EmployeeProfileRecord` | L | ✅ **Selesai** `fabric-engineer` |
| **23** | `solution/fabric-network-design.md` §1–§8 | Tiga org, channel-per-tenant | L | ✅ **Selesai** `fabric-architect` |
| **24** | `solution/integration-design.md` + `api-contracts.md` | Alur in-band, kontrak chaincode | L | ✅ **Selesai** `fabric-engineer` |
| **25** | `00-architecture/diagrams/*.mmd` | Gambar ulang dari nol | M | ✅ **Selesai** — `network-topology.mmd`, `anchor-write-sequence.mmd`, `integration-flow.mmd` |
| **26** | `07-repo-structure/repository-structure.md` | `chaincode/anchorcc`→`employeeprofilerecord`; tambah `ipfs/`, Caliper, *store* relasional | M | ⏳ **Belum dikerjakan** |
| **27** | `08-security/security-architecture.md` + `control-matrices.md` | Turunkan ulang model ancaman | L | ✅ **Selesai** `security-architect` — T9/T10 ditandai "kemungkinan larut" (bukan dihapus, S-4 belum final); T11/T15 pensiun; **T16 dipecah T16a/T16b**; T17-T21 baru (IPFS ×4, onboarding) |
| **28** | `06-roadmap/test-strategy.md` §0–§4 | ST-1 tetap satu uji P0 | L | ✅ **Selesai** `security-architect` |
| **29** | `implementation-backlog.md` (tulis ulang penuh) + `context/DSRM.md` §4 | Item baru NET-6/7/8, QA-8 | L | ⏳ **Sebagian** — `context/DSRM.md §4` ✅ selesai (`dsrm-researcher`); `implementation-backlog.md` penuh belum |
| **30** | `10-risk/risk-register.md` R-01…R-15 | Kelas risiko baru | M | ✅ **Selesai** `security-architect` — **dua** kelas baru (INTEGRITAS RISET R-16…R-21, Legal/Kepatuhan R-22/R-23) + R-24…R-30 risiko baru dari IPFS/in-band |

### Gelombang 5 — Agen, Konteks, Front Matter

| # | Artefak | Tindakan | Effort | Status |
|---|---|---|---|---|
| **31** | `01-agents/` + `agents/` + `skills/` | Arahkan ulang roster | L | ✅ **Selesai** 6 Agt — `fabric-architect`/`fabric-engineer`/`security-architect`/`dsrm-researcher` masing-masing memperbaiki spec sendiri + `architect.md`/`reviewer.md`/`backend-engineer.md`. **Temuan:** `.claude/agents/*.md` adalah *symlink* ke `agent-suite/agents/*.md` (bukan salinan terpisah) — perbaikan di sana otomatis ikut ke registri operasional |
| **32** | `skills/{privacy-by-design, dsrm-research-assistant, zkp-designer}` | Tulis-ulang terdalam + template DPIA | M | ✅ **Selesai** 6 Agt `security-architect` — template DPIA (`assets/dpia-template.md`) ditulis ulang, siap menutup **G-25** begitu diinstansiasi |
| **33** | `agent-suite/context/` — dokumen inti | Tulis ulang pustaka *grounding* | L | ✅ **Selesai** 6 Agt — terbagi 3 agen: `fabric-engineer` (BLOCKCHAIN-DATA-MODEL, BLOCKCHAIN-INTEGRATION, PHP-INTEGRATION, GO-ARCHITECTURE§8), `security-architect` (CRYPTOGRAPHY, PRIVACY-BY-DESIGN, SECURITY-BY-DESIGN, OWASP-ASVS, OWASP-API, ZERO-KNOWLEDGE-PROOF), `fabric-architect` (10 stub FABRIC-*, 6 diperbaiki), `dsrm-researcher` (DSRM §1-3, ADR.md meta) |
| **34** | `project-context.md` — aturan lengkap | Lengkapi *stub* #6 | M | ✅ **Selesai** 6 Agt — ditulis ulang penuh langsung (34 aturan) |
| **35** | `agent-suite/README.md` (status board) | Perbaiki *status board* usang | M | ✅ **Selesai** 6 Agt — dipecah jadi tabel historis (G0-G9, ditandai G8/G9 VOID) + tabel rekonsiliasi berjalan |
| **36** | `prd.md` §9, §10, §11 | Lengkapi §9 | — | ✅ **Selesai** — §9 matriks 12-baris terkoreksi, §10 lengkap, §11 terisi manifest + T16b/ADR-0021 |

### Gelombang 5 — hasil tambahan di luar rencana awal

Ditemukan dan diperbaiki dalam perjalanan, bukan direncanakan sejak awal:

- **`knowledge-graph.md` (Gelombang 2 #11)** — dikerjakan `dsrm-researcher`. Perbaikan terpenting:
  aktor **A7 — pemverifikasi independen milik klien** sebelumnya **tidak ada** di Layer A, padahal
  seluruh objektif prototipe (§2.1 PRD) melayaninya.
- **`implementation-backlog.md` (Gelombang 4 #21/#29, tulis ulang penuh)** — dikerjakan
  `fabric-engineer`. 38 item build final (`NET`×8 · `CC`×5 · `REC`×7 · `INT`×5 · `DEP`×5 · `QA`×8),
  item lama diarsipkan di §6 Lampiran (jejak audit, bukan dihapus).
- **`repository-structure.md` (Gelombang 4 #26)** — dikerjakan `fabric-architect`. Ditemukan dan
  diperbaiki **dua kebocoran nama nyata yang sudah ada sebelum sesi hari ini** (`../talenta-core/`,
  `employee-management-service`) — bukan kebocoran baru, tapi warisan lama yang baru terlihat.

### Gelombang 6 — Tesis, Gate, Pengukuran

| # | Artefak | Tindakan | Effort | Status |
|---|---|---|---|---|
| **37** | Naskah tesis | Eksekusi errata **E-1…E-14** menurut *tier*-nya | L | ⏳ **Tidak dapat dieksekusi agen** — berkas `.docx`, butuh Chandra langsung |
| **38** | `g8-review-report.md` → `g8-review-report-r2.md` | Tandai r1 **SUPERSEDED**, tulis r2 | M | ✅ **Selesai** 6 Agt — r2 menyatakan jujur: **G8a lulus, G8b belum dimulai** (belum ada kode, belum ada Caliper) |
| **39** | `quality-gates-and-approval.md` §5 | **Persetujuan ulang G9** (S-5) | S | ⏳ **Keputusan sponsor** — belum diputuskan Chandra |
| **40** | `06-roadmap/` — loop QA-5 → QA-8 | Jalankan Caliper, tulis balik ke tesis | L | ⏳ **Terhalang S-1 + infrastruktur nyata** — tidak dapat dijalankan di lingkungan ini |

---

## 4. Jalur Kritis

**Sepuluh langkah** menuju build yang sah. Sisanya *defence-critical* dan berjalan **paralel**, bukan
mendahului.

| # | Langkah | Item |
|---|---|---|
| 1 | Sesi sponsor — S-1…S-5 | 1 |
| 2 | Mekanika tata kelola | 2, 3, 5 |
| 3 | Pengaman | 6, 7, 8 |
| 4 | Empat ADR *blocking* | 13, 14, 15, 16 |
| 5 | Tutup gate *pre-build* | 21 |
| 6 | Dua dokumen yang dibaca item build pertama | 22, 23 |
| 7 | Bangun jaringan | NET-1 → NET-2 → NET-6 → NET-4/7 → NET-8 |
| 8 | Chaincode | CC-1 → CC-2 (dengan *retry* MVCC) → CC-6 |
| 9 | Aplikasi + pengukuran | INT-6 → INT-3 → INT-4 → Caliper → QA-8 |
| 10 | G8b + persetujuan ulang G9 | 38, 39 |

**Aturan paralelisasi:** langkah 7–9 boleh mulai **begitu langkah 6 mendarat DAN langkah 1 menyelesaikan
gate**. Pekerjaan keamanan, pustaka konteks, *risk register*, dan PRD §9 harus selesai **sebelum
langkah 10**, bukan sebelum langkah 7.

### 4.1. Dua temuan teknis yang masuk jalur kritis

| Temuan | Isi | Item |
|---|---|---|
| **`SUT_BIND` salah versi** | `run-caliper.sh` mem-*bind* ke `fabric:2.2`, bukan **2.5** | 40 |
| **`hotKeyFraction` sudah ada** | `caliper-workload-write.js` sudah mengimplementasikannya — uji kontensi MVCC adalah **konfigurasi, bukan kode baru** | 40 |

---

## 5. Yang Sudah Dikerjakan

| Tanggal | Item | Perbaikan | Bukti |
|---|---|---|---|
| 2 Agt 2026 | **#4** | 8 symlink `.claude/skills/fabric-*` diarahkan dari `fabric-talenta` (tidak ada) ke `../../fabric-skill-suite/skills/` | Kedelapan skill Fabric terverifikasi termuat harness |
| 2 Agt 2026 | **#36** (sebagian) | PRD §10 lengkap; §11 terisi manifest; §9 sebagian | `prd.md` 819 baris |
| 2 Agt 2026 | **OQ-1** | Diadopsi → §5.2 PRD; **ST-1 lulus utuh tanpa dipecah** | `prd.md` §5.2 |

---

## 6. Catatan Validitas Rencana Ini

**Sapuan dijalankan dengan asumsi "tanpa *salt*"** — ratifikasi yang berlaku saat itu. Karena OQ-1
kemudian **diadopsi**, sekitar **40 dari 315** rekomendasi berubah arah:

| Rekomendasi sapuan | Status |
|---|---|
| *"Hapus salt, catat sebagai risiko residual diterima"* | ❌ **Tidak berlaku** |
| *"ST-1 harus dipecah menjadi ST-1a / ST-1b"* | ❌ **Tidak berlaku** — ST-1 tetap satu uji P0 |
| *"T16 INVERTS — tersedia bagi setiap pembaca channel"* | ❌ **Tidak berlaku** — `pseudonymKey` kembali menjadi kontrol nyata |
| *"Tambah keterbatasan kelima ke BAB IV 4.5 soal hash dapat dibalik"* | ❌ **Tidak berlaku** |

**§5.2 PRD yang mengikat.**

Selain itu: **62 dari 68 divergensi** dari sapuan pertama **tidak diverifikasi adversarial** — batas
`VERIFY_CAP=6` yang saya tetapkan sendiri, bukan temuan bahwa masalahnya hanya lima. Item-item dalam
rencana ini yang berasal dari 62 itu **belum melewati penyangkalan**.
