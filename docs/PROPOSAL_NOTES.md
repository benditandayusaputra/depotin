# Bahan Proposal GTNIC 2026

Disusun mengikuti urutan isi wajib proposal di `PRD.md` bagian 15. Batas 15 halaman, jadi tiap bagian ringkas dan siap salin. Format berkas: `GTNIC2026_WebDev_NamaTim.pdf`, A4, Times New Roman 12 pt, spasi 1,5, margin kiri 4 cm dan lainnya 3 cm, logo GTNIC 2026 di sampul.

## 1. Judul karya dan nama tim

**Depotin: Sistem Operasional Depot Air Isi Ulang dengan Prediksi Galon Habis**

Nama tim: (isi). Anggota: (isi, maksimal sesuai aturan panitia).

## 2. Latar belakang masalah

Air isi ulang menjadi sumber air minum sekitar 31,7% rumah tangga Indonesia (Survei Kesehatan Indonesia 2023, dikutip dalam [S4]), dan di DKI Jakarta saja tercatat 2.541 depot (Dinas Kesehatan Jakarta, dikutip dalam [S3]). Depot adalah usaha mikro: pemilik merangkap kasir, pelanggan tetap berada dalam radius beberapa ratus meter, pesanan masuk lewat WhatsApp atau telepon.

Lima masalah operasional yang terdokumentasi:

| Kode | Masalah | Bukti |
|---|---|---|
| M1 | Pengantaran tidak tepat waktu dan salah komunikasi | [S1] |
| M2 | Petugas berkeliling mencari galon kosong tanpa tahu siapa yang butuh | [S2] |
| M3 | Galon milik depot yang dipinjam pelanggan tidak tercatat | [S5] |
| M4 | Kupon loyalitas kertas mudah hilang | [S2] |
| M5 | Omzet harian dan pelanggan aktif tidak terpantau | [S6] |

Akar masalahnya: depot bekerja **reaktif**. Pesanan baru diketahui saat pelanggan sudah kehabisan air, sehingga permintaan menumpuk di jam yang sama dan pelanggan yang lupa memesan diam-diam pindah depot. Padahal konsumsi tiap rumah tangga relatif stabil dan bisa diperkirakan dari riwayat pesanannya.

Catatan: angka 31,7% dan 2.541 berasal dari sumber sekunder; cek ke sumber primer sebelum dicetak.

## 3. Tujuan, manfaat, dan solusi

| Tujuan | Tuntutan GTNIC yang dijawab | Ukuran |
|---|---|---|
| Semua pesanan, pengantaran, pembayaran, dan galon pinjaman tercatat di satu sistem | Mengelola proses operasional | selisih galon catatan dan kenyataan nol untuk transaksi lewat sistem |
| Depot menawarkan isi ulang sebelum pelanggan kehabisan, kurir mengantar dalam urutan efisien | Meningkatkan produktivitas | konversi pengingat menjadi pesanan dalam 48 jam tampil di laporan |
| Pelanggan memesan satu ketukan dan memantau statusnya | Layanan lebih baik | pesan ulang di bawah 15 detik, maksimal 2 ketukan |

Solusi: aplikasi web dengan tiga sisi. **Pemilik** mengelola pesanan, kurir, galon pinjaman, dan menerima daftar pelanggan yang perlu ditawari hari ini beserta pesan WhatsApp siap kirim. **Kurir** memakai halaman HP dengan tombol besar dan antrean terurut. **Pelanggan** memesan ulang lewat satu link pribadi tanpa akun dan tanpa aplikasi.

Satu kalimat nilai: depot tahu siapa yang akan kehabisan air sebelum pelanggannya sendiri sadar.

## 4. Target pengguna

