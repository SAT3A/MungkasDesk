package main

import (
	"net/http"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============================================================
// VALIDASI — helper yang bisa dipakai lebih dari satu handler.
// Dipisah dari handler supaya handler tinggal manggil dan fokus ke
// alur request, sedangkan aturan validasi hidup di satu tempat.
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
