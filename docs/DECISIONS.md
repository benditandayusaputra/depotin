# Catatan Keputusan

Format tiap butir: tanggal, konteks, pilihan, alasan.

## 2026-10-06: Ringkasan pemahaman

Depotin adalah sistem multi depot untuk usaha air isi ulang dengan tiga peran: pemilik (dasbor penuh), kurir (antrean antar di HP), dan pelanggan (tanpa akun, lewat link bertoken). Inti pembeda adalah prediksi galon habis per pelanggan yang diturunkan dari jarak antar pengantaran, lalu dipakai menyusun antrean pengingat WhatsApp harian yang konversinya diukur.

Batas arsitektur yang paling menentukan:

- Peramban hanya berbicara ke origin web (SvelteKit di Vercel). Semua `/api/*` diteruskan ke API Go lewat proxy yang membubuhkan `X-Edge-Key`. Satu pengecualian: SSE dibuka langsung ke API dengan tiket sekali pakai.
- Server menentukan `depot_id`, `role`, harga, dan total. Klien tidak dipercaya.
- Logika domain (prediksi, mesin status, buku galon, loyalitas, urutan antar) adalah fungsi murni tanpa basis data.
- SQL ditulis tangan dan dikompilasi lewat sqlc. Tidak ada ORM.
- Satu proses API tanpa Redis. Hub SSE, pembatas laju, cache, dan tiket hidup di memori.

Urutan pengerjaan mengikuti 13 tahap di `MASTER_PROMPT.md`, tiap tahap irisan vertikal dengan gerbang tes.

## 2026-10-06: Versi pustaka

| Pustaka | Versi | Catatan |
|---|---|---|
| Go | 1.25.5 | toolchain lokal, minimal 1.25 sesuai SPEC |
| github.com/gofiber/fiber/v3 | v3.5.0 | stabil terbaru |
| github.com/jackc/pgx/v5 | v5.11.0 | |
| github.com/pressly/goose/v3 | v3.28.0 | |
| github.com/sqlc-dev/sqlc | v1.31.1 | alat generate, tidak masuk biner |
| github.com/golang-jwt/jwt/v5 | v5.3.1 | |
| golang.org/x/crypto | v0.57.0 | argon2 |
| github.com/go-playground/validator/v10 | v10.30.5 | |
| github.com/goccy/go-json | v0.11.2 | |
| github.com/google/uuid | v1.6.0 | UUIDv7 |
| golangci-lint | v2.14.0 | alat lint |
| PostgreSQL | 18 | Neon menjalankan 18.6, lokal `postgres:18-alpine` |
| Node | 24 | |
| @sveltejs/kit | 2.70.3 | lihat keputusan di bawah |
| svelte | 5.57.1 | runes |
| @sveltejs/adapter-vercel | 6.3.4 | runtime Node |
| @sveltejs/vite-plugin-svelte | 6.2.4 | |
| vite | 7.3.7 | |
| tailwindcss, @tailwindcss/vite | 4.3.3 | |
| typescript | 5.9.3 | |
| openapi-typescript | 7.13.0 | |
| vitest | 4.1.11 | |
| @playwright/test | 1.63.0 | |

## 2026-10-06: Tetap di SvelteKit 2, bukan 3

Konteks: SvelteKit 3.0.0 terbit 1 Oktober 2026, lima hari sebelum proyek dimulai. Adapter Vercel 7 untuk Kit 3 juga baru terbit.

Pilihan: SvelteKit 2.70.3 dengan adapter-vercel 6.3.4 dan Vite 7.

Alasan: SPEC bagian 3 menetapkan SvelteKit 2 dan melarang mengganti tumpukan. Rilis mayor berumur beberapa hari berisiko membawa regresi pada adapter dan alat pendukung (svelte-check, eslint-plugin-svelte) saat tenggat lomba dekat.

## 2026-10-06: TypeScript 5.9, bukan 7

Konteks: TypeScript 7 (kompilator native) sudah terbit, tetapi svelte-check dan @sveltejs/kit menyatakan dukungan peer `^5 || ^6`.

Pilihan: TypeScript 5.9.3.

## 2026-10-06: Basis data lokal di port 54329

