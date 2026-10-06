# Deployment Depotin

Tiga komponen: basis data di Neon, API di VPS di belakang Caddy, dan web di Vercel. Urutannya mengikuti SPEC bagian 14.

## 1. Basis data (Neon)

1. Buat proyek Neon di region Singapura (`ap-southeast-1`), PostgreSQL 17 atau 18.
2. Buat dua peran lewat SQL Editor:

```sql
CREATE ROLE depotin_migrator LOGIN PASSWORD '<rahasia-1>';
CREATE ROLE depotin_app LOGIN PASSWORD '<rahasia-2>';
GRANT ALL ON SCHEMA public TO depotin_migrator;
ALTER DEFAULT PRIVILEGES FOR ROLE depotin_migrator IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO depotin_app;
ALTER DEFAULT PRIVILEGES FOR ROLE depotin_migrator IN SCHEMA public
  GRANT USAGE ON SEQUENCES TO depotin_app;
```

   Bila ingin cepat untuk demo, boleh memakai satu peran `neondb_owner` untuk keduanya.
3. Salin string koneksi **langsung** (tanpa `-pooler`) untuk tiap peran. Pakai `MIGRATE_DATABASE_URL` untuk `depotin_migrator` dan `DATABASE_URL` untuk `depotin_app`. Pooler Neon juga bisa dipakai bila hanya itu yang tersedia.
4. Jalankan migrasi dari mesin mana pun yang punya Go:

```sh
cd apps/api
MIGRATE_DATABASE_URL='postgres://...' go run ./cmd/migrate up
DATABASE_URL='postgres://...' WEB_ORIGIN='https://<web>' LINK_ENC_KEY='<base64 32 byte>' go run ./cmd/seed
```

## 2. API (VPS)

1. Daftarkan subdomain gratis, misalnya di DuckDNS, dan arahkan ke alamat IP VPS.
2. Pasang Docker Engine dan Docker Compose plugin. Aktifkan firewall:

```sh
sudo ufw allow 22/tcp && sudo ufw allow 80/tcp && sudo ufw allow 443/tcp && sudo ufw allow 443/udp && sudo ufw enable
```

3. Salin folder `deploy/` ke server, lalu `cp .env.example .env` dan isi:

| Variabel | Cara mengisi |
|---|---|
| `API_DOMAIN` | subdomain yang mengarah ke VPS |
| `DATABASE_URL`, `MIGRATE_DATABASE_URL` | dari Neon |
| `WEB_ORIGIN` | alamat Vercel final, tanpa garis miring di akhir |
| `EDGE_KEY`, `JWT_SECRET`, `LINK_ENC_KEY` | `openssl rand -base64 32`, masing-masing berbeda |
| `DEMO_RESET_HOUR` | jam WIB setel ulang depot contoh, kosongkan untuk mematikan |
| `DB_KEEPALIVE_UNTIL` | waktu ISO akhir jendela penjurian, misalnya `2026-11-14T12:00:00Z` |

4. Bangun atau tarik citra, lalu jalankan:

```sh
docker compose build api
docker compose run --rm --no-deps --entrypoint /app/migrate api up
docker compose run --rm --no-deps --entrypoint /app/seed api
docker compose up -d
```

   Caddy mengambil sertifikat Let's Encrypt otomatis untuk `API_DOMAIN`.
5. Periksa `https://<API_DOMAIN>/healthz`. Permintaan ke `/api/v1/readyz` tanpa `X-Edge-Key` harus ditolak dengan `403`.

Pembaruan selanjutnya cukup `./update.sh`: menarik citra, migrasi, mengganti kontainer, memeriksa kesehatan, dan kembali ke citra lama bila gagal.

## 3. Web (Vercel)

1. Impor repositori, setel **Root Directory** ke `apps/web`. Framework terdeteksi sebagai SvelteKit.
2. Di Settings, pilih region fungsi Singapura (`sin1`).
3. Isi variabel lingkungan:

| Variabel | Nilai |
|---|---|
| `API_ORIGIN` | `https://<API_DOMAIN>` |
| `EDGE_KEY` | sama dengan di API |
| `PUBLIC_STREAM_ORIGIN` | `https://<API_DOMAIN>` |
| `PUBLIC_DEMO_MODE` | `true` untuk menampilkan akun demo di halaman masuk |

4. Deploy. Setelah alamat final diketahui, perbarui `WEB_ORIGIN` di `.env` API lalu `docker compose up -d api`.

## Akun demo

| Peran | Nomor | Kata sandi |
|---|---|---|
| Pemilik | 081200000001 | demo-depotin-2026 |
| Kurir | 081200000002 | demo-depotin-2026 |
| Kurir | 081200000003 | demo-depotin-2026 |

Data depot contoh disetel ulang otomatis setiap hari pada `DEMO_RESET_HOUR`.
