# Kemajuan

## Tahap selesai

- Tahap 0: orientasi dan kerangka.
- Tahap 1: fondasi API, migrasi skema, fondasi web dengan proxy dan CSP, CI.
- Tahap 2: akun dan depot (API dan web): daftar, masuk, rotasi token, penguncian, pengaturan depot, kurir, produk, penjaga rute.
- Tahap 3: produk dan pelanggan, link pribadi terenkripsi dan rotasi, halaman pelanggan dengan prediksi.
- Tahap 4: pesanan pemilik dengan idempotensi dan mesin status, papan pesanan, formulir satu layar, rincian pesanan.
- Tahap 5: halaman publik `/d/[slug]`, pelacakan `/t/[token]`, pesanan publik, Playwright Alur B.
- Tahap 6: kurir, penyelesaian satu transaksi, buku galon, loyalitas, antrean luring IndexedDB, halaman Galon.
- Tahap 7: prediksi, antrean pengingat, kirim WA, atribusi konversi, `/p/[token]` pesan ulang satu ketukan.
- Tahap 8: hub SSE, tiket sekali pakai, klien stream dengan sambung ulang dan mode berkala, bunyi pesanan baru.
- Tahap 9: dasbor Hari ini, laporan rentang tanggal, ekspor CSV, log audit.
- Tahap 10: data contoh deterministik, setel ulang demo terjadwal, akun demo di halaman masuk.
- Tahap 12 (sebagian): Dockerfile, compose, Caddyfile, update.sh, deploy/README, ARCHITECTURE, PROPOSAL_NOTES, README.

## Tahap berjalan

- Tahap 7 gerbang: Playwright Alur A lolos (`tests/alur-a.spec.ts`).
- Tahap 11: pengerasan. Sudah: EXPLAIN, pengukuran p95 lokal, govulncheck dan npm audit, SECURITY_CHECKLIST, cakupan tes. Berjalan: audit aksesibilitas, uji lebar 360/768/1280, Lighthouse.
- Tahap 12 sisa: tangkapan layar untuk README dan proposal (berjalan).

## Ditunda

- Fitur Could (`PRD.md` 5.3) tidak dikerjakan: F-17 jual dadakan oleh kurir, F-18 impor pelanggan CSV, F-19 manifest PWA. Alasan: prioritas Must dan Should lebih dulu; ketiganya tidak dibutuhkan tiga alur utama.
- Deploy sungguhan ke Neon (sudah dimigrasi dan diisi data contoh), VPS, dan Vercel menunggu akses dan keputusan pengguna. Langkahnya di `deploy/README.md`.

## Menjalankan aplikasi

```sh
make dev
make migrate-up
make seed
cd apps/api && go run ./cmd/api
cd apps/web && npm install && npm run dev
```

Akun demo setelah `make seed`: pemilik 081200000001, kurir 081200000002 dan 081200000003, kata sandi `demo-depotin-2026`.

API membaca variabel di `apps/api/.env.example`, web membaca `apps/web/.env.example`. Nilai bawaan pengembangan sudah bekerja tanpa mengisi rahasia.
