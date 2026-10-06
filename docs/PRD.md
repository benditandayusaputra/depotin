# Depotin: Product Requirements Document

| | |
|---|---|
| Versi | 1.0 |
| Tanggal | 6 Oktober 2026 |
| Lomba | GTNIC 2026, kategori Web Development |
| Subtema | Accelerating Digital Transformation for Business and SMEs |
| Dokumen terkait | `SPEC.md` (spesifikasi teknis), `MASTER_PROMPT.md` (prompt Claude Code) |

## 1. Ringkasan

Depotin adalah sistem web untuk depot air minum isi ulang. Pemilik depot mengelola pesanan, kurir, dan galon pinjaman dari satu dasbor. Pelanggan memesan ulang lewat satu link tanpa membuat akun dan tanpa memasang aplikasi.

Fitur pembedanya adalah **prediksi galon habis**: sistem menghitung pola konsumsi tiap pelanggan dari riwayat pesanannya, lalu menyiapkan daftar pelanggan yang perlu ditawari hari ini, lengkap dengan pesan WhatsApp siap kirim. Depot tidak lagi menunggu pelanggan kehabisan air, dan kurir tidak lagi berkeliling mencari galon kosong.

Satu kalimat nilai: **depot tahu siapa yang akan kehabisan air sebelum pelanggannya sendiri sadar.**

## 2. Latar belakang dan masalah

### 2.1 Konteks pasar

- Air isi ulang adalah sumber air minum sehari-hari bagi sekitar 31,7% rumah tangga Indonesia (Survei Kesehatan Indonesia 2023, dikutip dalam [S4]).
- Di DKI Jakarta saja tercatat 2.541 depot air minum isi ulang (data Inspeksi Kesehatan Lingkungan Puskesmas Dinas Kesehatan Jakarta, dikutip dalam [S3]).
- Depot air adalah usaha mikro yang khas: pemilik merangkap kasir, pelanggan tetap berada dalam radius beberapa ratus meter, dan sebagian besar pesanan masuk lewat WhatsApp atau telepon.

Catatan: kedua angka di atas berasal dari sumber sekunder. Sebelum masuk proposal, cek ulang ke sumber primernya.

### 2.2 Masalah yang terdokumentasi

| Kode | Masalah | Bukti |
|---|---|---|
| M1 | Pengantaran tidak tepat waktu dan sering terjadi salah komunikasi antara depot dan pelanggan | Studi Telkom University menyebut keluhan pengantaran galon yang tidak tepat waktu, stok yang tidak menentu, dan miskomunikasi pemilik depot dengan pelanggan [S1] |
| M2 | Petugas berkeliling mencari galon kosong karena tidak tahu siapa yang sudah butuh isi ulang | Disebut sebagai pekerjaan yang ingin dihindari lewat pemesanan daring [S2] |
| M3 | Galon milik depot yang dipinjam pelanggan tidak tercatat rapi | Aplikasi pesaing menjadikan "lupa melacak galon yang dipinjam pelanggan" sebagai masalah utama yang dijualnya [S5] |
| M4 | Kupon loyalitas kertas (misalnya isi 10 gratis 1) mudah hilang atau sobek | Disebut dalam [S2], dan promo "isi 10 kali gratis 1 kali" umum dipakai depot |
| M5 | Pencatatan transaksi manual membuat omzet harian dan pelanggan aktif tidak terpantau | Disebut dalam panduan aplikasi kasir depot [S6] |

### 2.3 Akar masalah

Depot bekerja secara **reaktif**. Pesanan baru diketahui ketika pelanggan sudah kehabisan air, sehingga permintaan datang mendadak dan menumpuk di jam yang sama, sementara pelanggan yang lupa memesan diam-diam pindah ke depot lain. Padahal konsumsi air tiap rumah tangga relatif stabil dan bisa diperkirakan dari riwayat pesanannya.

### 2.4 Mengapa solusinya harus web

1. Masalahnya melibatkan tiga pihak (pemilik, kurir, pelanggan), dan pelanggan tidak akan memasang aplikasi demi satu depot. Link yang dibuka di peramban HP adalah cara paling ringan untuk melibatkan mereka.
2. Kurir bekerja dari HP sendiri di jalan. Halaman web ramah seluler bisa dipakai tanpa instalasi.
3. Pemilik perlu melihat pesanan masuk secara langsung dari perangkat apa pun.

