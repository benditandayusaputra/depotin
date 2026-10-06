# Daftar Periksa Keamanan

Tinjauan butir demi butir terhadap `SPEC.md` bagian 10. Status: **ya** terpenuhi dan diuji, **ya (manual)** terpenuhi dan diperiksa tanpa tes otomatis, **pengguna** harus dilakukan pengguna saat deploy.

## 10.1 Lapisan jaringan

| Butir | Status | Bukti |
|---|---|---|
| TLS 1.2 ke atas, sertifikat otomatis, HSTS, HTTP/2 dan HTTP/3 oleh Caddy | ya (manual) | `deploy/Caddyfile`: header HSTS, Caddy 2 bawaan TLS otomatis dan HTTP/3 |
| Firewall VPS hanya 22, 80, 443 dan SSH berkunci | pengguna | langkah di `deploy/README.md` |
| API hanya mendengarkan di jaringan Docker internal | ya (manual) | `deploy/docker-compose.yml`: layanan `api` memakai `expose`, tanpa `ports` |
| API menolak permintaan tanpa `X-Edge-Key` kecuali `/healthz` dan `GET /api/v1/stream`, perbandingan waktu tetap | ya | `httpx.EdgeKeyMiddleware` memakai `crypto/subtle`; tes `TestReadyzRejectsWithoutEdgeKey`, `TestHealthzWithoutEdgeKey`, `TestStreamTicketIsSingleUse` |
| `X-Client-IP` hanya dipercaya bila edge key benar | ya | `httpx.ClientIP` memeriksa `EdgeTrusted`; jalur stream memakai alamat dari Caddy lewat `X-Forwarded-For` dengan `TrustProxyConfig{Private, Loopback}` |

## 10.2 Autentikasi

| Butir | Status | Bukti |
|---|---|---|
| Argon2id 19 MiB, 2 iterasi, 1 paralel, salt 16 byte, keluaran 32 byte, format PHC | ya | `crypto/password.go`; tes `TestHashAndVerifyPassword` memeriksa awalan PHC |
| Kata sandi 10 sampai 128 karakter | ya | validasi `min=10,max=128` di register, kurir, ganti kata sandi; tes `TestRegisterValidation` |
| JWT HS256 15 menit dengan klaim `sub`, `did`, `role`, `fid`, `iat`, `exp`; algoritma dikunci | ya | `auth/token.go` memakai `WithValidMethods`; tes `TestParseRejectsOtherAlgorithms` menolak `none` dan HS512 |
| Token penyegar 32 byte acak, 30 hari, disimpan SHA-256, dirotasi, pemakaian ulang mencabut keluarga | ya | `auth/service.go` `Refresh`; tes `TestRefreshRotatesAndDetectsReuse` |
| Cookie `__Host-dp_at` Lax Path=/ dan `__Secure-dp_rt` Strict Path=/api/v1/auth, HttpOnly, Secure | ya | `auth/cookies.go`; awalan dan `Secure` dimatikan saat `COOKIE_SECURE=false` |
| Pembatas 5 per menit per IP dan nomor, kunci 15 menit setelah 5 gagal, pesan tidak membedakan nomor dan sandi, waktu respons disamakan | ya | tes `TestLoginRateLimitPerIPAndPhone`, `TestLoginLockoutAfterFiveFailures`, `TestLoginDoesNotRevealUnknownPhone`; hash tiruan diverifikasi saat nomor tak dikenal |

## 10.3 Otorisasi

| Butir | Status | Bukti |
|---|---|---|
| `RequireRole` memeriksa peran dari token | ya | `httpx.RequireRole`; tes `TestCourierCannotReachOwnerRoutes` |
| Setiap kueri data bisnis menerima `depot_id` | ya | semua kueri di `db/queries/*.sql` menyaring `depot_id` atau memakai kunci yang sudah terikat depot (token pelacakan, token pribadi) |
| Tes isolasi antar depot untuk tiap sumber daya | ya | `TestUsersAreIsolatedPerDepot`, `TestProductsCrudAndIsolation`, `TestCustomersCreateSearchAndIsolation`, `TestOrdersAreIsolatedPerDepot`, `TestLedgerNeverNegativeAndAdjustments`, `TestReminderQueueSendSkipAndConversion`, `TestCourierQueueAndLocation` |
| Kurir hanya menerima kolom untuk mengantar, tanpa omzet | ya | `order.CourierView` tanpa subtotal, diskon, dan ongkir; tes memeriksa `subtotal` tidak ada; dasbor dan laporan ditolak untuk kurir |

## 10.4 CSRF

| Butir | Status | Bukti |
|---|---|---|
| Cookie `SameSite` | ya | Lax untuk akses, Strict untuk penyegar |
| Metode pengubah data mewajibkan `Origin` sama dengan `WEB_ORIGIN` dan `Content-Type: application/json` | ya | `httpx.RequireOrigin`; tes `TestOriginCheckOnMutations` |

## 10.5 Validasi dan keluaran

