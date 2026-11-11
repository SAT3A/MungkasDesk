package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// MODEL — bentuk data user, identik dengan tabel users di database
// ============================================================

type User struct {
	ID           string     `json:"id"` // unique UUID, di-generate backend
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"` // "-" = TIDAK pernah dikirim ke client
	IsActive     bool       `json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"` // pointer → bisa null di JSON
	CreatedAt    time.Time  `json:"created_at"`
	DeletedAt    *time.Time `json:"deleted_at"` // pointer → NULL = belum dihapus (soft delete)
}

// CreateUserInput = bentuk BODY request untuk POST /users.
// Password diterima mentah sekali saja di sini, langsung di-hash, gak disimpan mentah.
type CreateUserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ============================================================
// KONEKSI DATABASE
// ============================================================

// db = connection pool: sekumpulan koneksi yang dipakai bersama semua request,
// supaya tiap request gak perlu membuka koneksi sendiri-sendiri.
var db *pgxpool.Pool

func connectDB() {
	// godotenv.Load() membaca file .env di folder kerja (backend/) dan
	// mengisinya sebagai environment variable. Error-nya sengaja diabaikan
	// (pakai _ =) karena file .env opsional — di deployment, env var bisa
	// diset langsung tanpa file. Yang menentukan tetap os.Getenv di bawah.
	//
	// Kenapa password gak boleh hardcode lagi? Karena kode ini masuk GitHub:
	// kredensial harusnya hidup di file yang di-gitignore (.env), bukan di
	// source yang ke-push.
	_ = godotenv.Load()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL belum diset. Salin .env.example jadi .env di folder backend/, lalu isi alamat PostgreSQL lu.")
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		log.Fatal("Gagal membuat pool:", err)
	}
	// Ping = cek koneksi benar-benar tersambung ke server, bukan cuma alamatnya valid.
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("Gagal terhubung ke PostgreSQL:", err)
	}
	db = pool
	log.Println("Koneksi PostgreSQL OK")
}

// ============================================================
// HANDLER — fungsi yang dipanggil Gin tiap ada request masuk
// ============================================================

func getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "API berjalan sukses",
	})
}

// getUsers: GET /users — list semua user langsung dari database
func getUsers(c *gin.Context) {
	ctx := context.Background()

	// WHERE deleted_at IS NULL = user yang sudah soft delete gak ikut ke list.
	// Barisnya masih ada di database, tapi API memperlakukannya sebagai tidak ada.
	// Urutan kolom SELECT harus sama dengan urutan Scan di bawah.
	rows, err := db.Query(ctx,
		"SELECT id, username, password_hash, is_active, last_login_at, created_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal mengambil data user",
		})
		return
	}
	defer rows.Close()

	var data []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.DeletedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal membaca data user",
			})
			return
		}
		data = append(data, u)
	}
	if err := rows.Err(); err != nil { // error yang mungkin terjadi saat iterasi
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal selesai membaca data user",
		})
		return
	}
	if data == nil {
		data = []User{} // tabel kosong → JSON [] bukan null
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Data user berhasil diambil",
		"data":    data,
	})
}

// ============================================================
// USERS — DETAIL, UPDATE, SOFT DELETE
// Tiga endpoint yang bekerja pada SATU user tertentu (bukan koleksi).
// ============================================================

// parseUserID: ambil teks ":id" dari URL dan pastikan bentuknya UUID yang valid.
// Kenapa perlu? c.Param("id") selalu mengembalikan STRING mentah dari URL.
// Kalau langsung dipakai di query, nilai salah format akan ditolak PostgreSQL
// dengan error yang membingungkan. Jadi validasi dilakukan di sini → HTTP 400.
//
// Nilai kembalian `ok bool` memakai pola "helper yang mengirim response sendiri":
// kalau false, response 400 SUDAH dikirim di dalam fungsi ini, jadi caller
// cukup `return` tanpa menulis response lagi.
func parseUserID(c *gin.Context) (string, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID harus berupa UUID yang valid",
		})
		return "", false
	}
	return id.String(), true
}