## 3. Tujuan dan ukuran keberhasilan

### 3.1 Tujuan produk

| Kode | Tujuan | Tuntutan tantangan GTNIC yang dijawab |
|---|---|---|
| T1 | Semua pesanan, pengantaran, pembayaran, dan galon pinjaman tercatat di satu sistem | Mengelola proses operasional |
| T2 | Depot menawarkan isi ulang sebelum pelanggan kehabisan, dan kurir mengantar dalam urutan yang efisien | Meningkatkan produktivitas |
| T3 | Pelanggan memesan dalam satu ketukan dan bisa memantau status pesanannya | Layanan lebih baik bagi pelanggan |

### 3.2 Ukuran keberhasilan

Untuk demo lomba, angka di bawah diukur pada data contoh dan pada uji coba dengan minimal satu depot nyata.

| Ukuran | Target |
|---|---|
| Waktu pelanggan lama membuat pesanan ulang dari link pribadi | Di bawah 15 detik, maksimal 2 ketukan |
| Waktu pemilik mencatat pesanan telepon | Di bawah 20 detik |
| Waktu kurir menyelesaikan satu pengantaran di aplikasi | Di bawah 15 detik |
| Konversi pengingat menjadi pesanan dalam 48 jam | Terukur dan tampil di laporan (target awal 30%) |
| Selisih galon pinjaman antara catatan dan kenyataan | Nol untuk transaksi yang lewat sistem |
| Pesanan baru tampil di dasbor pemilik | Di bawah 2 detik sejak dibuat |

## 4. Pengguna

### 4.1 Pemilik depot (peran `owner`)

Ibu atau bapak pemilik depot, usia 30 sampai 55 tahun, terbiasa dengan WhatsApp tetapi tidak dengan perangkat lunak akuntansi. Bekerja dari HP atau laptop tua di meja depot. Yang dicari: tidak ada pesanan terlewat, tahu galon ada di mana, tahu omzet hari ini.

### 4.2 Kurir (peran `courier`)

Karyawan atau anggota keluarga yang mengantar dengan motor. Memakai HP Android kelas menengah ke bawah, sering dengan satu tangan, sinyal tidak selalu bagus. Yang dicari: daftar antar yang jelas, tombol besar, tidak perlu mengetik.

### 4.3 Pelanggan (tanpa akun)

Rumah tangga, kos, warung, atau kantor kecil di sekitar depot. Yang dicari: pesan cepat tanpa mengetik alamat lagi, tahu kapan galon datang.

## 5. Lingkup

### 5.1 Wajib ada (Must)

| Kode | Fitur |
|---|---|
| F-01 | Pendaftaran depot dan login pemilik serta kurir |
| F-02 | Pengelolaan produk dan harga |
| F-03 | Pengelolaan pelanggan, termasuk link pribadi tiap pelanggan |
| F-04 | Pesanan: dibuat oleh pemilik, oleh pelanggan lewat halaman depot, dan oleh pelanggan lewat link pribadi |
| F-05 | Alur status pesanan dari masuk sampai selesai, dengan pembaruan langsung di dasbor |
| F-06 | Halaman kurir: antrean antar dan penyelesaian pengantaran |
| F-07 | Buku galon pinjaman per pelanggan |
| F-08 | Prediksi galon habis per pelanggan |
| F-09 | Antrean pengingat dengan pesan WhatsApp siap kirim |
| F-10 | Halaman pelacakan pesanan untuk pelanggan |
| F-11 | Dasbor hari ini dan laporan ringkas |

### 5.2 Sebaiknya ada (Should)

| Kode | Fitur |
|---|---|
| F-12 | Kartu loyalitas digital (isi N gratis 1) |
| F-13 | Urutan antar otomatis berdasarkan lokasi tersimpan |
| F-14 | Daftar pelanggan berisiko pindah (lama tidak memesan) |
| F-15 | Ekspor laporan ke CSV |
| F-16 | Mode demo dengan data contoh yang tersetel ulang otomatis |

### 5.3 Boleh ada bila waktu cukup (Could)

