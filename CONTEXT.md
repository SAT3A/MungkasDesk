# MungkasDesk — Context & Handover

> Dokumen "otak project". Target pembaca: coding agent / developer lain yang perlu kerja di
> repo ini **tanpa harus baca semua kode dari nol**. Ibarat handover ke tim IT berikutnya.
>
> Terakhir diperbarui: **12 November 2026** (setelah QA API 64 kasus; aturan wajib update §0 ditambahkan).
> Kalau ada perubahan besar di kode, perbarui dokumen ini juga — dokumen basi lebih bahaya daripada tidak ada.

---

## 0. ATURAN WAJIB — update dokumen ini tiap akhir task

Bukan saran; ini aturan project yang ditegakkan lewat `instructions` di `opencode.jsonc`
(dokumen ini di-load OpenCode sebagai instruksi untuk SEMUA agent tiap sesi) dan
aturan #10 di `.opencode/agents/executor.md`. Berlaku untuk setiap agent yang
selesai menjalankan task di repo ini:

1. **Update `CONTEXT.md`** sesuai perubahan yang terjadi — perbarui bagian/section yang kena.
2. Kalau tidak ada bagian lain yang berubah (task kecil, config, dokumentasi),
   cukup tambahkan 1 baris di **§13 Changelog** dan perbarui tanggal
   "Terakhir diperbarui" di atas.
3. **Sebut di laporan akhir**: "CONTEXT.md sudah di-update (§X)". Tidak menyebut =
   task dianggap BELUM selesai.

Alasannya: dokumen ini satu-satunya peta cepat untuk agent berikutnya. Dokumen basi
= agent berikutnya kerja dengan peta salah, lebih bahaya daripada tidak ada peta.

---

## 1. Ringkasan eksekutif

MungkasDesk = webapp manajemen konten sosial media: merencanakan konten teks, mengelola
persona & akun sosial (Threads/Facebook), melihat jadwal di kalender, dan mencatat postingan
yang sudah diterbitkan manual. Satu pengguna (pemilik aplikasi), tidak ada registrasi publik.

**Status realita saat ini (pahami dulu SEBELUM ngoding):**

- Yang **SUDAH ada**: satu REST API Go/Gin untuk **manajemen user** (CRUD + soft delete + login),
  seluruhnya di **satu file** `backend/cmd/api/main.go` (~705 baris) + **satu tabel** `users` di PostgreSQL.
  Tahap fondasi/pembelajaran, berjalan dan sudah lolos QA manual (58/64 PASS, 6 temuan lihat §9).
- Yang **BELUM ada** (seluruh isi `PLAN.md`): personas, social accounts, content_posts,
  kalender, session idle-timeout 2 jam, frontend Next.js/FullCalendar, sqlc, Goose.
  `PLAN.md` = **blueprint masa depan**, bukan deskripsi kode sekarang.
- **Jangan asumsikan schema di PLAN.md sama dengan database sekarang.** Divergensi penting:
  | Hal | PLAN.md (rencana) | Kode sekarang (aktif) |
  |---|---|---|
  | Identitas login | `email` + `display_name` | `username` |
  | Hash password | Argon2id | **bcrypt** (`golang.org/x/crypto/bcrypt`) |
  | Query | sqlc generate | SQL mentah via `pgx/v5` |
  | Migration | Goose versioned | File SQL manual `backend/db/init_users.sql` (idempotent, jalankan via psql) |
  | Session | cookie + tabel `user_sessions`, idle 2 jam | **Belum ada** — semua endpoint terbuka (lihat F5) |
  | Format error | `{"error":{code,message,fields}}` | `{"success","message"[,"data"]}` |

---

## 2. Tech stack (yang aktif)

| Komponen | Nilai |
|---|---|
| Bahasa | Go 1.27.1 (module `mungkasdesk/backend`) |
| HTTP framework | Gin v1.12.0 |
| Database | PostgreSQL lokal (nama db: `mungkasdesk`) |
| Driver | `pgx/v5` + `pgxpool` (connection pool global var `db`) |
| Password hashing | bcrypt, `bcrypt.DefaultCost` |
| UUID | `github.com/google/uuid` (UUID v4, dibuat backend, BUKAN db) |
| Env | `github.com/joho/godotenv` membaca `backend/.env` |
| Port server | `:8080` (hardcoded di `router.Run(":8080")`, belum ada env `HTTP_ADDR`) |