Konteks: mesin pengembang sudah punya PostgreSQL Homebrew di 5432.

Pilihan: kontainer dev memetakan ke `54329` agar tidak bertabrakan.

## 2026-10-06: Neon memakai string koneksi pooler

Konteks: SPEC 11.2 meminta koneksi langsung Neon. Pengguna memberikan string koneksi `-pooler` dan belum memisahkan peran `depotin_migrator` dan `depotin_app`.

Pilihan: untuk lingkungan yang disediakan pengguna, `DATABASE_URL` dan `MIGRATE_DATABASE_URL` memakai string yang diberikan. Pemisahan peran dan koneksi langsung dicatat sebagai langkah pengguna di `deploy/README.md`.

Alasan: akses ke konsol Neon hanya dimiliki pengguna. Pooler Neon (PgBouncer 1.21 ke atas) mendukung prepared statement tingkat protokol, sehingga mode bawaan pgx tetap bekerja.

## 2026-10-06: Font sistem, bukan berkas font

Konteks: SPEC 11.4 menyebut font disajikan sendiri dengan subset Latin dan dua ketebalan.

Pilihan: tumpukan font sistem (`system-ui, -apple-system, Segoe UI, Roboto`) tanpa berkas font.

Alasan: nol permintaan jaringan dan nol byte tambahan untuk HP kelas bawah, tidak ada skrip atau aset pihak ketiga, dan tidak ada lisensi font yang perlu dicatat. Font sistem Android (Roboto) dan iOS sudah terbaca baik untuk angka besar.

## 2026-10-06: style-src mengizinkan unsafe-inline

Konteks: SPEC 10.5 meminta CSP mode hash. SvelteKit 2 merender pengumum navigasi (`#svelte-announcer`) dengan atribut `style` sebaris dari templat internalnya. Dengan `style-src 'self'`, Firefox memblokirnya sehingga teks judul halaman tampil di bawah halaman setelah navigasi klien.

Pilihan: `style-src 'self' 'unsafe-inline'`. Skrip tetap ketat dengan hash.

Alasan: alternatifnya adalah menyematkan hash dari string internal SvelteKit yang bisa berubah saat pembaruan minor. Risiko `unsafe-inline` pada gaya jauh lebih kecil daripada pada skrip, dan tidak ada `{@html}` di aplikasi.

## 2026-10-06: Go 1.26

Konteks: Fiber v3.5.0 menuntut `go >= 1.26`, sehingga `go.mod` naik ke 1.26.0 saat dependensi ditambahkan.

Pilihan: proyek memakai Go 1.26 (toolchain lokal diunduh otomatis, Dockerfile memakai `golang:1.26-alpine`). Tabel versi di atas yang menyebut 1.25.5 sudah tidak berlaku.

## 2026-10-06: Penjadwal memakai satu zona waktu

Konteks: SPEC 6.6 meminta penjadwal berjalan pukul 06.00 zona waktu depot, dan SPEC 11.5 meminta penjadwal tidak menyentuh basis data tiap menit.

Pilihan: penjadwal menghitung jadwal berikutnya di memori untuk zona Asia/Jakarta, lalu menyusun antrean semua depot sekali sehari pada 06.00 WIB. Antrean juga disusun ulang secara idempoten setiap kali halaman Pengingat dibuka, sehingga depot di zona lain tetap mendapat antrean yang benar saat dipakai.

Alasan: semua depot bawaan memakai `Asia/Jakarta`, dan satu kali bangun per hari menjaga kuota komputasi Neon.

## 2026-10-06: Lewati pengingat hanya untuk hari itu

Konteks: PRD F-09 menyebut pemilik bisa melewati atau menunda pengingat.

Pilihan: Lewati menandai pengingat `skipped` dan pelanggan bisa muncul lagi besok. Untuk menunda beberapa hari, pakai Tunda yang mengisi `reminder_snoozed_until`. Ini mengikuti SPEC 6.6 yang hanya menjamin pengingat yang dilewati tidak muncul lagi di hari yang sama.

## 2026-10-06: Kurir melihat total tagihan