| Pengguna | Profil | Kebutuhan |
|---|---|---|
| Pemilik depot | 30 sampai 55 tahun, akrab WhatsApp, HP atau laptop tua | tidak ada pesanan terlewat, tahu galon ada di mana, tahu omzet hari ini |
| Kurir | karyawan atau keluarga, HP Android kelas menengah ke bawah, satu tangan, sinyal tidak stabil | daftar antar jelas, tombol besar, tanpa mengetik |
| Pelanggan | rumah tangga, kos, warung, kantor kecil di sekitar depot | pesan cepat tanpa mengetik alamat lagi, tahu kapan galon datang |

Pasar: depot air isi ulang sebagai UMKM, dimulai dari satu depot uji coba dan dapat dipakai banyak depot sekaligus karena data terisolasi per depot sejak awal.

## 5. Arsitektur sistem, basis data, dan teknologi

Salin diagram arsitektur dan diagram relasi dari `docs/ARCHITECTURE.md` (render Mermaid ke gambar). Ringkasan arsitektur:

- Peramban hanya berbicara dengan satu asal (web di Vercel). Semua permintaan API diteruskan lewat proxy yang menambahkan kunci rahasia, sehingga API menolak lalu lintas langsung.
- API Go di VPS di belakang Caddy (TLS otomatis, HTTP/3), satu proses tanpa Redis.
- PostgreSQL terkelola di Neon.
- Pembaruan langsung lewat Server-Sent Events dengan tiket sekali pakai.

Tabel teknologi dan alasan (salin dari `SPEC.md` bagian 3):

| Lapisan | Pilihan | Alasan |
|---|---|---|
| Bahasa API | Go 1.26 | biner tunggal, memori kecil, konkurensi untuk SSE |
| Kerangka HTTP | Fiber v3 | alokasi rendah, middleware SSE bawaan |
| Akses data | pgx v5 + sqlc + goose | SQL ditulis tangan, diperiksa saat kompilasi, migrasi tersemat |
| Basis data | PostgreSQL 18 di Neon | terkelola, cadangan otomatis, gratis untuk skala lomba |
| Keamanan | Argon2id, JWT HS256 15 menit, token penyegar dirotasi, AES-256-GCM untuk token link | rekomendasi OWASP, token bisa dicabut |
| Frontend | SvelteKit 2, Svelte 5, TypeScript, Tailwind CSS v4 | bundel kecil untuk HP kelas bawah |
| Proxy dan kontainer | Caddy, Docker distroless | HTTPS otomatis, permukaan serangan kecil |
| Pengujian | go test, Vitest, Playwright | unit, integrasi, ujung ke ujung di viewport HP dan desktop |

Tabel entitas utama (lengkap di `SPEC.md` bagian 5): depots, users, sessions, products, customers, orders, order_items, order_events, gallon_ledger, reminders, order_counters, audit_logs. Semua tabel bisnis menyimpan `depot_id` dan setiap kueri menyaringnya.

## 6. Fitur utama dan keunikan

Fitur wajib yang selesai:

1. Pendaftaran depot, masuk pemilik dan kurir, penguncian akun, keluar dari semua perangkat.
2. Pengelolaan produk dan harga; harga lama di pesanan tidak berubah.
3. Pelanggan dengan pencarian saat mengetik, link pribadi terenkripsi, rotasi link.
4. Pesanan dari pemilik, dari halaman publik depot, dan dari link pribadi; harga dan total dihitung server; idempoten terhadap klik ganda.
5. Alur status dari menunggu sampai selesai dengan riwayat peristiwa dan pembaruan langsung di dasbor.
6. Halaman kurir: antrean terurut tetangga terdekat, Berangkat dan Selesai tanpa mengetik, antrean aksi luring saat sinyal hilang.
7. Buku galon pinjaman per pelanggan dengan saldo yang selalu sama dengan jumlah bukunya.
8. Prediksi galon habis per pelanggan yang bisa dijelaskan dan punya tingkat keyakinan.
9. Antrean pengingat harian dengan pesan WhatsApp siap kirim dan konversi terukur.
10. Pelacakan pesanan untuk pelanggan tanpa akun.
11. Dasbor hari ini dan laporan rentang tanggal dengan ekspor CSV.