Dependency langsung di `backend/go.mod`: gin, uuid, pgx/v5, x/crypto.
Catatan kecil: `joho/godotenv` tercatat sebagai `// indirect` padahal di-import main.go —
kosmetik saja; `go mod tidy` akan memindahkannya ke blok direct.

---

## 3. Struktur repo

```text
MungkasDesk/
├── CONTEXT.md                  # dokumen ini
├── PLAN.md                     # blueprint aplikasi lengkap fase 1 (belum diimplementasi)
├── TEST-REPORT-API.md          # hasil QA manual 64 kasus, 12 Nov 2026 (BELUM di-commit)
├── AGENTS.md                   # aturan graphify + panduan sesi (dibaca agent)
├── opencode.jsonc, .opencode/  # config OpenCode; CONTEXT.md di-load via "instructions" + 3 agent custom: planner/executor/testing
├── graphify-out/               # knowledge graph project (query sebelum baca kode)
├── .gitignore                  # *.exe, backend/.env
└── backend/
    ├── cmd/api/main.go         # SELURUH backend ada di sini (single file, by design)
    ├── db/init_users.sql       # schema tabel users (idempotent, dijalankan manual via psql)
    ├── .env                    # kredensial lokal — GITIGNORED, jangan pernah di-commit
    ├── .env.example            # placeholder DATABASE_URL
    ├── go.mod / go.sum
    └── api.exe                 # build artifact (gitignored); build ulang kalau hilang
```

---

## 4. API endpoints

Semua route di bawah group `/api/v1` (daftarnya di `main()` bagian bawah main.go).
Server: `http://localhost:8080`.

| Method | Path | Handler | Sukses | Keterangan |
|---|---|---|---|---|
| GET | `/api/v1/health` | `getHealth` | 200 | Health check, tanpa data |
| POST | `/api/v1/login` | `login` | 200 | Lihat alur 4 langkah §8 |
| GET | `/api/v1/users` | `getUsers` | 200 | List user aktif saja, `data` = array (dijamin `[]` bukan `null` saat kosong) |
| POST | `/api/v1/users` | `createUser` | 201 | Body `{username, password}` |
| GET | `/api/v1/users/:id` | `getUserByID` | 200 | `:id` wajib UUID valid, else 400 |
| PATCH | `/api/v1/users/:id` | `updateUser` | 200 | Partial update; body `{username?/is_active?}` |
| DELETE | `/api/v1/users/:id` | `softDeleteUser` | 200 | Soft delete (baris tetap ada), sukses tanpa field `data` |

### Bentuk response (envelope konsisten)

```jsonc
// sukses
{ "success": true, "message": "…", "data": { …user object / array… } }
// gagal
{ "success": false, "message": "…" }
```

User object tidak pernah membocorkan password: `PasswordHash` bertag `json:"-"`.

### Semantik status code (WAJIB dipertahankan agent berikutnya)

| Code | Kapan dipakai |
|---|---|
| 400 | Body bukan JSON valid, ID bukan UUID, field kosong, password gagal policy, PATCH tanpa field yang mau diubah |
| 401 | Login gagal (pesan **generik** "Username atau password salah" — username tidak ada & password salah dibalas PESAN SAMA untuk cegah username enumeration) |
| 403 | Login user yang `is_active = false` (dicek sebelum bcrypt) |
| 404 | Data tidak ada **atau sudah soft delete** — via `pgx.ErrNoRows`, bukan error server. (Pengecualian: 404 route Gin keluar plain text `404 page not found`, belum JSON — temuan F8) |
| 409 | Username duplikat — ditangkap dari SQLSTATE **23505** (`errors.As` → `*pgconn.PgError`), cek-nya di index unique DB, tidak di-dobel di aplikasi |
| 500 | Error DB/hash lainnya |

---

## 5. Database

### Tabel `users` (satu-satunya, file `backend/db/init_users.sql`)

```sql
id            UUID PRIMARY KEY          -- dibuat backend: uuid.NewString()
username      VARCHAR(100) NOT NULL     -- unique case-insensitive via index lower(username)
password_hash TEXT NOT NULL             -- bcrypt, BUKAN password asli
is_active     BOOLEAN NOT NULL DEFAULT true
last_login_at TIMESTAMPTZ NULL          -- NULL = belum pernah login
created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
deleted_at    TIMESTAMPTZ NULL          -- NULL = belum dihapus (soft delete marker)
```

