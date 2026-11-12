package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// ============================================================
// KONEKSI DATABASE
// Satu file khusus urusan koneksi, supaya handler-handler di file lain
// tidak perlu tahu BAGAIMANA koneksi dibuat — mereka cukup pakai `db`.
// Kalau nanti mau ganti cara konfigurasi (mis. tambah env var), urusan
// itu berhenti di file ini, tidak menular ke handler.
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