Fitur tambahan: kartu loyalitas digital, urutan antar berdasarkan lokasi, daftar pelanggan berisiko pindah, mode demo dengan data contoh yang disetel ulang tiap malam.

Keunikan dibanding sistem serupa (tabel pembanding di `PRD.md` bagian 9):

1. **Proaktif, bukan reaktif.** Sistem memberi tahu siapa yang perlu ditawari hari ini dan mengukur berapa yang menjadi pesanan.
2. **Tanpa akun untuk pelanggan.** Kepemilikan nomor WhatsApp menjadi bukti identitas lewat link pribadi.
3. **Galon sebagai aset yang dilacak.** Setiap galon depot yang keluar dan kembali tercatat per pelanggan.

## 7. Batasan sistem

Salin dari `PRD.md` bagian 11:

1. Pembayaran hanya dicatat, belum diproses daring.
2. Pesan WhatsApp dikirim pemilik dengan satu ketukan, belum otomatis.
3. Prediksi akurat setelah minimal dua pesanan selesai; sebelumnya memakai angka bawaan depot.
4. Prediksi tidak memperhitungkan perubahan mendadak kecuali pengingatnya ditunda manual.
5. Urutan antar memakai jarak garis lurus.
6. Memerlukan internet; hanya aksi penyelesaian kurir yang ditahan saat sinyal hilang.
7. Satu akun pemilik mengelola satu depot.

## 8. Tangkapan layar yang perlu diambil

Ambil pada lebar 360 piksel (HP) kecuali disebut lain, tema terang, dengan data contoh "Depot Tirta Sejuk". Beri keterangan satu kalimat per gambar.

| No | Halaman | Yang harus terlihat |
|---|---|---|
| 1 | `/app/pengingat` | antrean pengingat dengan tombol Kirim WA, perkiraan habis, tingkat keyakinan |
| 2 | `/p/<token>` | tombol besar "Pesan 2 galon seperti biasa", saldo galon, stempel |
| 3 | `/app` (desktop 1280) | kartu angka hari ini dan indikator tersambung |
| 4 | `/app/pesanan` (desktop 1280) | papan per status |
| 5 | `/app/pesanan/baru` | formulir satu layar dengan pencarian pelanggan |
| 6 | `/kurir` | kartu antar dengan Peta, WhatsApp, Berangkat |
| 7 | `/kurir` lembar Selesai | penghitung galon kosong dan tombol cara bayar |
| 8 | `/app/pelanggan/<id>` | kalimat prediksi, keyakinan, saldo galon, link pribadi |
| 9 | `/app/galon` | total dipinjam dan galon mengendap |
| 10 | `/d/depot-tirta-sejuk` | halaman publik dengan harga dan formulir |
| 11 | `/t/<token>` | garis waktu status |
| 12 | `/app/laporan` (desktop 1280) | omzet, konversi pengingat, grafik harian |

## 9. Rencana pengembangan lanjutan

Salin dari `PRD.md` bagian 12: pengiriman pengingat otomatis lewat WhatsApp Business API, pembayaran QRIS dinamis, langganan terjadwal, prediksi musiman, banyak cabang dan peran kasir, catatan perawatan filter dan sertifikat higiene sanitasi, rute berbasis jalan dan pelacakan kurir.

## Lampiran: pustaka pihak ketiga

Salin tabel dari `docs/THIRD_PARTY.md`. Semua berlisensi MIT, BSD-3-Clause, atau Apache-2.0.

## Lampiran: tautan

- Repositori: https://github.com/benditandayusaputra/depotin
- Website: (isi alamat Vercel)
- Akun demo: pemilik 081200000001, kurir 081200000002, kata sandi `demo-depotin-2026`
