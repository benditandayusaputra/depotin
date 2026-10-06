# Depotin: Prompt Master untuk Claude Code

## Cara pakai

1. Buat repositori kosong bernama `depotin`, lalu taruh `PRD.md` dan `SPEC.md` di folder `docs/`.
2. Buka Claude Code di akar repositori.
3. Tempel seluruh isi di bawah garis sebagai pesan pertama.
4. Bila sesi terputus, buka sesi baru dan kirim: `Lanjutkan pekerjaan sesuai docs/PROGRESS.md dan aturan di docs/MASTER_PROMPT.md.` Simpan juga berkas ini ke `docs/MASTER_PROMPT.md` supaya bisa dirujuk.

---

# Tugas

Bangun **Depotin**, sistem web untuk depot air minum isi ulang, dari repositori kosong sampai siap di-deploy. Kerjakan sampai seluruh tahap di bawah selesai dan lolos gerbangnya.

## Sumber kebenaran

Baca dua berkas ini **seluruhnya** sebelum menulis kode apa pun:

- `docs/PRD.md`: apa yang dibangun, untuk siapa, dan kriteria terimanya.
- `docs/SPEC.md`: arsitektur, model data, aturan domain, API, keamanan, performa, deployment, pengujian.

Aturan saat ragu:

1. Soal perilaku produk, ikuti `PRD.md`. Soal cara membangun, ikuti `SPEC.md`.
2. Bila keduanya diam, pilih jalan paling sederhana yang konsisten dengan Prinsip rancangan di `SPEC.md` bagian 1, lalu catat keputusan itu di `docs/DECISIONS.md` (tanggal, konteks, pilihan, alasan).
3. Jangan berhenti untuk bertanya, kecuali kamu butuh rahasia atau akses yang hanya dimiliki pengguna (Neon, VPS, Vercel). Pengembangan dan pengujian memakai PostgreSQL lokal lewat Docker, jadi hampir semua pekerjaan bisa selesai tanpa itu.

## Tumpukan teknologi

Sudah diputuskan di `SPEC.md` bagian 3 dan tidak boleh diganti: Go dengan Fiber v3, pgx v5, sqlc, goose, PostgreSQL, SvelteKit 2 dengan Svelte 5 runes dan TypeScript, Tailwind CSS v4, Caddy, Docker.

Sebelum memakai sebuah pustaka, periksa dokumentasi resminya untuk versi stabil terbaru. Pengetahuanmu bisa tertinggal, terutama untuk:

- Fiber v3: API berbeda dari v2 (handler menerima `fiber.Ctx`, pengikatan permintaan, konfigurasi `Listen`, middleware SSE).
- Svelte 5: pakai runes (`$state`, `$derived`, `$effect`, `$props`), bukan store gaya lama dan bukan `export let`.
- Tailwind v4: konfigurasi lewat CSS, bukan `tailwind.config.js`.
- adapter Vercel untuk SvelteKit: runtime edge sudah usang, pakai runtime Node.

Kunci semua versi di `go.mod` dan `package-lock.json`.

## Aturan kerja

### Orisinalitas

- Tulis semua kode dari nol. Jangan menyalin kode dari repositori, templat, atau contoh mana pun.
- Pustaka sumber terbuka boleh dipakai. Catat setiap pustaka, versinya, dan lisensinya di `docs/THIRD_PARTY.md`.

### Gaya kode

- **Tanpa komentar di kode.** Kode harus jelas dari penamaan dan strukturnya. Pengecualian hanya untuk arahan yang dibutuhkan alat: anotasi kueri sqlc (`-- name: ... :one`), arahan goose (`-- +goose Up`), dan arahan kompilator Go (`//go:embed`, `//go:generate`, `//go:build`). Penjelasan rancangan ditulis di `docs/`, bukan di kode.
- Go: lolos `gofmt`, `go vet`, dan `golangci-lint`. Galat selalu dibungkus dengan konteks. Tidak ada `panic` di jalur permintaan. `context.Context` diteruskan sampai ke kueri. Tidak ada variabel global yang bisa berubah.
- TypeScript: mode `strict`, tanpa `any`, tanpa `@ts-ignore`.
- SQL: hanya lewat sqlc. Tidak ada penyusunan SQL dari string.
- Tidak ada kode mati, tidak ada `TODO` tertinggal, tidak ada `console.log` atau cetak debug.
- Nama pengenal dalam bahasa Inggris. Semua teks antarmuka dalam bahasa Indonesia.
- Teks antarmuka singkat dan ramah, tanpa tanda pisah panjang.

