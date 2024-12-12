# Plugin Cache by URL

## Deskripsi
Plugin Cache by URL digunakan untuk meningkatkan performa Sidra Api dengan menyimpan hasil respons dari backend berdasarkan URL di Redis. Plugin ini memungkinkan pengambilan data langsung dari cache jika data tersedia, mengurangi beban pada backend.

---

## Cara Kerja
1. **Fase Akses (Access)**
   - Plugin memeriksa apakah data sudah tersedia di Redis menggunakan key yang dihasilkan dari kombinasi method, path, dan hash body request.
   - Jika data ditemukan (*cache hit*), respons dikembalikan langsung dari Redis.
   - Jika tidak ditemukan (*cache miss*), request diteruskan ke backend.

2. **Fase Header (Header)**
   - Respons dari backend disimpan ke Redis dengan TTL (Time to Live) default selama 5 menit.

---

## Konfigurasi
- **Redis**: Plugin memerlukan Redis untuk menyimpan data cache.
  - **Host**: `localhost:6379` (default)
  - **Password**: Tidak ada password (default)
  - **DB**: `0` (default)

---

## Cara Menjalankan
1. **Pastikan Redis berjalan** di alamat `localhost:6379`.
2. Tambahkan file `main.go` ini ke direktori plugin Sidra Gateway, misalnya: `plugins/cache/main.go`.
3. Kompilasi dan jalankan Sidra Gateway.
4. Plugin akan otomatis terhubung melalui UNIX socket pada path `/tmp/cache.sock`.

---

## Pengujian

### Endpoint
- **URL**: `http://localhost:3080/api/v1/resource`

### Langkah Pengujian
1. Kirim request GET dengan body tertentu menggunakan Postman:
   ```plaintext
   GET http://localhost:3080/api/v1/resource
   Body: {"data":"example"}
   ```
2. Periksa Redis:
   - Gunakan perintah `redis-cli KEYS cache:*` untuk melihat key yang tersimpan.
   - Gunakan `redis-cli GET <key>` untuk memeriksa nilai yang tersimpan.
3. Kirim request GET yang sama lagi dan pastikan data diambil dari cache (*cache hit*).

---

## Catatan Penting
- **TTL Cache**: Data cache akan disimpan selama 5 menit secara default.
- **Kegunaan**: Gunakan plugin ini hanya untuk data yang jarang berubah.

---

## Lisensi
MIT License
