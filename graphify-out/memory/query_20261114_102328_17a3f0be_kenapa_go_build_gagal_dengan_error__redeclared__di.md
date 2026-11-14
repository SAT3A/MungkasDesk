---
type: "query"
date: "2026-11-14T10:23:28.288757+00:00"
question: "Kenapa go build gagal dengan error 'redeclared' di stdlib internal/strconv setelah upgrade Go via winget MSI, dan bagaimana fix-nya? Serta bagaimana cara benar install opencode V2 di Windows via npm?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["CONTEXT.md,backend"]
---

# Q: Kenapa go build gagal dengan error 'redeclared' di stdlib internal/strconv setelah upgrade Go via winget MSI, dan bagaimana fix-nya? Serta bagaimana cara benar install opencode V2 di Windows via npm?

## Answer

1) Upgrade MSI in-place Go 1.26.5(386)->1.27.4 meninggalkan file stdlib campuran dua versi (pow10Tab & uint64pow10 dideklarasi dobel di internal/strconv) sehingga build gagal walau 'go version' sudah benar. Fix terverifikasi: msiexec /X{ProductCode} (butuh UAC/admin; exit 1730 kalau non-elevated), hapus C:\Program Files\Go, install ulang winget --source winget (msstore error 0x80072ee2). 2) opencode V2 (config project pakai format permissions V2) di Windows harus npm install -g --allow-scripts=@opencode/cli @opencode/cli, karena tanpa flag itu postinstall (pemilih binary native) diblokir policy allowScripts npm dan CLI tidak berfungsi. Hasil: build+vet backend OK, opencode v2.3.0 + MCP playwright connected.

## Outcome

- Signal: useful

## Source Nodes

- CONTEXT.md,backend