### Git

- Satu commit untuk satu unit kerja yang utuh dan lolos tes. Jangan menumpuk satu tahap dalam satu commit raksasa.
- Format pesan: gitmoji, lalu kalimat yang menyebut pekerjaan nyata. Contoh: `✨ tambah perhitungan prediksi galon habis`, `🔒 tolak permintaan tanpa edge key`, `✅ uji isolasi data antar depot`. Jangan menyebut nomor tahap.
- Jangan menambahkan footer `Co-Authored-By`, teks "Generated with", atau jejak alat apa pun di pesan commit, kode, maupun dokumentasi.
- Jangan pernah meng-commit `.env`, rahasia, atau data pribadi.

### Keamanan yang tidak boleh dilanggar

Rinciannya ada di `SPEC.md` bagian 10. Yang paling sering terlewat:

1. `depot_id` dan `role` selalu dari token, tidak pernah dari badan atau kueri permintaan.
2. Setiap kueri data bisnis menyaring `depot_id`.
3. Harga dan total dihitung di server.
4. Token pelanggan dan token penyegar tidak pernah tercatat di log.
5. API menolak permintaan tanpa `X-Edge-Key`, kecuali dua jalur yang disebut di `SPEC.md`.
6. Tidak ada `{@html}` di Svelte.

### Cara maju

- Kerjakan tahap secara berurutan. Tiap tahap adalah irisan vertikal: basis data, API, antarmuka, dan tes untuk fitur itu selesai bersama, sehingga aplikasi selalu bisa didemokan.
- Jangan lanjut ke tahap berikutnya sebelum **gerbang** tahap itu lolos.
- Setelah tiap tahap, perbarui `docs/PROGRESS.md`: tahap selesai, tahap berjalan, hal yang ditunda, dan perintah untuk menjalankan aplikasi. Berkas ini adalah ingatanmu antar sesi.
- Bila sebuah tes gagal, perbaiki penyebabnya. Jangan melemahkan atau menghapus tes agar lolos.
- Bila kamu menemukan kesalahan di `SPEC.md`, perbaiki dokumennya dalam commit terpisah dan catat di `docs/DECISIONS.md`.

## Tahap pengerjaan

### Tahap 0: orientasi dan kerangka

- Baca `docs/PRD.md` dan `docs/SPEC.md`. Tulis ringkasan pemahamanmu dan daftar versi pustaka yang dipilih ke `docs/DECISIONS.md`.
- Buat struktur repositori sesuai `SPEC.md` bagian 4.
- Buat `docker-compose.dev.yml` (PostgreSQL), `Makefile` dengan target `dev`, `check`, `test`, `lint`, `migrate-up`, `migrate-down`, `sqlc`, `seed`, `types`.
- Buat `.gitignore`, `.env.example` untuk API dan web, `LICENSE`, dan kerangka `README.md`.

**Gerbang**: `make dev` menyalakan basis data lokal. Struktur folder sesuai spesifikasi.

### Tahap 1: fondasi

- API: konfigurasi, logger, pool basis data, paket `httpx` (respons, galat baku, validasi, paginasi kursor), `idgen`, `clock`, middleware (request ID, pemulihan, log, batas ukuran badan, edge key), `/healthz`, `/api/v1/readyz`, penghentian anggun.
- Migrasi seluruh tabel dan indeks di `SPEC.md` bagian 5, termasuk ekstensi `pg_trgm`. Siapkan sqlc.
- Web: SvelteKit, Tailwind v4, tata letak dasar, token desain (lihat Arah desain), proxy `/api/[...path]`, CSP lewat `kit.csp` mode `hash`, `hooks.server.ts` dengan header keamanan lainnya, klien API, generator tipe dari `openapi.yaml`.
- CI: alur kerja GitHub Actions sesuai `SPEC.md` bagian 15.3.

**Gerbang**: permintaan ke `/api/v1/readyz` lewat proxy web berhasil. Permintaan langsung ke API tanpa edge key ditolak dengan tes yang membuktikannya. `make check` hijau.

### Tahap 2: akun dan depot

- Pendaftaran depot, masuk, penyegaran token dengan rotasi dan deteksi pemakaian ulang, keluar, keluar dari semua perangkat, ganti kata sandi, penguncian setelah gagal berulang, pembatasan laju.
- Pengaturan depot dan pengelolaan akun kurir.
- Web: halaman daftar, masuk, penyiapan awal, pengaturan depot dan kurir, penjaga rute per peran.

