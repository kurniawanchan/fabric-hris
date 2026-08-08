# Peta Fase–Artefak DSRM — Prototipe Integritas Data Profil Karyawan

> **Peta MASTER.** Dokumen ini memetakan **keenam aktivitas DSRM** (Peffers, Tuunanen, Rothenberger &
> Chatterjee, 2007; direvisi 2022) **secara spesifik untuk prototipe ini** — bukan definisi ulang
> metodologinya (itu tetap satu rumah: [`../context/DSRM.md`](../context/DSRM.md) §1–§2) — ke tiga
> hal sekaligus untuk setiap aktivitas: **(a)** artefak konkret yang sudah ada atau harus ada di
> repo ini, **(b)** kerangka teori BAB II tesis yang mengikat aktivitas/keputusan itu (CIA Triad,
> Privacy-by-Design, Security-by-Design, TOE Framework, Trust Theory, atau metodologi DSR/DSRM itu
> sendiri), dan **(c)** status kejujuran saat ini.
>
> **Otoritas.** Sejak 2026-08-02/03, **tesis menang** di mana pun dokumen ini tumpang tindih dengan
> paket desain lama — lihat blok "AUTHORITY NOTICE" di
> [`_bmad-output/planning-artifacts/project-context.md`](../../_bmad-output/planning-artifacts/project-context.md).
> Sumber pengesahan: `prd.md` (final) + `errata-tesis.md` + `rencana-rekonsiliasi.md`, ketiganya di
> `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/`.
>
> **Fakta yang paling mudah dilupakan saat membaca peta ini:** **Gate G8 dan persetujuan G9 BATAL**
> `[prd: §11.1]` — bukan "perlu ditinjau ulang", tapi batal menurut definisinya sendiri
> (`00-architecture/quality-gates-and-approval.md` §5.1(b): G9 adalah persetujuan atas *satu set
> asumsi*, dan enam anggota set itu kini diketahui salah). Artinya paket ini **hari ini** berada di
> posisi *sebelum* G8/G9 tercapai lagi, walau status board `README.md` masih mencatat kedua gate itu
> ✅ dari siklus lama. **Tidak ada baris di bawah yang boleh dibaca sebagai "selesai" jika ia berada
> di sisi G8/G9 atau sesudahnya** — paling jauh ia "selesai untuk versi desain yang sudah disupersedi".

---

## 0. Cara membaca dokumen ini

**Kelas sitasi** (mengikuti `project-context.md` AUTHORITY NOTICE):

| Kelas | Arti |
|---|---|
| `[thesis: BAB <bab> §<n>]` | Kutipan/fakta dari naskah tesis. Sitasi di dokumen ini merujuk pada teks yang benar-benar bisa diverifikasi ulang di `source-thesis-extract.txt` (ekstraksi teks pra-revisi-InfoSec, dipakai `errata-tesis.md` untuk audit) — lihat **§11** untuk batas verifikasi terhadap revisi InfoSec terbaru. |
| `[prd: §<n>]` | Bagian dari `prd.md` (status: final) — sumber pengesahan yang berlaku. |
| `[errata: E-<n>]` | Temuan bernomor di `errata-tesis.md`. |
| `[rekonsiliasi: Gelombang <n> #<item>]` | Item bernomor di `rencana-rekonsiliasi.md`. |
| `[code: …]` | Artefak nyata di repo `agent-suite/` — path relatif terhadap `agent-suite/`. |
| `[ASSUMPTION] (gap G-##)` | Spesifik berbentuk-kebutuhan yang belum diratifikasi — lihat [`../11-execution/grounding-gaps.md`](../11-execution/grounding-gaps.md). |

**Legenda status:** ✅ Selesai & masih berlaku · 🟡 Selesai untuk versi lama, perlu ditulis ulang ·
⏳ Menunggu keputusan sponsor (S-1..S-5, §9) · ⛔ Terhalang / belum boleh dimulai.

**Register.** Dokumen ini memakai register generik yang sama dengan `prd.md` — "platform SaaS HRIS
multi-tenant komersial", "penyedia platform", "modul Employee" — dan **tidak** menyebut nama
platform/perusahaan nyata secara literal, termasuk saat merujuk grounding internal. Lihat **§11**.

**Tipe artefak** (March & Smith, 1995, dibawa ke Hevner et al., 2004 — lihat `DSRM.md` §1): setiap
aktivitas diberi label **construct** (kosakata/konsep), **model** (abstraksi/representasi), **method**
(praktik/algoritma), atau **instantiation** (sistem berjalan — hanya Fase 4, pasca-G9).

---

## 1. Tabel ringkas

| # | Aktivitas DSRM (Peffers et al., 2007) | Gate paket | Tipe artefak dominan | Kerangka teori BAB II utama | Status hari ini |
|---|---|---|---|---|---|
| 1 | Identifikasi Masalah & Motivasi | **G0** | construct | CIA Triad (Integrity) · Trust Theory (motivasi) | 🟡 substansi diratifikasi; representasi paket (`knowledge-graph.md`) basi |
| 2 | Perumusan Tujuan Solusi | **G1** | construct | Trust Theory · CIA Triad (P1) · UU PDP/PbD (P4) | ✅ diratifikasi manusia 2026-08-02 (`prd.md` §2–§4) |
| 3 | Perancangan & Pengembangan | **G2–G7** | model + method | CIA · PbD · SbD · TOE Framework | 🟡/⛔ — 6 dari 10 ADR lama disupersedi; 10 ADR baru **belum ditulis** |
| 4 | Demonstrasi | **pasca-G9** | instantiation | Trust Theory (dipentaskan) · CIA-Integrity (dibuktikan) | ⛔ belum boleh dimulai — menunggu S-1 & S-5 |
| 5 | Evaluasi | **G8** | method | CIA · UU PDP (Analytical baru) · Trust Theory (FGD) · DSR/DSRM sendiri | ⛔ PASS **batal** (`prd.md` §11.1); G8a/G8b belum berjalan |
| 6 | Komunikasi | **G9** | construct | DSR/DSRM sendiri (Fase 6) · Trust Theory (narasi UJ-1) | ⛔ persetujuan **batal**; republikasi menunggu S-5 |

---

## 2. Aktivitas 1 — Identifikasi Masalah & Motivasi (G0)

**Definisi Peffers:** mendefinisikan masalah dan mengapa ia penting; menjustifikasi nilai artefak
`[brief: agents-guide.md]` / `DSRM.md` §1. Tesis sendiri menempatkan aktivitas ini di **Tahap 1**,
dioperasionalkan sebagai *"analisis dokumentasi keamanan publik, wawancara semi-terstruktur,
threat modeling STRIDE per section profil"* `[thesis: BAB III §3.4.1]`.