// getUserByID: GET /users/{id} — ambil SATU user berdasarkan ID.
//
// Bedanya dengan getUsers:
//
//	getUsers    → db.Query()   → bisa 0..N baris → wajib iterasi rows
//	getUserByID → db.QueryRow() → paling banyak 1 baris → cukup Scan sekali
//
// Kalau tidak ada baris yang cocok, err dari QueryRow bukan error server,
// melainkan pgx.ErrNoRows — sinyal resmi pgx untuk "data tidak ada".
// Inilah kenapa responsenya 404 (Not Found), bukan 500.
//
// `AND deleted_at IS NULL` → user yang sudah soft delete juga dianggap tidak
// ada. Datanya masih di database (bisa dilihat via query manual di psql),
// tapi dari sisi API sudah "hilang".
func getUserByID(c *gin.Context) {
	ctx := context.Background()

	id, ok := parseUserID(c)
	if !ok {
		return // response 400 sudah dikirim di dalam parseUserID
	}

	var u User
	err := db.QueryRow(ctx,
		"SELECT id, username, password_hash, is_active, last_login_at, created_at, deleted_at FROM users WHERE id = $1 AND deleted_at IS NULL",
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.DeletedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "User tidak ditemukan",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal mengambil data user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Data user berhasil diambil",
		"data":    u,
	})
}

// UpdateUserInput = bentuk BODY request PATCH /users/{id}.
//
// Field-nya pakai POINTER (*string / *bool). Ini kunci dari partial update:
//
//	nil  (field tidak ada di JSON) = "jangan ubah field ini"
//	ada isinya                     = "ubah field ini ke nilai ini"
//
// Kalau pakai string/bool biasa, kita tidak bisa membedakan "tidak dikirim"
// dengan "dikirim kosong/false". Contoh: is_active false itu request yang sah,
// dan harus bisa dibedakan dari is_active yang sama sekali tidak dikirim.
type UpdateUserInput struct {
	Username *string `json:"username"`
	IsActive *bool   `json:"is_active"`
}

// updateUser: PATCH /users/{id} — ubah sebagian field user (partial update).
//
// Trik utamanya ada di SQL: SET username = COALESCE($2, username).
//   - $2 = NULL → COALESCE memilih nilai LAMA → field tidak berubah
//   - $2 = isi  → COALESCE memilih nilai BARU → field berubah
//
// Jadi SATU query statis bisa menangani kombinasi update apa pun,
// tanpa harus merakit string SQL secara dinamis.
//
// RETURNING di akhir = minta PostgreSQL mengembalikan baris yang SUDAH diubah,
// sehingga kita tidak perlu SELECT ulang untuk response. Sekaligus berfungsi
// sebagai deteksi 404: kalau ID tidak cocok, UPDATE tidak menyentuh baris
// apa pun dan QueryRow mengembalikan pgx.ErrNoRows.
//
// Syarat WHERE juga mencakup deleted_at IS NULL → user yang sudah soft delete
// otomatis 404, jadi tidak bisa di-update lagi walau ID-nya masih valid.
func updateUser(c *gin.Context) {
	ctx := context.Background()

	id, ok := parseUserID(c)
	if !ok {
		return
	}

	var input UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Body tidak valid, butuh JSON berisi username dan/atau is_active",
		})
		return
	}

	// Validasi username HANYA kalau field-nya dikirim (tidak nil).
	var newUsername *string
	if input.Username != nil {
		trimmed := strings.TrimSpace(*input.Username)
		switch {
		case trimmed == "":
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Username tidak boleh kosong",
			})
			return
		case utf8.RuneCountInString(trimmed) > 100:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Username maksimal 100 karakter",
			})
			return
		}
		newUsername = &trimmed
	}

	// Tidak ada satu pun field yang mau diubah → tolak dengan 400,
	// supaya client sadar request-nya tidak melakukan apa-apa.
	if input.Username == nil && input.IsActive == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Tidak ada field yang diubah, kirim username dan/atau is_active",
		})
		return
	}

	var u User
	err := db.QueryRow(ctx,
		`UPDATE users
		 SET username  = COALESCE($2, username),
		     is_active = COALESCE($3, is_active)
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING id, username, password_hash, is_active, last_login_at, created_at, deleted_at`,
		id, newUsername, input.IsActive,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.DeletedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "User tidak ditemukan",
		})
		return
	}
	// Catatan: kalau username baru == username user ini sendiri, index unique
	// lower(username) tetap akan protes (23505) karena menabrak barisnya sendiri.
	// Keterbatasan ini sengaja dibiarkan sederhana untuk sekarang.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "Username sudah dipakai",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal menyimpan perubahan user",
		})
		return
	}

	// Pesan response menyesuaikan aksi yang terjadi. Kalau yang terjadi adalah
	// penonaktifan, formatnya: "User {nama} dinonaktifkan {tanggal & waktu}".
	// time.Now() = waktu server saat request diproses.
	// "2006-01-02 15:04:05" adalah layout acuan Go, BUKAN tanggal sembarangan:
	// 01 = bulan, 02 = hari, 15 = jam format 24 jam, 04 = menit, 05 = detik.
	message := "User berhasil diperbarui"
	if input.IsActive != nil && !*input.IsActive {
		message = fmt.Sprintf("User %s dinonaktifkan %s",
			u.Username, time.Now().Format("2006-01-02 15:04:05"))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    u,
	})
}