**Gerbang**: tes untuk rotasi token, deteksi pemakaian ulang, penguncian, pemeriksaan Origin, dan penolakan akses kurir ke rute pemilik.

### Tahap 3: produk dan pelanggan

- CRUD produk dengan produk bawaan saat depot dibuat.
- CRUD pelanggan, pencarian trigram, normalisasi nomor HP, link pribadi (hash dan enkripsi), rotasi link.
- Web: pengaturan produk, daftar dan rincian pelanggan, tombol salin link dan kirim lewat WhatsApp.

**Gerbang**: tes isolasi antar depot untuk produk dan pelanggan. Rotasi membuat link lama tidak berlaku.

### Tahap 4: pesanan oleh pemilik

- Pembuatan pesanan dengan salinan harga, kode pesanan, idempotensi, mesin status, penugasan kurir, pembatalan, tandai lunas, riwayat peristiwa.
- Web: papan pesanan per status, formulir pesanan baru satu layar, rincian pesanan.

**Gerbang**: tes berbasis tabel untuk semua perpindahan status sah dan tidak sah. Permintaan ganda dengan `Idempotency-Key` sama menghasilkan satu pesanan.

### Tahap 5: pesanan oleh pelanggan

- Halaman publik depot, pesanan publik dengan aturan `SPEC.md` bagian 6.9, pelacakan bertoken, pembatalan oleh pelanggan.
- Web: `/d/[slug]` dan `/t/[token]` dengan SSR, header cache dan privasi yang benar, polling status.

**Gerbang**: Playwright untuk Alur B di `PRD.md`. Respons pesanan publik tidak membocorkan data pelanggan yang sudah ada.

### Tahap 6: kurir, penyelesaian, buku galon, loyalitas

- Antrean kurir terurut, berangkat, selesai dengan transaksi di `SPEC.md` bagian 6.2, buku galon, penyesuaian manual, loyalitas, simpan lokasi.
- Web: halaman kurir ramah seluler, lembar penyelesaian tanpa mengetik, antrean aksi luring, halaman Galon untuk pemilik.

**Gerbang**: tes integrasi yang membuktikan saldo galon selalu sama dengan jumlah buku dan tidak pernah negatif. Tes dua penyelesaian bersamaan pada pesanan yang sama hanya meloloskan satu. Permintaan ulang `deliver` dengan `Idempotency-Key` yang sama tidak menulis buku galon dua kali.

### Tahap 7: prediksi, pengingat, halaman pribadi

- Paket `prediction` sebagai fungsi murni dengan seluruh kasus uji wajib di `SPEC.md` bagian 6.5.
- Antrean pengingat, kirim, lewati, tunda, penjadwal harian, atribusi pesanan dari pengingat, daftar pelanggan berisiko.
- Web: halaman Pengingat, tampilan prediksi dan penjelasannya di rincian pelanggan, `/p/[token]` dengan pesan ulang satu ketukan.

**Gerbang**: Playwright untuk Alur A di `PRD.md`, dari antrean pengingat sampai pesanan selesai dan konversi tercatat.

### Tahap 8: pembaruan langsung

- Hub SSE, tiket sekali pakai, peristiwa, batas koneksi.
- Web: klien stream dengan sambung ulang dan cadangan polling, indikator koneksi, bunyi dan tanda pesanan baru.

**Gerbang**: pesanan yang dibuat di satu peramban muncul di dasbor peramban lain tanpa memuat ulang. Tiket tidak bisa dipakai dua kali.

### Tahap 9: dasbor, laporan, audit

- Dasbor hari ini, ringkasan rentang tanggal, ekspor CSV, log audit.
- Web: halaman Hari ini dan Laporan.

**Gerbang**: angka laporan cocok dengan data contoh yang dihitung manual di tes.

### Tahap 10: data contoh dan mode demo

- `cmd/seed` sesuai `SPEC.md` bagian 16, penyetelan ulang depot contoh terjadwal, tampilan akun demo di halaman masuk.

**Gerbang**: setelah `make seed`, setiap halaman menampilkan data yang berarti: ada pengingat, ada pelanggan berisiko, ada galon mengendap, ada pesanan di tiap status.

### Tahap 11: pengerasan

