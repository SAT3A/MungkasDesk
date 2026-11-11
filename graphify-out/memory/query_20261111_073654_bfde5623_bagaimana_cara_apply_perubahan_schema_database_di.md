---
type: "query"
date: "2026-11-11T07:36:54.417306+00:00"
question: "Bagaimana cara apply perubahan schema database di project ini?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["main.go"]
---

# Q: Bagaimana cara apply perubahan schema database di project ini?

## Answer

Jalankan ulang backend/db/init_users.sql via psql (path: C:\Program Files\PostgreSQL\18\bin\psql.exe, psql TIDAK di PATH, set \ dulu, DB: mungkasdesk). File SQL-nya idempotent (CREATE/ALTER/INDEX semuanya IF NOT EXISTS) jadi aman dijalankan berulang. PENTING: setelah kolom baru ditambahkan ke SELECT handler Go, DB harus di-update dulu sebelum api.exe dijalankan, kalau tidak semua endpoint terkait 500 (column does not exist). Terverifikasi 11 Nov 2026 saat menambah kolom deleted_at.

## Outcome

- Signal: useful

## Source Nodes

- main.go