| Kode | Fitur |
|---|---|
| F-17 | Jual dadakan oleh kurir di jalan |
| F-18 | Impor pelanggan dari CSV |
| F-19 | Tampilan bisa dipasang ke layar utama HP (manifest PWA) |

### 5.4 Tidak dikerjakan di versi ini (Won't)

- Pembayaran daring lewat payment gateway. Pembayaran hanya dicatat.
- Pengiriman WhatsApp otomatis lewat WhatsApp Business API.
- Pelacakan posisi kurir secara langsung di peta.
- Aplikasi native Android atau iOS.
- Manajemen banyak cabang dalam satu akun.
- Akuntansi lengkap, stok bahan baku, dan penggajian.

## 6. Alur utama

### 6.1 Alur A: pelanggan lama memesan ulang dari pengingat

1. Pagi hari, sistem menandai Bu Rina akan kehabisan air besok.
2. Pemilik membuka halaman Pengingat, menekan **Kirim WA** di baris Bu Rina. WhatsApp terbuka dengan pesan yang sudah terisi dan link pribadi Bu Rina.
3. Bu Rina membuka link, melihat tombol **Pesan 2 galon seperti biasa**, lalu menekannya.
4. Pesanan langsung muncul di dasbor pemilik dengan bunyi notifikasi, tertandai berasal dari pengingat.
5. Pemilik menugaskan kurir. Kurir menekan **Berangkat**, lalu di lokasi menekan **Selesai**, mengisi jumlah galon kosong yang diterima dan cara bayar.
6. Buku galon, stempel loyalitas, dan prediksi Bu Rina diperbarui otomatis. Bu Rina melihat status selesai di halamannya.

### 6.2 Alur B: pelanggan baru memesan dari halaman depot

1. Calon pelanggan membuka halaman publik depot dari link di bio atau spanduk.
2. Ia mengisi nama, nomor WhatsApp, alamat, dan jumlah galon, lalu mengirim.
3. Ia mendapat halaman pelacakan untuk pesanan itu.
4. Pemilik melihat pesanan berstatus **Menunggu konfirmasi**, memeriksa alamat, lalu mengonfirmasi.
5. Setelah pesanan pertama selesai, pemilik mengirim link pribadi ke nomor WhatsApp pelanggan itu. Sejak saat itu ia memakai Alur A.

### 6.3 Alur C: pesanan lewat telepon atau datang langsung

1. Pemilik menekan **Pesanan baru**, mengetik beberapa huruf nama atau nomor pelanggan, memilih jumlah, lalu menyimpan.
2. Untuk ambil sendiri, pemilik langsung menandai selesai dan lunas.

## 7. Kebutuhan fungsional

Setiap kebutuhan ditulis sebagai cerita pengguna dengan kriteria terima (KT) yang bisa diuji.

### F-01 Akun dan akses

- Sebagai calon pengguna, saya bisa mendaftarkan depot dengan nama depot, nama saya, nomor HP, dan kata sandi.
  - KT1: nomor HP dinormalkan ke format 62 dan harus unik.
  - KT2: kata sandi minimal 10 karakter.
  - KT3: setelah daftar, saya langsung masuk dan diarahkan ke penyiapan awal (harga isi ulang, jam buka).
- Sebagai pemilik, saya bisa menambah akun kurir, menonaktifkannya, dan menyetel ulang kata sandinya.
- Sebagai pengguna, saya tetap masuk selama 30 hari di perangkat yang sama, dan bisa keluar dari semua perangkat.
  - KT4: 5 kali gagal masuk berturut-turut mengunci akun 15 menit.
  - KT5: kurir hanya bisa membuka halaman kurir. Semua akses lain ditolak di sisi server.

### F-02 Produk dan harga

- Sebagai pemilik, saya bisa mengatur produk dengan nama, harga, dan jenis (`isi ulang`, `galon baru`, `lainnya`).
  - KT1: depot baru otomatis punya produk "Isi ulang galon".
  - KT2: perubahan harga tidak mengubah pesanan lama.

### F-03 Pelanggan

- Sebagai pemilik, saya bisa menambah, mencari, dan mengubah pelanggan (nama, nomor WhatsApp, alamat, patokan, area, jumlah biasa).
  - KT1: pencarian berdasarkan nama, nomor, atau alamat menampilkan hasil saat mengetik.
  - KT2: nomor WhatsApp unik per depot.