- Jalankan `EXPLAIN` pada kueri papan pesanan, antrean kurir, dan antrean pengingat. Pastikan memakai indeks.
- Ukur target performa di `SPEC.md` bagian 11.1. Perbaiki yang meleset.
- Audit aksesibilitas: kontras, fokus, label, navigasi papan ketik.
- Uji tampilan di lebar 360, 768, dan 1280 piksel.
- Tinjau ulang seluruh daftar keamanan di `SPEC.md` bagian 10, butir demi butir, dan tulis hasilnya di `docs/SECURITY_CHECKLIST.md`.
- Jalankan `govulncheck` dan `npm audit`.

**Gerbang**: semua target tercapai atau selisihnya tercatat beserta alasannya di `docs/DECISIONS.md`.

### Tahap 12: deployment dan dokumentasi

- `apps/api/Dockerfile` multi tahap, `deploy/docker-compose.yml`, `deploy/Caddyfile`, `deploy/update.sh`, dan `deploy/README.md` berisi langkah persis untuk Neon, VPS dengan subdomain gratis, dan Vercel.
- `README.md` lengkap: deskripsi, tangkapan layar, arsitektur singkat, cara menjalankan lokal, akun demo, perintah tes.
- `docs/ARCHITECTURE.md` dengan diagram Mermaid dan diagram relasi basis data.
- `docs/PROPOSAL_NOTES.md`: bahan siap salin untuk proposal lomba, disusun mengikuti urutan isi wajib proposal di `PRD.md` bagian 15. Sertakan tabel alasan teknologi, arsitektur dan diagram basis data, daftar fitur dan keunikan, batasan sistem, rencana pengembangan, daftar pustaka pihak ketiga beserta lisensinya, dan daftar tangkapan layar yang perlu diambil. Proposal dibatasi 15 halaman, jadi tulis ringkas.

**Gerbang**: citra Docker berhasil dibangun dan berjalan lokal di belakang Caddy. Seseorang yang hanya membaca `README.md` bisa menjalankan proyek.

Jangan melakukan deploy sungguhan kecuali pengguna memberikan akses. Bila sampai di titik ini tanpa akses, tulis daftar variabel lingkungan yang perlu diisi dan berhenti.

## Arah desain

- Pengguna utama adalah pemilik depot dan kurir yang memakai HP kelas menengah ke bawah, sering di luar ruangan. Utamakan keterbacaan dan kecepatan, bukan efek visual.
- Tema terang sebagai bawaan dengan dukungan tema gelap mengikuti sistem. Hindari `backdrop-filter` dan animasi berat.
- Satu warna aksen biru air, netral abu hangat, serta warna status yang konsisten (menunggu, dikonfirmasi, diantar, selesai, batal), selalu disertai teks.
- Tipografi besar dan lega: teks isi minimal 16 piksel, angka penting dibuat menonjol.
- Target sentuh minimal 48 piksel di halaman kurir dan halaman pelanggan.
- Definisikan token desain (warna, jarak, radius, bayangan, ukuran huruf) sekali di CSS, lalu pakai konsisten.
- Halaman pelanggan harus terasa seperti satu tombol besar, bukan aplikasi.

## Definisi selesai

Proyek selesai bila semua butir ini benar:

1. Seluruh fitur Must di `PRD.md` bagian 5.1 memenuhi kriteria terimanya. Fitur Should dikerjakan sesuai urutan, dan yang tidak sempat dicatat di `docs/PROGRESS.md`.
2. Tiga alur utama di `PRD.md` bagian 6 lolos Playwright pada viewport HP dan desktop.
3. `make check` hijau dan CI hijau.
4. Definisi selesai di `SPEC.md` bagian 17 terpenuhi untuk setiap fitur.
5. `docs/SECURITY_CHECKLIST.md` terisi lengkap.
6. Repositori bersih: tanpa rahasia, tanpa komentar kode, tanpa kode mati, tanpa jejak alat di riwayat commit.

## Laporan akhir

Saat selesai, tulis ringkasan berisi:

1. Fitur yang selesai dan yang ditunda, dengan alasannya.
2. Hasil pengukuran performa terhadap target.
3. Keputusan penting yang menyimpang dari `SPEC.md`.
4. Langkah yang harus dilakukan pengguna sendiri: membuat proyek Neon, menyiapkan VPS dan subdomain, mengisi variabel lingkungan, deploy ke Vercel.
5. Saran urutan adegan untuk video demo 3 sampai 5 menit, dibuka dengan halaman Pengingat.

Mulai dari Tahap 0 sekarang.