Konteks: SPEC 10.3 menyebut kurir tidak menerima data omzet, sedangkan PRD F-06 menyebut kurir melihat total yang harus ditagih.

Pilihan: tampilan kurir (`CourierOrder`) memuat `total`, `refill_qty`, dan `free_qty`, tetapi tidak memuat `subtotal`, `discount`, `delivery_fee`, maupun data agregat omzet. Endpoint dasbor dan laporan ditolak untuk kurir.

## 2026-10-06: Hasil EXPLAIN pada data contoh

Konteks: Tahap 11 meminta memastikan kueri papan pesanan, antrean kurir, dan antrean pengingat memakai indeks.

Hasil: pada data contoh perencana memilih pemindaian berurutan karena setiap tabel hanya beberapa halaman (waktu eksekusi di bawah 0,2 ms). Dengan `enable_seqscan = off`, papan pesanan memakai `orders_board_idx`, antrean kurir memakai `orders_courier_queue_idx`, dan antrean pengingat memakai `customers_reminder_idx`, `orders_customer_active_idx`, serta `reminders_depot_status_due_idx`. Tidak ada kueri yang membutuhkan indeks tambahan.

## 2026-10-06: govulncheck bersih setelah mengunci toolchain

Konteks: `govulncheck` melaporkan 20 kerentanan pustaka standar karena `go.mod` hanya menyebut `go 1.26.0`, sehingga toolchain 1.26.0 yang dipakai.

Pilihan: menambahkan `toolchain go1.26.8` di `go.mod`. Setelah itu `govulncheck ./...` melaporkan nol kerentanan yang dipanggil kode. `npm audit --omit=dev` juga nol.

## 2026-10-06: Pengukuran performa API lokal

Konteks: SPEC 11.1 menargetkan p95 baca di bawah 150 ms, p95 tulis pesanan di bawah 300 ms, dan memori API saat diam di bawah 60 MB.

Hasil pada data contoh, API dan PostgreSQL di mesin yang sama, 100 permintaan berurutan per endpoint:

| Endpoint | p50 | p95 |
|---|---|---|
| `GET /orders` | 3,0 ms | 4,6 ms |
| `GET /customers` | 1,4 ms | 2,4 ms |
| `GET /dashboard/today` | 3,8 ms | 8,9 ms |
| `GET /reminders` (termasuk susun ulang antrean) | 0,4 ms | 0,7 ms |
| `GET /public/depots/{slug}` (cache) | 0,3 ms | 0,5 ms |
| `POST /orders` | 0,4 ms | 0,5 ms |

Memori proses API setelah pengukuran: 31 MB. Di produksi, waktu tambahan berasal dari jarak jaringan VPS ke Neon (sekitar 1 sampai 5 ms per kueri di region yang sama) dan bangun tidur Neon yang dikecualikan dari target.

## 2026-10-06: Batas pendaftaran dilonggarkan di lingkungan pengembangan

Konteks: SPEC 10.7 membatasi `POST /auth/register` 3 per jam per IP. Di pengembangan lokal semua peramban dan Playwright berbagi satu alamat (`::1`), sehingga menjalankan e2e dua kali dalam satu jam sudah terblokir.

Pilihan: pada `APP_ENV=development` batasnya 100 per jam per IP; produksi tetap 3 per jam. Aturan lain tidak berubah.

## 2026-10-06: Neon lewat pooler sudah diuji

Konteks: skema dimigrasikan dan data contoh dimuat ke proyek Neon milik pengguna lewat string koneksi `-pooler` (ap-southeast-1).

Hasil: API berjalan tanpa galat terhadap pooler, termasuk prepared statement bawaan pgx. Dari mesin pengembang di Indonesia, `GET /orders` 130 ms pada permintaan pertama dan 67 ms setelahnya, `GET /dashboard/today` 295 ms karena beberapa kueri berurutan. Seed penuh memakan 190 detik karena ribuan kueri kecil melintasi jaringan, sehingga setel ulang demo harian dijalankan di VPS yang satu region dengan Neon. Target p95 SPEC diukur di dalam satu region, bukan dari luar negeri.

`apps/api/.env` lokal (tidak di-commit) mengarah ke Neon agar `make api` memakai basis data yang sama dengan demo, sementara `make test-api` tetap memakai PostgreSQL Docker.