- Sebagai pemilik, saya bisa menyalin atau mengirim link pribadi pelanggan lewat WhatsApp, dan membuat link baru bila link lama bocor.
  - KT3: link lama langsung tidak berlaku setelah link baru dibuat.
- Sebagai pemilik, di halaman pelanggan saya melihat riwayat pesanan, saldo galon pinjaman, stempel loyalitas, dan perkiraan tanggal habis beserta tingkat keyakinannya.

### F-04 Pembuatan pesanan

- Sebagai pemilik, saya bisa membuat pesanan untuk pelanggan dalam satu layar.
  - KT1: jumlah terisi otomatis dari jumlah biasa pelanggan.
  - KT2: total dihitung di server. Angka dari peramban tidak dipercaya.
- Sebagai pelanggan lama, saya bisa memesan ulang dari link pribadi dengan satu tombol, dengan pilihan mengubah jumlah, hari antar (hari ini atau besok), dan catatan.
- Sebagai calon pelanggan, saya bisa memesan dari halaman publik depot.
  - KT3: pesanan dari halaman publik berstatus menunggu konfirmasi.
  - KT4: pengiriman ganda karena tombol tertekan dua kali tidak membuat dua pesanan.
  - KT5: depot bisa menutup penerimaan pesanan sementara, dan halaman publik menampilkan statusnya.

### F-05 Alur pesanan dan pembaruan langsung

- Status pesanan: `Menunggu konfirmasi`, `Dikonfirmasi`, `Sedang diantar`, `Selesai`, `Dibatalkan`.
  - KT1: perpindahan status yang tidak sah ditolak server.
  - KT2: setiap perpindahan tercatat dengan waktu dan pelakunya.
- Sebagai pemilik, saya melihat pesanan baru muncul sendiri di dasbor tanpa memuat ulang, dengan bunyi dan tanda visual.
  - KT3: bila koneksi langsung putus, dasbor beralih ke pemeriksaan berkala dan menampilkan tanda "menyambung ulang".
- Sebagai pemilik, saya bisa mengonfirmasi, menugaskan kurir, membatalkan dengan alasan, dan menandai lunas.

### F-06 Halaman kurir

- Sebagai kurir, saya melihat antrean antar hari ini dengan nama, alamat, patokan, jumlah galon, dan total yang harus ditagih.
- Sebagai kurir, saya menekan **Berangkat** lalu **Selesai**, mengisi galon kosong yang diterima dan cara bayar (tunai, transfer, belum bayar).
  - KT1: seluruh penyelesaian bisa dilakukan dengan ibu jari, tanpa mengetik, dengan tombol minimal 48 piksel.
  - KT2: bila sinyal hilang saat menekan Selesai, aplikasi menyimpan aksi itu dan mengirim ulang saat sinyal kembali, tanpa membuat data ganda.
- Sebagai kurir, saya bisa menyimpan lokasi pelanggan saat berada di rumahnya, dan membuka alamat di aplikasi peta.

### F-07 Buku galon pinjaman

- Setiap pengantaran isi ulang otomatis mencatat galon penuh yang diserahkan dan galon kosong yang diterima.
  - KT1: saldo galon pinjaman pelanggan sama dengan jumlah seluruh catatan bukunya.
  - KT2: saldo tidak pernah negatif.
- Sebagai pemilik, saya bisa mencatat penyesuaian manual (galon hilang, dikembalikan di luar pesanan) dengan catatan wajib.
- Sebagai pemilik, saya melihat total galon yang sedang dipinjam dan daftar "galon mengendap" (pelanggan yang memegang galon depot tetapi tidak memesan lebih dari 30 hari).

### F-08 Prediksi galon habis

- Sistem menghitung rata-rata hari per galon tiap pelanggan dari riwayat pengantaran, lalu memperkirakan tanggal habis.
  - KT1: perhitungan bisa dijelaskan. Halaman pelanggan menampilkan kalimat seperti "Biasanya 1 galon habis dalam 3,5 hari, dihitung dari 6 pesanan terakhir".
  - KT2: tingkat keyakinan ditampilkan (rendah, sedang, tinggi).
  - KT3: pelanggan yang baru punya satu pengantaran memakai angka bawaan depot dan ditandai keyakinan rendah. Pelanggan tanpa pengantaran belum punya prediksi.
  - KT4: jeda yang tidak wajar (misalnya pelanggan mudik) tidak merusak perkiraan.