### Artefak konkret

| Artefak | Status keberadaan | Sitasi |
|---|---|---|
| Tiga kelemahan (W-1 tidak ada jaminan integritas kriptografis, W-2 pembuktian forensik terbatas, W-3 isolasi antartenant logis bukan kriptografis) | ✅ ada — `prd.md` §1.2 | `[prd: §1.2]`, cocok dengan `[thesis: BAB IV §4.1.2–§4.1.4]`, `[thesis: BAB I §1.2]` (RQ1) |
| Dampak bisnis (kerugian finansial, kerentanan hukum, kepatuhan tidak terbukti) | ✅ ada | `[prd: §1.3]` |
| `11-execution/knowledge-graph.md` (peta aktor→PII→layanan→kapabilitas Fabric→frame→fase) | 🟡 ada tapi **basi** — masih memuat topologi dua-org, anchoring berbasis event, single-channel+PDC yang sudah di-STOP-LIST-kan | `[code: 11-execution/knowledge-graph.md]`; perbaikan = `[rekonsiliasi: Gelombang 2 #11]` (**belum dikerjakan**, bukan tugas peta ini) |
| `11-execution/grounding-gaps.md` (register 24→sekarang ~29 gap) | ✅ ditulis ulang bersamaan dengan peta ini, mode THESIS-AUTHORITATIVE | `[code: 11-execution/grounding-gaps.md]`, `[rekonsiliasi: Gelombang 2 #10]` |
| Risk register (seeded) | 🟡 ada, belum menambahkan kelas risiko ketiga "Integritas Riset" yang direkomendasikan sapuan | `[code: 10-risk/risk-register.md]`, `[rekonsiliasi: Gelombang 4 #30]` (**belum dikerjakan** — bukan berkas yang boleh disentuh peta ini, lihat batasan di README repo) |
| Instrumen RQ1 yang dijanjikan (wawancara 10–15 partisipan, STRIDE per section, analisis tematik Miles–Huberman–Saldaña) | ⛔ dijanjikan `[thesis: BAB III §3.4.1]`, **tidak pernah dilaporkan** di BAB IV — `[errata: E-4]`. **Diklaim sudah diperbaiki di revisi InfoSec dengan menghapus janji (bukan menambah laporan)** — lihat **§11** untuk batas verifikasi klaim ini di lingkungan ini. |

### Kerangka teori BAB II yang mengikat

- **CIA Triad — dimensi Integrity.** BAB II §2.3 menyatakan *Integrity* sebagai dimensi paling
  relevan penelitian ini: *"data profil karyawan tidak dapat diubah tanpa otorisasi selama
  penyimpanan, transmisi, maupun pemrosesan"* `[thesis: BAB II §2.3]`. W-1/W-2 di `prd.md` §1.2
  **adalah** pelanggaran dimensi ini secara konkret (SQL `UPDATE` langsung, tanpa jejak yang dapat
  dibuktikan forensik) — problem framing aktivitas 1 secara langsung adalah "Integrity is broken,
  and nothing detects it."
- **Trust Theory — pemicu motivasi, bukan hanya kesimpulan.** BAB I §1.4 (Manfaat Praktis) sudah
  menyebut pergeseran *institutional trust* → *code-based trust* sebagai motivasi awal, bukan
  temuan belakangan `[thesis: BAB I §1.4]` — jadi Trust Theory mengikat **aktivitas 1**, bukan
  cuma aktivitas 5/6 (lihat juga `[thesis: BAB II §2.4.2]`).
- **UU PDP Pasal 8** (kebenaran/akurasi/konsistensi data pribadi) disebut sebagai bagian dampak
  bisnis sejak BAB I `[thesis: BAB I §1.2]`, `[prd: §1.3]` — anteseden langsung ke P4 (aktivitas 2).

### Status kejujuran

**Substansi masalah** (`prd.md` §1) sudah diratifikasi dan konsisten dengan tesis. **Representasi
paket** (`knowledge-graph.md`) belum menyusul — ia masih menceritakan masalah dalam kerangka desain
lama (anchoring event, dua org) yang sudah dinyatakan mati oleh STOP-LIST. Ini **bukan** blocker
untuk peta ini (di luar cakupan tugas — lihat batasan berkas), tapi **adalah** utang yang harus
ditagih di Gelombang 2 #11 sebelum siapa pun mengklaim aktivitas 1 "selesai" dalam arti penuh.

---

## 3. Aktivitas 2 — Perumusan Tujuan Solusi (G1)

**Definisi Peffers:** apa yang harus dicapai solusi yang baik, kuantitatif atau kualitatif
(`DSRM.md` §1). Tesis: **Tahap 2**, *"menetapkan kebutuhan solusi berbasis blockchain yang
meningkatkan keamanan dan kepatuhan UU PDP"* `[thesis: BAB III §3.4.2]`.

### Artefak konkret

| Artefak | Status | Sitasi |
|---|---|---|
| Objektif utama — pergeseran *institutional trust* → *code-based trust*, dapat diverifikasi independen oleh klien | ✅ diratifikasi 2026-08-02 sebagai keputusan manusia eksplisit, **menutup gap G-01** | `[prd: §2.1]` |
| Objektif sekunder OS-1..OS-3 (postur keamanan, pembeda komersial, kontribusi teoretis DSRM+Compliance-by-Design) | ✅ ada, dinyatakan **tidak diukur** sebagai penentu utama | `[prd: §2.2]` |
| Empat predikat teruji P1–P4 (bukan KPI — predikat yang dapat digagalkan satu *counterexample*) | ✅ diratifikasi, **menutup bagian "demo success metric" dari gap G-01** | `[prd: §3]` |
| Cakupan anchoring: unit = **section profil** (bukan event), lima section (PERSONAL/EMPLOYMENT/EDUCATION/ADDITIONAL/PAYROLL) | ✅ diratifikasi 2026-08-02, **menutup gap G-05**, membatalkan permukaan `EVENT_*` | `[prd: §4]` |
| Ruang Lingkup Penelitian (Fabric v2.5, channel-per-tenant, Caliper v0.5, evaluasi UU PDP) | ✅ ada di tesis, konsisten dengan `prd.md` | `[thesis: BAB I §1.5]` |
| Tujuan Penelitian (identifikasi kelemahan → rancang prototipe → evaluasi 3 dimensi) | ✅ ada, memetakan langsung ke DSRM 1/3/5 | `[thesis: BAB I §1.3]` |

### Kerangka teori BAB II yang mengikat

