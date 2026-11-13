package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================
// MAIN — kerangka aplikasi (titik masuk).
//
// Dulu SELURUH backend hidup di file ini. Sekarang dipecah per
// tanggung jawab, tapi tetap SATU package main — semua file di
// folder cmd/api saling bisa manggil tanpa perlu import:
//
//	database.go        → var db + connectDB() (koneksi PostgreSQL)
//	model.go           → struct data (User, input request)
//	handlers_users.go  → endpoint user (list/detail/create/update)
//	handlers_auth.go   → endpoint login
//	validate.go        → helper validasi (UUID, aturan password)
//
// File ini yang menjadi "pusat": satu-satunya tempat yang tahu
// urutan inisialisasi dan daftar route. Kenapa pecah file dulu,
// belum ke package/folder (mis. internal/handler)? Karena pecah
// file = manfaat besar (cari kode cepat, perubahan terisolasi)
// dengan biaya nol. Package baru menambah import path + aturan
// antar-package; itu layak dipikirkan saat file-nya sudah bener
// banyak, bukan sekarang.
// ============================================================

// getHealth: GET /api/v1/health — sinyal "API hidup".
// Cuma pasang di main.go karena isinya app-level (bukan urusan user),
// dan bodinya hanya 1 fungsi kecil — buat file khusus justru lebay.
func getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "API berjalan sukses",
	})
}

// ============================================================
// ROUTE & SERVER — satu-satunya peta endpoint yang hidup
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

		// Catatan: TIDAK ada route DELETE — user diminta nonaktif via
		// PATCH {is_active: false}. Menghapus data dari API sengaja tidak
		// disediakan supaya tidak ada jalur hilangnya data dari aplikasi.
		// (Baris user yang sudah soft delete di DB tetap di-filter
		// deleted_at IS NULL di semua handler.)
	}

	log.Println("Server jalan di http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
