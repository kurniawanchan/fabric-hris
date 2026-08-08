# Errata Tesis — Daftar Cacat yang Harus Diperbaiki Sebelum Sidang

| | |
|---|---|
| **Dokumen sumber** | *Tesis — Perancangan dan Implementasi Prototipe Sistem Keamanan Data Karyawan Berbasis Blockchain pada SaaS HRIS menggunakan Metode Design Science Research* |
| **Penulis** | Chandra Kurniawan (2602655404) |
| **Tanggal audit** | 2 Agustus 2026 |
| **Metode** | 6 lensa pembaca independen + audit internal tesis; setiap temuan diserang 3 penyangkal dari sudut berbeda (*misread*, *reconcilable*, *consequence*) |
| **Dokumen terkait** | [`prd.md`](prd.md) — PRD yang meratifikasi PB-4 / PB-5 |

> **Cara membaca.** Ini bukan daftar kritik gaya penulisan. Setiap butir di bawah adalah hal yang
> **akan ditanyakan penguji** dan yang jawabannya saat ini belum tersedia di dokumen. Diurutkan
> dari yang paling menentukan.

---

## E-1 — Hash tanpa *salt* dan pseudonim tanpa kunci ⛔ MEMBLOKIR SIDANG **DAN** BUILD

**Lokasi:** BAB IV 4.2.3 (definisi `EmployeeProfileRecord`), didukung BAB II (klaim PbD), BAB V 5.2.2, Tabel 4.5.

**Isi cacat:**
- `DataHash = SHA-256(JSON kanonik section)` — **tanpa *salt***
- `EmployeeID = SHA-256(internal_emp_id)` — **tanpa kunci**
- `UpdatedBy = SHA-256(user_id)` — **tanpa kunci**

**Bukti empiris (diukur, bukan diperkirakan).** Verifikator menjalankan serangan nyata:

| Pengujian | Hasil |
|---|---|
| Memulihkan `emp_id = 456` dari digest SHA-256-nya | **0,0002 detik** |
| Menyapu **seluruh** ruang ID internal 10⁷ | **4,4 detik** |
| Lingkungan | 1 core CPU, Python interpretatif (~2,3 juta SHA-256/detik) |
| Implementasi C atau GPU | **3–4 orde magnitudo lebih cepat** |

`EmployeeID` dan `UpdatedBy` karena itu **bukan pseudonim** — keduanya adalah **encoding yang dapat
dibalik**. Hal yang sama berlaku untuk `DataHash` atas field ber-entropi rendah: nomor rekening bank,
NIK 16 digit berstruktur, grade jabatan, jenjang pendidikan, nominal gaji.

**Akibat berantai:**

1. **Kontradiksi internal.** BAB II menyebut pseudonimisasi via "SHA-256 dari ID internal" sebagai
   kontrol *Privacy-by-Design* untuk minimalisasi data. Konstruksinya tidak memberikan properti itu.
   Jadi ini **cacat internal tesis**, bukan sekadar perbedaan dengan paket desain.
2. **Ledger menjadi direktori identitas yang dapat ditautkan** — siapa mengubah data siapa, dapat
   dibaca oleh **setiap** peer pemegang salinan ledger. Menurut topologi tesis sendiri (BAB IV 4.2.2),
   itu termasuk **Org2 (klien)** dan **Org3 (auditor)**.
3. **Klaim BAB V 5.2.2** — *"yang tersimpan on-chain adalah hash kriptografis … sehingga privasi
   karyawan tetap terjaga"* — tidak terdukung.
4. **Klaim Pasal 26 di Tabel 4.5** — bahwa hash yang tertinggal *"menjadi orphan tidak identifiable"*
   — tidak terdukung, karena hash tersebut masih dapat dibalik.
5. **Klaim Pasal 16(2)** melemah: hash tanpa *salt* atas data pribadi kemungkinan masih tergolong
   **data pribadi ter-pseudonimisasi**, sejalan dengan pedoman ENISA yang tesis ini kutip sendiri.

**Perbaikan — dua konstruksi berbeda untuk dua tujuan berbeda.** Ini koreksi penting atas rekomendasi
awal:

| Field | Konstruksi yang benar | Alasan |
|---|---|---|
| `DataHash` (isi data) | **`SHA-256(salt ‖ JSON kanonik)`**, *salt* ≥128 bit dari CSPRNG, **per-record**, disimpan hanya off-chain | *Salt* acak per-record mematahkan serangan kamus atas isi ber-entropi rendah |
| `EmployeeID`, `UpdatedBy` (pengenal) | **`HMAC-SHA256(pseudonymKey, …)`** dengan kunci rahasia di domain kunci terpisah | Harus **deterministik** agar pencarian record & rantai `PrevHash` per karyawan tetap bekerja — *salt* acak akan merusaknya. HMAC memberi determinisme **dan** ketidak-tertautan tanpa kunci |

> ⚠️ **Jangan memakai *salt* untuk pengenal.** *Salt* acak per-record membuat `EmployeeID` berbeda
> setiap kali, sehingga record seorang karyawan tidak bisa lagi ditemukan atau dirantai. Untuk
> pengenal, yang dibutuhkan adalah **kunci**, bukan *salt*.

**Biaya edit:** definisi struct di BAB IV 4.2.3, satu kalimat klaim PbD di BAB II, satu kalimat di
BAB V 5.2.2, dan sel Pasal 26 + Pasal 16(2) di Tabel 4.5.

**Yang TIDAK berubah:** nilai P1–P4. Deteksi tetap 100%, latensi praktis tidak terpengaruh, isolasi
tidak tersentuh, kepatuhan **menguat**.

---

## E-2 — Klaim "100% pada lima section", padahal hanya empat yang diuji ⛔

**Lokasi:** Abstrak (ID & EN), BAB V 5.1, versus Tabel 4.4.

Tabel 4.4 memuat lima baris, tetapi hanya **empat** yang menguji *hash mismatch*:

| ID | Section | Jenis uji |
|---|---|---|
| SC-A | PAYROLL | ✅ uji hash |
| SC-B | PERSONAL | ✅ uji hash |
| SC-C | EMPLOYMENT | ✅ uji hash |
| SC-D | EDUCATION | ✅ uji hash |
| SC-E | *ALL* | ❌ **uji isolasi channel — bukan uji hash** |

**Section `ADDITIONAL` tidak pernah dimanipulasi dalam pengujian apa pun.** Penguji yang mencocokkan
jumlah baris dengan klaim akan langsung menemukannya.

**Perbaikan — pilih satu:**
- **(a)** Tambahkan satu baris manipulasi untuk section `ADDITIONAL` (mis. mengubah status perkawinan
  atau data tanggungan), sehingga klaim "lima section" menjadi benar. **Direkomendasikan** — biayanya
  satu skenario uji tambahan.
- **(b)** Persempit klaim menjadi "empat section yang diuji".

> **Dampak ke PRD:** predikat **P1** sudah dikoreksi menyesuaikan temuan ini.

---

## E-3 — Klaim rekonstruksi nilai asli tidak mungkin dari ledger ⛔

**Lokasi:** Tabel 4.4 SC-A — *"Versi valid terakhir (rekening asli) dapat direkonstruksi."*

Ledger menyimpan `DataHash`, sebuah **fungsi satu arah**. Nilai aslinya tidak dapat dipulihkan dari
digest. Klaim ini juga **bertentangan dengan argumen privasi tesis sendiri di bab yang sama** —
kalau nilai bisa direkonstruksi dari ledger, maka ledger memang menyimpan data pribadi.

| Ledger **bisa** | Ledger **tidak bisa** |
|---|---|
| Membuktikan nilai sekarang ≠ nilai sah terakhir | Memulihkan nomor rekening aslinya |
| Menunjuk **versi mana** yang terakhir valid | Mengembalikan isi datanya |
| **Memverifikasi** kandidat nilai itu benar asli | **Menghasilkan** kandidat itu |

**Perbaikan:** hapus klausa tersebut. Klaim yang benar: manipulasi **terdeteksi**, dan nilai sah
terakhir dipulihkan dari **backup off-chain** — sementara ledger berperan **membuktikan bahwa nilai
yang dipulihkan itu benar**. Properti terakhir ini justru lebih kuat daripada sekadar menyimpan
salinan, karena salinan pun bisa dimanipulasi.

---

## E-4 — Metodologi BAB III tidak dilaporkan di BAB IV ⛔ PENYEBAB REVISI VIVA PALING UMUM