- Soft delete: `DELETE` endpoint hanya mengisi `deleted_at = now()`. **Tidak ada hard delete dari aplikasi.**
- Semua query user menambahkan `AND deleted_at IS NULL` → user terhapus "hilang" dari API tapi datanya masih di DB.
- Restore manual (dari psql): `UPDATE users SET deleted_at = NULL WHERE id = '<uuid>';`
- Lihat data terhapus: `SELECT * FROM users WHERE deleted_at IS NOT NULL;`

### Aturan perubahan schema (penting!)

1. Edit `backend/db/init_users.sql` dengan pola **idempotent**: `CREATE TABLE IF NOT EXISTS`,
   `ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`.
2. **Jalankan ulang file itu via psql SEBELUM server dipakai** — kolom baru di SELECT/Scan
   handler tanpa ALTER yang sesuai membuat endpoint 500.
3. Urutan kolom SELECT harus sama persis dengan urutan `Scan()` di handler.

Contoh jalanin migration (PowerShell, sesuaikan user/password):

```powershell
psql -U <user> -d mungkasdesk -f backend/db/init_users.sql
```

---

## 6. Konvensi kode (gaya yang dipakai project ini)

- Komentar **Bahasa Indonesia yang menjelaskan KENAPA** (rasional, trade-off), bukan cuma mengulang sintaks. Ini disengaja karena user sedang belajar kode pelan-pelan — pertahankan gayanya.
- Header section dalam file: `// ============================================================` + judul section (MODEL, KONEKSI DATABASE, HANDLER, AUTH, MAIN, dll).
- Tetap **satu file dulu** (`cmd/api/main.go`) sebelum memecah ke struktur folder; pecah hanya kalau memang sudah terasa berat, dan pecahnya mengikuti section yang sudah ada.
- Query selalu berparameter (`$1, $2`), tidak ada string SQL dirakit dari input.
- Trik SQL yang dipakai: `COALESCE($n, kolom)` untuk partial update; `UPDATE … RETURNING` untuk dapat nilai balik tanpa SELECT kedua.

---

## 7. Cara jalanin lokal (Windows)

```powershell
# 1. Siapkan config: salin .env.example → .env di backend/, isi DATABASE_URL
#    Format: postgres://<user>:<password>@localhost:5432/mungkasdesk?sslmode=disable

# 2. Buat/update schema (idempotent, boleh diulang)
psql -U <user> -d mungkasdesk -f backend/db/init_users.sql

# 3. Build & run (dari folder backend/)
cd backend
go build -o api.exe ./cmd/api
.\api.exe
# Server jalan di http://localhost:8080 — log sukses: "Koneksi PostgreSQL OK"
```

- `connectDB()` akan `log.Fatal` kalau `DATABASE_URL` kosong.
- Kredensial TIDAK PERNAH masuk source code; hanya lewat env var / `.env` (gitignored).
- Menambah env var baru? Update `backend/.env.example` juga (placeholder saja).

---

## 8. Keputusan desain penting + alasannya

1. **Soft delete, bukan hard delete** — data bisa diperiksa/dipulihkan; FK tidak ikut runtuh.
2. **Pesan login generik** — username tidak ada dan password salah sama-sama 401 dengan pesan
   identik, supaya orang tidak bisa mengintip username mana yang terdaftar.
3. **Cek `is_active` sebelum bcrypt di login** — user nonaktif langsung 403
   (trade-off: bocor status akun tanpa password; disadari, lihat F6).
4. **Password divalidasi isinya, bukan cuma panjang**: minimal 8 karakter, ≥1 huruf besar
   (`unicode.IsUpper`), ≥1 simbol (karakter yang bukan huruf/digit/spasi). Password TIDAK di-trim
   (spasi boleh jadi bagian password); username di-trim.
5. **UUID dibuat backend** (`uuid.NewString()`), bukan `gen_random_uuid()` di DB — tidak
   bergantung extension PostgreSQL.
6. **Partial update via pointer struct** — `*string`/`*bool` di `UpdateUserInput` supaya bisa
   bedakan "field tidak dikirim" (nil = jangan ubah) vs "dikirim false/kosong".
   Di SQL ditangani `COALESCE($2, username)`.
