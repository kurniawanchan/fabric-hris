---
title: 'PRD — Prototipe Sistem Integritas Data Profil Karyawan Berbasis Blockchain'
status: final
created: '2026-08-02'
updated: '2026-08-06'
project: fabric-hris
bahasa: 'Indonesia'
tujuan_publikasi: 'Confluence'
fase_dsrm: 'Fase 2 (Define Objectives) — meratifikasi objektif untuk Fase 4 Demonstration'
sumber_otoritatif: 'Tesis — Chandra Kurniawan (2602655404), Universitas Bina Nusantara'
menutup_blocker: ['PB-4 / G-01', 'PB-5 / G-05']
---

# PRD — Prototipe Sistem Integritas Data Profil Karyawan Berbasis Blockchain

| | |
|---|---|
| **Penulis** | Chandra Kurniawan |
| **Tanggal** | 2–3 Agustus 2026 |
| **Status** | **Final** — PB-4/PB-5 diratifikasi, Bagian 1–12 lengkap. 10 item terbuka (OQ-2…OQ-6, S-1, S-4, S-5, PB-1, PB-3) tercatat dengan pemilik & kondisi tinjau-ulang eksplisit — lihat Bagian 12 dan `rencana-rekonsiliasi.md` §2 |
| **Sumber otoritatif** | Tesis *"Perancangan dan Implementasi Prototipe Sistem Keamanan Data Karyawan Berbasis Blockchain pada SaaS HRIS menggunakan Metode Design Science Research"* |
| **Fase DSRM** | Fase 2 — *Define Objectives of a Solution* (meratifikasi target untuk Fase 4 *Demonstration*) |

