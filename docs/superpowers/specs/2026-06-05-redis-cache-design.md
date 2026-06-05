# Redis 查询缓存设计

## 概述

在 Service 层为所有 List 查询接口添加 Redis 缓存。采用 TTL 过期（默认 60 秒）+ 写入时主动失效的双重策略。

## 目标

- 降低重复 List 查询的数据库压力
- 通过 TTL + 事件驱动的缓存失效保证数据新鲜度
- 最小化代码改动，遵循现有项目规范

## 范围

- **缓存**：5 个事件类型的 List 接口（分页查询）
- **不缓存**：GetByID（单行查询开销小）、health 接口、写入操作

## 架构

```
main.go (redis client)
  └─ api.RegisterRoutes(r, db, redis)
       └─ service.NewXxxService(repo, redis)
            └─ service.List() → cache.Get / cache.Set
            └─ cache.DeleteByPrefix（写入时）

listener handler.Handle()
  └─ repo.Create() → cache.DeleteByPrefix(ctx, rdb, "{event}:list:")
```

## 新增文件：`internal/cache/cache.go`

通用缓存工具，提供以下函数：

```go
// Get 读取并反序列化缓存值。命中时返回 (value, true, nil)。
func Get[T any](ctx context.Context, rdb *redis.Client, key string) (T, bool, error)

// Set 序列化并存储值，带 TTL。
func Set(ctx context.Context, rdb *redis.Client, key string, value any, ttl time.Duration) error

// DeleteByPrefix 通过 SCAN + DEL 删除匹配前缀的所有 key。
func DeleteByPrefix(ctx context.Context, rdb *redis.Client, prefix string) error

// BuildListKey 根据事件类型和查询参数生成缓存 key。
// 格式："{eventType}:list:{md5(sorted_params)}"
func BuildListKey(eventType string, query any) string
```

`BuildListKey` 将 query 结构体序列化为 JSON，排序后取 MD5 哈希。

## 缓存 Key 格式

| 类型 | Key 模式 | 示例 |
|------|----------|------|
| List | `{eventType}:list:{hash}` | `staked:list:a1b2c3d4...` |
| 失效前缀 | `{eventType}:list:*` | `staked:list:*` |

## TTL 配置

在 `config.yaml` 的 `redis` 下新增 `cache_ttl`：

```yaml
redis:
  addr: localhost:6379
  password:
  db: 0
  cache_ttl: 60  # 秒，默认 60
```

更新 `internal/config/config.go` 的 Redis 结构体：

```go
type Redis struct {
    Addr     string `mapstructure:"addr"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
    CacheTTL int    `mapstructure:"cache_ttl"` // 秒
}
```

## Service 层改造

每个 service 新增 `rdb *redis.Client` 和 `cacheTTL time.Duration` 字段。

示例（`StakedEventService`）：

```go
type StakedEventService struct {
    repo     *repository.StakedEventRepository
    rdb      *redis.Client
    cacheTTL time.Duration
}

func (s *StakedEventService) List(ctx context.Context, query repository.StakedEventQuery) (*StakedEventListResult, error) {
    key := cache.BuildListKey("staked", query)

    // 1. 尝试读缓存
    if result, ok, err := cache.Get[StakedEventListResult](ctx, s.rdb, key); err == nil && ok {
        return &result, nil
    }

    // 2. 查数据库
    events, total, err := s.repo.List(ctx, query)
    if err != nil {
        return nil, err
    }
    result := &StakedEventListResult{Items: events, Total: total, Page: query.Page, PageSize: query.PageSize}

    // 3. 写缓存
    _ = cache.Set(ctx, s.rdb, key, result, s.cacheTTL)
    return result, nil
}
```

5 个 service 均按此模式改造：`StakedEventService`、`RewardClaimedEventService`、`WithdrawnEventService`、`MinStakeAmountUpdatedEventService`、`RewardRateUpdatedEventService`。

## Listener Handler 改造

每个 listener handler 新增 `rdb *redis.Client` 字段。`repo.Create()` 成功后调用：

```go
cache.DeleteByPrefix(ctx, h.rdb, "staked:list:")
```

确保下次 List 查询能拿到最新数据。

## 依赖注入改动

### `main.go`

- 将 `redisClient` 传给 `api.RegisterRoutes(r, db, redisClient)`
- 将 `redisClient` 传给每个 listener handler 的构造函数

### `internal/api/router.go`

- `RegisterRoutes` 签名改为：`func RegisterRoutes(r *gin.Engine, db *gorm.DB, rdb *redis.Client) error`
- 将 `rdb` 和 `cacheTTL` 传给每个 `service.NewXxxService(repo, rdb, cacheTTL)`

## 改动文件清单

| 文件 | 改动 |
|------|------|
| `internal/cache/cache.go` | **新增** — 通用缓存工具 |
| `internal/cache/cache_test.go` | **新增** — 单元测试 |
| `internal/config/config.go` | Redis 结构体新增 `CacheTTL` |
| `config.yaml` | 新增 `cache_ttl: 60` |
| `internal/service/staked_event_service.go` | 新增 rdb/cacheTTL 字段，List 方法加缓存 |
| `internal/service/reward_claimed_event_service.go` | 同上 |
| `internal/service/withdrawn_event_service.go` | 同上 |
| `internal/service/min_stake_amount_updated_event_service.go` | 同上 |
| `internal/service/reward_rate_updated_event_service.go` | 同上 |
| `internal/listener/staked_event_handler.go` | 新增 rdb 字段，Create 后删缓存 |
| `internal/listener/reward_claimed_event_handler.go` | 同上 |
| `internal/listener/withdrawn_event_handler.go` | 同上 |
| `internal/listener/min_stake_amount_updated_event_handler.go` | 同上 |
| `internal/listener/reward_rate_updated_event_handler.go` | 同上 |
| `internal/api/router.go` | 传递 rdb 给 service |
| `main.go` | 传递 rdb 给 RegisterRoutes 和 listener handler |

## 错误处理

- 缓存读取失败：打印警告日志，降级查数据库（缓存是尽力而为）
- 缓存写入失败：打印警告日志，直接返回数据库结果（不影响请求）
- `DeleteByPrefix` 失败：打印警告日志，不阻塞事件处理
- Redis 不可用：所有请求降级到数据库，不影响正确性

## 测试

- `cache.BuildListKey` 单元测试（验证哈希确定性）
- `cache.Get`/`cache.Set`/`cache.DeleteByPrefix` 单元测试（需要 redis 或 mock）
- 集成测试：List → 缓存命中 → 写入事件 → 缓存失效 → List 返回最新数据