7. **404 vs 500 dipisah disiplin** — `pgx.ErrNoRows` = 404; error lain = 500. Duplikat = 409 dari SQLSTATE 23505.
8. **Kondisi `deleted_at IS NULL` di SEMUA endpoint user** — user terhapus tidak bisa login,
   tidak bisa di-patch, tidak muncul di list/detail.

---

## 9. Backlog: temuan QA terbuka (12 Nov 2026, dari TEST-REPORT-API.md)

Hasil: 58 PASS / 6 FAIL / 64 kasus. Semua FAIL = temuan perancangan, bukan crash.
JANGAN di-rediscover — ini daftarnya:

| ID | Temuan | Saran |
|---|---|---|
| **F1** | Username user soft-deleted ter-reserve PERMANEN (create ulang → 409) karena unique index `lower(username)` mencakup semua baris, termasuk yang `deleted_at` terisi | Ganti ke partial index `... WHERE deleted_at IS NULL`, atau validasi konflik hanya vs user aktif |
| **F2** | Password > 72 byte → **500** (bukan 400): `bcrypt.GenerateFromPassword` mengembalikan `ErrPasswordTooLong`, dan validasi create tidak punya batas panjang | Validasi `len(password) <= 72` → 400 dengan pesan jelas |
| **F3** | Content-Type tidak divalidasi — JSON valid dengan header `text/plain` tetap diproses 201 | Standar: tolak dengan 415, atau minimal dokumentasikan perilaku permissive ini |
| **F4** | Method salah (mis. `PUT /users`) dibalas 404, bukan 405 | Set `router.HandleMethodNotAllowed = true` |
| **F5** | **Tidak ada autentikasi/otorisasi di SEMUA endpoint** — CRUD user terbuka penuh. Paling besar di backlog; ini sejalan dengan rencana session di PLAN.md §13 | Implementasi session cookie + tabel `user_sessions` (idle 2 jam) sesuai PLAN.md |
| **F6** | Login user nonaktif + password salah tetap 403 → bocor keberadaan akun (by design, tapi perlu disadari) | Trade-off sadar; revisi kalau mau lebih strict |
| **F7** | Komentar main.go ~baris 311-313 menyesatkan: klaim update username ke nilai sendiri akan kena 23505 — faktanya PostgreSQL tidak konflik dengan baris sendiri (terbukti P9/P15 = 200). Perilaku KODE-nya benar, komentarnya yang perlu dikoreksi | Perbaiki komentar |
| **F8** | Error format tidak konsisten: error bisnis JSON envelope, tapi 404 route Gin plain text | Tambah fallback handler JSON |
| **F9** | Observasi minor: `GET /users` tanpa pagination (semua baris sekaligus); trailing slash `/users/` auto-redirect 200 | Tambah pagination saat data tumbuh |

Detail lengkap + bukti DB per kasus: `TEST-REPORT-API.md`.
Skrip tes reusable (dibuat saat QA, ada di temp folder, bisa sudah terhapus):
`C:\Users\irvansah.pamungkas\AppData\Local\Temp\opencode\qa_api_tests.ps1`.

---

## 10. Aturan project (wajib untuk coding agent)

1. **Graphify dulu**: sebelum grep/baca source untuk pertanyaan kodebase, jalankan
   `graphify query "<topik>"`.
   **Cara panggil yang terverifikasi (Nov 2026)**: shim `graphify.exe` HILANG dari
   `C:\Users\irvansah.pamungkas\.local\bin` (hanya `graphify-mcp.exe` yang ada) meskipun
   `uv tool list` menampilkan `graphifyy v0.9.106`. Solusi tanpa reinstall:
   ```powershell
   uvx --from graphifyy graphify query "<topik>"
   uvx --from graphifyy graphify update .
   ```
   atau repair dulu: `uv tool install graphifyy --reinstall`.
   `graphify-out/wiki/index.md` belum ada; untuk review arsitektur luas baca `graphify-out/GRAPH_REPORT.md`.
2. Cek `graphify-out/reflections/LESSONS.md` sebelum menelusuri topik yang pernah dikerjakan.
3. Setelah perubahan kode final: **build dulu** (`go build -o api.exe ./cmd/api` dari `backend/`),
   lalu `graphify update .` (AST-only, gratis) supaya peta kode akurat.