- **Trust Theory adalah kerangka objektif utama**, secara literal: objektif utama `prd.md` §2.1
  ("membuktikan ... dapat dilakukan sendiri oleh klien ... tanpa harus mempercayai penyedia
  platform") **adalah** operasionalisasi §2.4.2 BAB II kata demi kata — *"klien dapat secara
  mandiri menghitung hash ... tanpa bergantung pada jaminan vendor"* `[thesis: BAB II §2.4.2]`.
  Tidak ada terjemahan yang diperlukan; objektif dan teori adalah kalimat yang sama diucapkan dua
  kali.
- **CIA Triad — P1 sebagai operasionalisasi Integrity.** P1 ("Setiap manipulasi langsung di basis
  data terdeteksi melalui *hash mismatch*", ambang 100% pada lima section) `[prd: §3]` adalah
  bentuk falsifiable dari klaim BAB II §2.3 bahwa *"blockchain menegakkan Integrity melalui hash
  SHA-256 yang dirantai antar-versi ... perubahan sekecil apa pun menghasilkan hash yang berbeda
  (avalanche effect)"* `[thesis: BAB II §2.3]`.
- **UU PDP / Privacy-by-Design — P4.** P4 (kewajiban UU PDP dipenuhi secara teknis, diukur via
  matriks bertingkat §9, bukan "5 pasal terpenuhi") `[prd: §3]` adalah operasionalisasi klaim BAB II
  §2.3 tentang PbD *data minimization* — *"menyimpan hanya hash kriptografis ... bukan plaintext
  data personal"* `[thesis: BAB II §2.3]`. **Catatan kejujuran:** BAB II mengklaim SHA-256(ID
  internal) tanpa kunci sebagai kontrol PbD — klaim ini **bertentangan dengan dirinya sendiri**
  secara empiris (dapat dibalik 0,0002 detik) — lihat `[errata: E-1]`. Diratifikasi sebaliknya di
  `prd.md` §5.2 (hierarki *salt* + HMAC dua lapis).
- **TOE Framework** ikut membentuk cakupan-anggaran objektif secara implisit: tekanan *environment*
  (UU PDP + ekspektasi klien enterprise) adalah alasan **mengapa** objektif ini dipilih di atas
  alternatif lain `[thesis: BAB II §2.4.1]` — lihat aktivitas 3 untuk pengikatan TOE yang lebih
  eksplisit pada keputusan desain.

### Status kejujuran

**Selesai dan berlaku.** Ini satu-satunya aktivitas yang benar-benar dapat ditandai ✅ tanpa
kualifikasi — gap G-01 dan G-05 ditutup oleh keputusan manusia eksplisit (bukan artefak HLF-6), dan
peta ini sendiri (§0 di atas, `grounding-gaps.md`) mentranskripsi ratifikasi itu verbatim, bukan
memparafrasenya. Satu catatan tersisa: BAB III tesis Tahap 2 masih menyebut **tiga** section
("personal, kepegawaian, pendidikan") padahal seluruh dokumen lain menyebut lima — `[errata: E-5]`
— ini murni utang editorial naskah tesis, tidak mempengaruhi objektif yang diratifikasi.

---

## 4. Aktivitas 3 — Perancangan & Pengembangan (G2–G7)

**Definisi Peffers:** artefak itu sendiri — construct, model, method, atau instantiation
(`DSRM.md` §1). Tesis: **Tahap 3**, *"Keputusan desain utama: Hyperledger Fabric v2.5, channel-per-tenant,
chaincode GoLang `EmployeeProfileRecord`, IPFS Private Cluster"* `[thesis: BAB III §3.4.3]`,
diperinci di `[thesis: BAB IV §4.2.1–§4.2.5]`.

Aktivitas ini **membentang enam sub-gate**; setiap sub-gate diberi baris sendiri karena masing-masing
punya artefak dan status supersesi berbeda.

### 4.1. G2 — Skill baru

| Artefak | Status | Sitasi |
|---|---|---|
| `dsrm-research-assistant`, `zkp-designer`, `privacy-by-design` (3 skill baru) | ✅ ditulis, terinstal | `[code: skills/*]` |
| Template DPIA (`skills/privacy-by-design/assets/dpia-template.md`) | ✅ ada, **belum pernah diisi** untuk artefak ini — lihat gap baru **G-25** (§10) | `[code: skills/privacy-by-design/assets/dpia-template.md]` |

**Kerangka teori:** Privacy-by-Design secara langsung — template DPIA **adalah** instrumen PbD
prinsip *"proactive not reactive"*.

### 4.2. G3 — Arsitektur + spesifikasi agen

| Artefak | Status | Sitasi |
|---|---|---|
| `00-architecture/multi-agent-architecture.md`, `orchestration-and-collaboration.md`, `quality-gates-and-approval.md`, 4 diagram | 🟡 struktur proses masih berlaku; **tapi** memuat 24 kemunculan aturan "0 baris kode" di 17 berkas yang menciptakan kebuntuan tata kelola S-1 (§9) | `[rekonsiliasi: Gelombang 2 #9]`, `[rekonsiliasi: §1.2]` |
| 12 spesifikasi agen (`01-agents/specs/*`) + 4 agen operasional (`agents/*.md`) | 🟡 tiga di antaranya (`fabric-engineer.md`, `fabric-architect.md`, `security-architect.md`) menanamkan aturan "0 baris kode" yang sama — akan menolak pekerjaan yang sudah diratifikasi selama S-1 belum diputuskan | `[rekonsiliasi: §1.2]` |

**Kerangka teori:** metodologi DSR/DSRM sendiri — pembagian tugas AUTHOR/VERIFIER per gate
**adalah** siklus desain (*design cycle*) Hevner (2007) yang diinstansiasi sebagai proses organisasi
`[DSRM.md §3]`.

### 4.3. G4 — ADR

| Artefak | Status | Sitasi |
|---|---|---|
| ADR-0001..0010 (skema *digest*, org topologi, tenancy, anchor-service, LevelDB, ZKP-defer, key separation, erasure) | 🟡 **enam dari sepuluh** disupersedi oleh ratifikasi tesis — status index masih "Accepted (pending supersession)" karena ADR pengganti **belum ditulis** | `[code: 05-adr/README.md]` (dibaca, **tidak disentuh** — dimiliki agen lain) |
| ADR-0011..0020 (digest per section+HMAC hierarkis; 3-org+Raft; channel-per-tenant; in-band recording; erasure+IPFS+relational tier+domain kunci; performa dalam cakupan; kontrak chaincode+batas hashing) | ⛔ **belum ditulis** — ini item paling kritis di jalur kritis rekonsiliasi | `[rekonsiliasi: Gelombang 3 #13–20]` |

**Kerangka teori:**
- **Security-by-Design** mengikat langsung ADR-0012 (topologi 3-org + Raft) — BAB II §2.3 menyebut
  SbD diterapkan lewat *"topologi multi-organisasi yang secara arsitektur mencegah manipulasi
  unilateral"* dan *"endorsement policy yang mensyaratkan persetujuan multi-pihak"*
  `[thesis: BAB II §2.3]`. Ini **persis** apa yang ADR-0012 harus mencatat.
- **Privacy-by-Design** mengikat ADR-0011 (skema digest) dan ADR-0015/16/17/19 (erasure + domain
  kunci) — prinsip *data minimization* dan *full lifecycle protection* `PRIVACY-BY-DESIGN.md` §1.
- **TOE Framework** mengikat ADR-0018 (performa masuk cakupan) — dimensi *organization* (biaya
  infrastruktur node, ketersediaan keahlian) dan *technology* (kematangan Fabric v2.5)
  `[thesis: BAB II §2.4.1]` adalah argumen mengapa Caliper v0.5 dijadikan *harness* wajib, bukan
  sekadar nice-to-have.

### 4.4. G5 — Desain solusi

| Artefak | Status | Sitasi |
|---|---|---|
| `00-architecture/solution/{data-model,fabric-network-design,integration-design,api-contracts}.md` | 🟡 seluruhnya perlu ditulis ulang total — **tidak satu pun** dari 12 nama *field* lama (`AnchorRecord`, dst.) bertahan | `[rekonsiliasi: Gelombang 4 #22–24]` — dimiliki `fabric-architect`/`fabric-engineer`, **bukan berkas peta ini** |
| Definisi struct `EmployeeProfileRecord` (BAB IV) — sumber tunggal yang HARUS diikuti desain baru | ✅ ada sebagai rujukan tesis, **dengan tiga koreksi wajib**: *salt* pada `DataHash`, hierarki `employeeKey_i` pada `EmployeeID`/`UpdatedBy`, digest dihitung **off-chain** (bukan oleh chaincode) | `[thesis: BAB IV §4.2.3]` dikoreksi oleh `[prd: §5.2]`, `[errata: E-1]` |

**Kerangka teori:** CIA Triad (ketiga dimensi sekaligus) — *Integrity* via hash chaining per section,
*Confidentiality* via *private channels* + enkripsi off-chain, *Availability* via desentralisasi
jaringan Fabric `[thesis: BAB II §2.3]`.

### 4.5. G6 — Arsitektur keamanan

| Artefak | Status | Sitasi |
|---|---|---|
| `08-security/{security-architecture,control-matrices}.md`, `trust-boundaries.mmd` | 🟡 model ancaman lama kehilangan objek (T9/T10/T11/T15); ancaman IPFS + operator *node cluster* sebagai batas kepercayaan keempat **belum dimodelkan** | `[rekonsiliasi: Gelombang 4 #27]` — **dimiliki `security-architect`, bukan berkas peta ini** |

**Kerangka teori:** Security-by-Design (ketiga tenet — *secure defaults*, *least privilege*,
*defense-in-depth*, per `SECURITY-BY-DESIGN.md`) adalah kerangka native G6; Privacy-by-Design
mengikat bagian erasure/DPIA-nya.

### 4.6. G7 — Tugas + strategi uji

| Artefak | Status | Sitasi |
|---|---|---|
| `06-roadmap/test-strategy.md`, `implementation-backlog.md` | 🟡 ST-1 (invariant kerahasiaan, P0) **tetap satu uji yang dapat dipenuhi** setelah §5.2 diadopsi, tapi permukaan pindaian harus diperluas (buang PDC, tambah `IPFSCIDs`+digest pengenal) — **belum dikerjakan** | `[rekonsiliasi: Gelombang 4 #28–29]` — **dimiliki `qa`/`architect`, bukan berkas peta ini** |
| Rencana eksperimen G7 (feed dari mode Experiment Planner skill ini) | ⏳ belum diminta secara eksplisit dalam tugas peta ini — lihat hand-off §12 | — |

### Status kejujuran aktivitas 3 secara keseluruhan

**Blocked, bukan selesai.** README status board masih mencatat G2–G7 ✅ — sertifikasi itu berlaku
untuk desain **yang sudah tidak ada** (`prd.md` §11.1: enam asumsi yang disetujui G9 lama diketahui
salah). Sebelum satu pun sub-gate di atas boleh diklaim ✅ lagi, sepuluh ADR baru (§4.3) harus
ditulis lebih dulu — mereka adalah dependensi keempat di jalur kritis `[rekonsiliasi: §4]`, dan tanpa
kosakata status `Superseded by ADR-NNNN` (sudah ditambahkan ke template 3 Agustus 2026) tidak satu
pun dari sepuluh ADR baru itu bisa dicatat secara sah.

---

## 5. Aktivitas 4 — Demonstrasi (pasca-G9)

**Definisi Peffers:** artefak digunakan pada satu instans masalah — studi kasus, eksperimen, atau
bukti (`DSRM.md` §1). **Urutan yang dibalik secara sengaja** di paket ini: DSRM menaruh Demonstrasi
sebelum Evaluasi, tapi batas desain-saja memaksa Evaluasi (G8) + Komunikasi/persetujuan (G9)
mendahului prototipe berjalan mana pun `[DSRM.md §2]`. Tesis: **Tahap 4**, *"lima skenario
operasional: onboarding karyawan baru, pembaruan section kepegawaian, verifikasi integritas data
penggajian, deteksi manipulasi data pendidikan, dan isolasi profil antartenant"*
`[thesis: BAB III §3.4.4]`.

### Artefak konkret

| Artefak | Status | Sitasi |
|---|---|---|
| UJ-1 (skrip demonstrasi deteksi manipulasi penggajian) + UJ-2 (isolasi antartenant) | ✅ **skrip** sudah ditulis — ini adalah rencana demonstrasi, bukan demonstrasi itu sendiri | `[prd: §6]` |
| SC-A..SC-D (manipulasi PAYROLL/PERSONAL/EMPLOYMENT/EDUCATION) + skenario `ADDITIONAL` baru | ⛔ SC-A..SC-D dijalankan **pada versi desain yang sudah disupersedi** (BAB IV, hash tanpa *salt*); `ADDITIONAL` **belum pernah dijalankan sama sekali** | `[thesis: BAB IV §4.3.2]` Tabel 4.4; kekurangan `ADDITIONAL` = `[errata: E-2]`; kebutuhan re-run atas §5.2 = `[prd: §3]` |
| SC-E (isolasi channel) | ⛔ mensyaratkan Tenant B + peer Tenant B nyata; belum ada bukti dua channel benar-benar dijalankan | `[errata: E-11]` |
| Bagian hasil Tahap 4 yang eksplisit di naskah tesis | ⛔ **tidak ada** — label `SC-A..SC-E` dipakai ulang untuk himpunan skenario yang berbeda (uji keamanan Tabel 4.4), sehingga Tahap 4 kehilangan bagian hasilnya sendiri — celah struktural, bukan format | `[errata: E-9]` |
| Grounding jalur nyata modul Employee (generik) | 🟡 tersedia **secara internal** untuk memetakan lima section (PERSONAL/EMPLOYMENT/EDUCATION/ADDITIONAL/PAYROLL) ke domain data nyata platform, **tidak dituliskan literal** di artefak mana pun sejak register generik 2026-08-06 — lihat **§11** | `[ASSUMPTION]` (gap baru **G-29**, §10) |

### Kerangka teori BAB II yang mengikat

- **Trust Theory dipentaskan, bukan sekadar dinarasikan.** UJ-1 langkah 7 — *"Yang menghentikan
  saya menutupi ini bukan kebijakan perusahaan, tapi kriptografi"* `[prd: §6]` — adalah dramatisasi
  langsung §2.4.2 BAB II. Demonstrasi aktivitas 4 **adalah** wahana operasional Trust Theory,
  bukan cuma ilustrasi teknis.
- **CIA-Integrity dibuktikan, bukan diargumentasikan.** Aktivitas 4 adalah satu-satunya titik di
  seluruh peta di mana klaim Integrity BAB II §2.3 berhenti menjadi argumen (aktivitas 1–3, ranah
  Descriptive/Analytical) dan menjadi bukti empiris (Testing/Experimental, `DSRM.md` §4).

### Status kejujuran

**Belum boleh dimulai.** Batas desain-saja `[brief: agents-guide.md]` mengunci aktivitas ini di
belakang G9 — dan G9 batal (`prd.md` §11.1). Bahkan setelah G9 disetujui ulang, dua prasyarat
tambahan harus terpenuhi sebelum demonstrasi punya nilai evidentiary apa pun: **(1)** ADR-0011..0020
(aktivitas 3) harus ada, karena skenario mana pun yang dijalankan terhadap desain lama tidak
membuktikan apa-apa tentang desain yang diratifikasi; **(2)** skenario `ADDITIONAL` (`errata: E-2`)
dan Tenant-B nyata (`errata: E-11`) harus disediakan agar P1/P3 sebagaimana diratifikasi benar-benar
teruji. **Jangan pernah merencanakan Testing/Experimental seolah jaringan sudah ada hari ini** —
setiap rencana eksperimen untuk aktivitas ini diberi label **Fase-4 (pasca-G9)**, mengikuti aturan
operasional skill ini.

---

## 6. Aktivitas 5 — Evaluasi (G8)

**Definisi Peffers:** bukti terukur seberapa baik artefak memenuhi objektif; beri makan kembali ke
aktivitas 3 (`DSRM.md` §1). Tesis: **Tahap 5**, tiga dimensi — *"benchmark kinerja, pengujian
keamanan, pemetaan kepatuhan UU PDP dan validasi kualitatif FGD"* `[thesis: BAB III §3.4.5]`,
dilaporkan di `[thesis: BAB IV §4.3.1–§4.3.4]`.

### Artefak konkret

| Artefak | Status | Sitasi |
|---|---|---|
| `prd.md` §10 (Metode Evaluasi DSRM) — pemetaan 4 jalur tesis ke 5 keluarga Hevner, 2 metode yang harus ditambahkan ke `DSRM.md` §4 | ✅ ditulis lengkap di `prd.md`; **dieksekusi** oleh peta ini terhadap `context/DSRM.md` §4 (lihat berkas sibling hasil tugas ini) | `[prd: §10]` |
| `09-review/verification/g8-review-report.md` | ❌ **PASS batal** — mensertifikasi desain yang sudah tidak ada (skema digest ber-*salt* per-event, permukaan PDC, dst.) | `[prd: §11.1]` |
| `g8-review-report-r2.md` | ⛔ **belum ditulis**; r1 harus ditandai SUPERSEDED (bukan diedit di tempat) sebelum r2 ada | `[rekonsiliasi: Gelombang 6 #38]` |
| Pembagian G8 → **G8a** (sapuan dokumen, 0 baris kode) + **G8b** (evaluasi terukur, kode diharapkan) | ⏳ **direkomendasikan**, menunggu keputusan sponsor **S-1** — lihat §9 | `[rekonsiliasi: §1.3]` |
| Benchmark kinerja Caliper v0.5 (850,4 TPS tulis / 1.950,5 TPS baca / 2,85 detik) | ⛔ **placeholder, bukan hasil pengukuran** — angka ini tidak boleh dikutip artefak mana pun sampai *harness* benar-benar dijalankan | `[prd: §3.1]`, `[errata: E-8]` |
| Matriks kepatuhan UU PDP (5 pasal diklaim) | ❌ **4 dari 5 nomor pasal salah kutip** — diperbaiki jadi 12 baris tiga-tingkat status di `prd.md` §9.2 | `[thesis: BAB IV §4.3.3]` Tabel 4.5, dikoreksi `[prd: §9]`, `[errata: E-14]` |
| FGD (3 ahli, checklist 7 kriteria) | 🟡 **instrumen tidak konsisten** — BAB III menjanjikan 4 kriteria, Tabel 4.6 menilai 7; "kepercayaan" (kriteria Trust Theory) tidak konsisten diukur | `[thesis: BAB III §3.4 Instrumen Penelitian]`, `[thesis: BAB IV §4.3.4]` Tabel 4.6, `[errata: E-6]` |

### Kerangka teori BAB II yang mengikat

- **DSR/DSRM sendiri adalah kerangka aktivitas ini** — tesis §2.5 secara eksplisit menyatakan
  Evaluasi *"menguji efektivitas artefak berdasarkan kriteria keamanan (CIA Triad), kepatuhan
  (UU PDP), dan **kepercayaan pengguna (Trust Theory)**"* `[thesis: BAB II §2.5.2 / Proses dan
  Tahapan]`. Ini kalimat tunggal yang mengikat **tiga** frame sekaligus ke satu aktivitas — dan
  ini pula alasan mengapa "kepercayaan" pada FGD **wajib** diukur konsisten (`errata: E-6`): kalau
  sebuah lapisan teori dideklarasikan, penguji DSRM menuntutnya dioperasionalkan, bukan didekorasi.
- **CIA Triad** mengikat dua dari lima keluarga Hevner secara langsung: benchmark kinerja (dimensi
  Availability/kinerja praktis) dan simulasi manipulasi (dimensi Integrity, P1).
- **UU PDP / Privacy-by-Design** mengikat jalur evaluasi keempat — analisis kepatuhan (P4) — yang
  **DITAMBAHKAN** ke `context/DSRM.md` §4 sebagai instans konkret keluarga **Analytical** oleh
  tugas ini (lihat berkas sibling).
- **Trust Theory** mengikat jalur evaluasi kelima — FGD — sebagai instans konkret keluarga
  **Descriptive + Observational**, juga ditambahkan ke `context/DSRM.md` §4 oleh tugas ini.

### Status kejujuran

**PASS batal, evaluasi belum bisa berjalan ulang.** `prd.md` §11.1 menyatakan ini definitif, bukan
sekadar "perlu ditinjau ulang". Lima prasyarat harus terpenuhi sebelum evaluasi menghasilkan bukti
yang dapat dipertahankan (`prd.md` §10.4): skenario `ADDITIONAL` (E-2), Tenant-B nyata (E-11),
*harness* Caliper nyata (E-8), kepemilikan Org2 dijawab jujur (E-11), dan instrumen FGD disamakan
(E-6). **Kebuntuan tata kelola** (§9, S-1) memblokir seluruhnya: P1/P2/P4 menuntut prototipe
berjalan, tapi aturan "0 baris kode" adalah syarat kelulusan G8 versi lama — lingkaran yang hanya
dapat dipecah keputusan sponsor, bukan pekerjaan teknis.

---

## 7. Aktivitas 6 — Komunikasi (G9)

**Definisi Peffers:** masalah, artefak, rigor, dan utilitasnya dikomunikasikan ke peneliti dan
praktisi (`DSRM.md` §1). Tesis: **Tahap 6**, *"Temuan dikomunikasikan melalui penulisan tesis dan
presentasi kepada pemangku kepentingan [platform] sebagai tahap validasi akhir"*
`[thesis: BAB III §3.4.6]`.

### Artefak konkret

| Artefak | Status | Sitasi |
|---|---|---|
| `agent-suite/README.md` status board | ❌ masih mencatat *"G4 · 10 ADR semua Accepted"* dan *"G5 · confidentiality invariant verified"* dan *"G9 · APPROVED 2026-07-13"* — ketiganya tidak berlaku lagi | `[code: README.md]`, perbaikan = `[rekonsiliasi: Gelombang 5 #35]` (**bukan berkas peta ini**) |
| Naskah tesis (BAB V Kesimpulan & Saran, DAFTAR PUSTAKA) | 🟡 ada, tapi mewarisi seluruh klaim yang errata temukan cacat (5-pasal, 100%-lima-section, rekonstruksi nilai asli, dst.) di ringkasan Kesimpulan-nya | `[thesis: BAB V §5.1]` |
| Publikasi Confluence (HLF-6) | ⛔ target tidak terjangkau dari environment ini (hanya `jurnal.atlassian.net` yang diberi akses; HLF-6 ada di `chandrakurniawan.atlassian.net`) | `[ASSUMPTION]` (gap G-01/G-14, dicatat ulang di `grounding-gaps.md`) |
| Kontribusi teoretis "DSRM + Compliance-by-Design" (OS-3) | ✅ dinyatakan sebagai objektif sekunder; tesis membahasnya di BAB IV §4.4.3 | `[prd: §2.2]` OS-3, `[thesis: BAB IV §4.4.3]` |
| UJ-1/UJ-2 sebagai naskah komunikasi ke audiens (dosen penguji, forum arsitektur/keamanan internal, ahli FGD) | ✅ ada | `[prd: §6]` |

### Kerangka teori BAB II yang mengikat

- **DSR/DSRM sendiri** — Komunikasi adalah aktivitas keenam yang dideklarasikan tesis sendiri
  sebagai bagian metodologi, bukan formalitas administratif: *"Mendokumentasikan hasil penelitian
  dan kontribusi teoretis maupun praktisnya"* `[thesis: BAB II §2.5.2]`.
- **Trust Theory** membentuk isi pesan yang dikomunikasikan, bukan cuma metodologinya — bagian
  Pembahasan tesis sendiri menyebut pergeseran *institutional trust* → *code-based trust* sebagai
  temuan yang dikomunikasikan ke pembaca `[thesis: BAB IV §4.4.2]`.

### Status kejujuran

**Persetujuan batal; republikasi menunggu sponsor.** G9 bukan sekadar "belum disetujui ulang" — ia
secara definitif batal menurut kriterianya sendiri (`prd.md` §11.1). Status board `README.md` yang
menyatakan G9 ✅ per 2026-07-13 adalah artefak dari siklus rekonsiliasi **sebelum** ratifikasi tesis
— ia harus diperlakukan sebagai **sejarah**, bukan otorisasi hari ini. Publikasi Confluence pun
tidak dapat dieksekusi dari environment ini terlepas dari status gate — itu keterbatasan akses,
dicatat sebagai `[ASSUMPTION]`, bukan diselesaikan diam-diam.

---

## 8. Keseimbangan Rigor vs Relevansi (tiga siklus Hevner)

`DSRM.md` §3 mendefinisikan tiga siklus (Hevner, 2007). Berikut penerapannya **khusus untuk
prototipe ini**, bukan pengulangan definisinya:

| Siklus | Sebelum ratifikasi tesis (s.d. 2026-08-01) | Sesudah ratifikasi tesis (2026-08-02 → sekarang) |
|---|---|---|
| **Relevansi** (lingkungan ↔ desain) | Provisional — objektif `[ASSUMPTION] G-01`, event mana yang di-*anchor* `[ASSUMPTION] G-05`; grounding hanya lewat dua repo nyata + korpus Fabric | **Lebih kuat, tapi tidak murni.** Objektif dan cakupan *anchoring* kini diratifikasi manusia (bukan lagi `[ASSUMPTION]`), dan grounding jalur nyata modul Employee sudah dilakukan (2026-08-06) untuk memverifikasi pemetaan SC-A..SC-D+`ADDITIONAL`. **Tapi** representasi tertulisnya (`knowledge-graph.md`) belum menyusul, dan register generik baru (§11) berarti sebagian bukti grounding kini disimpan tidak-literal — sebuah bentuk provisional yang baru, bukan hilangnya provisionalitas |
| **Rigor** (basis pengetahuan ↔ desain) | Kuat — korpus Fabric 2.5 dipin, tujuh frame keamanan dimandatkan, literatur DSRM oleh penulis/tahun | **Bertambah kuat pada satu sumbu (BAB II tesis + verifikasi hukum UU PDP 7-pembaca-independen), tapi terbuka gap baru**: `context/DSRM.md` §4 sendiri sampai sebelum tugas ini **tidak mencantumkan** dua metode evaluasi yang benar-benar dipakai tesis (analisis kepatuhan, FGD) — celah rigor-terhadap-pustaka-sendiri, ditutup oleh berkas sibling tugas ini |
| **Desain** (bangun ↔ evaluasi) | Loop G-gate rutin, semua ✅ hingga G9 | **Loop putus di G8.** `prd.md` §11.1 membatalkan PASS G8 dan persetujuan G9 — siklus desain untuk paket ini **secara harfiah kembali ke titik sebelum G8**, meski sepuluh gate sempat dilewati sekali |

**Fakta.** Karena relevansi kini bersandar pada ratifikasi manusia (bukan lagi tebakan tim desain)
tapi rigor-terhadap-pustaka-metodologi-sendiri baru saja ditemukan berlubang (dua metode evaluasi
hilang dari `DSRM.md` §4), **rekomendasi `DSRM.md` §3 untuk "menimbang rigor lebih berat"** kini
punya makna baru: bobot rigor yang tepat bukan lagi "korpus Fabric + tujuh frame" saja, melainkan
**korpus Fabric + tujuh frame + kerangka BAB II tesis (CIA/PbD/SbD/TOE/Trust Theory/DSR-DSRM) +
verifikasi hukum UU PDP** — empat pilar, bukan dua. **Rekomendasi:** setiap ADR baru (§4.3) yang
lahir dari rekonsiliasi ini harus mengutip pilar BAB II yang relevan secara eksplisit (seperti yang
dilakukan §2–§7 peta ini), bukan hanya korpus Fabric — supaya rigor baru ini tidak hilang lagi di
revisi desain berikutnya.

---

## 9. Keputusan Sponsor yang Menggerbangi Peta Ini (S-1..S-5)

Lima keputusan ini **bukan** keputusan teknis — mereka menentukan **apakah** dan **kapan** Aktivitas
3–6 di atas boleh melanjut. Karena "Decisions need alternatives" adalah aturan keras peran ini,
setiap baris berikut mencatat alternatif yang dipertimbangkan, bukan hanya rekomendasi tunggal.
**Keputusan final tetap milik sponsor** (Chandra Kurniawan) — peta ini membingkai, tidak memutuskan.

| # | Keputusan | Alternatif dipertimbangkan | Rekomendasi (dan mengapa yang lain ditolak) | Yang tertahan sampai diputuskan |
|---|---|---|---|---|
| **S-1** | Pecah G8 → G8a (sapuan dokumen, 0 kode) + G8b (evaluasi terukur, kode diharapkan); pensiunkan aturan "0 baris kode" di 17 berkas | (a) pertahankan G8 tunggal apa adanya; (b) hapus aturan "0 baris kode" total, tanpa pembagian gate | (a) ditolak — P1/P2/P4 semuanya menuntut prototipe berjalan, sehingga G8 tunggal menciptakan lingkaran tak-terpecahkan (§6). (b) ditolak — menghapus total menghilangkan jejak keputusan *mengapa* batas desain-saja pernah ada, dan mencampur "desain konsisten" dengan "artefak terukur" jadi satu pertanyaan. **Rekomendasikan: pembagian G8a/G8b** `[rekonsiliasi: §1.3]` | **Seluruh build** — S-1 harus diputuskan lebih dulu dari S-2..S-5 |
| **S-2** | Basis data operasional otoritatif: PostgreSQL/AWS (Tabel 4.2 tesis) atau MySQL/existing (BAB IV §4.1.1 tesis) | (a) migrasi ke PostgreSQL sesuai Tabel 4.2 | (a) ditolak — migrasi tak terjustifikasi bertentangan dengan posisi "berdampingan, bukan menggantikan" (`prd.md` Ringkasan Eksekutif) dan tidak pernah dijelaskan tesis (`errata: E-10`). **Rekomendasikan: MySQL existing** | ADR-0017, `integration-design.md`, FR terkait |
| **S-3** | Klaim "rekonstruksi nilai asli" (OQ-6) | (A) pulihkan dari backup/PITR; (B) simpan *change log* append-only terpisah (butuh di-*anchor* juga); (C) persempit klaim (hapus kata "direkonstruksi", nyatakan ledger membuktikan kandidat pemulihan, tidak menghasilkannya); (D) simpan section terenkripsi versi-per-versi di IPFS | (B) ditolak — log itu sendiri mutable, jadi butuh anchoring lagi (menambah kerumitan tanpa nilai baru). (D) ditolak — mengubah arsitektur untuk masalah yang punya jawaban murah. **Rekomendasikan: C (klaim) + A (jawaban operasional)** — paling murah, dan justru memperkuat posisi karena "membuktikan kandidat benar" lebih kuat metodologis daripada "menyimpan salinan" | `FR-17`, UJ-1 langkah 6 |
| **S-4** | PB-2/G-23 (*read-back* delta PII oleh anchor-service) — larut atau dipertahankan | (a) larutkan karena tanpa anchor-service/Kafka tidak ada lagi jalur *read-back* yang perlu diamankan; (b) pertahankan sebagai gap terbuka untuk desain in-band yang mungkin masih perlu membaca nilai section saat menghitung digest | (b) tidak bisa ditolak sepenuhnya — pencatatan in-band (ADR-0014) tetap harus **membaca** nilai section terkini dari basis data operasional untuk menghitung `DataHash`; pertanyaannya berpindah bentuk (bukan hilang), bukan otomatis larut. **Peta ini TIDAK merekomendasikan menutup G-23** — lihat status "menunggu" di `grounding-gaps.md`, bukan "ditutup" | Gate *pre-build* |
| **S-5** | Persetujuan ulang G9 terhadap himpunan asumsi baru | (a) setujui sekarang berdasarkan `prd.md` saja; (b) setujui setelah Gelombang 1–5 `rencana-rekonsiliasi.md` selesai (10 ADR baru ditulis, dokumen desain ditulis ulang) | (a) ditolak — menyetujui sebelum sepuluh ADR baru ada berarti menyetujui *placeholder*, mengulang persis kegagalan G9 lama (`prd.md` §11.1: "menyetujui satu set asumsi" yang kemudian berubah). **Rekomendasikan: (b)** | Otorisasi build sah |

---

## 10. Celah Baru Ditemukan Saat Menyusun Peta Ini

Ditambahkan ke [`../11-execution/grounding-gaps.md`](../11-execution/grounding-gaps.md) sebagai
**G-25..G-29** (rincian lengkap tiap gap ada di berkas itu; ringkasan di sini untuk kelengkapan
peta):

| Gap | Ringkasan | Muncul dari |
|---|---|---|
| **G-25** | DPIA (Pasal 34 UU PDP) belum disusun, padahal terpicu ≥4 pintu ayat (2) sekaligus — termasuk "penggunaan teknologi baru", klaim kebaruan tesis sendiri. Template sudah ada (`skills/privacy-by-design/assets/dpia-template.md`), belum pernah diisi. | `[prd: §9.3]` baris 10 |
| **G-26** | Kebijakan retensi ledger (Pasal 42 UU PDP) tidak ada — *crypto-shred* dipicu **permintaan**, sedangkan Pasal 42 mewajibkan penghentian pemrosesan **otomatis** berbasis waktu/tujuan. Ledger *append-only* tidak pernah berhenti menyimpan. | `[prd: §9.3]` baris 11 |
| **G-27** | Peran hukum Org1/Org2 (Pengendali vs Prosesor, Pasal 52) belum ditetapkan — menentukan pasal mana yang mengikat pihak mana; ditambah dasar pemrosesan untuk replikasi ledger ke Org3 (yang secara harfiah adalah "transfer" per Penjelasan Pasal 16(1)(e)) belum dinyatakan. | `[prd: §9.4]` |
| **G-28** | Verifikasi independen atas revisi InfoSec tesis (klaim bahwa E-4 "sudah diperbaiki dengan menghapus janji, bukan menambah laporan") **tidak dapat dilakukan** di lingkungan ini — tooling tidak dapat membaca `.docx` biner; rekonsiliasi bersandar pada ekstraksi teks pra-revisi (`source-thesis-extract.txt`) + karakterisasi `errata-tesis.md`. | Ditemukan saat menyusun peta ini — lihat **§11** |
| **G-29** | Bagaimana bukti grounding internal (path repo/tabel/kolom nyata) dipertahankan sebagai jejak audit tanpa melanggar register generik yang berlaku sejak 2026-08-06 — perlu keputusan format (mis. lampiran internal berakses terbatas) agar rigor tidak diam-diam hilang saat register berubah. | Ditemukan saat menyusun peta ini — lihat **§11** |

---

## 11. Batas Kerahasiaan & Register Generik

Dua batasan eksplisit yang membentuk *bagaimana* peta ini ditulis, dicatat di sini untuk transparansi
(bukan disembunyikan sebagai detail teknis):

1. **Register generik (berlaku sejak 2026-08-06).** Peta ini boleh digrounding terhadap modul
   Employee nyata milik platform SaaS HRIS multi-tenant yang menjadi rujukan tesis, tapi **tidak
   menyebut nama platform/perusahaan atau nama tabel/kolom/repo nyata secara literal** di badan
   dokumen — mengikuti `project-context.md` AUTHORITY NOTICE. Konsekuensinya: beberapa baris di §5
   (Aktivitas 4) dan gap **G-29** (§10) mencatat bahwa grounding *ada* tanpa mereproduksinya —
   sebuah bentuk kejujuran evidentiary yang berbeda dari sitasi `[code: …]` biasa di dokumen lain
   dalam paket ini yang ditulis sebelum aturan register ini berlaku.
2. **Batas alat baca berkas biner.** Revisi tesis "InfoSec" terbaru (`.docx`) **tidak dapat dibuka**
   oleh peralatan yang tersedia di lingkungan penyusunan peta ini (bukan PDF; pembaca berkas ini
   menolak biner `.docx` secara eksplisit). Seluruh kutipan `[thesis: BAB … §…]` di dokumen ini
   diverifikasi terhadap `source-thesis-extract.txt` — ekstraksi teks dari **versi tesis sebelum
   revisi InfoSec**, yang juga menjadi basis audit `errata-tesis.md`. Klaim bahwa revisi InfoSec
   "sudah memperbaiki" E-4 (dengan menghapus janji instrumen, bukan menambah laporan) **belum
   diverifikasi ulang secara independen oleh peta ini** — dicatat sebagai gap **G-28** (§10), bukan
   diterima sebagai fakta ratifikasi. **Rekomendasi:** jatuhkan ekstraksi teks revisi InfoSec ke
   `11-execution/sources/` agar klaim ini dapat diverifikasi tanpa alat pembaca `.docx`.

---

## 12. Hand-off

- **`architect`** — sepuluh ADR baru (§4.3, ADR-0011..0020) adalah dependensi jalur-kritis nomor
  4 di `rencana-rekonsiliasi.md` §4; peta ini menyediakan pengikatan teori BAB II per ADR sebagai
  bahan bagian "Context"/"Consequences" masing-masing ADR, tidak menggantikan penulisannya.
- **`pm`** — keputusan sponsor S-1..S-5 (§9) perlu difasilitasi sebagai satu sesi keputusan manusia,
  sesuai catatan `rencana-rekonsiliasi.md` §2 ("S-1 harus diputuskan lebih dulu; empat lainnya
  dapat diputuskan dalam satu sesi yang sama").
- **`security-architect`** — gap **G-25** (DPIA belum disusun) dan **G-26** (retensi) berada tepat
  di persimpangan Privacy-by-Design dan kepatuhan UU PDP; template DPIA sudah tersedia
  (`skills/privacy-by-design/assets/dpia-template.md`) dan tinggal diisi begitu ADR-0011/0015/0017
  (§4.3) mengunci bentuk final skema digest + tier relasional.
- **`qa`** — Aktivitas 5 (§6) dan Aktivitas 4 (§5) memberi bahan mentah untuk mode **Experiment
  Planner** skill ini (belum dijalankan sebagai tugas terpisah): skenario `ADDITIONAL` (E-2) dan
  Tenant-B (E-11) adalah kandidat kuat objek eksperimen begitu S-1/S-5 (§9) diputuskan.
- **Sesi berikutnya dengan skill ini** — begitu S-1..S-5 diputuskan, jalankan mode **Experiment
  Planner** untuk skenario `ADDITIONAL` + Tenant-B secara formal (`experimentPlan`, memberi makan
  G7 test strategy), lalu mode **Evaluation Report Generator** untuk `g8-review-report-r2.md`
  begitu ADR-0011..0020 dan dokumen desain Gelombang 4 selesai ditulis ulang.

---

*Traceability. Peta ini mengeksekusi permintaan pemetaan enam-aktivitas dari orkestrator (2026-08-06)
tanpa mendefinisikan ulang metodologi — definisi tetap satu rumah di
[`../context/DSRM.md`](../context/DSRM.md) §1–§2 dan literatur Peffers et al. (2007, 2022) /
Hevner et al. (2004) / Hevner (2007) / March & Smith (1995). Sumber pengesahan:
[`prd.md`](../../_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/prd.md),
[`errata-tesis.md`](../../_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/errata-tesis.md),
[`rencana-rekonsiliasi.md`](../../_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/rencana-rekonsiliasi.md).
Gap register sibling: [`../11-execution/grounding-gaps.md`](../11-execution/grounding-gaps.md).*