### F-09 Pengingat

- Setiap pagi sistem menyusun antrean pelanggan yang perlu ditawari hari ini.
  - KT1: pelanggan dengan pesanan aktif, yang baru diingatkan, atau yang ditunda tidak masuk antrean.
- Sebagai pemilik, saya menekan **Kirim WA** untuk membuka WhatsApp dengan pesan terisi, lalu baris itu tertandai terkirim.
- Sebagai pemilik, saya bisa melewati atau menunda pengingat seorang pelanggan selama beberapa hari.
- Pesanan yang dibuat dari link di pesan pengingat, dalam 48 jam sejak pengingat dikirim, tercatat sebagai hasil pengingat.
  - KT2: laporan menampilkan jumlah pengingat terkirim dan persentase yang menjadi pesanan.

### F-10 Pelacakan untuk pelanggan

- Sebagai pelanggan, saya melihat status pesanan, perkiraan hari antar, total, dan nomor WhatsApp depot.
  - KT1: halaman memperbarui status sendiri selama pesanan belum selesai.
  - KT2: halaman tidak menampilkan data pelanggan lain dan tidak terindeks mesin pencari.
- Sebagai pelanggan lama, di link pribadi saya juga melihat riwayat singkat, saldo galon pinjaman, dan stempel loyalitas.

### F-11 Dasbor dan laporan

- Dasbor hari ini menampilkan pesanan per status, omzet hari ini, galon terjual, perkiraan pesanan hari ini, jumlah pengingat menunggu, dan total galon dipinjam.
- Laporan rentang tanggal menampilkan omzet, galon terjual, pesanan per sumber, konversi pengingat, pelanggan baru, dan pelanggan aktif.

### F-12 Loyalitas (Should)

- Pemilik menyetel "isi N gratis 1". Stempel bertambah otomatis tiap galon isi ulang selesai diantar.
  - KT1: saat stempel cukup, pesanan berikutnya otomatis mendapat potongan satu galon dan stempel berkurang.
  - KT2: pelanggan melihat kemajuan stempelnya di link pribadi.

### F-13 Urutan antar (Should)

- Antrean kurir diurutkan dari titik terdekat ke titik berikutnya untuk pelanggan yang lokasinya tersimpan. Pelanggan tanpa lokasi dikelompokkan per area.

### F-14 Pelanggan berisiko (Should)

- Daftar pelanggan yang sudah melewati dua kali perkiraan habisnya tanpa memesan.

### F-16 Mode demo (Should)

- Depot contoh berisi sekitar 40 pelanggan dengan riwayat 8 minggu, sehingga prediksi dan pengingat langsung menampilkan hasil yang berarti bagi juri.
- Data depot contoh tersetel ulang otomatis tiap malam.

## 8. Kebutuhan non-fungsional

| Aspek | Kebutuhan |
|---|---|
| Performa | Halaman publik depot dan pelacakan tampil di bawah 2,5 detik pada jaringan 4G lambat. API merespons di bawah 150 ms (p95) untuk daftar dan di bawah 300 ms untuk penyimpanan pesanan, di luar waktu bangun database |
| Keamanan | Lihat bagian Keamanan di `SPEC.md`. Ringkasnya: kata sandi ber-hash Argon2id, sesi di cookie HttpOnly, akses berbasis peran dan per depot, pembatasan laju, link pelanggan bertoken acak yang disimpan sebagai hash dan salinan terenkripsi |
| Responsif | Berfungsi baik di lebar 360 piksel sampai desktop, di Chrome, Firefox, Safari, dan Edge |
| Aksesibilitas | Kontras teks minimal WCAG AA, semua kontrol bisa dipakai dengan papan ketik, label formulir lengkap |
| Keandalan | Tidak ada data ganda saat permintaan diulang. Kegagalan jaringan menampilkan pesan yang jelas dan tombol coba lagi |
| Bahasa | Seluruh antarmuka berbahasa Indonesia, mata uang Rupiah, zona waktu Asia/Jakarta |
| Privasi | Nomor dan alamat pelanggan hanya terlihat oleh depotnya. Halaman publik menyamarkan data pribadi |

