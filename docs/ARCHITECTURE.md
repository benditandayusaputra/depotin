# Arsitektur Depotin

## Gambaran

```mermaid
flowchart LR
  subgraph Klien
    B["Peramban pemilik, kurir, pelanggan"]
  end
  subgraph Vercel["Vercel (sin1)"]
    W["SvelteKit 2: halaman SSR dan CSR, proxy /api/v1/*"]
  end
  subgraph VPS["VPS"]
    C["Caddy: TLS, HTTP/3, zstd/gzip, log tanpa tiket"]
    A["API Go Fiber v3: auth, domain, SSE hub, penjadwal"]
  end
  subgraph Neon
    D[(PostgreSQL 18)]
  end
  B -->|"HTTPS satu asal, cookie HttpOnly"| W
  W -->|"HTTPS + X-Edge-Key + X-Client-IP"| C
  C --> A
  A -->|"pgx pool, MaxConns 10, MinConns 0"| D
  B -.->|"SSE langsung dengan tiket sekali pakai"| C
```

Peramban hanya berbicara dengan origin web. Proxy di `apps/web/src/routes/api/[...path]/+server.ts` menambahkan `X-Edge-Key`, dan API menolak permintaan tanpa kunci itu kecuali `/healthz` dan `GET /api/v1/stream`.

## Lapisan di API

```mermaid
flowchart TB
  H["handler: baca permintaan, validasi, respons"] --> S["service: aturan bisnis, transaksi"]
  S --> R["sqlc: kueri SQL bertipe, selalu menyaring depot_id"]
  S --> P["fungsi murni: prediction, order.state, order.pricing, order.routing, gallon"]
  R --> DB[(PostgreSQL)]
```

| Paket | Tanggung jawab |
|---|---|
| `internal/platform/httpx` | respons baku `{data}` dan `{error}`, dekode JSON ketat, validasi, paginasi kursor, middleware request ID, pemulihan, log, edge key, Origin, batas laju |
| `internal/auth` | Argon2id, JWT HS256 15 menit, token penyegar 30 hari dengan rotasi dan deteksi pemakaian ulang, penguncian akun |
| `internal/order` | pembuatan pesanan dengan salinan harga, kode `DP-YYMMDD-NNN`, idempotensi, mesin status, transaksi penyelesaian, urutan antar |
| `internal/prediction` | rata-rata bergerak eksponensial atas jarak antar pengantaran, pembuangan jeda tidak wajar, tingkat keyakinan |
| `internal/reminder` | antrean harian idempoten, kirim WA, lewati, atribusi konversi 48 jam |
| `internal/gallon` | buku galon pinjaman, penyesuaian manual, galon mengendap |
| `internal/public` | halaman depot ber-cache dan ETag, pesanan publik, pelacakan bertoken, halaman pribadi |
| `internal/stream` | hub SSE dalam memori, tiket 30 detik sekali pakai, batas 3 koneksi per pengguna |
| `internal/report` | dasbor hari ini, ringkasan rentang, ekspor CSV, log audit |
| `internal/scheduler` | 06.00 WIB susun antrean pengingat, setel ulang depot demo, penjaga hidup Neon |
| `internal/seed` | depot contoh deterministik dengan riwayat 8 minggu lewat service asli |

## Alur penyelesaian pesanan

```mermaid
sequenceDiagram
  participant K as Kurir (HP)
  participant W as Web proxy
  participant A as API
  participant DB as PostgreSQL
  K->>W: POST /orders/{id}/deliver + Idempotency-Key
  W->>A: + X-Edge-Key, X-Client-IP
  A->>DB: BEGIN, SELECT order FOR UPDATE, SELECT customer FOR UPDATE
  A->>DB: UPDATE orders SET status='delivered' WHERE status=$from
  A->>DB: INSERT gallon_ledger, UPDATE customers (saldo, stempel, prediksi)
  A->>DB: INSERT order_events (idempotency_key), COMMIT
  A-->>K: 200 pesanan + saldo galon + prediksi baru
  A-->>A: hub.Publish(order.updated) ke pemilik dan kurir
```

