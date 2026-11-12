# Graph Report - MungkasDesk  (2026-11-12)

## Corpus Check
- 3 files · ~15,031 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 3 file(s) not represented in the graph (top: (none) 1, .example 1, .jsonc 1)

## Summary
- 38 nodes · 49 edges · 6 communities (3 shown, 3 thin omitted)
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `e59c517f`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- github.com/gin-gonic/gin.Context
- createUser
- User
- connectDB
- mungkasdesk/backend

## God Nodes (most connected - your core abstractions)
1. `parseUserID()` - 5 edges
2. `createUser()` - 4 edges
3. `getUserByID()` - 3 edges
4. `updateUser()` - 3 edges
5. `softDeleteUser()` - 3 edges
6. `User` - 2 edges
7. `connectDB()` - 2 edges
8. `getHealth()` - 2 edges
9. `getUsers()` - 2 edges
10. `containsUpper()` - 2 edges

## Surprising Connections (you probably didn't know these)
- `createUser()` --references--> `github.com/gin-gonic/gin.Context`  [EXTRACTED]
  backend/cmd/api/main.go →   _Bridges community 2 → community 1_

## Import Cycles
- None detected.

## Communities (6 total, 3 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.09
Nodes (3): CreateUserInput, LoginInput, UpdateUserInput

### Community 1 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.36
Nodes (7): getHealth(), getUserByID(), getUsers(), login(), parseUserID(), softDeleteUser(), updateUser()

### Community 2 - "createUser"
Cohesion: 0.67
Nodes (3): containsSymbol(), containsUpper(), createUser()

## Knowledge Gaps
- **1 isolated node(s):** `mungkasdesk/backend`
  These have ≤1 connection - possible missing edges. (Counts symbols only; 23 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `User` connect `User` to `main.go`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **What connects `mungkasdesk/backend` to the rest of the system?**
  _1 weakly-connected nodes found - possible missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09090909090909091 - nodes in this community are weakly interconnected._
- **Why does `createUser()` connect `createUser` to `main.go`, `github.com/gin-gonic/gin.Context`?**
  _High betweenness centrality (0.009) - this node is a cross-community bridge._
- **Why does `parseUserID()` connect `github.com/gin-gonic/gin.Context` to `main.go`?**
  _High betweenness centrality (0.006) - this node is a cross-community bridge._