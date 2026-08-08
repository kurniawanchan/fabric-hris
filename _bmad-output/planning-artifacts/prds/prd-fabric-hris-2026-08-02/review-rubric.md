# PRD Quality Review — HRIS-on-Fabric PII-Anchoring Prototype

> Dilakukan langsung di konteks utama (subagent ditangguhkan sesuai instruksi proyek), 3 Agustus 2026.
> Rujukan: `prd.md` (922→~970 baris setelah perbaikan §5.2), `errata-tesis.md`, `rencana-rekonsiliasi.md`.

## Overall verdict

PRD ini kuat pada dimensi yang jarang kuat bersamaan — kejujuran cakupan dan koherensi strategis —
karena mandatnya bukan "usulkan produk baru" melainkan "ratifikasi dua keputusan yang sudah diblokir
dan rekonsiliasikan dua artefak yang berbeda." Kelemahan nyatanya ada di *downstream usability*:
tidak ada Glosarium terpusat untuk istilah kriptografis yang dipakai di lebih dari sepuluh tempat, dan
satu NFR masih memakai kata sifat tanpa ambang angka. Tidak ada temuan *critical* yang menahan
finalisasi; ada satu temuan *high* (Glosarium) yang layak diperbaiki sebelum dokumen diserahkan ke
tim lain.

## Decision-readiness — strong