**Lokasi:** BAB III versus BAB IV 4.1.

BAB III menjanjikan **tiga instrumen** untuk menjawab RQ1:

| Dijanjikan di BAB III | Dilaporkan di BAB IV? |
|---|---|
| Wawancara semi-terstruktur, *purposive sample* **10–15 partisipan** | ❌ tidak ada temuan wawancara |
| *Threat modeling* STRIDE per section profil | ❌ tidak ada tabel STRIDE |
| Analisis tematik Miles, Huberman & Saldaña (reduksi, penyajian, penarikan kesimpulan) | ❌ tidak ada tema terkode |

BAB IV menjawab RQ1 **hanya** dengan analisis arsitektur *as-is*.

**Ketidaksesuaian metodologi-versus-hasil adalah penyebab tunggal paling umum tuntutan revisi
sidang.** Penguji DSRM melacak keenam aktivitas satu per satu.

**Perbaikan — pilih satu:**
- **(a)** Laporkan instrumennya (jika memang dijalankan, hasilnya harus muncul di BAB IV).
- **(b)** Revisi BAB III agar menggambarkan apa yang **benar-benar** dilakukan. Lebih jujur dan lebih
  murah — dan tidak melemahkan kontribusi, karena analisis arsitektur *as-is* memang cukup untuk RQ1.

---

## E-5 — Tahap 2 DSRM menyebut tiga section, bukan lima ⛔

**Lokasi:** BAB III, pemetaan tahapan DSRM — Tahap 2 *(Define Objectives of a Solution)* menyebut
hanya **"personal, kepegawaian, pendidikan"**.

Judul tesis, kedua abstrak, Ruang Lingkup 1.5, dan seluruh BAB IV menyebut **lima** section.

**Mengapa ini paling berbahaya di antara inkonsistensi angka:** Tahap 2 adalah **tahap yang tepat
tempat objektif diratifikasi**. Inkonsistensi ini duduk persis di kalimat yang menanggung beban.

**Perbaikan:** samakan menjadi lima section.

---

## E-6 — Instrumen FGD yang dijanjikan berbeda dari yang dilaporkan

**Lokasi:** BAB III (Daftar Evaluasi FGD) versus Tabel 4.6.

| BAB III menjanjikan **4 kriteria** | Tabel 4.6 menilai **7 kriteria** |
|---|---|
| kegunaan | kelengkapan cakupan lima section |
| kemudahan penggunaan | keandalan deteksi manipulasi |
| **kepercayaan** | minimalisasi dampak kinerja |
| kelayakan | kekuatan isolasi antartenant |
| | kelayakan implementasi skala produksi |
| | kesesuaian dengan Pasal 8 |
| | efektivitas *crypto-shredding* Pasal 26 |

**"Kepercayaan" tidak pernah diukur** — padahal **Trust Theory** adalah salah satu lapisan teori yang
dideklarasikan di BAB II. Penguji yang melihat sebuah lapisan teori dideklarasikan akan menuntutnya
dioperasionalkan.

**Perbaikan:** samakan instrumen, atau tambahkan kriteria "kepercayaan" ke Tabel 4.6 dan hubungkan
eksplisit ke Trust Theory.

---

## E-7 — Jumlah peserta FGD saling bertentangan di halaman pertama

| Lokasi | Angka |
|---|---|
| Abstract (Inggris) | **tujuh** praktisi |
| Abstrak (Indonesia) | tidak menyebut angka |
| BAB III (target sampel) | **3–5** orang |
| BAB IV 4.3.4 | **tiga** ahli |
| Tabel 4.6 (jumlah kolom penilai) | **tiga** |

Angka yang benar tampaknya **tiga**. **Abstract Inggris harus dikoreksi** — kontradiksi numerik antara
dua abstrak pada dokumen yang sama, di halaman pertama.

---

## E-8 — Angka throughput bertentangan dengan tabelnya sendiri

**Dua arah kesalahan:**

1. **Puncak salah.** Kedua abstrak menyatakan throughput tulis maksimum **850,4 TPS**. Tetapi Tabel
   4.3 mencatat **920,1 TPS** pada *send rate* 2.000 TPS, dan narasi BAB IV sendiri menyebut ~920 TPS
   sebagai titik saturasi.
