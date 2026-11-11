-- ============================================================
-- TABEL: users
-- Dipetakan dari model User di cmd/api/main.go.
-- Ini versi "migration sederhana": file SQL dieksekusi manual via psql,
-- nantinya kalau mau lebih terstruktur bisa naik ke tool migration (mis. Goose).
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY,               -- di-generate backend (uuid.NewString())
    username      VARCHAR(100) NOT NULL,
    password_hash TEXT         NOT NULL,                  -- hash bcrypt, BUKAN password asli
    is_active     BOOLEAN      NOT NULL DEFAULT true,     -- akun baru otomatis aktif
    last_login_at TIMESTAMPTZ  NULL,                      -- NULL = belum pernah login
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),    -- otomatis = waktu akun dibuat
    deleted_at    TIMESTAMPTZ  NULL                       -- NULL = belum dihapus; soft delete = diisi now()
);

-- Username unik secara case-insensitive (Budi = budi = BUDI).
CREATE UNIQUE INDEX IF NOT EXISTS users_username_unique
    ON users (lower(username));

-- Catatan: blok seed data dummy sudah dihapus.
-- Data user baru dibuat lewat endpoint POST /api/v1/users.

-- ============================================================
-- PERUBAHAN SCHEMA: kolom deleted_at (soft delete)
-- Untuk database yang tabel users-nya SUDAH dibuat dari versi lama file ini,
-- jalankan file ini lagi via psql. Semua perintah di sini idempotent:
-- CREATE TABLE IF NOT EXISTS + ADD COLUMN IF NOT EXISTS + CREATE INDEX IF NOT
-- EXISTS → dijalankan berulang pun hasilnya sama, tidak ada yang rusak.
-- ============================================================

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

-- Cara lihat user yang sudah dihapus (data tetap ada, cuma disembunyikan API):
--   SELECT * FROM users WHERE deleted_at IS NOT NULL;
-- Cara pulihkan (restore) lewat query manual:
--   UPDATE users SET deleted_at = NULL WHERE id = '<uuid>';