Setiap keputusan besar (PB-4, PB-5, "tesis menang", adopsi §5.2) dinyatakan sebagai keputusan eksplisit
dengan tanggal dan pengambil keputusan — bukan terkubur sebagai "pertimbangan". *Trade-off* dinyatakan
dengan apa yang dikorbankan, bukan cuma apa yang dipilih (mis. §5.2: "Yang tidak berubah karena
keputusan ini... tidak ada nilai P1–P4"). *Open Questions* di Bagian 12 benar-benar terbuka — OQ-2,
OQ-5, OQ-6 punya rekomendasi tapi tidak dipaksa jadi keputusan; hanya OQ-1 yang ditutup, dan itu
karena Chandra benar-benar memutuskannya.

### Findings
- **low** Peringatan ⚠️ dipakai untuk tiga kelas hal berbeda (placeholder data, cacat kriptografis,
  koreksi editorial) tanpa pembeda visual. *Fix:* pertimbangkan simbol berbeda per kelas jika dokumen
  ini terus tumbuh — tidak mendesak pada ukuran saat ini.

## Substance over theater — strong

Tidak ada *persona theater* — UJ-1 punya satu protagonis bernama, terikat skenario sidang yang
konkret. Tidak ada *NFR theater* — NFR-1..NFR-11 sebagian besar punya ambang angka spesifik (< 3
detik @ 500 TPS, ≥128 bit, dsb.), bukan "harus skalabel". Pernyataan objektif §2.1 spesifik pada
artefak ini (pergeseran *institutional* → *code-based trust*) — tidak bisa ditukar ke PRD lain tanpa
perubahan.

### Findings
- **medium** NFR-9 (*"Pencatatan tidak menurunkan pengalaman pemakaian aplikasi utama"*) adalah satu
  pengecualian — kata sifat tanpa ambang. *Fix:* beri angka (mis. "penambahan latensi pada jalur
  tulis HRIS < X ms dibanding tanpa pencatatan").

## Strategic coherence — strong

Tesis punya satu *thesis* yang tidak ambigu (§2.1, ditegaskan ulang §11 sebagai basis rekonsiliasi).
Pengelompokan FR (Kelompok A–F, B′) mengikuti langsung empat fungsi chaincode tesis — bukan daftar
keinginan. Kriteria sukses (P1–P4, matriks §9) memvalidasi klaim yang sesungguhnya diajukan (deteksi
manipulasi, isolasi, kepatuhan), bukan aktivitas yang tidak terkait. Baris "Tidak dapat diklaim" di
§9.2 berfungsi sebagai *counter-metric* yang jujur — pengakuan eksplisit atas apa yang **tidak**
terbukti, alih-alih menghaluskannya.

### Findings
Tidak ada temuan pada dimensi ini.

## Done-ness clarity — adequate

Sebagian besar FR punya konsekuensi yang dapat diuji (FR-3: *salt* ≥128 bit CSPRNG; FR-14: tidak
boleh mengembalikan *salt* dalam bentuk apa pun). Namun beberapa titik lebih lemah.

### Findings
- **medium** NFR-9 — lihat *Substance over theater* di atas; masalah yang sama juga membuat "done"
  untuk kriteria ini tidak terverifikasi.
- **low** FR-24 (*provisioning* channel tenant baru) tidak punya ambang waktu/SLA eksplisit. *Fix:*
  tambahkan target (mis. "tenant baru siap menerima anchor dalam < N menit setelah provisioning
  dimulai") saat FR ini diimplementasikan — tidak mendesak untuk versi PRD saat ini karena
  *provisioning* tenant belum masuk cakupan G8a/G8b manapun.
- **low** FR-9 ("konten section identik tidak menghasilkan record baru") tidak menyebut bagaimana
  "identik" dibandingkan — sebelum atau sesudah kanonikalisasi JCS? *Fix:* rujuk eksplisit ke FR-2/JCS
  saat spesifikasi teknis chaincode ditulis (di luar cakupan PRD ini).

## Scope honesty — strong (standout dimension)

Ini dimensi terkuat PRD ini. Kepadatan *open items* tinggi (OQ-1..OQ-6, S-1..S-5 di
`rencana-rekonsiliasi.md`, 14 butir errata, manifes rekonsiliasi 315 item) — tapi taruhannya juga
tinggi (dokumen ini melayani sidang tesis, forum keamanan internal, DAN build teknik sekaligus),
sehingga kepadatan tinggi ini **tepat**, bukan tanda bahaya, selama setiap butir punya pemilik dan
jalur resolusi — dan memang punya (`rencana-rekonsiliasi.md` §2 mencantumkan rekomendasi + konsekuensi
"tertahan" untuk tiap satu). Baris "Tidak dapat diklaim" di §9.2 adalah model *scope honesty*: bukan
disembunyikan, dinyatakan terbuka.

### Findings
- **low** Satu keputusan tidak tercatat formal di badan PRD: sintesis Workflow 1 (sebelum ratifikasi
  "tesis menang") merekomendasikan **menolak** jalur tesis-verbatim. Ini disampaikan ke Chandra secara
  langsung dalam percakapan dan Chandra melanjutkan dengan "tesis menang" — jadi secara substansi
  sudah diputuskan ulang — tapi PRD sendiri tidak mencatat bahwa rekomendasi yang berlawanan pernah
  ada dan dipertimbangkan. *Fix:* tambahkan satu baris di Lampiran A mencatat ini untuk kelengkapan
  jejak audit.

## Downstream usability — thin

Ini dimensi terlemah. ID kontinu dan tidak ada duplikat (FR-1..37, NFR-1..11, OQ-1..6, P1..4 —
diverifikasi via grep, semua bersih; FR-10 sengaja dipertahankan bertanda usang alih-alih
penomoran ulang, praktik yang wajar). Empat tautan jangkar internal semuanya *resolve*. Namun:

### Findings
- **high** Tidak ada Glosarium terpusat. Istilah `DataHash`, `EmployeeID`, `employeeKey_i`,
  `pseudonymKey`, *salt*, `KEY_EMPLOYEE` didefinisikan inline di §5.2 tapi tidak dikumpulkan sebagai
  satu bagian bernama — padahal dipakai lintas Kelompok D/E/F, §9, dan kedua berkas pendamping. *Fix:*
  tambahkan bagian "Glosarium" pendek setelah §5.2 atau di lampiran, terutama karena dokumen ini akan
  menjadi rujukan bagi `rencana-rekonsiliasi.md` (10 ADR baru) dan kerja arsitektur berikutnya.
- **low** Istilah "DataHash" dan "*digest*" dipakai bergantian di §9 dan `rencana-rekonsiliasi.md`
  tanpa dinyatakan sinonim. *Fix:* akan terselesaikan otomatis begitu Glosarium dibuat.

## Shape fit — fit, bentuk hibrida yang layak dinyatakan eksplisit

PRD ini tidak masuk kategori tunggal manapun di rubrik: ia sekaligus (a) **pembaruan
regulasi/kepatuhan** (§9 — keterlacakan kewajiban ke pasal adalah non-negotiable, dan PRD memenuhi
ini dengan baik), dan (b) **brownfield chain-top** (memberi makan 315 item manifes rekonsiliasi dan
10 ADR mendatang). Kategori rubrik tidak punya slot bernama untuk "catatan ratifikasi bagi tesis yang
sudah selesai + rekonsiliasi teknik yang berjalan bersamaan" — tapi bentuk PRD yang sekarang (berat
di keterlacakan batasan, kelompok FR yang dipetakan ke fungsi chaincode konkret, satu UJ dengan
protagonis nyata untuk keperluan sidang) cocok dengan hibrida itu. Ini bukan cacat, tapi layak
dinyatakan eksplisit di pembaca pertama (mis. di Ringkasan Eksekutif) agar pembaca tidak mencari
bentuk "PRD produk konsumen" yang tidak akan pernah muncul.

### Findings
- **low** Tidak ada temuan yang menuntut perbaikan; catatan di atas murni untuk orientasi pembaca.

## Mechanical notes

- **Glossary drift:** "DataHash" vs "*digest*" (lihat *Downstream usability*).
- **ID continuity:** bersih — diverifikasi via grep untuk FR/NFR/OQ/P, tidak ada celah atau duplikat.
- **Assumptions Index roundtrip:** tesis PRD ini tidak memakai tag `[ASSUMPTION]` inline standar
  skill (memakai OQ-1..6 dan tanda `[T]/[D]/[BARU]/[R]` sebagai gantinya) — pola ini konsisten
  dan diterapkan sepanjang dokumen, jadi tidak dianggap penyimpangan, hanya konvensi lokal yang layak
  dicatat sekali di sini agar pembaca lain tidak bingung mencari tag `[ASSUMPTION]` standar.
- **UJ protagonist naming:** UJ-1 dan UJ-2 (turunan) keduanya memenuhi syarat — protagonis nyata
  (Chandra Kurniawan) untuk UJ-1; UJ-2 memakai peran generik ("peer tenant B") yang wajar karena ia
  turunan teknis, bukan narasi manusia.
- **Bagian yang diperlukan untuk taraf ini:** hadir semua — Konteks, Objektif, Kriteria Sukses,
  Cakupan, Invarian, UJ, FR, NFR, Kepatuhan Regulasi, Metode Evaluasi, Keputusan Terbuka, Open
  Questions, dua Lampiran. Tidak ada bagian template yang dipaksakan tanpa isi nyata.

## Addendum — Sapuan Struktural Penuh (`bmad-editorial-review-structure`)

Dijalankan setelah tinjauan rubrik di atas, membaca dokumen penuh baris demi baris. Dua temuan
**diperbaiki langsung** (bukan hanya diusulkan) karena keduanya faktual/konsistensi, bukan pilihan
subjektif:

- **[SUDAH DIPERBAIKI] Kontradiksi UJ-1 langkah 6 vs OQ-6.** UJ-1 mengklaim *"nomor rekening asli
  direkonstruksi"* dari ledger — persis klaim yang OQ-6 (ditulis di bagian bawah dokumen yang sama)
  buktikan **tidak mungkin**, karena ledger hanya menyimpan `DataHash` satu arah. Ini kontradiksi
  yang bisa langsung ditemukan pembaca yang membaca dokumen top-to-bottom, termasuk penguji sidang.
  Diperbaiki mengikuti rekomendasi OQ-6 (opsi C + A): ledger menunjuk versi valid terakhir,
  pemulihan nilai dari PostgreSQL PITR, ledger membuktikan kandidat pemulihan itu benar.
- **[SUDAH DIPERBAIKI] Angka "dua belas titik penegakan" di §11.2 sudah usang.** Verifikasi
  independen (dicatat di `rencana-rekonsiliasi.md` §1.2) mengoreksi ini menjadi **24 kemunculan di
  17 berkas** sejak awal, tapi §11.2 di `prd.md` tidak pernah disinkronkan dengan koreksi itu.
  Diperbaiki, dengan tambahan konteks bahwa aturan ini tertanam di definisi agen aktif.

**Rekomendasi tersisa (belum dieksekusi, menunggu keputusan penulis):**

- **QUESTION** — Penomoran subbagian `3.1a` dan `5a` menyisipkan huruf di luar urutan numerik murni.
  *Rasional:* praktik umum untuk menghindari menomori ulang seluruh dokumen (dan merusak tautan
  jangkar yang sudah ada) saat menyisipkan bagian baru di tengah proses penulisan. *Dampak jika
  dirapikan:* nol secara substansi, hanya kerapian; berisiko merusak 2 tautan jangkar yang sudah
  *resolve*. **Rekomendasi: biarkan apa adanya** kecuali PRD ini mengalami restrukturisasi besar
  berikutnya.
- **PRESERVE** — §9.4 (pertanyaan hukum) ditulis sebagai paragraf padat, bukan tabel. *Rasional:*
  argumen hukum di bagian ini butuh nuansa (silogisme legal, rangkaian pasal) yang akan hilang kalau
  dipaksa ke format tabel MECE. Mempertahankan bentuk prosa di sini adalah keputusan yang benar,
  bukan kelalaian.

**Ringkasan:** dokumen sudah rapat secara struktural. Tidak ada bagian yang perlu dipotong, digabung,
atau dipindah pada skala berarti — dua perbaikan di atas adalah koreksi faktual yang ditemukan
sepanjang jalan, bukan hasil dari analisis struktur murni.