2. **Keberhasilan tidak dilaporkan.** Target BAB I 1.3 (< 3 detik @ 500 TPS) **terpenuhi dengan
   margin lebar** di Tabel 4.3 — **485,2 TPS / 1,12 detik / 100%** — tetapi angka ini **tidak pernah
   disebut** di abstrak maupun BAB V. Artinya tesis tidak pernah menyatakan bahwa ia menutup objektif
   teknisnya sendiri.

**Perbaikan:** koreksi angka puncak, **dan** laporkan hasil pada ambang target. Butir kedua lebih
penting — itu prestasi yang saat ini tidak diklaim.

> **Catatan penting:** semua angka Caliper saat ini adalah **placeholder** (lihat `prd.md` §3.1).
> Butir E-8 berlaku untuk versi final setelah pengukuran nyata dilakukan.

---

## E-9 — DSRM Tahap 4 (Demonstration) tidak punya bagian hasil

**Lokasi:** BAB III menamai **lima skenario** untuk Tahap 4; BAB IV tidak memiliki bagian hasil
Tahap 4. Label `SC-A`…`SC-E` justru **dipakai ulang** untuk himpunan skenario yang berbeda (uji
keamanan di Tabel 4.4).

Pada tesis DSRM, **bagian Fase-4 yang hilang adalah celah struktural, bukan masalah format.** Penguji
DSRM melacak keenam aktivitas satu per satu.

**Perbaikan:** tambahkan bagian hasil Tahap 4 yang eksplisit, dan gunakan label berbeda untuk dua
himpunan skenario tersebut.

---

## E-10 — Basis data operasional disebut tiga cara berbeda

| Lokasi | Basis data |
|---|---|
| BAB IV 4.1.1 (*as-is*) | **MySQL** di Alicloud RDS |
| Tabel 4.2 (artefak) | **PostgreSQL** di AWS RDS |
| Kedua abstrak | **PostgreSQL** |

Sementara itu tesis menegaskan prototipe bekerja **berdampingan** dengan HRIS yang ada, bukan
menggantikannya — sehingga peralihan basis data tidak konsisten dengan posisi itu, dan migrasinya
tidak pernah dijelaskan.

**Perbaikan:** jelaskan migrasinya, atau koreksi Tabel 4.2 menjadi MySQL. Opsi kedua lebih konsisten
dengan posisi "berdampingan".

---

## E-11 — Klaim isolasi antartenant akan diprobe langsung

**Dua pertanyaan yang pasti muncul:**

1. **"Apakah demonstrasinya benar-benar menjalankan dua channel dan dua tenant?"** Tabel 4.4 SC-E
   mengklaim isolasi antartenant terbukti — yang mensyaratkan adanya Tenant B, peer milik Tenant B,
   dan channel kedua.
2. **"Siapa yang mengoperasikan Org2?"** BAB IV 4.2.2 menjadikan peer yang dioperasikan klien secara
   independen sebagai **alasan** SC-A…SC-D dapat mendeteksi manipulasi DBA. Manfaat Praktis 1.4 —
   *"menghilangkan ketergantungan pada kepercayaan institusional terhadap vendor"* — bersandar pada
   premis yang sama. Kalau Org2 ternyata dioperasikan vendor, kedua klaim itu gugur.

**Ada juga ketegangan logis:** RQ1 menyebut isolasi logis berbasis `company_id` sebagai **kelemahan
yang diperbaiki**. Klaim "lebih kuat dari model shared-table" harus dijelaskan mekanismenya, bukan
diasumsikan.

**Perbaikan:** siapkan jawaban jujur, dan **nyatakan isolasinya sebagai *identity-bound* /
ditegakkan-chaincode**, bukan "kriptografis", kecuali dua channel dan dua tenant benar-benar
dijalankan di demonstrasi.

---

## E-12 — Integritas daftar pustaka

| Temuan | Jumlah |
|---|---|
| Sumber dikutip di badan teks tetapi **tidak ada** di DAFTAR PUSTAKA | **5** |
| Entri di DAFTAR PUSTAKA yang **tidak pernah dikutip** di badan teks | **16 dari 48** |
| Pasangan nama-sama/tahun-sama dengan inisial berbeda dan judul tidak konsisten | **1** — `Adhiatma, A. (2022)` vs `Adhiatma, F. (2022)` |

