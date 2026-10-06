# Depotin

Sistem web untuk depot air minum isi ulang. Pemilik mengelola pesanan, kurir, dan galon pinjaman dari satu dasbor. Pelanggan memesan ulang lewat satu link tanpa akun. Fitur pembeda: prediksi galon habis dan antrean pengingat WhatsApp.

Dokumen produk dan teknis ada di [docs/PRD.md](docs/PRD.md) dan [docs/SPEC.md](docs/SPEC.md).

## Menjalankan lokal

```sh
make dev          # PostgreSQL lokal lewat Docker
make migrate-up   # migrasi skema
make seed         # data contoh
make api          # API Go di :8080
make web          # SvelteKit di :5173
```

Salin `apps/api/.env.example` ke `apps/api/.env` dan `apps/web/.env.example` ke `apps/web/.env`, lalu isi rahasianya.

## Perintah

| Perintah | Fungsi |
|---|---|
| `make check` | lint, sqlc diff, dan seluruh tes |
| `make test` | tes API dan web |
| `make lint` | gofmt, go vet, golangci-lint, svelte-check, ESLint |
| `make sqlc` | generate kode dari SQL |
| `make types` | generate tipe TypeScript dari openapi.yaml |

## Lisensi

MIT. Lihat [LICENSE](LICENSE).