## 9. Keunikan dibanding sistem serupa

| Pembanding | Yang dilakukan | Yang tidak ada |
|---|---|---|
| Aplikasi kasir depot luring (contoh: Buku Galon [S5]) | Kasir, stok galon, catatan galon pinjaman, di satu HP tanpa internet | Sisi pelanggan, pesanan daring, kurir terpisah, prediksi |
| Aplikasi kasir umum [S6] | Pencatatan penjualan dan laporan | Konsep galon pinjaman, pengantaran, pengingat |
| Proyek akademik pemesanan galon [S1][S2] | Pelanggan memesan lewat aplikasi atau web | Prediksi konsumsi, buku galon, pelanggan harus mendaftar atau memasang aplikasi |
| **Depotin** | Operasional depot, kurir, dan pelanggan dalam satu alur, ditambah prediksi galon habis dan pengingat terukur | Lihat Batasan sistem |

Tiga pembeda utama:

1. **Proaktif, bukan reaktif.** Sistem memberi tahu depot siapa yang perlu ditawari, dan mengukur berapa pengingat yang menjadi pesanan.
2. **Tanpa akun untuk pelanggan.** Link pribadi dikirim ke nomor WhatsApp pelanggan, sehingga kepemilikan nomor menjadi bukti identitas.
3. **Galon sebagai aset yang dilacak.** Setiap galon milik depot yang keluar dan kembali tercatat per pelanggan.

## 10. Pemetaan ke rubrik GTNIC

| Kriteria | Bobot | Jawaban Depotin |
|---|---|---|
| Relevansi masalah dan inovasi | 20% | Masalah M1 sampai M5 berbukti, prediksi galon habis sebagai inovasi, pengukuran konversi pengingat sebagai bukti dampak |
| Fungsionalitas dan kualitas implementasi | 25% | Tiga alur utama berjalan utuh dari ujung ke ujung, dengan data contoh yang realistis |
| Kualitas teknis dan arsitektur | 20% | API Go berlapis, kueri bertipe aman, keamanan berlapis, tes otomatis, dokumentasi di repositori |
| UX dan desain antarmuka | 15% | Satu ketukan untuk pesan ulang, layar kurir tanpa mengetik, umpan balik jelas di setiap aksi |
| Dampak bisnis dan skalabilitas | 10% | Multi depot sejak awal, ukuran dampak tampil di laporan, jalur pengembangan jelas |
| Dokumentasi, demo, presentasi | 10% | README, diagram arsitektur dan basis data, akun demo, catatan siap salin untuk proposal |

## 11. Batasan sistem

Bagian ini bisa disalin ke subbab "Batasan sistem aplikasi" di proposal.

1. Pembayaran hanya dicatat, belum diproses secara daring.
2. Pesan WhatsApp dikirim oleh pemilik dengan satu ketukan, belum terkirim otomatis.
3. Prediksi membutuhkan minimal dua pesanan selesai agar akurat. Sebelum itu sistem memakai angka bawaan depot.
4. Prediksi tidak memperhitungkan perubahan mendadak seperti tamu menginap atau pelanggan bepergian, kecuali pengingatnya ditunda manual.
5. Urutan antar memakai jarak garis lurus, bukan rute jalan sebenarnya.
6. Sistem memerlukan koneksi internet. Hanya aksi penyelesaian kurir yang ditahan sementara saat sinyal hilang.
7. Satu akun pemilik mengelola satu depot.

## 12. Rencana pengembangan lanjutan

1. Pengiriman pengingat otomatis lewat WhatsApp Business API.
2. Pembayaran QRIS dinamis dengan konfirmasi otomatis.
3. Langganan terjadwal (antar tiap Senin dan Kamis tanpa perlu memesan).
4. Prediksi yang memperhitungkan musim dan hari libur.
5. Banyak cabang dan peran kasir.
6. Catatan perawatan filter dan sertifikat laik higiene sanitasi, karena kepatuhan sanitasi adalah masalah nyata di industri ini [S3].
7. Rute berbasis jalan dan pelacakan kurir.