// softDeleteUser: DELETE /users/{id} — "menghapus" user secara LOGIS (soft delete).
//
// Baris di database TIDAK benar-benar dihapus. Yang berubah hanya kolom
// deleted_at: dari NULL jadi waktu sekarang (now() dari PostgreSQL).
// Keuntungannya:
//   - data masih ada kalau suatu saat perlu diperiksa atau dipulihkan
//   - tidak ada data lain yang ikut hilang
//
// Konsekuensinya: endpoint user lain memakai syarat deleted_at IS NULL,
// jadi user yang sudah dihapus logis gak muncul lagi di list/detail/update.
//
// QueryRow + RETURNING dipakai (bukan Exec) supaya kita dapat username dan
// waktu hapusnya untuk pesan response. ErrNoRows di sini artinya user tidak
// ada ATAU sudah pernah dihapus — keduanya dibalas 404.
func softDeleteUser(c *gin.Context) {
	ctx := context.Background()

	id, ok := parseUserID(c)
	if !ok {
		return
	}

	var username string
	var deletedAt time.Time
	err := db.QueryRow(ctx,
		`UPDATE users
		 SET deleted_at = now()
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING username, deleted_at`,
		id,
	).Scan(&username, &deletedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "User tidak ditemukan (atau sudah pernah dihapus)",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal menghapus user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("User %s dihapus %s",
			username, deletedAt.Format("2006-01-02 15:04:05")),
	})
}

// ============================================================
// VALIDASI PASSWORD — cek isi password, gak cuma panjangnya
// ============================================================

// containsUpper: true kalau ada minimal 1 huruf besar.
// unicode.IsUpper cocok untuk huruf besar apapun (A-Z dan juga macam-macam huruf non-Inggris).
func containsUpper(password string) bool {
	for _, r := range password {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// containsSymbol: true kalau ada minimal 1 simbol,
// yaitu karakter yang bukan huruf, bukan digit, dan bukan spasi.
func containsSymbol(password string) bool {
	for _, r := range password {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

// createUser: POST /users — bikin user baru.
// - id            : di-generate backend (UUID v4)
// - is_active     : selalu true (akun baru otomatis aktif)
// - last_login_at : gak diisi → NULL (belum pernah login)
// - created_at    : dari RETURNING created_at (= waktu baris dibuat oleh PostgreSQL)
// - password      : di-hash bcrypt sebelum masuk database
func createUser(c *gin.Context) {
	ctx := context.Background()

	// ShouldBindJSON = baca body request + masukkan ke struct + cek format JSON.
	var input CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Body tidak valid, butuh JSON berisi username dan password",
		})
		return
	}

	// ==========================================================
	// Validasi input (HTTP 400 kalau gak lolos)
	// ==========================================================
	username := strings.TrimSpace(input.Username)
	password := input.Password

	switch {
	case username == "":
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Username tidak boleh kosong",
		})
		return
	case utf8.RuneCountInString(username) > 100:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Username maksimal 100 karakter",
		})
		return
	case len(password) < 8:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Password minimal 8 karakter",
		})
		return
	case !containsUpper(password):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Password harus mengandung minimal 1 huruf besar (A-Z)",
		})
		return
	case !containsSymbol(password):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Password harus mengandung minimal 1 simbol (contoh: ! @ # $ % & ? *)",
		})
		return
	}

	// Hash password (bcrypt.DefaultCost = biaya standar).
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal memproses password",
		})
		return
	}

	// UUID v4 = acak, kemungkinannya bisa dobel terlalu kecil untuk dipikirkan.
	id := uuid.NewString()

	var createdAt time.Time
	err = db.QueryRow(ctx,
		`INSERT INTO users (id, username, password_hash, is_active)
		 VALUES ($1, $2, $3, true)
		 RETURNING created_at`,
		id, username, string(hashed),
	).Scan(&createdAt)

	if err != nil {
		// SQLSTATE 23505 = unique violation. Index lower(username) di database
		// yang menangkapnya, jadi aturan "username unik" gak di-dobel ceknya di sini.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": "Username sudah dipakai",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal menyimpan data user",
		})
		return
	}

	created := User{
		ID:           id,
		Username:     username,
		PasswordHash: string(hashed), // tandanya json:"-" → gak akan bocor ke response
		IsActive:     true,
		LastLoginAt:  nil,
		CreatedAt:    createdAt,
	}

	// 201 Created = makna REST yang bener untuk "data baru berhasil dibuat".
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "User berhasil dibuat",
		"data":    created,
	})
}

