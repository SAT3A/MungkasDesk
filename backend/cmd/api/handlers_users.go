package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// HANDLER — USERS (list, detail, create, update)
// Semua endpoint yang bekerja pada data user ada di sini. Satu file
// per "domain" (users vs auth) supaya saat backend tumbuh, tiap domain
// punya rumah sendiri tanpa langsung perlu package/folder baru.
// ============================================================

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
	// (Perilaku sebenarnya sudah teruji: PostgreSQL tidak menganggap baris
	// konflik dengan dirinya sendiri, jadi update ke nilai sendiri tetap 200.)
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