4. Response API konsisten `{"success","message","data"}` + status code semantik (§4).
5. Kredensial tidak pernah masuk source; env baru → update `.env.example`.
6. Perubahan schema → `backend/db/init_users.sql` idempotent + ingatkan user jalankan ulang via psql (§5).
7. Git: boleh `git add`/`commit` dengan pesan jelas; **`git push` WAJIB konfirmasi user dulu**.
8. Rencana ambigu / bertentangan dengan kode ada → **BERHENTI dan tanya**, jangan menebak.
9. Dilarang keras: hardcode secret, SQL penghapus data dari kode aplikasi, perubahan schema
   yang menghapus data tanpa diminta eksplisit.
10. Temuan baru yang sudah diverifikasi → simpan via `graphify save-result --question "..." --answer "..." --nodes "..." --outcome useful`, lalu `graphify reflect --graph graphify-out/graph.json`.
11. **Tiap akhir task: update CONTEXT.md** (detail + alasan di §0) — dan WAJIB menyebut
    di laporan akhir bahwa CONTEXT.md sudah di-update, sebut bagiannya.

---

## 11. Gotcha environment (Windows / PowerShell 5.1)

- Shell adalah **Windows PowerShell**; error body dari `Invoke-WebRequest` perlu cara baca
  khusus (lihat LESSONS.md / TEST-REPORT-API.md metode QA).
- `api.exe` adalah build artifact, gitignored (`*.exe`); build ulang kalau hilang.
- Repo path mengandung **spasi** (`C:\Satria Punya\MungkasDesk`) — selalu quote path di shell.
- `TEST-REPORT-API.md` masih untracked; keputusan commit-nya milik user.

---

## 12. Peta cepat: "kalau mau X, sentuh Y"

| Mau… | Sentuh |
|---|---|
| Tambah kolom di users | `backend/db/init_users.sql` (idempotent) → jalankan ulang via psql → update SELECT+Scan di main.go |
| Tambah endpoint | Handler baru di `backend/cmd/api/main.go` + daftarkan route di `main()` (group `api`) |
| Perbaiki F1–F9 | Lihat §9; mulai dari F5 (auth) dan F2 (bcrypt 72 byte) kalau mau dampak terbesar |
| Mulai fase 1 sesungguhnya (personas, content_posts, kalender) | Ikuti `PLAN.md` — tapi sadari divergensi stack di §1 (bcrypt vs Argon2id, username vs email, SQL manual vs sqlc/Goose): putuskan dulu mau porsi mana yang diadopsi |
| Paham alur login | `login()` di main.go — 4 langkah: cari user (case-insensitive) → 401 generik → 403 nonaktif → bcrypt → update `last_login_at` |
| Lanjutkan sesi agent sebelumnya | Dari root repo ini: `opencode -c` (sesi terakhir) atau `opencode -s <sessionID>`; cari ID lewat `opencode session list`. Sesi disimpan di background service — terminal ditutup pun masih bisa dilanjutkan |
| Restore user terhapus | `UPDATE users SET deleted_at = NULL WHERE id = '<uuid>';` (psql manual, by design tidak ada endpoint-nya) |

---

## 13. Changelog dokumen

Jejak update CONTEXT.md — WAJIB ditambah baris baru di sini tiap task selesai
(lihat §0), minimal 1 baris.

| Tanggal | Perubahan | Agent |
|---|---|---|
| 12 Nov 2026 | Dokumen dibuat (handover pertama: status, API, schema, konvensi, backlog QA F1–F9, aturan agent) | executor |
| 12 Nov 2026 | Aturan wajib update dokumen ditambahkan: §0 (hukum tiap akhir task), §10 item 11, §13 Changelog ini. `opencode.jsonc` sekarang memuat CONTEXT.md sebagai `instructions` (kebukti live: CONTEXT.md ke-inject tanpa restart); aturan #10 ditambahkan di `.opencode/agents/executor.md` | executor |
| 12 Nov 2026 | Tambah baris §12: cara lanjutkan session sebelumnya (`opencode -c` / `opencode -s <sessionID>`, daftar ID via `opencode session list`) — dari pertanyaan user, tidak ada kode yang diubah | executor |