// ============================================================
// AUTH — LOGIN
// ============================================================

// LoginInput = bentuk BODY request untuk POST /login.
type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// login: POST /login — verifikasi username + password, catat last_login_at.
//
// Alurnya 4 langkah:
//  1. Cari user berdasarkan username (case-insensitive, sama seperti index unique)
//  2. User tidak ada → 401 dengan pesan GENERIK
//  3. User ada tapi is_active false → 403
//  4. Verifikasi password dengan bcrypt → cocok → isi last_login_at
//
// Kenapa "username tidak ada" dan "password salah" dibalas pesan yang SAMA?
// Kalau pesannya beda (mis. "user tidak ditemukan" vs "password salah"),
// orang bisa memakai perbedaan itu untuk mengintip username mana saja yang
// terdaftar. Pesan generik = informasi minim untuk pihak yang gak berhak.
// Detail keamanan seperti ini murah diterapkan, jadi kebiasaan yang baik.
func login(c *gin.Context) {
	ctx := context.Background()

	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Body tidak valid, butuh JSON berisi username dan password",
		})
		return
	}

	// Trim + wajib isi keduanya. Login gak punya alasan menerima field kosong,
	// jadi ini 400 sebelum menyentuh database sama sekali.
	username := strings.TrimSpace(input.Username)
	if username == "" || input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Username dan password wajib diisi",
		})
		return
	}

	// ==========================================================
	// 1. Cari user berdasarkan username
	// ==========================================================
	// lower(username) = lower($1) membuat pencarian case-insensitive,
	// konsisten dengan index unique lower(username) di database.
	// deleted_at IS NULL = user yang sudah soft delete gak bisa login lagi,
	// meskipun baris datanya masih ada.
	var u User
	err := db.QueryRow(ctx,
		`SELECT id, username, password_hash, is_active, last_login_at, created_at, deleted_at
		 FROM users
		 WHERE lower(username) = lower($1) AND deleted_at IS NULL`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.DeletedAt)

	// ErrNoRows = username tidak ada (atau user sudah dihapus logis).
	// Sengaja dibalas 401 "username/password salah", BUKAN 404 "user tidak
	// ada" — lihat alasan pesan generik di komentar atas fungsi.
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Username atau password salah",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal mengambil data user",
		})
		return
	}

	// ==========================================================
	// 2. User nonaktif ditolak SEBELUM password dicek → 403
	// ==========================================================
	// 403 Forbidden = "aku tahu kamu siapa, tapi gak boleh".
	// Beda dengan 401 yang berarti "aku gak bisa memastikan kamu siapa".
	if !u.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Akun dinonaktifkan, hubungi admin",
		})
		return
	}

	// ==========================================================
	// 3. Verifikasi password dengan bcrypt
	// ==========================================================
	// CompareHashAndPassword membandingkan password MENTAH dari request
	// dengan hash di database. Caramnya bukan "dekripsi lalu banding" —
	// bcrypt memang gak bisa dibalik. Fungsi ini meng-hash input dengan
	// parameter yang tersimpan di dalam hash itu sendiri, lalu cek hasilnya.
	// nil = cocok; error lain (mis. ErrMismatchedHashAndPassword) = gak cocok.
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Username atau password salah",
		})
		return
	}

	// ==========================================================
	// 4. Sukses → catat waktu login terakhir (akhirnya keisi!)
	// ==========================================================
	// UPDATE ... RETURNING last_login_at = satu query untuk dua kerjaan:
	// mengisi last_login_at DAN mengambil nilai barunya ke struct,
	// tanpa perlu SELECT kedua.
	err = db.QueryRow(ctx,
		`UPDATE users
		 SET last_login_at = now()
		 WHERE id = $1
		 RETURNING last_login_at`,
		u.ID,
	).Scan(&u.LastLoginAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal mencatat waktu login",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login berhasil",
		"data":    u,
	})
}

// ============================================================
// MAIN — konek DB, daftarkan route, jalankan server
// ============================================================

func main() {
	connectDB()
	defer db.Close() // tutup pool saat aplikasi berhenti

	router := gin.Default()

	// Route group: semua URL di bawah otomatis punya prefix "/api/v1"
	api := router.Group("/api/v1")
	{
		api.GET("/health", getHealth)
		api.POST("/login", login)
		api.GET("/users", getUsers)
		api.POST("/users", createUser)
		api.GET("/users/:id", getUserByID)
		api.PATCH("/users/:id", updateUser)
		api.DELETE("/users/:id", softDeleteUser) // route DELETE, tapi isinya soft delete (baris tetap ada)
	}

	log.Println("Server jalan di http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
