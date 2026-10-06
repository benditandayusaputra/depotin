# Pustaka Pihak Ketiga

Semua pustaka di bawah berlisensi permisif (MIT, BSD-3-Clause, atau Apache-2.0) dan dipakai sesuai lisensinya. Tidak ada kode yang disalin dari repositori atau templat lain.

## API (Go)

| Pustaka | Versi | Lisensi | Kegunaan |
|---|---|---|---|
| github.com/gofiber/fiber/v3 | v3.5.0 | MIT | kerangka HTTP, middleware SSE |
| github.com/jackc/pgx/v5 | v5.11.0 | MIT | driver PostgreSQL dan pool koneksi |
| github.com/pressly/goose/v3 | v3.28.0 | MIT | migrasi skema |
| github.com/sqlc-dev/sqlc | v1.31.1 | MIT | alat generate kode dari SQL, hanya saat pengembangan |
| github.com/golang-jwt/jwt/v5 | v5.3.1 | MIT | token akses JWT HS256 |
| golang.org/x/crypto | v0.57.0 | BSD-3-Clause | Argon2id |
| github.com/go-playground/validator/v10 | v10.30.5 | MIT | validasi struct permintaan |
| github.com/goccy/go-json | v0.11.2 | MIT | enkoder JSON cepat |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause | UUIDv7 |
| github.com/golangci/golangci-lint/v2 | v2.14.0 | GPL-3.0 | alat lint, hanya saat pengembangan, tidak masuk biner |
| golang.org/x/vuln | terbaru | BSD-3-Clause | govulncheck di CI |

Dependensi transitif Go tercatat di `apps/api/go.sum`.

## Web (Node)

| Pustaka | Versi | Lisensi | Kegunaan |
|---|---|---|---|
| svelte | 5.57.1 | MIT | komponen antarmuka |
| @sveltejs/kit | 2.70.3 | MIT | kerangka aplikasi, routing, SSR |
| @sveltejs/adapter-vercel | 6.3.4 | MIT | deploy ke Vercel |
| @sveltejs/vite-plugin-svelte | 6.2.4 | MIT | integrasi Vite |
| vite | 7.3.7 | MIT | bundler |
| tailwindcss, @tailwindcss/vite | 4.3.3 | MIT | gaya |
| typescript | 5.9.3 | Apache-2.0 | bahasa |
| openapi-typescript | 7.13.0 | MIT | tipe dari openapi.yaml |
| vitest | 4.1.11 | MIT | tes unit |
| @playwright/test | 1.63.0 | Apache-2.0 | tes ujung ke ujung |
| eslint, @eslint/js | 10.12.0, 10.0.1 | MIT | lint |
| eslint-plugin-svelte | 3.23.0 | MIT | lint Svelte |
| typescript-eslint | 8.71.1 | MIT | lint TypeScript |
| svelte-check | 4.7.6 | MIT | pemeriksaan tipe Svelte |
| prettier, prettier-plugin-svelte | 3.9.9, 4.1.1 | MIT | format kode |
| globals | 17.13.0 | MIT | daftar global untuk ESLint |

Dependensi transitif Node tercatat di `apps/web/package-lock.json`.

## Infrastruktur

| Komponen | Lisensi | Kegunaan |
|---|---|---|
| PostgreSQL 18 | PostgreSQL License | basis data |
| Caddy 2 | Apache-2.0 | proxy balik dan TLS |
| Docker, citra distroless | Apache-2.0 | kontainer |

## Aset

Tidak ada font, ikon, atau gambar pihak ketiga. Ikon berupa SVG sebaris buatan sendiri dan font memakai tumpukan font sistem.
