# AGENTS.md

Guidance for AI coding assistants (Claude Code, Cursor, Copilot, etc.) working in this
repository. Read this before editing. It captures the architecture, conventions, and
non-obvious gotchas that are easy to get wrong.

## What this library is

`yaaf-common-redis` is the **Redis-backed implementation** of three abstract interfaces
defined in the parent framework [`go-yaaf/yaaf-common`](https://github.com/go-yaaf/yaaf-common):

| Interface | Package in yaaf-common | What it provides |
|-----------|------------------------|------------------|
| `database.IDataCache` | `database` | Key/value + hash + list cache operations over entities |
| `messaging.IMessageBus` | `messaging` | Pub/Sub, work queues, producers & consumers |
| `database.ILocker` | `database` | Distributed mutual-exclusion lock |

A **single struct — `RedisAdapter` — implements both `IDataCache` and `IMessageBus`.**
`NewRedisDataCache` and `NewRedisMessageBus` return the same underlying object type; pick the
factory that matches the interface you want. This library is a **dependency of other services**,
not a standalone app — treat exported signatures as a public API and avoid breaking changes.

## ⚠️ Critical gotcha: directory name ≠ package name

The Go source lives in the `redis/` directory, but the **package is named `facilities`**:

```go
package facilities   // in every file under redis/
```

So consumers import the path `.../redis` but reference symbols via `facilities` (often aliased):

```go
import cr "github.com/go-yaaf/yaaf-common-redis/redis"   // path ends in /redis
// ...
cache, err := cr.NewRedisDataCache("redis://localhost:6379/0")   // package identifier is facilities/alias
```

Do **not** "fix" the package name to `redis` — it would collide with the imported
`github.com/redis/go-redis/v9` package and break every file.

## Repository layout

```
redis/
  redis_adapter.go       # RedisAdapter struct, factory methods, connection helpers, (de)serialization helpers
  redis_datacache.go     # IDataCache impl: key / hash / list operations + ObtainLocker
  redis_message_bus.go   # IMessageBus impl: Publish/Subscribe, Push/Pop queue, producer & consumer
  redis_locker.go        # ILocker impl: distributed lock (Lua-scripted Refresh/Release/TTL)
test/                    # Integration tests — REQUIRE a live Redis (see "Testing")
  docker-compose.yml     # Local Redis for tests
  test_cluster/          # Cluster variant compose
examples/                # Runnable usage examples (pubsub, message_queue)
llms.txt / README.md     # Human/LLM-facing docs — keep in sync with code changes
```

## Core concepts & conventions

### Entities and serialization
- Values are **`entity.Entity`** objects from yaaf-common, serialized with the framework's
  `entity.Marshal` / `entity.Unmarshal` (standard-library JSON under the hood).
- Reads take an **`EntityFactory`** (a `func() Entity`) so the adapter can construct the concrete
  type before unmarshalling. Messages use `MessageFactory` the same way.
- Every typed method has a **`...Raw` sibling** that skips (de)serialization and works on `[]byte`
  (e.g. `Get`/`GetRaw`, `Set`/`SetRaw`, `HGet`/`HGetRaw`). Keep this pairing when adding methods.

### The three feature areas (grouped with `// region` comments)
1. **Data cache** (`redis_datacache.go`): keys (`Get/Set/Del/Scan/Exists/Rename`), hashes
   (`HGet/HSet/HDel/...`), lists (`RPush/LPush/RPop/LPop/BRPop/BLPop/LRange/LLen`).
2. **Message bus** (`redis_message_bus.go`):
   - **Pub/Sub** (fan-out, non-durable): `Publish` + `Subscribe` (callback) / `CreateConsumer` (`Read`).
     A topic containing `*` triggers Redis **pattern** subscription (`PSubscribe`).
   - **Queue** (point-to-point, durable-ish via lists): `Push` (LPush) + `Pop` (RPop/BRPop).
3. **Distributed lock** (`redis_locker.go`): `RedisAdapter.ObtainLocker(key, ttl)` → `Locker`
   with `Refresh` / `Release` / `TTL`, each guarded by a Lua script that verifies the caller's
   token before mutating, so a process can only release/extend a lock **it** holds.

### Code style (match the existing code)
- **Dot imports** of yaaf-common subpackages are the established pattern
  (`. "github.com/go-yaaf/yaaf-common/entity"`, `.../database`, `.../messaging`). Follow it.
- Heavy **`if x, err := ...; err != nil { ... } else { ... }`** nesting is the house idiom.
- Return the **actual error** — return `cmd.Err()` / the inner `er`, never a stale outer `err`.
- Sections are delimited by `// region ... ` / `// endregion` comments.

## Build, vet, and test

```bash
go build ./...        # must pass
go vet ./...          # must pass (also compiles the tests)
```

**Tests are integration tests and need a running Redis** on `localhost:6379`:

```bash
docker compose -f test/docker-compose.yml up -d   # start Redis (loopback-bound)
go test ./...                                      # run tests
docker compose -f test/docker-compose.yml down
```

- Tests call `skipCI(t)` (in `test/def_test.go`) which **skips when the `CI` env var is set**, so
  the GitHub Actions build does not require a Redis service. Do not remove those guards.
- There are **no unit tests inside the `redis/` package** — coverage lives in `test/`.

## Security conventions (do not regress these)

This library was hardened; keep the invariants below when editing.

- **Lock tokens must be cryptographically unguessable.** `ObtainLocker` uses `GUID()` (UUID v4),
  **not** timestamp-based `ID()`/`NanoID()`. The token is the lock's ownership credential — a
  predictable token lets other processes forge `Release`/`Refresh`. Never weaken this.
- **Bound goroutines on message receipt.** The pub/sub `subscriber` dispatches callbacks through a
  semaphore (`maxConcurrentHandlers`) to prevent unbounded goroutine growth (DoS) under load.
  Preserve the bound if you touch that loop.
- **Never log the connection URI verbatim** — it embeds the password. `RedisAdapter` implements
  `fmt.Stringer` via `redactURI` to mask it; use `redactURI` for any new URI output.
- **Production connections should use `rediss://` (TLS) with auth.** Docs and compose files show
  plaintext `redis://localhost` for **local dev only**; compose ports are bound to `127.0.0.1`.
- **Do not commit** `.DS_Store`, `.idea/`, or other OS/editor artifacts (already in `.gitignore`).
- CI runs `go vet` and `govulncheck`; keep both green. Stdlib CVEs are fixed by **upgrading the Go
  toolchain**, not by code changes.

## Dependencies

- `github.com/go-yaaf/yaaf-common` — the parent framework (interfaces, entities, serialization,
  `logger`, id generators). Interface changes there ripple here.
- `github.com/redis/go-redis/v9` — the Redis client (`redis.ParseURL`, `redis.Client`, `redis.NewScript`).
- `github.com/stretchr/testify` — test assertions.
- Go **1.25+** (see `go.mod`).

## When you change behavior

Update **`README.md`** and **`llms.txt`** alongside code — they are the human- and LLM-facing
contract and are expected to stay in sync with the exported API and connection-string guidance.