Penguji melakukan *spot-check* referensi. Daftar yang menggelembung atau tidak cocok memicu pertanyaan
integritas yang lebih luas — risiko yang jauh lebih besar daripada biaya memperbaikinya.

**Perbaikan:** lengkapi 5 yang hilang, hapus atau kutip 16 yang menganggur, dan selesaikan pasangan
Adhiatma.

---

## E-13 — Kerja lapangan versus simulasi

**Lokasi:** Ruang Lingkup 1.5 butir terakhir menyatakan wawancara dan FGD dilakukan **"dalam konteks
simulasi"**. Namun §Tempat Penelitian menyebut tiga divisi PT. XYZ memberikan akses ke detail
implementasi teknis, dan Tabel 4.6 mengatribusikan skor kepada **"VP Engineering PT. XYZ"** — seorang
individu bernama jabatan nyata.

Penguji yang bertanya *"apakah partisipannya nyata?"* harus mendapat **satu** jawaban yang konsisten.

**Perbaikan:** rekonsiliasi eksplisit di BAB III **dan** di §Keterbatasan Penelitian.

---

## E-14 — Tabel 4.5 (matriks UU PDP): 4 dari 5 nomor pasal salah kutip ⛔⛔ TEMUAN TERBESAR

**Lokasi:** Tabel 4.5, BAB IV 4.3.3, Abstrak, BAB V.

**Metode:** verifikasi terhadap naskah resmi UU No. 27 Tahun 2022 — 7 pembaca independen (satu per
pasal yang diklaim, plus satu audit kelengkapan), 3 Agustus 2026. Naskah diperoleh dari salinan
dwibahasa ABNR (abnrlaw.com), dikonfirmasi silang via pasal.id, uupdp-info.id, bplawyers.co.id.

**Temuan:**

| Baris tesis | Pasal diklaim | Isi diklaim | Status verifikasi |
|---|---|---|---|
| 1 | Pasal 4(2) | Hak memperbarui/memperbaiki | ❌ **Salah total** — Pasal 4 adalah klasifikasi jenis data, tidak memuat hak/kewajiban/larangan apa pun |
| 2 | Pasal 8 | Kewajiban akurasi | ❌ **Salah total** — Pasal 8 adalah hak menghapus/memusnahkan (isi untuk baris 4) |
| 3 | Pasal 16(2) | Kewajiban lindungi akses | ⚠️ **Nomor benar, kategori salah** — hanya prinsip (Bab V); kewajibannya di Pasal 39 |
| 4 | Pasal 26(1) | Hak penghapusan | ❌ **Salah total** — Pasal 26 mengatur data penyandang disabilitas |
| 5 | Pasal 43 | Larangan proses tidak sah | ❌ **Salah kategori & bertukar isi dengan baris 4** — Pasal 43 = kewajiban menghapus |

**Nomor pasal yang benar** (klaster, karena satu baris tesis sering menggabungkan hak + kewajiban +
mekanisme yang tersebar di beberapa pasal):

- Baris 1 → **Pasal 6** (hak) jo. **Pasal 30** (kewajiban Pengendali) jo. **Pasal 14** (mekanisme permohonan)
- Baris 2 → **Pasal 29 (1)–(2)**
- Baris 3 → **Pasal 39 (1)–(2)** jo. Pasal 35
- Baris 4 → **Pasal 8** jo. **Pasal 43 (1)** jo. **Pasal 44 (1)** jo. Pasal 45
- Baris 5 → **Pasal 38** jo. Pasal 39(1) jo. Pasal 37

**Perbaikan tidak boleh menambal sel — ganti seluruh tabel.** Matriks terkoreksi lengkap (12 baris,
tiga tingkat status) ada di [`prd.md`](prd.md) **§9.2**. Ringkasannya:

- **5 baris tesis, semua status ditinjau ulang** (turun ke *Comply sebagian* atau *Comply dengan catatan*)
- **5 baris baru ditambahkan** — Pasal 31 (perekaman), Pasal 47 (akuntabilitas — **klaim terkuat di
  seluruh matriks**), Pasal 36 (kerahasiaan), Pasal 4(2)(f)+35(b) (klasifikasi data spesifik)