> **Keputusan pengarah.** Di mana pun dokumen ini berbeda dengan paket desain `agent-suite/`
> (gate G0–G9, disetujui 2026-07-13), **tesis yang berlaku**. Paket desain direkonsiliasi
> mengikuti tesis, bukan sebaliknya. Konsekuensinya terhadap ADR dan artefak desain dicatat di
> [Bagian 11](#11-keputusan-yang-dibuka-ulang).
>
> **Satu pengecualian eksplisit — konstruksi kriptografis.** Diratifikasi 2 Agustus 2026: skema
> *hash* dan pseudonimisasi **tidak** mengikuti tesis, melainkan mengikuti `ADR-0001` +
> `data-model.md` dari paket desain. Alasannya bukan preferensi desain — konstruksi di tesis
> **bertentangan dengan klaim tesis sendiri** (BAB II menyebutnya sebagai kontrol
> *Privacy-by-Design*, padahal ia dapat dibalik dalam hitungan milidetik). Rinciannya di
> [§5.2](#52-konstruksi-kriptografis-diratifikasi) dan [`errata-tesis.md`](errata-tesis.md) butir
> **E-1**.

---

## Ringkasan Eksekutif

Platform SaaS HRIS menyimpan data profil karyawan di basis data relasional yang **mutable**. Tanpa
mekanisme kriptografis, tidak ada cara membuktikan bahwa data yang tersimpan saat ini identik dengan
data yang terakhir diotorisasi secara sah. Administrator berprivilese tinggi dapat mengubah field
profil langsung melalui SQL, melewati alur kerja aplikasi, tanpa meninggalkan jejak yang dapat
dibuktikan secara forensik.

Prototipe ini menambahkan **lapisan jaminan integritas** (*integrity assurance layer*) di samping
HRIS yang sudah berjalan — bukan menggantikannya. Setiap kali salah satu dari lima section profil
karyawan dibuat atau diperbarui, sidik jari kriptografis section tersebut dicatat permanen di ledger
Hyperledger Fabric v2.5. Karena klien enterprise dan auditor masing-masing memegang salinan ledger
secara independen, penyedia platform tidak dapat memanipulasi catatan itu sendirian.

Nilai intinya bukan teknis, melainkan **pergeseran basis kepercayaan**: klien tidak lagi perlu
mempercayai reputasi vendor, karena dapat memverifikasi integritas datanya sendiri secara matematis,
kapan saja.

---

## 1. Konteks & Masalah

### 1.1. Kondisi saat ini

Platform beroperasi dengan arsitektur tiga lapis — presentasi (web + mobile), aplikasi (microservice
PHP dan GoLang), dan data (MySQL di Alicloud RDS). Model multi-tenant memakai kolom `company_id`
pada tabel bersama sebagai pemisah antartenant.

Data profil karyawan tersebar di lima tabel: `employees`, `emp_employment`, `emp_education`,
`emp_additional`, dan `emp_payroll`.

### 1.2. Tiga kelemahan yang diidentifikasi

| # | Kelemahan | Wujud konkret |
|---|---|---|
| **W-1** | Tidak ada jaminan integritas kriptografis | `UPDATE emp_payroll SET bank_account='rekening_pelaku' WHERE emp_id='123'` dieksekusi langsung ke DB. Gaji dialihkan. Tidak ada mekanisme teknis yang mendeteksinya setelah terjadi — kecuali laporan manual dari karyawan yang dirugikan. |
| **W-2** | Pembuktian forensik terbatas | Dalam sengketa hubungan industrial, sistem hanya dapat menunjukkan nilai data **saat ini**, tanpa dapat membuktikan nilai tersebut belum pernah dimanipulasi. UU ITE dan UU PDP mensyaratkan bukti elektronik dengan integritas yang dapat diverifikasi. |
| **W-3** | Isolasi antartenant bersifat logis, bukan kriptografis | Model *shared-table* berbasis `company_id` berisiko *cross-tenant leakage* bila terjadi kesalahan query. Klien enterprise — khususnya perbankan — menuntut jaminan yang lebih kuat. |

### 1.3. Dampak bisnis

- **Kerugian finansial langsung** — manipulasi nominal gaji, potongan, atau nomor rekening bank.
- **Kerentanan hukum** — tidak ada bukti elektronik yang dapat diverifikasi secara independen.
- **Kepatuhan tidak terbukti** — UU PDP No. 27 Tahun 2022 Pasal 8 mewajibkan organisasi menjaga
  kebenaran, keakuratan, dan konsistensi data pribadi, tetapi tidak ada mekanisme teknis yang
  membuktikan pemenuhannya secara otomatis.

---

## 2. Objektif

> **Ini adalah ratifikasi PB-4 / gap G-01.** Menggantikan `[ASUMSI]` yang sebelumnya tercatat di
> `agent-suite/11-execution/grounding-gaps.md` baris G-01. Diputuskan oleh Chandra Kurniawan,
> 2 Agustus 2026, sebagai keputusan manusia yang eksplisit.

### 2.1. Objektif utama

> **Membuktikan bahwa data profil karyawan pada SaaS HRIS tidak pernah dimanipulasi sejak terakhir
> diotorisasi secara sah — dan bahwa pembuktian itu dapat dilakukan sendiri oleh klien, secara
> independen, tanpa harus mempercayai penyedia platform.**

### 2.2. Objektif sekunder

Diakui dan dinyatakan, tetapi **tidak diukur** sebagai penentu keberhasilan utama:

| Kode | Objektif sekunder | Pemangku kepentingan |
|---|---|---|
| **OS-1** | Postur keamanan dan kelayakan integrasi memenuhi standar forum arsitektur/keamanan internal | Tim internal |
| **OS-2** | Kemampuan ini menjadi pembeda komersial pada segmen klien enterprise di sektor teregulasi | Produk / komersial |
| **OS-3** | Model integrasi DSRM + *Compliance-by-Design* dapat digunakan ulang untuk sistem SaaS teregulasi lain | Kontribusi teoretis |

### 2.3. Bukan tujuan (*non-goals*)

- Mengembangkan sistem HRIS secara keseluruhan.
- *Logging* aktivitas pengguna. Yang dicatat adalah **sidik jari data profil**, bukan jejak aktivitas.
- Menggantikan enkripsi at-rest yang sudah ada. Lapisan ini **menambah**, tidak menggantikan.
- Menyimpan data profil itu sendiri di blockchain.

---

## 3. Kriteria Sukses

Empat **predikat teruji** (*testable predicate*) — pernyataan yang dapat **digagalkan oleh sebuah
pengujian**, bukan indikator kinerja. Bentuk ini mengikuti rekomendasi `context/DSRM.md §4`.

| Kode | Predikat | Ambang lulus | Cara menggagalkan | Rujukan tesis |
|---|---|---|---|---|
| **P1** ⭐ | Setiap manipulasi langsung di basis data terdeteksi melalui *hash mismatch* | **100% pada kelima section** ⚠️ | **Satu** skenario manipulasi lolos tanpa terdeteksi | Tabel 4.4, SC-A…SC-D |
| **P2** | Pencatatan integritas tidak mengganggu operasional HRIS | Latensi tulis **< 3 detik pada beban 500 TPS** | Latensi melewati 3 detik pada beban target | BAB I 1.3 |
| **P3** | Isolasi data antartenant bersifat kriptografis | Akses lintas-tenant **ditolak** oleh MSP | Peer tenant B berhasil membaca record tenant A | Tabel 4.4, SC-E |
| **P4** | Kewajiban UU PDP dipenuhi secara teknis | Matriks bertingkat **§9** — bukan "5 pasal terpenuhi" | Ada kewajiban dalam matriks yang berstatus **Tidak dapat diklaim** tanpa keterbatasan dinyatakan | §9, matriks terkoreksi |

**P1 adalah metrik utama.** Ia dipilih karena falsifikasinya paling tegas: satu *counterexample*
sudah cukup meruntuhkan klaim. Untuk artefak keamanan, predikat semacam ini lebih kuat secara
metodologis daripada tolok ukur rata-rata.

> ⚠️ **P1 menuntut satu skenario uji tambahan.** Tabel 4.4 tesis hanya menguji *hash mismatch* pada
> **empat** section — PAYROLL (SC-A), PERSONAL (SC-B), EMPLOYMENT (SC-C), EDUCATION (SC-D). SC-E
> adalah uji isolasi channel, **bukan** uji hash. Section **`ADDITIONAL` belum pernah dimanipulasi
> dalam pengujian apa pun**, sementara abstrak dan BAB V mengklaim "100% pada lima section".
>
> Agar P1 sebagaimana diratifikasi benar-benar terbukti, **satu skenario manipulasi untuk section
> `ADDITIONAL` harus ditambahkan** (mis. mengubah status perkawinan atau data tanggungan). Lihat
> [`errata-tesis.md`](errata-tesis.md) butir **E-2**.

### 3.1. Status angka kinerja — WAJIB DIBACA

> ⚠️ **`[ASUMSI]`** Angka Hyperledger Caliper yang tercetak di tesis — **850,4 TPS** tulis,
> **1.950,5 TPS** baca, latensi **2,85 detik** (Tabel 4.3, diulang di Abstrak dan BAB V 5.1) —
> adalah **nilai placeholder, bukan hasil pengukuran.**
>
> Penyiapan Hyperledger Caliper v0.5 adalah **tugas build** (lihat Bagian 10.2). Hasil pengukuran
> yang sebenarnya akan **dituliskan kembali ke dalam tesis** menggantikan angka-angka tersebut.
>
> **Yang diratifikasi sebagai kriteria sukses adalah ambang P2 (< 3 detik @ 500 TPS), bukan angka
> hasilnya.** Ambang inilah yang dipakai untuk menyatakan lulus atau gagal.
>
> **Konsekuensi metodologis:** karena evaluasi kinerja kini masuk ke dalam cakupan, pernyataan di
> `context/DSRM.md §4` bahwa evaluasi kinerja *out of scope* (ditunda di balik gap G-08) **tidak lagi
> berlaku** dan harus direvisi. Gap **G-08 dibuka kembali**.

### 3.1a. Status matriks UU PDP — WAJIB DIBACA

> ⛔ **Klaim "5 pasal UU PDP terpenuhi" TIDAK BENAR — verifikasi terhadap naskah undang-undang
> menemukan 4 dari 5 nomor pasal salah kutip.** Lihat **Bagian 9** untuk matriks terkoreksi (12
> baris, tiga tingkat status) dan [`errata-tesis.md`](errata-tesis.md) butir **E-14**.
>
> **P4 karena itu tidak lagi didefinisikan sebagai "5 pasal terpenuhi".** Ambang yang berlaku:
> setiap kewajiban dalam matriks Bagian 9 berstatus *Comply*, *Comply sebagian*, atau *Comply
> dengan catatan* — dan setiap kewajiban berstatus **Tidak dapat diklaim** dinyatakan terbuka
> sebagai keterbatasan penelitian, bukan disembunyikan.

### 3.2. Celah yang diakui: kerahasiaan belum punya predikat

Keempat predikat di atas menguji **integritas, isolasi, dan kepatuhan**. **Tidak ada satu pun yang
menguji kerahasiaan** — yaitu bahwa tidak ada PII yang bocor ke ledger dalam bentuk yang dapat
dibalik.

Paket `agent-suite` memiliki pengujian untuk ini (**ST-1**, prioritas P0: pemindaian ledger penuh
untuk membuktikan 0 byte PII plaintext atau turunan yang reversibel). Klaim privasi di tesis BAB V
5.2.2 saat ini berdiri di atas **argumen**, belum di atas bukti pengujian. Lihat **OQ-1** di
[Bagian 12](#12-open-questions) — ini isu terbuka paling penting dalam dokumen ini.

---

## 4. Cakupan Anchoring

> **Ini adalah ratifikasi PB-5 / gap G-05.** Menggantikan `[ASUMSI]` di baris G-05, dan
> **membatalkan** permukaan berbasis event (`EVENT_UPDATE_PERSONAL`, `EVENT_ADD`,
> `EVENT_UPDATE_EMPLOYMENT`, `EVENT_RESIGN`) yang diasumsikan paket desain sebelumnya.

### 4.1. Unit anchoring: section profil, bukan event

Unit yang dicatat di ledger adalah **section profil**. Setiap section punya siklus pembaruan sendiri
dan rantai versi sendiri.

| Section | Tabel sumber | Frekuensi perubahan | Sensitivitas |
|---|---|---|---|
| **PERSONAL** | `employees` | Jarang | Sangat tinggi |
| **EMPLOYMENT** | `emp_employment` | Saat promosi / mutasi | Tinggi |
| **EDUCATION** | `emp_education` | Saat pendidikan baru selesai | Sedang |
| **ADDITIONAL** | `emp_additional` | Saat status perkawinan / tanggungan berubah | Tinggi |
| **PAYROLL** | `emp_payroll` | Saat penyesuaian gaji / ganti rekening | **Paling kritis** |

### 4.2. Pemicu pencatatan

Pencatatan dipicu **setiap kali sebuah section dibuat atau diperbarui**. Rantai `PrevHash` disusun
**per section per karyawan** — bukan satu rantai untuk seluruh profil.

### 4.3. Alasan granularitas per section

1. **Efisiensi** — pembaruan section `PAYROLL` tidak memicu pencatatan ulang section `EDUCATION`
   yang tidak berubah. Pada skala ribuan karyawan, beban on-chain jauh lebih ringan dibanding hash
   profil monolitik.
2. **Presisi forensik** — auditor dapat menunjuk **section mana** yang dimanipulasi tanpa memeriksa
   seluruh profil.

*(Rujukan: tesis BAB IV 4.2.3 dan pembahasan 4.4.1.)*

---

## 5. Invarian & Batasan Desain

| Kode | Invarian | Status |
|---|---|---|
| **INV-1** | Tidak ada data profil (plaintext) **maupun turunan yang dapat dibalik** yang disimpan di ledger. Yang on-chain hanya sidik jari ber-*salt* + CID dokumen terenkripsi. | Wajib |
| **INV-2** | Identitas on-chain dipseudonimisasi dengan **HMAC berkunci**, bukan hash telanjang. `EmployeeID` dan `UpdatedBy` tidak boleh dapat dibalik tanpa `employeeKey_i` milik karyawan tersebut. | Wajib — diratifikasi (§5.2, final 6 Agustus) |
| **INV-3** | Ledger bersifat *append-only*. Penghapusan data **tidak** dilakukan dengan mengubah state on-chain, melainkan melalui *crypto-shredding* off-chain. | Wajib |
| **INV-4** | Salinan ledger dipegang minimal tiga organisasi independen, sehingga penyedia platform tidak dapat mengubah catatan sendirian. | Wajib |
| **INV-5** | Isolasi antartenant ditegakkan secara kriptografis melalui *channel-per-tenant* + MSP, bukan secara logis. | Wajib |
| **INV-6** | Kanonikalisasi JSON harus deterministik dan identik antara penulis dan pemverifikasi. | Wajib — RFC-8785 (JCS), lihat PB-3 |

### 5.1. Batasan teknologi

- **Blockchain:** Hyperledger Fabric v2.5, state database **LevelDB**
- **Basis data operasional:** PostgreSQL (AWS RDS) — *lihat OQ-2*
- **Penyimpanan dokumen:** IPFS Private Cluster, terenkripsi per karyawan (`KEY_EMPLOYEE`)
- **Topologi:** 3 organisasi — penyedia platform (2 peer + 3 orderer Raft), klien enterprise
  (1 peer), auditor (1 peer *read-only*)

### 5.2. Konstruksi Kriptografis (diratifikasi)

> **Diratifikasi 2 Agustus 2026 oleh Chandra Kurniawan.** Menutup OQ-1 / errata **E-1**. Ini
> **satu-satunya** bagian yang mengikuti paket desain `agent-suite`, bukan tesis.

**Prinsip: pengenal dan isi data butuh perlakuan kriptografis yang berlawanan.** Isi harus
**non-deterministik** agar tidak dapat dikamuskan; pengenal harus **deterministik** agar dapat
ditelusuri dan dirantai. Satu primitif tidak dapat melayani keduanya.

**Kunci pengenal — acak independen, bukan diturunkan.** `employeeKey_i` dibuat sekali per karyawan
oleh CSPRNG (≥128 bit) saat *record* pertamanya ditulis, disimpan off-chain — **tidak ada kunci
induk** yang menurunkannya. Ini konstruksi final setelah **dua** koreksi:

1. **2 Agustus 2026** — draf awal menghitung `EmployeeID`/`UpdatedBy` langsung dari satu
   `pseudonymKey` yang dipakai bersama lintas karyawan. Ditemukan cacat: menghancurkan kunci itu
   untuk memenuhi penghapusan **satu** karyawan merusak verifikasi **semua** karyawan lain yang
   berbagi kunci tersebut.
2. **3 Agustus 2026** — diperbaiki dengan hierarki dua lapis: `pseudonymKey` (induk, lingkup tenant)
   menurunkan `employeeKey_i` (per karyawan) via HMAC. Ini menutup cacat #1 — tapi verifikasi
   kepatuhan UU PDP (§9) menemukan cacat baru: `employeeKey_i` adalah fungsi **deterministik** dari
   `pseudonymKey` + `employeeInternalId` (yang dapat dienumerasi) — jadi kompromi `pseudonymKey`
   membalikkan ulang **seluruh** penghapusan yang pernah dilakukan tenant itu, bukan cuma karyawan
   yang belum dihapus. Ditandai **T16b** di `security-architecture.md`.
3. **6 Agustus 2026 (final) — `pseudonymKey` dihapus sepenuhnya.** `employeeKey_i` diperlakukan
   persis seperti *salt* `DataHash` yang sudah bekerja: acak independen, bukan diturunkan dari apa
   pun. Tidak ada lagi kunci induk yang bisa membuka ulang riwayat. **T16b tertutup**, bukan cuma
   dimitigasi. Diputuskan Chandra Kurniawan secara eksplisit di antara tiga opsi
   (lihat `errata: ADR-0021`) — dipilih atas pertimbangan bahwa pembalikan massal penghapusan yang
   sudah selesai adalah moda kegagalan yang lebih berat daripada kehilangan satu `employeeKey_i`
   secara tidak sengaja (yang sudah diterima untuk `KEY_EMPLOYEE` di **OQ-5**).

| Field on-chain | Konstruksi | Kunci / *salt* | Sifat yang dijamin |
|---|---|---|---|
| `DataHash` | `SHA-256(salt ‖ JSON kanonik section)` | *salt* **≥128 bit**, CSPRNG, **baru untuk setiap record**, disimpan **hanya off-chain** | Isi ber-entropi rendah tidak dapat dipulihkan dengan serangan kamus |
| `EmployeeID` | `HMAC-SHA256(employeeKey_i, "id")` | `employeeKey_i` — CSPRNG ≥128 bit, **sekali per karyawan**, off-chain | **Deterministik** (pencarian & rantai bekerja), tidak tertaut tanpa kunci, **dapat dihapus per karyawan tanpa memengaruhi siapa pun** |
| `UpdatedBy` | `HMAC-SHA256(employeeKey_i, "actor" ‖ user_id)` | idem | idem |
| `PrevHash` | Merujuk `DataHash` record sebelumnya dari section yang sama | — | Rantai per section tetap utuh |

**Tiga domain kunci yang tidak boleh pernah dicampur:**

| Domain | Kunci | Fungsi |
|---|---|---|
| **A** | Kunci MSP / TLS Fabric | Penandatanganan identitas & transport |
| **B** | `KEY_EMPLOYEE` | Enkripsi dokumen pendukung di IPFS |
| **C** | `employeeKey_i` (acak independen, per karyawan) + *salt store* | Pseudonimisasi pengenal & pembukaan sidik jari |

Tidak satu pun boleh diturunkan dari yang lain **lintas domain**. Kunci yang dipakai untuk
*anchoring* tidak boleh juga dapat mendekripsi PII.

> ⚠️ **Trade-off yang diterima secara sadar.** Karena `employeeKey_i` tidak diturunkan dari apa pun,
> kehilangan **satu** `employeeKey_i` secara tidak sengaja (bukan karena permintaan penghapusan)
> menjadi **permanen dan tidak dapat dibedakan dari penghapusan yang disengaja** — persis moda
> kegagalan yang sudah diterima untuk `KEY_EMPLOYEE` (OQ-5). Bukan risiko baru; instansi kedua dari
> risiko yang sudah ada, dan sebaiknya ditangani disiplin *backup* yang sama untuk keduanya.

**Kanonikalisasi:** RFC-8785 (JCS), satu implementasi yang dipin untuk penulis **dan** pemverifikasi
(lihat PB-3 / gap G-24).

**Yang tidak berubah karena keputusan ini:** **tidak ada nilai P1–P4.** Deteksi tetap 100%, latensi
praktis tidak terpengaruh, isolasi tidak tersentuh — dan kepatuhan kerahasiaan (§9, Pasal 36 &
Pasal 39) justru **menguat**, karena sidik jari yang tertinggal setelah *crypto-shredding* kini
benar-benar *non-identifiable*.

> ⚠️ **Nomor pasal di paragraf sebelumnya sudah dikoreksi.** Draf sebelumnya menyebut "Pasal 16(2)
> serta Pasal 26" — verifikasi §9 menemukan kedua nomor itu **salah kutip**. Lihat Bagian 9 untuk
> pemetaan yang benar dan errata **E-14**.

**Edit tesis yang diperlukan:** definisi struct BAB IV 4.2.3 (`employeeKey_i` acak independen, bukan
diturunkan), satu kalimat klaim PbD di BAB II, satu kalimat BAB V 5.2.2, serta **seluruh Tabel 4.5** —
lihat Bagian 9. Pertimbangkan menambah `employeeKey_i` ke daftar keterbatasan BAB IV 4.5 sebagai
instansi kedua dari risiko kehilangan kunci yang sudah diakui untuk `KEY_EMPLOYEE`.

---

## 5a. Glosarium

| Istilah | Arti |
|---|---|
| **`DataHash`** *(= digest)* | Sidik jari `SHA-256(salt ‖ JSON kanonik section)` yang tersimpan on-chain untuk satu section profil satu versi |
| **`salt`** | Nilai acak ≥128 bit CSPRNG, baru per record, disimpan **hanya off-chain**; mencegah `DataHash` dibalik dengan serangan kamus |
| **`employeeKey_i`** | Kunci **acak independen** (CSPRNG ≥128 bit), **satu per karyawan**, dibuat sekali saat *record* pertama ditulis, disimpan off-chain; dipakai untuk `EmployeeID`/`UpdatedBy`; dihancurkan saat karyawan itu minta penghapusan. **Tidak diturunkan dari kunci induk apa pun** (final 6 Agustus 2026 — lihat §5.2) |
| **`EmployeeID`** | `HMAC-SHA256(employeeKey_i, "id")` — pengenal karyawan on-chain, deterministik namun tidak tertaut tanpa kunci |
| **`UpdatedBy`** | `HMAC-SHA256(employeeKey_i, "actor" ‖ user_id)` — pengenal aktor pengubah on-chain |
| **`KEY_EMPLOYEE`** | Kunci simetris per karyawan untuk mengenkripsi dokumen pendukung sebelum diunggah ke IPFS — **domain kunci terpisah** dari `employeeKey_i` |
| **`PrevHash`** | Referensi ke `DataHash` versi sebelumnya, dirantai **per section per karyawan** |
| **`ProfileSection`** | Salah satu dari lima nilai: `PERSONAL`, `EMPLOYMENT`, `EDUCATION`, `ADDITIONAL`, `PAYROLL` |
| **JCS** | RFC-8785 *JSON Canonicalization Scheme* — memastikan `SHA-256`/`HMAC` dihitung atas byte yang identik antara penulis dan pemverifikasi |
| **`[T]` / `[D]` / `[BARU]` / `[R]`** | Penanda asal FR: eksplisit-di-tesis / turunan / tambahan-baru / diratifikasi-mengikuti-ADR (§7) |

---

## 6. User Journey — Verifikasi

> **UJ-1 — Demonstrasi deteksi manipulasi data penggajian**
>
> **Protagonis:** Chandra Kurniawan, penulis, mempresentasikan langsung.
> **Audiens:** dosen penguji (utama), forum arsitektur/keamanan internal, dan ahli FGD.
> **Tujuan:** membuat audiens melihat sendiri bahwa manipulasi yang hari ini tidak terdeteksi,
> menjadi terdeteksi seketika.

**Alurnya:**

1. Chandra menampilkan profil karyawan `emp_id=123` di HRIS. Nomor rekening bank terlihat normal.
2. Ia membuka terminal dan mengeksekusi **langsung ke basis data, melewati aplikasi**:
   `UPDATE emp_payroll SET bank_account='rekening_pelaku' WHERE emp_id='123'`
3. Ia me-*refresh* HRIS. Nomor rekening **sudah berubah**. **Aplikasi tidak protes. Tidak ada
   peringatan. Tidak ada jejak.** — *"Inilah kondisi hari ini. Gaji karyawan ini akan masuk ke
   rekening pelaku, dan tidak ada sistem yang tahu."*
4. Ia memanggil `VerifyProfileIntegrity('123','PAYROLL')`.
5. Hasilnya muncul: **HASH MISMATCH** — `computedHash` ≠ `expectedHash` yang tersimpan di ledger.
   **Inilah momen pembuktiannya.**
6. Ia memanggil `GetProfileHistory('123','PAYROLL')` → riwayat versi muncul kronologis. Ledger
   **menunjuk versi mana yang terakhir valid** — `Version`, `Timestamp`, `UpdatedBy`-nya — tapi
   **tidak mengembalikan nilai rekening itu sendiri**, karena yang tersimpan hanyalah `DataHash`
   satu arah. Chandra memulihkan nomor rekening asli dari *snapshot* PostgreSQL pada `Timestamp`
   tersebut, lalu memakai `VerifyProfileIntegrity` untuk **membuktikan** kandidat pemulihan itu
   cocok dengan `DataHash` di ledger — properti yang lebih kuat daripada sekadar menyimpan salinan,
   karena salinan pun bisa dimanipulasi. *(Lihat OQ-6 §12 — ini klaim yang sudah dipersempit dari
   draf awal.)*
7. Penutup: *"Yang menghentikan saya menutupi ini bukan kebijakan perusahaan, tapi kriptografi.
   Saya harus menguasai ketiga organisasi sekaligus untuk mengubah catatan itu."*

**Mengapa ini meyakinkan:** langkah 3 memperlihatkan kerugian, langkah 5 memperlihatkan
penyelesaiannya — dan jaraknya hanya beberapa detik. Verifikasi tidak memerlukan konsensus
*ordering*, cukup membaca *world state* peer lokal, sehingga responsnya seketika.

**UJ-2 (turunan) — Isolasi antartenant.** Peer tenant B mencoba men-*query* record profil karyawan
tenant A → **ACCESS DENIED** oleh *channel membership* MSP. Ditolak di lapisan kriptografis, bukan
lapisan aplikasi.

---

## 7. Kebutuhan Fungsional

Diturunkan dari empat fungsi chaincode di tesis BAB IV 4.2.3 dan struktur
`EmployeeProfileRecord`. Setiap FR diberi penanda asal:

| Penanda | Arti |
|---|---|
| **`[T]`** | Eksplisit tertulis di tesis |
| **`[D]`** | Turunan logis dari desain tesis |
| **`[BARU]`** | Tambahan — belum ada di tesis, perlu keputusan apakah masuk cakupan |
| **`[R]`** | **Diratifikasi 2 Agustus 2026** — mengikuti `ADR-0001`/`data-model.md`, bukan tesis (§5.2) |

### Kelompok A — Pencatatan Integritas Section (`RecordProfileSection`)

| ID | Kebutuhan | Asal |
|---|---|---|
| **FR-1** | Sistem HARUS mencatat satu `EmployeeProfileRecord` baru setiap kali salah satu dari lima section profil dibuat atau diperbarui melalui alur aplikasi HRIS yang sah. | `[T]` |
| **FR-2** | Sidik jari section HARUS dihitung dari representasi **JSON kanonik** section tersebut, memakai satu skema kanonikalisasi deterministik (RFC-8785 / JCS) yang **identik** antara penulis dan pemverifikasi. | `[T]` + PB-3 |
| **FR-3** | Sidik jari HARUS dihitung sebagai `SHA-256(salt ‖ JSON kanonik)` dengan ***salt* ≥128 bit** dari CSPRNG, **baru untuk setiap record**, disimpan **hanya off-chain**. | `[R]` |
| **FR-4** | Setiap record HARUS dirantai ke record sebelumnya **dari section yang sama pada karyawan yang sama** melalui `PrevHash`, dengan `Version` diinkremen. | `[T]` |
| **FR-5** | `EmployeeID` dan `UpdatedBy` HARUS dihitung dari **`employeeKey_i`** — kunci **acak independen** (CSPRNG ≥128 bit, bukan diturunkan dari kunci apa pun), dibuat sekali per karyawan, disimpan off-chain — sehingga bersifat deterministik (pencarian & rantai bekerja), tidak dapat dibalik tanpa kunci, **dan dapat dihapus per karyawan tanpa memutus karyawan lain maupun berisiko dibalik ulang oleh kompromi kunci induk**. | `[R]` |
| **FR-6** | Sistem HARUS **menolak** pencatatan bila identitas pemanggil tidak terverifikasi oleh MSP. | `[T]` → Pasal 43 |
| **FR-7** | Setiap record HARUS memiliki pengenal unik (`RecordID`, UUID v4) dan penanda waktu. | `[T]` |
| **FR-8** | Sistem HARUS memvalidasi bahwa `ProfileSection` adalah salah satu dari lima nilai sah, dan menolak nilai di luar itu. | `[D]` |
| **FR-9** | Pencatatan dengan konten section yang **identik** dengan versi terakhir TIDAK BOLEH menghasilkan record baru — mencegah pembengkakan ledger tanpa nilai forensik. | `[BARU]` |

### Kelompok B — Verifikasi Integritas (`VerifyProfileIntegrity`)

| ID | Kebutuhan | Asal |
|---|---|---|
| **FR-10** | ~~Sistem menerima data section profil dari pemanggil, menghitung ulang sidik jarinya, dan membandingkan di dalam chaincode.~~ **DIREVISI** → lihat Kelompok B′. Bentuk asli ini mengirim PII plaintext sebagai argumen chaincode. | `[T]` ❌ |
| **FR-11** | Hasil verifikasi HARUS memuat `{isValid, expectedHash, computedHash, version, timestamp}` — dirakit di sisi pemverifikasi, bukan dikembalikan chaincode. | `[T]` (direvisi) |
| **FR-12** | Verifikasi HARUS dapat dijalankan sebagai **operasi baca** terhadap *world state* peer lokal, tanpa memerlukan konsensus *ordering* — sehingga responsnya seketika. | `[T]` |
| **FR-13** | Verifikasi HARUS dapat dipanggil oleh **karyawan (subjek data)**, HR manager, dan **auditor**. | `[T]` → Pasal 4(2) |
| **FR-14** | Endpoint verifikasi TIDAK BOLEH mengembalikan *salt* dalam bentuk apa pun, pada respons mana pun. Penyerahan *salt* kepada pemverifikasi berwenang berjalan lewat jalur terpisah (FR-36). | `[R]` |
| **FR-15** | Kegagalan verifikasi HARUS dapat dibedakan antara **data tidak cocok** (manipulasi) dan **record tidak ditemukan** (belum pernah di-*anchor*). | `[BARU]` |

### Kelompok B′ — Protokol Verifikasi Sisi-Klien (revisi 2 Agustus 2026)

> **Mengapa direvisi.** Dua cacat bertemu di satu titik, dan solusinya sama.
>
> **Cacat 1 — PII plaintext jadi argumen chaincode.** Bentuk `VerifyProfileIntegrity(sectionData)` di
> tesis mengirim isi section ke *peer* untuk dihitung di sana. Di bawah topologi tiga organisasi,
> *peer* itu bisa milik Org2 atau Org3 — artinya **PII plaintext melintasi batas organisasi**. Lebih
> buruk: objektif yang diratifikasi adalah *"klien memverifikasi tanpa mempercayai vendor"*, tetapi
> bentuk ini justru membuat klien **mengirim plaintext-nya ke peer vendor** setiap kali verifikasi.
> Kebalikan dari maksudnya.
>
> **Cacat 2 — *salt* memutus verifikasi independen.** Setelah §5.2 diadopsi, `DataHash` memerlukan
> *salt* untuk dihitung ulang. Kalau hanya vendor yang memegang *salt*, klien tidak bisa lagi
> memverifikasi sendiri — dan kepercayaan institusional kembali masuk lewat pintu belakang.
>
> **Kunci penyelesaiannya:** *salt* hanya perlu dirahasiakan dari pihak yang **belum** memegang
> plaintext. Pihak yang sudah sah memegang datanya tidak belajar apa pun dari *salt*-nya. Jadi *salt*
> boleh diserahkan kepada pemegang data yang berwenang — melalui jalur terkendali yang sama dengan
> datanya, **bukan** melalui endpoint verifikasi.

| ID | Kebutuhan | Asal |
|---|---|---|
| **FR-34** | Isi section profil (plaintext) **TIDAK BOLEH** pernah menjadi argumen transaksi chaincode — baik *submit* maupun *evaluate*. | `[R]` |
| **FR-35** | Sistem HARUS menyediakan operasi **baca saja** yang mengembalikan `{DataHash, Version, Timestamp, UpdatedBy}` tersimpan, tanpa menerima data profil apa pun. Penghitungan ulang dan pembandingan dilakukan **di sisi pemverifikasi**. | `[R]` |
| **FR-36** | Sistem HARUS menyediakan jalur terkendali dan ter-audit bagi pemverifikasi berwenang untuk memperoleh ***salt*** versi yang diperiksa — terpisah dari endpoint verifikasi (FR-14 tetap berlaku). Kewenangan mengikuti kepemilikan data: karyawan atas datanya sendiri, auditor sebatas lingkup auditnya. | `[R]` |
| **FR-37** | Pemverifikasi milik organisasi yang punya *peer* sendiri (Org2 klien, Org3 auditor) HARUS dapat menjalankan FR-35 terhadap ***peer* miliknya sendiri** — sehingga ia tidak perlu mempercayai *peer* vendor untuk menjawab jujur. | `[R]` |

**Properti yang diperoleh dibanding bentuk asli:**

| | Bentuk tesis | Protokol B′ |
|---|---|---|
| Plaintext masuk argumen chaincode | ✅ ya | ❌ **tidak pernah** |
| Plaintext transit ke *peer* / log *peer* | ✅ ya | ❌ tidak |
| Perlu percaya *peer* vendor menjawab jujur | ✅ ya | ❌ tidak (FR-37) |
| Tetap operasi baca (sub-detik) | ✅ | ✅ |
| Rahasia terhadap pengamat ledger-saja | ❌ (tanpa *salt*) | ✅ |
| Verifikasi independen oleh klien | ✅ | ✅ |

**Ini memperkuat klaim inti tesis** — *code-based trust* — pada dimensi yang justru diklaimnya.

### Kelompok C — Riwayat & Audit Forensik

| ID | Kebutuhan | Asal |
|---|---|---|
| **FR-16** | Sistem HARUS mengembalikan **seluruh riwayat versi** sebuah section profil secara kronologis (`GetHistoryForKey`). | `[T]` |
| **FR-17** | Sistem HARUS dapat menunjukkan **versi valid terakhir** sebuah section — yaitu versi terakhir yang sidik jarinya cocok. ⚠️ *Lihat OQ-6: ledger dapat mengidentifikasi versi valid terakhir, tetapi tidak dapat memulihkan nilainya.* | `[T]` |
| **FR-18** | Sistem HARUS mengembalikan sidik jari dan versi terkini **seluruh lima section** seorang karyawan dalam satu panggilan, untuk keperluan audit cepat. | `[T]` |
| **FR-19** | Setiap entri riwayat HARUS mencantumkan aktor terpseudonimisasi dan penanda waktunya. | `[D]` |

### Kelompok D — Isolasi Tenant

| ID | Kebutuhan | Asal |
|---|---|---|
| **FR-20** | Setiap tenant HARUS dipetakan ke **channel Fabric terpisah**. | `[T]` |
| **FR-21** | Upaya peer suatu tenant membaca record tenant lain HARUS **ditolak** oleh penegakan *channel membership* MSP — di lapisan kriptografis, bukan lapisan aplikasi. | `[T]` |
| **FR-22** | Klien enterprise HARUS memiliki peer yang menyimpan **salinan ledger independen** untuk tenant-nya, sehingga penyedia platform tidak dapat mengubah catatan sendirian. | `[T]` |
| **FR-23** | Auditor HARUS memiliki peer **read-only** untuk verifikasi kepatuhan. | `[T]` |
| **FR-24** | Sistem HARUS menyediakan proses **provisioning channel untuk tenant baru** (*onboarding*), termasuk pendaftaran identitas MSP dan penyebaran chaincode ke channel tersebut. | `[BARU]` |

### Kelompok E — Penghapusan Data (*Crypto-Shredding*)

| ID | Kebutuhan | Asal |
|---|---|---|
| **FR-25** | Atas permintaan penghapusan yang sah, sistem HARUS menghapus field data profil di basis data operasional. | `[T]` |
| **FR-26** | Sistem HARUS **menghancurkan `KEY_EMPLOYEE`** karyawan tersebut, sehingga seluruh dokumen pendukungnya di IPFS tidak dapat lagi didekripsi. | `[T]` |
| **FR-27** | Sistem **TIDAK BOLEH** mengubah, menimpa, atau menghapus state on-chain untuk memenuhi permintaan penghapusan. Sidik jari yang tertinggal menjadi *orphan* yang tidak dapat diidentifikasi. | `[T]` |
| **FR-28** | Penghapusan HARUS mencakup **penghancuran *salt* dan `employeeKey_i`** karyawan tersebut — tanpa keduanya, sidik jari dan pengenal masih dapat dibuka. Karena `employeeKey_i` tidak diturunkan dari kunci induk apa pun, penghancuran ini **final dan tidak dapat dibalik oleh siapa pun** (§5.2, final 6 Agustus). | `[R]` |
| **FR-29** | Sistem HARUS mencatat **bukti pelaksanaan** penghapusan, agar pemenuhan Pasal 26 dapat dibuktikan kepada regulator. | `[BARU]` |

### Kelompok F — Dokumen Pendukung (IPFS)

| ID | Kebutuhan | Asal |
|---|---|---|
| **FR-30** | Setiap dokumen pendukung HARUS dienkripsi dengan `KEY_EMPLOYEE` **sebelum** diunggah ke IPFS. | `[T]` |
| **FR-31** | **Hanya CID** yang boleh tercatat on-chain. Isi dokumen tidak boleh pernah menyentuh ledger. | `[T]` |
| **FR-32** | Setiap CID HARUS terkait ke section yang relevan (KTP → PERSONAL, ijazah → EDUCATION, slip gaji → PAYROLL, dst.). | `[T]` |
| **FR-33** | Sistem HARUS memastikan objek IPFS hanya direplikasi **di dalam** *private cluster* — objek yang bocor ke jaringan publik tidak dapat ditarik kembali, dan *crypto-shredding* menjadi tidak lengkap. | `[BARU]` |

---

## 8. Kebutuhan Non-Fungsional

| ID | Kebutuhan | Ambang / Ukuran | Terkait |
|---|---|---|---|
| **NFR-1** | Kinerja tulis pencatatan integritas | Latensi **< 3 detik pada beban 500 TPS** | **P2** |
| **NFR-2** | Kinerja verifikasi | Responsif untuk pemakaian interaktif (sub-detik), karena tidak memerlukan *ordering* | **P2**, FR-12 |
| **NFR-3** | Kerahasiaan | **0 byte** PII plaintext maupun turunan yang **dapat dibalik** di ledger — pengujian **ST-1 (P0)** | INV-1, §5.2 |
| **NFR-4** | Integritas | Ledger *append-only*; salinan independen di **minimal 3 organisasi** | INV-3, INV-4 |
| **NFR-5** | Isolasi | Penegakan kriptografis antartenant, bukan logis | INV-5, **P3** |
| **NFR-6** | Keamanan transport | **mTLS** pada seluruh endpoint peer, orderer, dan CA | — |
| **NFR-7** | Determinisme | Kanonikalisasi JSON menghasilkan byte identik antara penulis dan pemverifikasi | INV-6, PB-3 |
| **NFR-8** | Auditabilitas | Setiap perubahan terlacak ke aktor terverifikasi yang tidak dapat dihapus | Pasal 43 |
| **NFR-9** | Dampak minimal pada HRIS | Penambahan latensi pada jalur tulis HRIS akibat pencatatan integritas **< 500 ms** dibanding tanpa pencatatan *(ambang awal — kalibrasi ulang setelah Caliper §10.4 butir 3 berjalan)* | Kriteria FGD |
| **NFR-10** | Skalabilitas operasi massal | Operasi massal (mis. penyesuaian gaji tahunan ribuan karyawan) ditangani dengan *batching* | Tesis 4.3.1 |
| **NFR-11** | Reproducibilitas evaluasi | Lingkungan *benchmark* dapat dibangun ulang dan dijalankan ulang untuk menghasilkan angka yang dapat diperiksa penguji | Bagian 10.2 |

---

## 9. Kepatuhan UU PDP No. 27 Tahun 2022

> ⛔ **Tabel 4.5 tesis TIDAK DAPAT DIPERTAHANKAN dalam bentuk sekarang.** Verifikasi terhadap naskah
> resmi UU No. 27 Tahun 2022 (7 pembaca independen + 1 kritikus kelengkapan, 3 Agustus 2026)
> menemukan **4 dari 5 nomor pasal salah kutip** — dan Pasal 16(2), satu-satunya yang nomornya
> benar, salah **kategori norma**-nya (dikutip sebagai prinsip, padahal kewajibannya ada di pasal
> lain). Dua baris bahkan **saling tertukar isi**.
>
> **Kabar baiknya:** perbaikannya bukan mempersempit klaim, melainkan **memperluasnya**. Artefak
> yang diratifikasi memenuhi **lebih banyak** kewajiban daripada yang diklaim tesis — sekitar
> **12 pasal**, bukan 5 — asalkan disajikan sebagai matriks bertingkat yang jujur, bukan "*Comply*"
> seragam. Metode verifikasi lengkap ada di catatan kaki §9.5.

### 9.1. Kesalahan pada Tabel 4.5 tesis

| Baris tesis | Pasal yang diklaim | Isi yang diklaim | Nomor **benar** | Isi yang **sesungguhnya** diatur |
|---|---|---|---|---|
| 1 | Pasal 4 (2) | Hak memperbarui/memperbaiki data | ❌ **Salah total** | Pasal 4 adalah **klasifikasi jenis data** (spesifik/umum) — tidak memuat hak, kewajiban, atau larangan apa pun |
| 2 | Pasal 8 | Kewajiban menjaga kebenaran/akurasi data | ❌ **Salah total** | Pasal 8 adalah **hak mengakhiri pemrosesan/menghapus/memusnahkan** — persis isi yang tesis maksudkan untuk baris 4 |
| 3 | Pasal 16 (2) | Kewajiban melindungi dari akses tidak sah | ⚠️ **Nomor benar, kategori salah** | Pasal 16(2) huruf e hanya **prinsip** (Bab V); kewajiban konkretnya ada di Pasal 39 |
| 4 | Pasal 26 (1) | Hak penghapusan (*right to be forgotten*) | ❌ **Salah total** | Pasal 26 mengatur **pemrosesan data penyandang disabilitas** — tidak ada hubungan sama sekali dengan penghapusan |
| 5 | Pasal 43 | Larangan memproses data tidak sah | ❌ **Salah kategori & bertukar isi** | Pasal 43 adalah **kewajiban menghapus data** — bukan larangan, dan sudah dipakai (salah) di baris 4 |

**Pola kesalahannya sistemik**, bukan salah ketik satu-dua tempat — karena itu perbaikannya adalah
**mengganti seluruh tabel**, bukan menambal sel.

### 9.2. Matriks Kepatuhan Terkoreksi

12 baris, tiga tingkat status kejujuran: **Comply** (terpenuhi penuh) · **Comply sebagian** (mekanisme
menyentuh sebagian unsur pasal) · **Comply dengan catatan** (terpenuhi, tapi bersandar pada argumen
hukum yang belum teruji) · **Tidak dapat diklaim** (dinyatakan terbuka sebagai keterbatasan).

| # | Pasal | Jenis | Kewajiban/hak (ringkas) | Mekanisme artefak | Status |
|---|---|---|---|---|---|
| 1 | **Pasal 6** jo. **Pasal 30** jo. **Pasal 14** | Hak (P6) + Kewajiban Pengendali (P30) | Subjek berhak melengkapi/memperbarui/memperbaiki data; Pengendali wajib menuntaskan **3×24 jam** + memberitahukan hasil; diajukan via **permohonan tercatat** | `RecordProfileSection` mencatat pembaruan; data otoritatif **mutable** di RDBMS operasional (bukan ledger) — rektifikasi yuridis dimungkinkan; histori versi + `PrevHash` sebagai bukti kronologis | **Comply sebagian** |
| 2 | **Pasal 29 (1)–(2)** jo. Pasal 16(2)(d) | Kewajiban Pengendali | Wajib memastikan akurasi, kelengkapan, konsistensi; wajib **verifikasi** | *Digest* ber-*salt* per section + konsensus multi-pihak; `VerifyProfileIntegrity` sebagai operasi verifikasi | **Comply sebagian** |
| 3 | **Pasal 39 (1)–(2)** jo. Pasal 35 jo. Pasal 52 | Kewajiban Pengendali (+ Prosesor) | Wajib mencegah akses tidak sah; wajib menentukan tingkat keamanan sesuai sifat/risiko data | MSP validasi identitas; channel per tenant; dokumen IPFS terenkripsi `KEY_EMPLOYEE`; plaintext tidak pernah jadi argumen chaincode (§5.2 B′) | **Comply sebagian** |
| 4 | **Pasal 8** jo. **Pasal 43 (1)** jo. **Pasal 44 (1)** jo. Pasal 45 | Hak (P8) + Kewajiban Pengendali (P43/44/45) | Hak menghapus/memusnahkan; Pengendali wajib menghapus/memusnahkan + memberitahukan | *Crypto-shredding*: hapus field DB + hancurkan `KEY_EMPLOYEE` + hancurkan *salt* + `employeeKey_i` (§5.2, final 6 Agustus — `employeeKey_i` acak independen, penghancuran tidak dapat dibalik oleh kunci apa pun) | **Comply dengan catatan** ⚠️ |
| 5 | **Pasal 38** jo. Pasal 39(1) jo. Pasal 37 | Kewajiban Pengendali | Melindungi dari pemrosesan tidak sah; mengawasi setiap pihak yang terlibat | Validasi MSP sebelum mencatat; `UpdatedBy` (HMAC) terverifikasi; peer *read-only* Org3 mengawasi independen | **Comply dengan catatan** |
| 6 | **Pasal 31** jo. Pasal 52 *(baris baru)* | Kewajiban Pengendali (+ Prosesor) | Wajib **merekam seluruh kegiatan pemrosesan** | Ledger *append-only* per section; `PrevHash` berantai; `Version`+`Timestamp`+`UpdatedBy` tiap tulis | **Comply sebagian** |
| 7 | **Pasal 47** jo. Pasal 16(2)(h) *(baris baru)* | Kewajiban Pengendali | Wajib **menunjukkan** pertanggungjawaban; **dapat dibuktikan secara jelas** | Verifikasi sisi-klien; Org2/Org3 verifikasi terhadap peer masing-masing tanpa bergantung itikad baik Org1 | **Comply** ✅ *(terkuat di matriks)* |
| 8 | **Pasal 36** jo. Pasal 52 *(baris baru)* | Kewajiban Pengendali (+ Prosesor) | Wajib menjaga **kerahasiaan** | Tidak ada plaintext/turunan reversibel di ledger; *salt* + HMAC berkunci off-chain; dokumen terenkripsi per karyawan | **Comply** ✅ |
| 9 | **Pasal 4(2)(f)** jo. Pasal 35(b) *(baris baru)* | Klasifikasi (pemicu kewajiban lain) | Data keuangan = data **spesifik**; tingkat keamanan mengikuti sifat/risiko | Section `PAYROLL` + dokumen pendukung diperlakukan berbeda: terenkripsi, hanya CID on-chain | **Comply dengan catatan** |
| 10 | **Pasal 34 (1)–(2)** *(baris baru)* | Kewajiban Pengendali | Wajib **DPIA** untuk pemrosesan risiko tinggi (data spesifik, skala besar, teknologi baru — **4 pintu terpicu sekaligus**) | Tidak ada — DPIA belum disusun | **Tidak dapat diklaim** ⛔ |
| 11 | **Pasal 42 (1)(a)(b)** jo. Pasal 21(1)(d) *(baris baru)* | Kewajiban Pengendali | Wajib mengakhiri pemrosesan saat **retensi tercapai** atau **tujuan selesai** — berjalan **otomatis**, tanpa permintaan | Tidak ada — ledger menyimpan tanpa batas waktu, tanpa kebijakan retensi | **Tidak dapat diklaim** ⛔ |

### 9.3. Dua baris "Tidak dapat diklaim" — wajib masuk keterbatasan penelitian

**Baris 10 (DPIA, Pasal 34).** Artefak ini memicu kewajiban DPIA lewat **sekurang-kurangnya empat**
pintu ayat (2) sekaligus — termasuk huruf f *"penggunaan teknologi baru"*, yang justru **klaim
kebaruan tesis itu sendiri**. DPIA belum disusun.

**Baris 11 (Retensi, Pasal 42).** Ledger *append-only* **tidak pernah** berhenti menyimpan.
*Crypto-shred* **tidak menjawab ini** — ia dipicu **permintaan** (Pasal 43(1)(c)), sedangkan
kewajiban retensi Pasal 42 berjalan **otomatis berdasarkan waktu/tujuan**, tanpa permintaan siapa
pun. Kasus konkret yang akan ditanyakan: *"karyawan resign 2026, lima belas tahun kemudian
record-nya masih ada di ledger — atas dasar apa?"*

### 9.4. Pertanyaan hukum yang belum terjawab

**UU PDP TIDAK mengenal anonimisasi.** Pencarian menyeluruh atas naskah undang-undang: kata
"anonim", "anonimisasi", "pseudonim", "penyamaran", "enkripsi", "penyandian" — **nol kemunculan**,
baik di batang tubuh maupun Penjelasan. Tidak ada *safe harbour* seperti Pasal 4(5)/Recital 26 GDPR.

**Konsekuensi:** klaim *crypto-shredding* memenuhi kewajiban adalah **argumen hukum yang belum
diuji**, bukan pemenuhan tekstual — dan nasibnya **berbeda** antara dua kata kerja:

| Kata kerja UU | Dasar tekstual | Kekuatan argumen *crypto-shred* |
|---|---|---|
| **"Memusnahkan"** (Pasal 44) | Didefinisikan **berbasis hasil**: *"sehingga tidak lagi dapat digunakan untuk mengidentifikasi Subjek Data Pribadi"* | **Lebih kuat** — hasil akhirnya cocok |
| **"Menghapus"** (Pasal 43) | **Tidak didefinisikan** UU; Pasal 16(1)(f) memperlakukan hapus dan musnah sebagai **dua tindakan berbeda** | **Lebih lemah** — belum tentu setara |

**Penentu keabsahan seluruh matriks — peran hukum Org1 dan Org2 belum ditetapkan.** Pasal 52 memuat
daftar **tertutup** tujuh pasal (29, 31, 35, 36, 37, 38, 39) yang mengikat **Prosesor**. Menurut
Pasal 1 angka 4–5, Org2 (klien, yang menentukan tujuan pemrosesan) kemungkinan **Pengendali**, dan
Org1 (penyedia SaaS) kemungkinan **Prosesor**. Jika benar, kewajiban Pasal 30/42/43/44/45/46/47
**tidak mengikat Org1** — melainkan Org2. Ini **bukan detail redaksional**; ia menentukan pasal mana
yang boleh diklaim artefak, dan siapa yang menanggung kewajibannya.

**Replikasi ledger ke Org3 (auditor) adalah "transfer" secara harfiah.** Penjelasan Pasal 16(1)(e)
mendefinisikan transfer sebagai *"perpindahan, pengiriman, dan/atau penggandaan ... kepada pihak
lain"*. Dasar pemrosesan (Pasal 20(2)) untuk aliran data ke organisasi auditor **belum dinyatakan**.

**Paradoks yang harus diakui lebih dulu oleh penulis.** Penjelasan Pasal 46(1) memasukkan kegagalan
**integritas** dan **perubahan tidak sah** ke dalam definisi kegagalan Pelindungan Data Pribadi.
Artefak ini **dirancang untuk mendeteksinya** — sehingga setiap deteksi manipulasi yang berhasil
justru **mengaktifkan** kewajiban notifikasi 3×24 jam Pasal 46. Menjalankan sistem ini di produksi
**menaikkan** eksposur kepatuhan, bukan menurunkannya. Lebih baik penulis menyatakan ini sendiri
daripada ditemukan penguji.

### 9.5. Metode verifikasi

Naskah diperoleh dari salinan dwibahasa kantor hukum ABNR (abnrlaw.com), dikonfirmasi silang via
pasal.id, uupdp-info.id, dan bplawyers.co.id. Confidence **tinggi** untuk seluruh kutipan verbatim
di atas. Situs unduhan resmi (peraturan.bpk.go.id, hukumonline) menolak akses otomatis (HTTP 403)
dan PDF resmi yang berhasil diunduh berupa hasil pindai tanpa teks. **Sebelum sidang, verifikasi
ulang seluruh kutipan langsung terhadap Lembaran Negara RI Tahun 2022 Nomor 196** — terutama
Pasal 14, Pasal 30 (pembagian ayat), dan Pasal 44(1)(a)/(c) — sebagai sumber primer akhir; jangan
mengutip hasil tinjauan ini sebagai rujukan penutup.

### 9.6. Batas klaim yang harus dijaga di seluruh matriks

Dua hal yang **tidak boleh** diklaim lebih dari kenyataannya, berlaku lintas baris:

1. **Sistem membuktikan integritas, bukan akurasi** (baris 2). Verifikasi hash membuktikan data
   **tidak berubah sejak dicatat** — bukan bahwa datanya **benar**. Data yang salah sejak awal akan
   tetap lolos verifikasi.
2. **Isolasi antartenant: nyatakan mekanismenya, jangan labelnya** (baris 3, 5). Lihat
   [`errata-tesis.md`](errata-tesis.md) butir **E-11** — bila demonstrasi tidak benar-benar
   menjalankan dua channel dan dua tenant, isolasi harus dinyatakan sebagai *identity-bound* /
   ditegakkan-chaincode, bukan "kriptografis".
3. **§5.2 (*salt* + HMAC) memperkuat baris 8 (kerahasiaan) secara nyata** — sebelum adopsi, hash
   telanjang atas data ber-entropi rendah dapat dibalik oleh **setiap** pemegang replika ledger,
   termasuk Org2 dan Org3. Setelah adopsi, tidak dapat dibalik tanpa kunci off-chain di domain
   terpisah.

---

## 10. Metode Evaluasi (DSRM)

### 10.1. Pemetaan ke lima keluarga evaluasi Hevner

`context/DSRM.md §4` memetakan lima keluarga metode evaluasi desain (Hevner et al., 2004). Untuk
artefak **keamanan**, sebuah kontrol **diargumentasikan dan diuji**, bukan semata di-*benchmark* —
penekanannya berbeda dari artefak kinerja.

| Keluarga | Penerapan pada prototipe ini | Predikat | Status |
|---|---|---|---|
| **Descriptive** (*informed argument*, skenario) | Argumentasi setiap kontrol terhadap kerangka keamanan; penelusuran skenario *anchor* / kerahasiaan / akses / erasure | — | ✅ Selesai (G6) |
| **Analytical** (analisis arsitektur & statis) | Matriks keterlacakan kontrol→ancaman→uji; telaah statis chaincode dan *seam* integrasi | — | ✅ Selesai (G6/G8a) |
| **Testing** (*black-box* fungsional, *white-box* struktural) | *Black-box*: buktikan *anchor*-lalu-verifikasi tanpa mengekspos plaintext. *White-box*: uji unit chaincode + endorsement | **P1**, **P3** | ⏳ Fase 4 |
| **Experimental** (eksperimen terkendali, simulasi) | **Simulasi upaya manipulasi terhadap ledger imutabel** — SC-A…SC-D ditambah skenario `ADDITIONAL` yang baru | **P1** | ⏳ Fase 4 |
| **Observational** (studi kasus / lapangan) | Studi kasus satu tenant pilot | — | ⛔ Ditunda pasca-G9 |

**Konsekuensi:** metode yang *sekarang* berlaku adalah **Descriptive + Analytical**; **Testing +
Experimental menjadi utama di Fase 4 Demonstrasi.**

### 10.2. Dua metode yang harus ditambahkan ke `DSRM.md`

Tesis menggunakan dua metode evaluasi yang **tidak tercantum** di tabel `context/DSRM.md §4`:

| Metode | Predikat | Keluarga Hevner | Tindakan |
|---|---|---|---|
| **Analisis kepatuhan regulasi** (matriks UU PDP) | **P4** | *Analytical* — pemetaan artefak terhadap norma eksternal | Tambahkan sebagai baris eksplisit |
| **Focus Group Discussion** | — (kualitatif) | *Descriptive* + *Observational* | Tambahkan; lihat catatan §10.4 |

Tanpa penambahan ini, dua dari empat jalur evaluasi tesis **tidak punya rumah metodologis** — celah
yang akan ditanyakan penguji DSRM yang melacak keenam aktivitas satu per satu.

### 10.3. Empat jalur evaluasi dan pemetaan predikatnya

| # | Jalur | Instrumen | Predikat | Catatan penting |
|---|---|---|---|---|
| **1** | Pengujian keamanan | Simulasi manipulasi langsung di basis data | **P1** | Butuh **skenario `ADDITIONAL` tambahan** — errata **E-2** |
| **2** | Isolasi antartenant | Upaya akses lintas-tenant | **P3** | Butuh **tenant kedua + peer tenant kedua** agar reproducible — errata **E-11** |
| **3** | Benchmark kinerja | Hyperledger Caliper v0.5 | **P2** | Angka saat ini **placeholder** — §3.1 |
| **4** | Analisis kepatuhan | Matriks UU PDP | **P4** | Nomor pasal sedang diverifikasi — §9 |
| **5** | Evaluasi kualitatif | Focus Group Discussion | — | Instrumen tidak konsisten — errata **E-6** |

### 10.4. Prasyarat yang harus dipenuhi agar evaluasi valid

Lima hal yang saat ini **menghalangi** evaluasi menghasilkan bukti yang dapat dipertahankan:

| # | Prasyarat | Errata | Sifat |
|---|---|---|---|
| **1** | Skenario manipulasi section `ADDITIONAL` ditambahkan, agar klaim "100% pada lima section" benar | **E-2** | Satu skenario uji |
| **2** | Tenant kedua + peer tenant kedua disediakan, agar uji isolasi benar-benar dapat dijalankan | **E-11** | Provisioning |
| **3** | *Harness* Caliper v0.5 dibangun dan dijalankan; angka nyata **menggantikan** placeholder di Tabel 4.3, Abstrak, dan BAB V 5.1 | **E-8** | Tugas build |
| **4** | Kepemilikan Org2 dijawab jujur — mekanisme SC-A…SC-D bersandar pada peer yang **dioperasikan klien secara independen** | **E-11** | Klarifikasi + mungkin provisioning |
| **5** | Instrumen FGD disamakan antara yang dijanjikan (4 kriteria) dan yang dinilai (7 kriteria); **"kepercayaan" harus diukur** karena Trust Theory dideklarasikan sebagai lapisan teori | **E-6** | Revisi instrumen |

### 10.5. Kebuntuan tata kelola yang menghalangi seluruh Fase 4

**P1, P2, dan P4 semuanya menuntut prototipe berjalan** — tetapi aturan tata kelola paket ini
menjadikan "0 baris kode prototipe" sebagai syarat kelulusan G8, dan melanggarnya memblokir G9.

Rinciannya dan usulan pemecahannya (**G8a** dokumen / **G8b** evaluasi terukur) ada di
[§11.2](#112-kebuntuan-tata-kelola-yang-harus-dipecah-sponsor). **Ini keputusan sponsor, dan
menghalangi seluruh Bagian 10 sampai diselesaikan.**

### 10.6. Celah metodologi yang harus ditutup lebih dulu

Dua celah yang bukan soal cakupan, melainkan **integritas metodologis** — dan keduanya menyentuh
bagian yang sudah tertulis di tesis:

| Celah | Isi | Errata |
|---|---|---|
| **Instrumen RQ1 tidak dilaporkan** | BAB III menjanjikan wawancara 10–15 partisipan, STRIDE per section, dan analisis tematik Miles-Huberman. **BAB IV tidak melaporkan satu pun.** Ketidaksesuaian metodologi-versus-hasil adalah penyebab tunggal paling umum tuntutan revisi sidang. | **E-4** |
| **Fase 4 tidak punya bagian hasil** | BAB III menamai lima skenario untuk Tahap 4; BAB IV tidak memiliki bagian hasil Tahap 4. Label `SC-A`…`SC-E` justru dipakai ulang untuk himpunan skenario berbeda. Pada tesis DSRM ini **celah struktural**. | **E-9** |

Keduanya harus ditutup **sebelum** hasil Fase 4 ditulis — kalau tidak, bagian hasil yang baru akan
menempel pada metodologi yang tidak konsisten.

---

## 11. Keputusan yang Dibuka Ulang

Sapuan dampak otomatis atas seluruh paket `agent-suite/` (7 area, 9 agen) selesai 2 Agustus 2026.

| | Jumlah |
|---|---|
| Item terdampak | **315** |
| Di antaranya *blocking* | **127** |
| ADR baru yang dibutuhkan | **10** + 1 amandemen tata kelola |

> ⚠️ **Catatan validitas.** Sapuan dijalankan dengan asumsi "tanpa *salt*" (ratifikasi saat itu).
> Karena OQ-1 kemudian **diadopsi**, sekitar **40 dari 315** rekomendasi berubah arah — khususnya
> semua yang menyarankan "hapus *salt*, terima risiko residual". Rekomendasi tersebut **tidak lagi
> berlaku**. §5.2 dokumen ini yang mengikat.

### 11.1. Gate G8 dan persetujuan G9 tidak lagi berdiri

**G8 (*full package review*) — PASS-nya batal.** Laporan di
`agent-suite/09-review/verification/g8-review-report.md` mensertifikasi desain yang **sudah tidak
ada**:

- Perbaikan HIGH utamanya (S-1) merekonsiliasi paket ke `SHA-256(salt ‖ JCS(changedFieldValues))` —
  *digest* ber-*salt* atas **field yang berubah dalam satu event**. Yang diratifikasi sekarang adalah
  *digest* **per section**.
- Butir "*confidentiality invariant clean*" diverifikasi atas permukaan **PDC/transient** yang tidak
  ada dalam desain tesis.
- Klaimnya bahwa "*setiap referensi skill/context/MCP pada agent-spec resolve ke artefak nyata*"
  **terbukti salah di disk** — kedelapan symlink `.claude/skills/fabric-*` menunjuk ke
  `/Users/chan/www/hyperledger/fabric-talenta/…`, direktori yang **tidak ada**. Dan **tidak ada
  skrip linter mana pun** di seluruh tree, sehingga asersi "*machine-checked*" pada G3/G8 **tidak
  pernah benar-benar dijalankan**.
  → *Symlink sudah diperbaiki 2 Agustus 2026 (diarahkan ke `../../fabric-skill-suite/skills/`);
  linter masih belum ada.*

**G9 (persetujuan) — batal menurut definisinya sendiri.**
`00-architecture/quality-gates-and-approval.md` §5.1(b) mendefinisikan persetujuan G9 sebagai
penilaian sponsor bahwa *"asumsi kerja desain … dapat diterima untuk membangun prototipe"*. Jadi
G9 adalah **persetujuan atas sebuah himpunan asumsi** — dan **enam anggota himpunan itu kini
diketahui salah**: *commitment* ber-*salt* per-event, permukaan `EVENT_*`, dua org vendor,
*single-channel* + PDC, penghapusan via `PurgePrivateData`, dan kinerja *out of scope*. Objek yang
disetujui sudah tidak ada.

### 11.2. Kebuntuan tata kelola — ✅ **DIPECAHKAN 6 Agustus 2026 (S-1)**

> **Keputusan Chandra Kurniawan, 6 Agustus 2026: terima pemecahan G8a/G8b.** Kebuntuan di bawah ini
> **sudah tidak berlaku** — dipertahankan sebagai jejak historis, bukan status berjalan. Pencatatan
> resmi ada di `agent-suite/00-architecture/quality-gates-and-approval.md` §5.5. **Build G8b resmi
> terbuka** per item, mengikuti *Definition of Done* di `implementation-backlog.md`.

`quality-gates-and-approval.md` menjadikan **"0 baris kode prototipe"** sebagai butir *checklist*
kelulusan G8, dan melanggarnya sebagai temuan yang **memblokir G9**. Tetapi **P1, P2, dan P4 semuanya
menuntut prototipe berjalan** plus *harness* Caliper.

```
Build dibutuhkan → kode ada → G8 re-run gagal checklist-nya sendiri
      → G9 tak tercapai → build tidak berwenang → (kembali ke awal)
```

**Penyelesaian yang disarankan:** pecah G8 menjadi **G8a** (sapuan dokumen, tanpa kode) dan **G8b**
(evaluasi terukur, kode diharapkan ada), lalu pensiunkan aturan "0 baris kode" — verifikasi grep
menemukan **24 kemunculan di 17 berkas** untuk frasa ketat ini (bukan "dua belas" seperti dugaan awal
sapuan), termasuk tertanam di **tiga definisi agen aktif** (`fabric-engineer.md`, `fabric-architect.md`,
`security-architect.md`) dan tiga ADR (0005, 0008, 0009) — lihat `rencana-rekonsiliasi.md` §1.2.
Menandai risiko R-05 sebagai "*spent*" di *risk register* saja **tidak cukup** — itu meninggalkan
kriteria yang tetap ditegakkan padahal tidak mungkin dipenuhi, dan selama belum dipensiunkan, agen
mana pun yang di-*dispatch* untuk menulis kode akan **menolak** sesuai instruksinya sendiri.

### 11.3. ADR yang dibutuhkan

**Prasyarat tata kelola:** kosakata status ADR harus diperluas dengan `Superseded by ADR-NNNN` di
`_TEMPLATE.adr.md`, `05-adr/README.md`, dan `context/ADR.md`. **Tanpa ini tidak satu pun ADR di bawah
dapat dicatat.**

| ADR | Isi | Menggantikan |
|---|---|---|
| **ADR-0011** | Skema *digest* `EmployeeProfileRecord` — per section, ber-*salt*, pseudonim HMAC (§5.2) | ADR-0001 |
| **ADR-0012** | Konsorsium tiga organisasi + *ordering* Raft semua-Org1 + mekanisme *read-only* Org3 + `AND(Org1MSP.peer, Org2MSP.peer)` | ADR-0003 |
| **ADR-0013** | Satu channel per tenant, termasuk konsekuensi *lifecycle fan-out* dan hak istimewa *onboarding* | ADR-0004 |
| **ADR-0014** | Pencatatan *in-band* dari jalur tulis HRIS; tanpa Kafka, tanpa *anchor-service* | ADR-0006 |
| **ADR-0015** | Penghapusan via hapus field + hancurkan `KEY_EMPLOYEE` + hancurkan *salt*; tanpa PDC | ADR-0010 |
| **ADR-0016** | **IPFS Private Cluster** sebagai *tier* dokumen terenkripsi | *baru* |
| **ADR-0017** | *Tier* relasional — store mana yang otoritatif, skema, cerita migrasi (OQ-2) | *baru* |
| **ADR-0018** | Evaluasi kinerja masuk cakupan; Caliper v0.5 sebagai *harness*; angka Tabel 4.3 adalah *placeholder* yang tidak boleh dikutip artefak mana pun | *baru* |
| **ADR-0019** | **Empat** domain kunci (MSP/TLS, at-rest aplikasi, `KEY_EMPLOYEE`, pseudonimisasi pengenal) + kustodi & adjudikasi *destroy-ability* | ADR-0009 |
| **ADR-0021** *(baru 6 Agt)* | `employeeKey_i` **acak independen**, tanpa kunci induk — menutup T16b (risiko pembalikan penghapusan historis) yang diungkap ADR-0019 di hari yang sama | ADR-0011 (sebagian), ADR-0019 (sebagian) |
| **ADR-0020** | Kontrak chaincode + batas *hashing*: empat fungsi, *digest* dihitung **off-chain**, plaintext **tidak pernah** jadi argumen, verifikasi *evaluate-only*, kanonikalisasi sebagai antarmuka ber-versi | *baru* |

### 11.4. Temuan arsitektural baru dari IPFS

Tiga sifat IPFS yang **belum pernah dimodelkan** paket desain, dan semuanya berdampak pada P4:

1. **CID adalah kapabilitas pengambilan, bukan sekadar pengenal.** Karena *content-addressed*, siapa
   pun yang punya CID dapat mengambil objeknya. CID tercatat **on-chain**, sehingga dapat dibaca Org2,
   Org3, dan siapa pun yang memperoleh replika ledger. **Yang mencegah pengambilan publik hanyalah
   *swarm key*** — bukan CID-nya.
2. **`unpin` bukan `delete`.** Node mana pun yang pernah mengambil sebuah blok dapat menyimpannya.
   Jadi penghapusan **benar-benar bergantung pada enkripsi + penghancuran kunci**, bukan pada
   penghapusan objek. Ini justru menjadi alasan mengapa mekanisme Pasal 26 tetap sah — dan harus
   dinyatakan sebagai alasan utamanya.
3. **Objek IPFS imutabel terhadap alamatnya.** Enkripsi ulang dokumen dengan kunci baru menghasilkan
   **CID baru**, sehingga menuntut **versi `EmployeeProfileRecord` baru**.

**Operator node cluster adalah batas kepercayaan keempat** yang belum ada dalam model batas
kepercayaan paket ini.

---

## 12. Open Questions

### OQ-1 — Hash tanpa *salt* ✅ **SELESAI — DIADOPSI 2 Agustus 2026**

> **Keputusan Chandra Kurniawan, 2 Agustus 2026: adopsi.** *Salt* per-record untuk `DataHash`, HMAC
> berkunci untuk pengenal. **Skema final (setelah dua koreksi lanjutan, terakhir 6 Agustus) ada di
> [§5.2](#52-konstruksi-kriptografis-diratifikasi)** — bagian di bawah ini adalah **jejak keputusan
> pada tanggal 2 Agustus**, bukan konstruksi yang berlaku sekarang. Tabel rekomendasi di bawah masih
> menyebut `pseudonymKey` — itu sudah **digantikan** oleh `employeeKey_i` acak independen di §5.2;
> dipertahankan di sini apa adanya untuk kelengkapan jejak audit.

**Kondisi (sebelum keputusan).** Tesis menetapkan `DataHash = SHA-256(JSON kanonik section)` dan
`EmployeeID = SHA-256(internal_emp_id)` — **tanpa *salt***.

**Masalahnya.** Untuk field ber-entropi rendah, hash tanpa *salt* dapat dibalik dengan
*brute-force* atau serangan kamus. Nomor rekening bank, NIK 16 digit berstruktur, grade jabatan,
jenjang pendidikan, nominal gaji — ruang kemungkinannya kecil. `internal_emp_id` yang umumnya
integer kecil bahkan dapat dienumerasi habis dalam hitungan detik, sehingga ledger berpotensi
menjadi **direktori identitas yang dapat ditautkan**.

**Dampaknya berantai:**
- Klaim BAB V 5.2.2 — *"yang tersimpan on-chain adalah hash kriptografis … sehingga privasi
  karyawan tetap terjaga"* — menjadi **terlalu kuat**.
- Klaim kepatuhan **Pasal 16(2)** di Tabel 4.5 melemah, karena hash tanpa *salt* atas data pribadi
  kemungkinan masih tergolong **data pribadi ter-pseudonimisasi**, bukan anonim — posisi yang
  sejalan dengan pedoman ENISA yang tesis ini sendiri kutip di BAB V 5.2.1.
- **INV-1 dan INV-2 tidak terpenuhi** sebagaimana tertulis, dan pengujian **ST-1 (P0) tidak akan
  lulus**.

**Bukti empiris — sudah diukur, bukan diperkirakan.** Verifikasi adversarial menjalankan serangannya:

| Pengujian | Hasil |
|---|---|
| Memulihkan `emp_id = 456` dari digest SHA-256-nya | **0,0002 detik** |
| Menyapu **seluruh** ruang ID internal 10⁷ | **4,4 detik** |
| Lingkungan | 1 core CPU, Python interpretatif (~2,3 juta SHA-256/detik) |
| Implementasi C / GPU | **3–4 orde magnitudo lebih cepat** |

Jadi `EmployeeID` dan `UpdatedBy` **bukan pseudonim** — keduanya *encoding* yang dapat dibalik. Dan
karena setiap peer pemegang salinan ledger dapat melakukannya, itu termasuk **Org2 (klien)** dan
**Org3 (auditor)** menurut topologi tesis sendiri.

**Rekomendasi — dua konstruksi berbeda untuk dua tujuan berbeda.** Ini **koreksi** atas rekomendasi
saya sebelumnya, yang menyebut "*salt*" untuk keduanya:

| Field | Konstruksi yang benar | Alasan |
|---|---|---|
| `DataHash` (isi data) | **`SHA-256(salt ‖ JSON kanonik)`** — *salt* ≥128 bit CSPRNG, **per-record**, hanya off-chain | *Salt* acak per-record mematahkan serangan kamus atas isi ber-entropi rendah |
| `EmployeeID`, `UpdatedBy` (pengenal) | **`HMAC-SHA256(pseudonymKey, …)`** — kunci rahasia di domain kunci terpisah | Harus **deterministik** agar pencarian record dan rantai `PrevHash` per karyawan tetap bekerja. HMAC memberi determinisme **sekaligus** ketidak-tertautan tanpa kunci |

> ⚠️ **Jangan pakai *salt* untuk pengenal.** *Salt* acak membuat `EmployeeID` berbeda setiap kali,
> sehingga record seorang karyawan tidak bisa lagi ditemukan atau dirantai — FR-4 dan FR-18 langsung
> rusak. Untuk pengenal yang dibutuhkan adalah **kunci**, bukan *salt*.

Ini **cacat yang bisa dikoreksi**, bukan perbedaan preferensi desain yang kalah dari tesis — dan
tesis **bertentangan dengan dirinya sendiri** di sini, karena BAB II menyebut pseudonimisasi
SHA-256 ini sebagai kontrol *Privacy-by-Design*.

**Biaya edit:** definisi struct di BAB IV 4.2.3, satu kalimat klaim PbD di BAB II, satu kalimat di
BAB V 5.2.2, serta sel Pasal 16(2) dan Pasal 26 di Tabel 4.5.

**Yang TIDAK berubah:** **tidak ada satu pun nilai P1–P4** — deteksi tetap 100%, latensi praktis tidak
terpengaruh, isolasi tidak tersentuh, kepatuhan justru **menguat**.

✅ **DIADOPSI.** Konsekuensi lanjutan yang ikut terselesaikan: **ST-1 (P0) lulus utuh tanpa perlu
dipecah**, dan cacat "PII plaintext sebagai argumen chaincode" tertutup oleh protokol Kelompok B′.
Rincian: [`errata-tesis.md`](errata-tesis.md) butir **E-1**.

### OQ-2 — PostgreSQL vs MySQL ✅ **SELESAI — DIPUTUSKAN 6 Agustus 2026 (S-2)**

> **Keputusan Chandra Kurniawan, 6 Agustus 2026: pertahankan basis data operasional yang sudah
> ada, tanpa migrasi** (ADR-0017 dikonfirmasi, tidak lagi "rekomendasi kerja"). Konsisten dengan
> posisi "lapisan berdampingan, bukan menggantikan."

Tesis BAB IV 4.1.1 mendeskripsikan sistem *as-is* sebagai **MySQL di Alicloud RDS**, tetapi Tabel
4.2 menetapkan **PostgreSQL di AWS RDS** untuk data profil, tanpa penjelasan peralihannya.

**Edit tesis yang diperlukan:** koreksi Tabel 4.2 agar konsisten dengan BAB IV 4.1.1 (basis data
yang sudah ada), bukan basis data baru — penguji akan menanyakan migrasi yang tidak pernah
dijustifikasi.

### OQ-3 — Jumlah peserta FGD tidak konsisten

Tiga angka berbeda muncul: Abstract (Inggris) menyebut **tujuh** praktisi; BAB IV 4.3.4 menyebut
**tiga** ahli; BAB III 3.5 menargetkan **3–5** orang. Tabel 4.6 memuat **tiga** kolom penilai,
sehingga angka yang benar tampaknya tiga. **Abstract Inggris perlu dikoreksi sebelum sidang.**

### OQ-4 — Inkonsistensi nilai grade pada SC-C

Tabel 4.4 skenario SC-C menaikkan grade "dari **VI**", tetapi kolom deteksi menyebut "hash grade
**VII**". Salah satu salah tulis.

### OQ-5 — Mekanisme *key recovery* untuk `KEY_EMPLOYEE`

Diakui sendiri sebagai keterbatasan di tesis BAB IV 4.5. Kritis untuk kelangsungan operasional
*crypto-shredding*: jika `KEY_EMPLOYEE` hilang tanpa sengaja, dokumen karyawan hilang permanen —
efeknya identik dengan penghapusan yang tidak diminta.

### OQ-6 — "Rekonstruksi nilai asli" tidak dapat dilakukan dari ledger ✅ **SELESAI — DIKONFIRMASI 6 Agustus 2026 (S-3)**

> **Keputusan Chandra Kurniawan, 6 Agustus 2026: konfirmasi Opsi C + A.** Koreksi ini sudah
> diterapkan ke UJ-1 (§6, langkah 6) sebagai perbaikan faktual sejak *finalize* 3 Agustus; pada
> 6 Agustus Chandra secara eksplisit mengonfirmasi ini sebagai keputusannya sendiri, bukan sekadar
> koreksi yang diterapkan sepihak. Nilai P1 tidak berubah.

**Kondisi.** Tesis Tabel 4.4 skenario SC-A menyatakan: *"Versi valid terakhir (rekening asli) **dapat
direkonstruksi**."* Klaim ini juga menjadi langkah 6 pada UJ-1.

**Masalahnya.** Ledger hanya menyimpan `DataHash` — fungsi **satu arah**. Dari sebuah hash, nilai
aslinya tidak dapat dipulihkan. Yang sebenarnya bisa dilakukan ledger:

| Ledger **bisa** | Ledger **tidak bisa** |
|---|---|
| Membuktikan nilai saat ini ≠ nilai terakhir yang sah (*hash mismatch*) | Memulihkan nomor rekening aslinya |
| Menunjuk **versi mana** yang terakhir valid (`Version`, `Timestamp`, aktor) | Mengembalikan isi datanya |
| Membuktikan sebuah nilai kandidat **benar** nilai aslinya (dengan mencocokkan hash) | Menghasilkan nilai kandidat itu sendiri |

Untuk benar-benar memulihkan nilainya, dibutuhkan **sumber off-chain**: *backup* basis data, *change
log*, atau objek IPFS. Tesis tidak menyebutkan yang mana.

**Mengapa ini berisiko di sidang.** Kalau penguji bertanya *"direkonstruksi dari mana?"* pada langkah
6 UJ-1, harus ada jawaban. Tanpa itu, satu klaim di tabel hasil utama tidak terdukung.

**Empat pilihan:**

| Opsi | Isi | Biaya |
|---|---|---|
| **A** | Pulihkan dari *backup* / PITR PostgreSQL. Peran ledger: memberi tahu **timestamp backup mana** yang boleh dipercaya, dan **membuktikan** kandidat yang dipulihkan benar. | Rendah — hanya perlu dijelaskan |
| **B** | Simpan *change log* append-only di PostgreSQL. Tapi log itu sendiri mutable, jadi ia pun harus di-*anchor*. | Sedang |
| **C** | **Persempit klaimnya**: sistem membuktikan manipulasi dan menunjuk versi valid terakhir; pemulihan adalah prosedur operasional terpisah. | Rendah — edit satu sel Tabel 4.4 |
| **D** | Simpan nilai section terenkripsi di IPFS per versi, sehingga pemulihan mandiri. | Tinggi — mengubah arsitektur |

**Rekomendasi: C untuk klaim tesis + A sebagai jawaban operasional.** Kombinasi ini paling murah dan
paling mudah dipertahankan — dan justru memperkuat posisi Anda, karena kemampuan **membuktikan
kandidat pemulihan itu benar** adalah properti yang lebih kuat daripada sekadar menyimpan salinan.
Nilai P1 tidak berubah.

**Menunggu keputusan Chandra.**

---

## Lampiran A — Jejak Ratifikasi

| Butir | Keputusan | Dasar | Tanggal |
|---|---|---|---|
| **PB-4 / G-01** | ✅ Ditutup — objektif + P1…P4 | Keputusan manusia eksplisit (tesis sebagai sumber) | 2026-08-02 |
| **PB-5 / G-05** | ✅ Ditutup — lima section profil | Keputusan manusia eksplisit (tesis BAB IV 4.2.3) | 2026-08-02 |
| **PB-1 / G-18** | ⛔ **Masih terbuka** — batas kepercayaan header *gateway* API | — | — |
| **PB-2 / G-23** | ✅ **Ditutup — dilarutkan (S-4)** | Keputusan manusia eksplisit; tidak ada *anchor-service*/Kafka yang perlu diamankan | 2026-08-06 |
| **PB-3 / G-24** | ⛔ **Masih terbuka** — kanonikalisasi RFC-8785 JCS | — | — |
| **G-08** | ✅ Dibuka kembali, **ADR-0018 mengatur eksekusinya** | Angka Caliper adalah placeholder, akan diukur | 2026-08-02 |
| **OQ-1 / E-1** | ✅ Ditutup — *salt* + `employeeKey_i` acak independen (§5.2, final) | Keputusan manusia eksplisit; bukti empiris pembalikan hash; T16b ditutup ADR-0021 | 2026-08-02 → 06 |
| **OQ-2 / S-2** | ✅ **Ditutup** — pertahankan basis data yang ada, tanpa migrasi | Keputusan manusia eksplisit (ADR-0017 dikonfirmasi) | 2026-08-06 |
| **OQ-6 / S-3** | ✅ **Ditutup** — klaim rekonstruksi dipersempit (opsi C+A) | Keputusan manusia eksplisit, mengonfirmasi koreksi 3 Agustus | 2026-08-06 |
| **S-1** | ✅ **Ditutup** — G8 dipecah G8a (lulus)/G8b (belum dimulai); aturan "0 baris" dipensiunkan untuk G8b | Keputusan manusia eksplisit (`quality-gates-and-approval.md` §5.5) | 2026-08-06 |
| **S-5** | ✅ **Ditutup** — G9 disetujui ulang terhadap desain terkini | Keputusan manusia eksplisit (`quality-gates-and-approval.md` §5.5) | 2026-08-06 |
| **Gate G8** | ✅ **G8a lulus** 2026-08-06 (`g8-review-report-r2.md`); G8b **belum dimulai** — butuh kode nyata + Caliper | Sapuan dampak §11.1 → S-1 memecah gate | 2026-08-02 → 06 |
| **Gate G9** | ✅ **Disetujui ulang** 2026-08-06 terhadap desain saat ini — bukan kebangkitan persetujuan 2026-07-13 | Definisi G9 sendiri (§5.1b); S-5 | 2026-08-02 → 06 |

**Gate *blocking* pre-build: dari 5 butir menjadi 2** (PB-1 gateway header, PB-3 JCS) — keduanya
**item-scoped**, tidak lagi membekukan seluruh build.

**Catatan kejujuran jejak audit.** Sebelum "tesis menang" diratifikasi, sintesis Workflow 1
(rekonsiliasi tesis-vs-desain) sempat merekomendasikan **sebaliknya** — menolak jalur tesis-verbatim
dan merumuskan ulang objektif agar tidak membatalkan ADR mana pun, dengan alasan biaya *re-run*
G4–G7. Rekomendasi itu disampaikan ke Chandra secara langsung; ia memilih tetap pada "tesis menang",
dan seluruh pekerjaan sesudahnya (§5.2, Bagian 9, `rencana-rekonsiliasi.md`) dibangun di atas pilihan
itu. Dicatat di sini untuk kelengkapan, bukan sebagai keberatan yang belum selesai.

**Otorisasi build: TERBUKA sejak 6 Agustus 2026.** Kelima keputusan sponsor (S-1…S-5) sudah
diputuskan dalam satu sesi (lihat `quality-gates-and-approval.md` §5.5 untuk pencatatan resmi).
**G8b (evaluasi terukur) boleh dimulai** — per item, mengikuti *Definition of Done* dan keterlacakan
ADR/dokumen desain masing-masing di `implementation-backlog.md`. PB-1 dan PB-3 tetap menahan item
spesifik yang bergantung padanya, tapi **tidak lagi menahan seluruh build**.

---

## Lampiran B — Perbaikan yang Sudah Dijalankan

| Tanggal | Perbaikan | Bukti |
|---|---|---|
| 2026-08-02 | Kedelapan symlink `.claude/skills/fabric-*` diarahkan ulang dari `fabric-talenta` (tidak ada) ke `../../fabric-skill-suite/skills/` | Kedelapan skill Fabric kembali termuat oleh harness |

**Belum dikerjakan:** skrip *cross-reference linter* yang diasersikan G3/G8 masih **tidak ada** di
seluruh tree dan perlu ditulis sebelum asersi itu boleh diklaim lagi.