## 13. Risiko dan mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Pemilik depot tidak mau mengirim pengingat satu per satu | Fitur pembeda tidak terpakai | Satu ketukan per pelanggan, pesan sudah terisi. Uji saat wawancara. Jalur pengembangan: kirim otomatis |
| Pelanggan tidak membuka link | Konversi rendah | Link pendek, halaman ringan, tombol tunggal. Ukur konversi sejak awal |
| Prediksi meleset pada pelanggan dengan pola tidak teratur | Pengingat mengganggu | Tampilkan tingkat keyakinan, sediakan tunda dan lewati, saring jeda tidak wajar |
| Database gratis tertidur saat juri membuka | Permintaan pertama lambat | Halaman pertama tidak bergantung pada database, penjaga hidup diaktifkan hanya pada jendela penjurian |
| Waktu pengerjaan 23 hari sambil bekerja penuh waktu | Fitur tidak selesai | Urutan Must lebih dulu, setiap tahap menghasilkan aplikasi yang bisa didemokan |
| Tema pemesanan galon sudah sering jadi proyek mahasiswa | Dianggap biasa oleh juri | Pembukaan demo langsung menunjukkan prediksi dan pengingat, bukan formulir pesan |
| Juri merusak data demo | Demo berikutnya kacau | Setel ulang data contoh otomatis tiap malam |

## 14. Jadwal

| Periode | Kegiatan |
|---|---|
| 6 sampai 16 Oktober | Wawancara 3 sampai 5 pemilik depot dan beberapa pelanggan. Tanyakan biaya dan aturan tim ke panitia. Daftar paling lambat 16 Oktober |
| 18 Oktober | Technical Meeting. Tanyakan boleh tidaknya mulai menulis kode sebelum 19 Oktober |
| 19 sampai 25 Oktober | Fondasi, akun, produk, pelanggan, pesanan, alur status |
| 26 Oktober sampai 1 November | Halaman kurir, buku galon, prediksi, pengingat, pembaruan langsung, halaman pelanggan |
| 2 sampai 8 November | Laporan, loyalitas, urutan antar, data demo, tes, deploy, uji di HP nyata |
| 9 sampai 10 November | Proposal, video demo 3 sampai 5 menit, surat pernyataan, pengumpulan sebelum 23.59 WIB |
| 12 November | Pengumuman finalis |
| 14 November | Grand final lewat Zoom |

## 15. Syarat pengumpulan lomba

Diringkas dari guidebook GTNIC 2026. Semua dikirim lewat Google Form resmi paling lambat 10 November 2026 pukul 23.59 WIB.

| Yang dikumpulkan | Ketentuan |
|---|---|
| Tautan website live | Di-hosting di platform publik dan tetap bisa diakses juri sampai kompetisi berakhir |
| Tautan repositori kode | Akses publik |
| Proposal | PDF, maksimal 15 halaman, A4, Times New Roman 12 pt, jarak baris 1,5, margin kiri 4 cm dan lainnya 3 cm. Nama file `GTNIC2026_WebDev_NamaTim.pdf`. Logo GTNIC 2026 wajib ada di sampul |
| Video demo | 3 sampai 5 menit, diunggah ke YouTube atau Google Drive dengan akses publik, antara lain memuat perkenalan singkat anggota tim serta penjelasan masalah dan solusi |
| Surat pernyataan keaslian karya | Formatnya belum disebut di guidebook |

Isi wajib proposal, berurutan:

1. Judul karya dan nama tim
2. Latar belakang masalah operasional atau bisnis UMKM yang diangkat
3. Tujuan, manfaat, dan solusi teknologi yang ditawarkan
4. Target pengguna atau pasar aplikasi
5. Arsitektur sistem, diagram basis data, dan teknologi yang digunakan
6. Fitur utama aplikasi dan keunikan dibanding sistem serupa
7. Batasan sistem aplikasi
8. Tangkapan layar antarmuka yang sudah diimplementasikan
9. Rencana pengembangan lanjutan

Dua aturan yang memengaruhi cara kerja:

