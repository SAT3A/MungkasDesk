---
type: "query"
date: "2026-11-11T07:37:03.236775+00:00"
question: "Bagaimana membaca body respons error (4xx) dari Invoke-RestMethod di PowerShell 5.1 saat smoke test API?"
contributor: "graphify"
---

# Q: Bagaimana membaca body respons error (4xx) dari Invoke-RestMethod di PowerShell 5.1 saat smoke test API?

## Answer

Pakai \ .ErrorDetails.Message — di situ JSON body error tersedia. Pendekatan StreamReader di Exception.Response.GetResponseStream() menghasilkan string kosong. Terverifikasi saat smoke test POST /api/v1/login (backend/cmd/api/main.go): HTTP 401 :: {" message\:\Username atau password salah\,\success\:false}. --nodes login() --outcome useful