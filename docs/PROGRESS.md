# Kemajuan

## Tahap selesai

- Tahap 0: orientasi dan kerangka.
- Tahap 1: fondasi API, migrasi skema, fondasi web dengan proxy dan CSP, CI.
- Sisi API untuk Tahap 2 sampai 10: akun dan depot, produk dan pelanggan, pesanan pemilik, endpoint publik, penyelesaian kurir dengan buku galon dan loyalitas, prediksi dan pengingat, SSE, dasbor dan laporan, data contoh dan penjadwal. Semua dengan tes HTTP dan unit (`make test-api`).
- Berkas deployment: Dockerfile distroless, `deploy/docker-compose.yml`, `deploy/Caddyfile`, `deploy/update.sh`, `deploy/README.md`.

## Tahap berjalan

- Sisi web untuk Tahap 2 (daftar, masuk, penyiapan, pengaturan, penjaga rute) lalu Tahap 3 sampai 9 (pelanggan, pesanan, kurir, halaman publik, pengingat, stream, laporan).

## Ditunda

Belum ada.

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
