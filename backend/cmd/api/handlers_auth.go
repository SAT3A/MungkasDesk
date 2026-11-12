package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// HANDLER — AUTH (login)
// Autentikasi dipisah dari CRUD user karena kekhawatirannya beda:
// CRUD peduli data user, auth peduli "benar tidaknya identitas".
// Nanti kalau session (cookie + tabel user_sessions) diimplementasi,
// alurnya tetap hidup di file ini, tidak menumpuk di handlers_users.
// ============================================================

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
