## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- When graphify-out/graph.json exists, run `graphify query "<question>"` before your first grep, search or source-file read for a codebase question, and put this rule in every subagent prompt that explores code. Read raw files only for lines the graph or an earlier read already located. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
- A committed graphify-out/graph.json is rewritten whole on every build, so a large diff to it in a pull request is expected. Do not flag it as churn or suggest skipping the graph rebuild or its hooks.

## Panduan sesi berikutnya (MungkasDesk)

- `graphify-out/` sudah terisi — jangan jalankan `graphify extract` ulang dari nol; cukup `graphify update .` setelah mengubah kode (AST-only, gratis).
- Sebelum menelusuri topik yang pernah dikerjakan, cek `graphify-out/reflections/LESSONS.md` — temuan yang sudah terverifikasi tersimpan di sana (mis. cara apply perubahan schema DB, kebiasaan membaca error body PowerShell 5.1).
- Temuan baru yang sudah diverifikasi disimpan dengan `graphify save-result --question "..." --answer "..." --nodes "..." --outcome useful`, lalu diakhiri `graphify reflect --graph graphify-out/graph.json`.
- CLI diinstal via `uv tool install graphifyy`. Kalau perintah `graphify` tidak dikenali di sesi shell baru, prefix PATH-nya: `$env:PATH = "C:\Users\irvansah.pamungkas\.local\bin;$env:PATH"`.
