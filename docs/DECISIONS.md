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
