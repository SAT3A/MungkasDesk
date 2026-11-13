# Graph Report - MungkasDesk  (2026-11-13)

## Corpus Check
- 12 files · ~13,699 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 3 file(s) not represented in the graph (top: (none) 1, .example 1, .jsonc 1)

## Summary
- 85 nodes · 106 edges · 10 communities (7 shown, 3 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 11 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `d8955fff`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- database.go
- github.com/gin-gonic/gin.Context
- validate.go
- model.go
- package.json
- mungkasdesk/backend
- handlers_auth.go
- handlers_users.go
- helpers.js
- run.js

## God Nodes (most connected - your core abstractions)
1. `scripts` - 4 edges
2. `createUser()` - 4 edges
3. `parseUserID()` - 4 edges
4. `playwright` - 3 edges
5. `getUserByID()` - 3 edges
6. `updateUser()` - 3 edges
7. `getExtraHeadersFromEnv()` - 2 edges
8. `createContext()` - 2 edges
9. `playwright@^1.62.0` - 2 edges
10. `engines` - 2 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `connectDB()`  [INFERRED]
  backend/cmd/api/main.go → backend/cmd/api/database.go
- `getUserByID()` --calls--> `parseUserID()`  [INFERRED]
  backend/cmd/api/handlers_users.go → backend/cmd/api/validate.go
- `updateUser()` --calls--> `parseUserID()`  [INFERRED]
  backend/cmd/api/handlers_users.go → backend/cmd/api/validate.go
- `createUser()` --calls--> `containsSymbol()`  [INFERRED]
  backend/cmd/api/handlers_users.go → backend/cmd/api/validate.go
- `createUser()` --calls--> `containsUpper()`  [INFERRED]
  backend/cmd/api/handlers_users.go → backend/cmd/api/validate.go

## Import Cycles
- None detected.

## Communities (10 total, 3 thin omitted)

### Community 0 - "database.go"
Cohesion: 0.18
Nodes (3): connectDB(), getHealth(), main()

### Community 1 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.83
Nodes (3): getUserByID(), updateUser(), parseUserID()

### Community 2 - "validate.go"
Cohesion: 0.33
Nodes (3): createUser(), containsSymbol(), containsUpper()

### Community 3 - "model.go"
Cohesion: 0.29
Nodes (4): CreateUserInput, LoginInput, UpdateUserInput, User

### Community 4 - "package.json"
Cohesion: 0.12
Nodes (16): author, dependencies, playwright@^1.62.0, description, engines, node, keywords, license (+8 more)

### Community 8 - "helpers.js"
Cohesion: 0.15
Nodes (7): { chromium, firefox, webkit }, createContext(), fs, getExtraHeadersFromEnv(), http, os, path

### Community 9 - "run.js"
Cohesion: 0.17
Nodes (5): args, fs, nodeModules, path, { spawn }

## Knowledge Gaps
- **1 isolated node(s):** `mungkasdesk/backend`
  These have ≤1 connection - possible missing edges. (Counts symbols only; 44 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `playwright` connect `package.json` to `helpers.js`?**
  _High betweenness centrality (0.121) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `createUser()` (e.g. with `containsSymbol()` and `containsUpper()`) actually correct?**
  _`createUser()` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `mungkasdesk/backend` to the rest of the system?**
  _1 weakly-connected nodes found - possible missing edges._
- **Should `package.json` be split into smaller, more focused modules?**
  _Cohesion score 0.125 - nodes in this community are weakly interconnected._
- **Why does `createUser()` connect `validate.go` to `github.com/gin-gonic/gin.Context`, `handlers_users.go`?**
  _High betweenness centrality (0.018) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `parseUserID()` (e.g. with `getUserByID()` and `updateUser()`) actually correct?**
  _`parseUserID()` has 2 INFERRED edges - model-reasoned connections that need verification._