## 2026-10-06: Cakupan tes API

Diukur dengan `go test -race -coverpkg=./internal/... ./...` termasuk tes HTTP di `cmd/api` terhadap PostgreSQL lokal. Target SPEC 15.1 minimal 80% untuk `prediction`, `order`, `gallon`, dan `auth` terpenuhi.

| Paket | Cakupan |
|---|---|
| internal/prediction | 99,2% |
| internal/gallon | 92,0% |
| internal/customer | 90,6% |
| internal/report | 87,4% |
| internal/order | 85,3% |
| internal/auth | 84,5% |
| internal/stream | 81,8% |
| internal/public | 81,3% |
| internal/reminder | 80,3% |

Tambahan: batas pesanan publik per IP dan per nomor, serta batas masuk per IP dan nomor, juga dilonggarkan menjadi 100 pada `APP_ENV=development` dengan alasan yang sama (suite Playwright masuk sebagai pemilik berkali-kali dalam satu menit). Batas global 300 per menit per IP menjadi 5000 di development karena dua suite berturut-turut dari satu alamat melampauinya. Tes HTTP di `cmd/api` menyusun aplikasi tanpa pelonggaran sehingga tetap memverifikasi batas asli (5 per 10 menit per IP, 3 per jam per nomor, 3 pendaftaran per jam).

## 2026-10-06: Koneksi SSE terbaru menggantikan yang tertua

Konteks: SPEC 8.2 membatasi 3 koneksi serentak per pengguna. Setiap navigasi di dasbor membuka koneksi baru, dan koneksi lama baru terdeteksi putus pada detak jantung berikutnya (20 detik), sehingga muat ulang beberapa kali dalam 20 detik membuat koneksi baru ditolak dan klien jatuh ke mode berkala.

Pilihan: batas 3 tetap, tetapi saat penuh hub memutus koneksi tertua milik pengguna itu dan menerima yang baru. Tab yang masih hidup menyambung ulang sendiri.

## 2026-10-06: Urutan antrean pengingat

Antrean diurutkan dari pelanggan yang perkiraan habisnya paling dekat dengan hari ini (selisih absolut terkecil), bukan dari yang paling lama lewat. Pelanggan yang sudah lama lewat tetap masuk antrean, tetapi muncul di bawah, karena mereka juga tampil di filter Berisiko.

## 2026-10-06: Hasil audit aksesibilitas, responsif, dan Lighthouse

Diukur pada data contoh dengan axe-core 4.10.2 (WCAG 2.1 A dan AA), lebar 360 dan 1280, mode terang dan gelap, untuk 17 halaman: beranda, masuk, daftar, halaman publik depot, pelacakan, link pribadi, seluruh halaman pemilik, dan halaman kurir.

- axe: nol pelanggaran di semua halaman setelah perbaikan. Perbaikan yang dilakukan: kontras token `--color-status-pending` (#92400e) dan `--color-status-done` (#166534) menjadi di atas 6:1, warna placeholder, tautan Daftar di mode gelap, judul dokumen cadangan saat sesi gagal, target sentuh tautan sebaris minimal 44 piksel, badge dan teks halaman pelanggan dan kurir minimal 16 piksel, fokus masuk ke lembar penyelesaian kurir dan kembali ke tombol pembuka saat ditutup, Escape menutup lembar dan menu tunda, cincin fokus tidak terpotong di baris chip.
- Responsif: tidak ada luapan mendatar di 360, 768, dan 1280. Target sentuh 48 piksel di halaman kurir dan pelanggan, 44 piksel di dasbor. Pengecualian: kotak centang asli `Toggle` berukuran 24 piksel di dalam label setinggi 48 piksel.
- Lighthouse 13.5 (emulasi seluler, build produksi): beranda 100/100/100, halaman publik depot 99/100/100, link pribadi 99/100/100 untuk performa, aksesibilitas, praktik terbaik. LCP 1,5 sampai 2,0 detik, TBT 0 ms, CLS 0.
- JavaScript awal rute `/p/[token]`: 52,9 KB gzip (target 70 KB).