| Butir | Status | Bukti |
|---|---|---|
| Semua masukan divalidasi, jumlah galon 1 sampai 50 | ya | tag `validate` di tiap struct permintaan, `order.BuildQuote` menolak di luar rentang; tes `TestBuildQuote`, `TestOwnerCreatesOrderWithServerPricing` |
| Dekode JSON menolak kolom tak dikenal, badan maksimal 64 KB | ya | `httpx.DecodeJSON` dengan `DisallowUnknownFields`, `fiber.Config.BodyLimit`; tes mengirim `total` dari klien dan ditolak 422 |
| Hanya SQL berparameter lewat sqlc | ya | tidak ada penyusunan string SQL; `sqlc diff` di CI |
| Tanpa `{@html}` di Svelte | ya | aturan ESLint `svelte/no-at-html-tags: error` |
| CSP mode hash: `default-src 'self'`, `connect-src 'self' <stream>`, `frame-ancestors 'none'`, `base-uri 'self'`, `form-action 'self'` | ya | `apps/web/svelte.config.js`; `style-src` memakai `unsafe-inline` karena templat SvelteKit, lihat `DECISIONS.md` |
| `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy` | ya | `apps/web/src/hooks.server.ts` |

## 10.6 Token pelanggan

| Butir | Status | Bukti |
|---|---|---|
| Token 16 byte acak base62 | ya | `idgen.RandomToken(16)`; tes `TestRandomTokenIsBase62AndUnique` |
| Disimpan SHA-256, link pribadi juga AES-256-GCM dengan `LINK_ENC_KEY` | ya | `customer/link.go`, `crypto/token.go`; tes `TestSealerRoundTrip`, `TestCustomerLinkAndRotation` |
| Halaman bertoken `no-store`, `noindex`, `no-referrer` | ya (API) | API mengirim `Cache-Control: no-store` untuk `/public/track` dan `/public/me`; header `X-Robots-Tag` dan `Referrer-Policy` dipasang di rute web `/t/[token]` dan `/p/[token]` |
| Halaman pribadi menyamarkan nomor dan tidak menampilkan alamat lengkap | ya | `phone.Mask`; tes `TestPersonalPageAndReorder` memastikan nomor penuh tidak bocor |
| Log tanpa token, jalur dicatat sebagai pola rute | ya | `httpx.LoggerMiddleware` mencatat `c.Route().Path` seperti `/api/v1/public/me/:token`; Caddy menyamarkan kueri `ticket` |
| Rotasi mematikan token lama | ya | tes `TestCustomerLinkAndRotation` dan `TestPersonalPageAndReorder` (404 setelah rotasi) |

## 10.7 Pembatasan laju

| Jalur | Status | Bukti |
|---|---|---|
| Semua 300 per menit per IP | ya | `globalRule` di `cmd/api/app.go` |
| Login 5 per menit per IP dan nomor | ya | `TestLoginRateLimitPerIPAndPhone` |
| Register 3 per jam per IP | ya | `registerRule` |
| Pesanan publik 5 per 10 menit per IP dan 3 per jam per nomor | ya | `TestPublicOrderFlow` |
| `/public/me` dan `/public/track` 60 per menit per IP | ya | `tokenReadRule` |
| Tiket stream 20 per menit per pengguna | ya | `ticketRule` |
| Jendela geser di memori, `Retry-After` pada 429 | ya | `ratelimit.Limiter`; tes `TestAllowSlidingWindow`, `Retry-After` diperiksa di tes login |

## 10.8 Rahasia dan rantai pasok

| Butir | Status | Bukti |
|---|---|---|
| Rahasia dari variabel lingkungan, `.env.example` tanpa nilai, `.env` di `.gitignore` | ya | `config.Load`; `.gitignore` |
| Menolak berjalan di produksi bila rahasia kosong atau pendek | ya | tes `TestLoadProductionRejectsMissingOrShortSecrets` |
| CI menjalankan `govulncheck` dan `npm audit --omit=dev` | ya | `.github/workflows/ci.yml`; hasil lokal nol kerentanan setelah toolchain Go 1.26.8 |
| Citra distroless statis, non-root, sistem berkas hanya baca | ya | `apps/api/Dockerfile` (`distroless/static-debian12:nonroot`), `read_only: true`, `cap_drop: ALL`, `no-new-privileges` di compose |

## 10.9 Audit

| Butir | Status | Bukti |
|---|---|---|
| Masuk berhasil dan gagal | ya | `audit.ActionLoginSuccess`, `ActionLoginFailed` |
| Perubahan harga | ya | `product.Handler.update` |
| Perubahan pengguna dan setel ulang kata sandi | ya | `user.Handler` |
| Penyesuaian buku galon | ya | `gallon.Handler.adjust` |
| Rotasi link | ya | `customer.Handler.rotateLink`; tes memeriksa baris audit |
| Pembatalan pesanan | ya | `order.Service.Cancel` |
| Ekspor laporan | ya | `report.Handler.exportCSV`; tes `TestDashboardAndReportMatchManualNumbers` |

## Aturan yang paling sering terlewat (MASTER_PROMPT)

1. `depot_id` dan `role` dari token: ya, `httpx.CurrentPrincipal` adalah satu-satunya sumber.
2. Setiap kueri menyaring `depot_id`: ya.
3. Harga dan total dihitung di server: ya, `order.BuildQuote`, klien yang mengirim `total` ditolak.
4. Token pelanggan dan penyegar tidak di log: ya, log hanya memuat pola rute dan ID.
5. Edge key wajib kecuali dua jalur: ya.
6. Tanpa `{@html}`: ya, dijaga ESLint.