Permintaan ulang dengan `Idempotency-Key` yang sama menemukan peristiwa `delivered` dan membalas keadaan terkini tanpa menulis buku galon lagi. Dua permintaan bersamaan dengan kunci berbeda: yang kedua menemukan status sudah `delivered` dan dibalas `409 invalid_transition`.

## Diagram relasi basis data

```mermaid
erDiagram
  depots ||--o{ users : punya
  depots ||--o{ products : punya
  depots ||--o{ customers : punya
  depots ||--o{ orders : punya
  depots ||--o{ audit_logs : mencatat
  depots ||--o{ order_counters : menghitung
  users ||--o{ sessions : punya
  customers ||--o{ orders : membuat
  customers ||--o{ gallon_ledger : tercatat
  customers ||--o{ reminders : menerima
  orders ||--|{ order_items : berisi
  orders ||--o{ order_events : riwayat
  orders ||--o{ gallon_ledger : memicu
  reminders |o--o| orders : menghasilkan
  users ||--o{ orders : mengantar

  depots {
    uuid id PK
    text slug UK
    text timezone
    int loyalty_every
    numeric default_days_per_gallon
    boolean is_demo
  }
  users {
    uuid id PK
    uuid depot_id FK
    text role
    text phone UK
    text password_hash
    timestamptz locked_until
  }
  sessions {
    uuid id PK
    uuid user_id FK
    uuid family_id
    bytea token_hash UK
    timestamptz rotated_at
    timestamptz revoked_at
  }
  customers {
    uuid id PK
    uuid depot_id FK
    text phone
    bytea token_hash UK
    bytea token_enc
    int loan_balance
    int stamp_count
    numeric days_per_gallon
    text prediction_confidence
    timestamptz predicted_empty_at
  }
  orders {
    uuid id PK
    uuid depot_id FK
    uuid customer_id FK
    text code
    text source
    text status
    date scheduled_date
    bigint total
    uuid courier_id FK
    uuid reminder_id FK
    bytea track_token_hash UK
    text idempotency_key
  }
  order_items {
    uuid id PK
    uuid order_id FK
    uuid product_id FK
    bigint unit_price
    int qty
  }
  order_events {
    uuid id PK
    uuid order_id FK
    text type
    text actor_type
    text idempotency_key
  }
  gallon_ledger {
    uuid id PK
    uuid customer_id FK
    uuid order_id FK
    text kind
    int delta
    int balance_after
  }
  reminders {
    uuid id PK
    uuid customer_id FK
    date due_date
    text status
    timestamptz sent_at
    uuid order_id FK
  }
```

## Indeks dan kueri utama

| Kueri | Indeks yang dipakai (diperiksa dengan `EXPLAIN` pada data contoh) |
|---|---|
| papan pesanan per status dan tanggal | `orders_board_idx (depot_id, status, scheduled_date)` |
| antrean kurir | `orders_courier_queue_idx (courier_id, status) WHERE status IN ('confirmed','on_delivery')` |
| antrean pengingat | `customers_reminder_idx (depot_id, predicted_empty_at) WHERE is_active AND is_verified`, `orders_customer_active_idx`, `reminders_depot_status_due_idx` |
| pencarian pelanggan saat mengetik | indeks GIN trigram pada `name`, `phone`, `address` |
| riwayat dan prediksi | `orders_customer_delivered_idx (customer_id, delivered_at DESC) WHERE status = 'delivered'` |

Pada data contoh (40 pelanggan, sekitar 190 pesanan) perencana memilih pemindaian berurutan karena tabel muat dalam beberapa halaman. Dengan `SET enable_seqscan = off` semua kueri di atas memakai indeks yang dirancang, sehingga pada skala ribuan pesanan perencana akan beralih sendiri.

## Pembaruan langsung

```mermaid
sequenceDiagram
  participant B as Peramban pemilik
  participant W as Web proxy
  participant A as API
  B->>W: POST /api/v1/stream/tickets (cookie)
  W->>A: + X-Edge-Key
  A-->>B: {ticket} berlaku 30 detik
  B->>A: GET /api/v1/stream?ticket=... (EventSource, langsung ke API)
  A-->>B: event ready, lalu order.created / order.updated / reminder.queued
  Note over B,A: detak jantung tiap 20 detik, klien menyambung ulang 1, 2, 5, 10, 30 detik, lalu polling tiap 10 detik
```
