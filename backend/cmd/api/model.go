package main

import (
	"time"
)

// ============================================================
// MODEL — bentuk data user, identik dengan tabel users di database.
// Semua struct request/response dikumpulkan di file ini supaya kalau
// bentuk data berubah, cukup dicek dan diubah di SATU tempat —
// tidak tercecer di antara handler-handler.
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

// LoginInput = bentuk BODY request untuk POST /login.
type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
