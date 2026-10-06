# Kemajuan

## Tahap selesai

- Tahap 0: orientasi dan kerangka.
- Tahap 1: fondasi API, migrasi skema, fondasi web dengan proxy dan CSP, CI.

## Tahap berjalan

- Tahap 2: akun dan depot.

## Ditunda

Belum ada.

## Menjalankan aplikasi

```sh
make dev
make migrate-up
cd apps/api && go run ./cmd/api
cd apps/web && npm install && npm run dev
```

API membaca variabel di `apps/api/.env.example`, web membaca `apps/web/.env.example`. Nilai bawaan pengembangan sudah bekerja tanpa mengisi rahasia.
