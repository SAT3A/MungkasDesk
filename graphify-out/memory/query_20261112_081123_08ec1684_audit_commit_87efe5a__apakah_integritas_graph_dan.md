---
type: "query"
date: "2026-11-12T08:11:23.425998+00:00"
question: "Audit commit 87efe5a: apakah integritas graph dan lokasi simbol valid?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["backend_cmd_api_main"]
---

# Q: Audit commit 87efe5a: apakah integritas graph dan lokasi simbol valid?

## Answer

38 node dan 49 relasi; tidak ada ID duplikat, dangling edge, atau source node yang hilang. 18 anchor source cocok. ID node dan pasangan edge HTML cocok dengan graph.json; kode HTML di luar data tidak berubah. Semua enam JSON dalam diff valid. Scope hanya artefak Graphify, bukan audit runtime aplikasi.

## Outcome

- Signal: useful

## Source Nodes

- backend_cmd_api_main