1. Karya harus orisinal, buatan sendiri, dan belum pernah dipublikasikan atau memenangkan perlombaan lain.
2. Framework, library, dan aset pihak ketiga boleh dipakai selama sesuai lisensinya dan didokumentasikan dalam proposal. Karena itu `docs/THIRD_PARTY.md` wajib dijaga tetap lengkap.

## 16. Rencana validasi lapangan

Tujuan: menguji tiga asumsi sebelum menulis kode.

**Asumsi yang diuji**

1. Pemilik depot kesulitan mengetahui kapan pelanggan butuh isi ulang.
2. Pemilik depot kehilangan jejak galon yang dipinjam pelanggan.
3. Pelanggan mau memesan lewat link.

**Pertanyaan untuk pemilik depot**

1. Dari mana saja pesanan masuk dalam sehari, dan berapa banyak?
2. Pernahkah ada pesanan yang terlewat atau terlambat? Ceritakan kejadian terakhir.
3. Bagaimana Bapak atau Ibu tahu pelanggan mana yang sudah waktunya isi ulang?
4. Berapa galon milik depot yang sekarang ada di pelanggan? Bagaimana mencatatnya?
5. Pernahkah kehilangan galon? Berapa dalam setahun?
6. Apakah ada promo seperti isi 10 gratis 1? Bagaimana mencatatnya?
7. Kalau ada daftar "pelanggan ini besok kehabisan air", apakah Bapak atau Ibu mau mengirim pesan ke mereka?
8. Alat apa yang dipakai sekarang (buku, WhatsApp, aplikasi)? Apa yang paling merepotkan?

**Pertanyaan untuk pelanggan**

1. Bagaimana biasanya memesan air? Seberapa sering kehabisan sebelum sempat memesan?
2. Kalau depot mengirim link untuk pesan sekali ketuk, apakah mau memakainya?
3. Pernahkah pindah depot? Karena apa?

**Bukti yang dikumpulkan untuk proposal**: foto buku catatan depot (dengan izin), jumlah pesanan harian, jumlah galon pinjaman, dan kutipan langsung.

## 17. Pertanyaan terbuka

1. Berapa biaya pendaftaran? Di guidebook nominalnya kosong.
2. Aturan tim mana yang berlaku: maksimal 3 orang satu instansi, atau boleh lintas instansi?
3. Bolehkah mulai menulis kode sebelum masa pengerjaan 19 Oktober?
4. Apakah format file surat pernyataan keaslian disediakan panitia?
5. Apakah alat bantu AI untuk menulis kode diperbolehkan, dan apakah perlu dicantumkan di proposal? Guidebook tidak mengaturnya, sedangkan karya disyaratkan orisinal dan buatan sendiri.

## 18. Sumber

- [S1] Aplikasi Layanan Depot Air Minum Isi Ulang Berbasis Web (Modul Pengguna), Telkom University. https://repositori.telkomuniversity.ac.id/home/catalog/id/206143/slug/aplikasi-layanan-depot-air-minum-isi-ulang-berbasis-web-modul-pengguna-dalam-bentuk-buku-karya-ilmiah.html
- [S2] Aplikasi Pesan Antar Air Mineral Isi Ulang dan Gas Elpiji Berbasis Android. https://www.researchgate.net/publication/354548048_Aplikasi_Pesan_Antar_Air_Mineral_Isi_Ulang_dan_Gas_Elpiji_Berbasis_Android
- [S3] Kontan, Industri Depot Air Minum Isi Ulang Banyak yang Belum Penuhi Standar Sanitasi. https://industri.kontan.co.id/news/industri-depot-air-minum-isi-ulang-damiu-banyak-yang-belum-penuhi-standar-sanitasi
- [S4] Skripsi Universitas Hasanuddin yang mengutip Survei Kesehatan Indonesia 2023. https://repository.unhas.ac.id/id/eprint/51775/2/K011211087-1-2.pdf
- [S5] Buku Galon di Google Play. https://play.google.com/store/apps/details?id=com.cybersoft.bukugalon&hl=id
- [S6] Kasir Pintar, Rekomendasi Aplikasi Kasir untuk Bisnis Air Isi Ulang. https://kasirpintar.co.id/solusi/detail/aplikasi-kasir-bisnis-air-isi-ulang