- **2 baris berstatus "Tidak dapat diklaim"** yang **wajib** masuk BAB IV 4.5 Keterbatasan Penelitian:
  **DPIA (Pasal 34)** belum disusun padahal terpicu 4 pintu sekaligus, dan **retensi (Pasal 42)**
  tidak terjawab karena ledger *append-only* tanpa batas waktu

**Pertanyaan hukum yang belum terjawab, ditemukan bersamaan:**

1. **UU PDP tidak mengenal anonimisasi** — nol kemunculan kata "anonim"/"pseudonim"/"enkripsi" di
   seluruh naskah. Klaim *crypto-shredding* memenuhi kewajiban adalah argumen hukum belum teruji,
   dan lebih kuat untuk "memusnahkan" (Pasal 44, definisi berbasis hasil) daripada "menghapus"
   (Pasal 43, tidak didefinisikan).
2. **Peran hukum Org1/Org2 (Pengendali vs Prosesor) belum ditetapkan** — Pasal 52 memuat daftar
   tertutup 7 pasal yang mengikat Prosesor; jika Org1 = Prosesor, sebagian kewajiban yang diklaim
   tesis justru tidak mengikatnya, melainkan mengikat Org2.
3. **Replikasi ke Org3 (auditor) adalah "transfer" secara harfiah** menurut Penjelasan Pasal 16(1)(e)
   — dasar pemrosesan untuk aliran data ke auditor belum dinyatakan.
4. **Paradoks deteksi-integritas**: setiap deteksi manipulasi yang berhasil mengaktifkan kewajiban
   notifikasi 3×24 jam Pasal 46 — sistem ini menaikkan eksposur kepatuhan, bukan menurunkannya.

**Koreksi tambahan pada desain §5.2 yang ditemukan bersamaan:** `pseudonymKey` dalam skema HMAC
sebelumnya tidak disebut dalam prosedur penghapusan. Jika dipakai langsung (bukan diturunkan
per-karyawan), menghancurkannya untuk satu karyawan merusak karyawan lain. **Sudah diperbaiki**
menjadi hierarki dua lapis (`pseudonymKey` induk → `employeeKey_i` per karyawan) di `prd.md` §5.2.

**Dampak ke P4:** predikat P4 tidak lagi didefinisikan sebagai "5 pasal terpenuhi" — lihat `prd.md`
§3.1a. Rincian penuh, termasuk kutipan verbatim setiap pasal dan tiga koreksi silang antar-verifikator:
`prd.md` §9.

**Perbaikan tesis:** ganti seluruh Tabel 4.5; hapus setiap klaim "5 pasal" di Abstrak/BAB IV/BAB V;
tambahkan sub-bab keterbatasan untuk DPIA, retensi, peran Pengendali/Prosesor, dan status hukum
*crypto-shredding*; **verifikasi ulang seluruh kutipan langsung terhadap Lembaran Negara RI Tahun
2022 Nomor 196 sebelum sidang** — jangan mengutip tinjauan ini sebagai sumber primer akhir.

---

## Ringkasan prioritas

| Prioritas | Butir | Sifat |
|---|---|---|
| **1 — blokir sidang + build** | E-1 | Cacat kriptografis, terukur |
| **2 — blokir sidang** | **E-14**, E-2, E-3, E-4, E-5 | Klaim tidak terdukung / metodologi tidak dilaporkan |
| **3 — pasti ditanya** | E-6, E-9, E-11, E-13 | Celah struktural & premis |
| **4 — mudah diperbaiki, mahal jika terlewat** | E-7, E-8, E-10, E-12 | Inkonsistensi faktual |

**E-1 dan E-14 adalah dua butir dengan dampak terluas.** E-1 memblokir sidang **dan** build; **E-14
adalah temuan bervolume terbesar** (menggantikan seluruh Tabel 4.5) meski secara teknis "hanya"
memblokir sidang, bukan build. Selesaikan keduanya lebih dulu.

---

## Catatan keterbatasan audit ini

Audit menemukan **68 divergensi** antara tesis dan paket desain. Hanya **6** yang diverifikasi secara
adversarial (5 terkonfirmasi, 1 dibantah) — batas ini saya tetapkan sendiri saat menyusun sapuan,
**bukan** temuan bahwa masalahnya hanya lima.

**62 divergensi belum diverifikasi.** Daftar lengkapnya tersimpan di jurnal sapuan dan dapat
diverifikasi lanjutan bila diperlukan.
