# Redis Query Cache Design

## Overview

Add Redis caching to all List query endpoints at the service layer. Uses TTL-based expiration (60s default) with write-through invalidation when new events are persisted.

## Goals

- Reduce database load for repeated List queries
- Maintain data freshness via TTL + event-driven cache invalidation
- Minimal code changes, follow existing project conventions

## Scope

- **Cached**: All 5 event type List endpoints (paginated queries)
- **Not cached**: GetByID (single-row lookup is fast), health endpoint, write operations

## Architecture

```
main.go (redis client)
  └─ api.RegisterRoutes(r, db, redis)
       └─ service.NewXxxService(repo, redis)
            └─ service.List() → cache.Get / cache.Set
            └─ cache.DeleteByPrefix (on write)

listener handler.Handle()
  └─ repo.Create() → cache.DeleteByPrefix(ctx, rdb, "{event}:list:")
```

## New File: `internal/cache/cache.go`

Generic cache utility with three functions:

```go
// Get retrieves and deserializes a cached value. Returns (value, true, nil) on hit.
func Get[T any](ctx context.Context, rdb *redis.Client, key string) (T, bool, error)

// Set serializes and stores a value with TTL.
func Set(ctx context.Context, rdb *redis.Client, key string, value any, ttl time.Duration) error

// DeleteByPrefix removes all keys matching a prefix using SCAN + DEL.
func DeleteByPrefix(ctx context.Context, rdb *redis.Client, prefix string) error

// BuildListKey generates a cache key from event type and query parameters.
// Format: "{eventType}:list:{md5(sorted_params)}"
func BuildListKey(eventType string, query any) string
```

`BuildListKey` marshals the query struct to JSON, sorts keys, and takes MD5 hash.

## Cache Key Format

| Type | Key Pattern | Example |
|------|-------------|---------|
| List | `{eventType}:list:{hash}` | `staked:list:a1b2c3d4...` |
| Prefix for invalidation | `{eventType}:list:*` | `staked:list:*` |

## TTL Configuration

Add `cache_ttl` to `config.yaml` under `redis`:

```yaml
redis:
  addr: localhost:6379
  password:
  db: 0
  cache_ttl: 60  # seconds, default 60
```

Update `internal/config/config.go` Redis struct:

```go
type Redis struct {
    Addr     string `mapstructure:"addr"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
    CacheTTL int    `mapstructure:"cache_ttl"` // seconds
}
```

## Service Layer Changes

Each service gets a `rdb *redis.Client` and `cacheTTL time.Duration` field.

Example (`StakedEventService`):

```go
type StakedEventService struct {
    repo     *repository.StakedEventRepository
    rdb      *redis.Client
    cacheTTL time.Duration
}

func (s *StakedEventService) List(ctx context.Context, query repository.StakedEventQuery) (*StakedEventListResult, error) {
    key := cache.BuildListKey("staked", query)

    // 1. Try cache
    if result, ok, err := cache.Get[StakedEventListResult](ctx, s.rdb, key); err == nil && ok {
        return &result, nil
    }

    // 2. Query DB
    events, total, err := s.repo.List(ctx, query)
    if err != nil {
        return nil, err
    }
    result := &StakedEventListResult{Items: events, Total: total, Page: query.Page, PageSize: query.PageSize}

    // 3. Write cache
    _ = cache.Set(ctx, s.rdb, key, result, s.cacheTTL)
    return result, nil
}
```

Same pattern for all 5 services: `StakedEventService`, `RewardClaimedEventService`, `WithdrawnEventService`, `MinStakeAmountUpdatedEventService`, `RewardRateUpdatedEventService`.

## Listener Handler Changes

Each listener handler gains a `rdb *redis.Client` field. After successful `repo.Create()`, call:

```go
cache.DeleteByPrefix(ctx, h.rdb, "staked:list:")
```

This ensures the next List query sees fresh data.

## Wiring Changes

### `main.go`

- Pass `redisClient` to `api.RegisterRoutes(r, db, redisClient)`
- Pass `redisClient` to each listener handler constructor

### `internal/api/router.go`

- `RegisterRoutes` signature: `func RegisterRoutes(r *gin.Engine, db *gorm.DB, rdb *redis.Client) error`
- Pass `rdb` to each `service.NewXxxService(repo, rdb, cacheTTL)`

## Files Modified

| File | Change |
|------|--------|
| `internal/cache/cache.go` | **New** — generic cache utility |
| `internal/cache/cache_test.go` | **New** — unit tests |
| `internal/config/config.go` | Add `CacheTTL` to Redis struct |
| `config.yaml` | Add `cache_ttl: 60` |
| `internal/service/staked_event_service.go` | Add rdb/cacheTTL fields, cache in List |
| `internal/service/reward_claimed_event_service.go` | Same |
| `internal/service/withdrawn_event_service.go` | Same |
| `internal/service/min_stake_amount_updated_event_service.go` | Same |
| `internal/service/reward_rate_updated_event_service.go` | Same |
| `internal/listener/staked_event_handler.go` | Add rdb field, delete cache on Create |
| `internal/listener/reward_claimed_event_handler.go` | Same |
| `internal/listener/withdrawn_event_handler.go` | Same |
| `internal/listener/min_stake_amount_updated_event_handler.go` | Same |
| `internal/listener/reward_rate_updated_event_handler.go` | Same |
| `internal/api/router.go` | Pass rdb to services |
| `main.go` | Pass rdb to RegisterRoutes and listener handlers |

## Error Handling

- Cache read failures: log warning, fall through to DB query (cache is best-effort)
- Cache write failures: log warning, return DB result (don't fail the request)
- `DeleteByPrefix` failures: log warning, don't block event processing
- Redis down: all requests fall through to DB, no impact on correctness

## Testing

- Unit tests for `cache.BuildListKey` (deterministic hashing)
- Unit tests for `cache.Get`/`cache.Set`/`cache.DeleteByPrefix` (requires redis or mock)
- Integration test: List → cache hit → write event → cache invalidated → List returns fresh data
