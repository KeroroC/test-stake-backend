# Redis 查询缓存实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为所有 List 查询接口添加 Redis 缓存，降低数据库压力。

**Architecture:** 在 Service 层通过通用缓存工具（`internal/cache/cache.go`）实现缓存读写。List 查询先尝试 Redis，未命中则查数据库并回写缓存。Listener 写入新事件后主动删除对应前缀的缓存 key。TTL 默认 60 秒，可通过 config.yaml 配置。

**Tech Stack:** Go 1.26, github.com/redis/go-redis/v9, Gin, GORM

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `internal/cache/cache.go` | **新增** — 通用缓存工具：Get/Set/DeleteByPrefix/BuildListKey |
| `internal/cache/cache_test.go` | **新增** — 缓存工具单元测试 |
| `internal/config/config.go` | **修改** — Redis 结构体新增 CacheTTL 字段 |
| `config.yaml` | **修改** — 新增 cache_ttl 配置项 |
| `internal/service/staked_event_service.go` | **修改** — List 方法加缓存逻辑 |
| `internal/service/reward_claimed_event_service.go` | **修改** — 同上 |
| `internal/service/withdrawn_event_service.go` | **修改** — 同上 |
| `internal/service/min_stake_amount_updated_event_service.go` | **修改** — 同上 |
| `internal/service/reward_rate_updated_event_service.go` | **修改** — 同上 |
| `internal/listener/staked_event_handler.go` | **修改** — Handle 中 Create 后删缓存 |
| `internal/listener/reward_claimed_event_handler.go` | **修改** — 同上 |
| `internal/listener/withdrawn_event_handler.go` | **修改** — 同上 |
| `internal/listener/min_stake_amount_updated_event_handler.go` | **修改** — 同上 |
| `internal/listener/reward_rate_updated_event_handler.go` | **修改** — 同上 |
| `internal/api/router.go` | **修改** — RegisterRoutes 接收并传递 redis client |
| `main.go` | **修改** — 传递 redis client 给 RegisterRoutes 和 listener handler |

---

### Task 1: 新增缓存工具 `internal/cache/cache.go`

**Files:**
- Create: `internal/cache/cache.go`
- Test: `internal/cache/cache_test.go`

- [ ] **Step 1: 编写缓存工具测试**

```go
// internal/cache/cache_test.go
package cache

import (
	"testing"
)

func TestBuildListKey_Deterministic(t *testing.T) {
	query := struct {
		Page     int
		PageSize int
		User     string
	}{Page: 1, PageSize: 20, User: "0xabc"}

	key1 := BuildListKey("staked", query)
	key2 := BuildListKey("staked", query)
	if key1 != key2 {
		t.Errorf("BuildListKey not deterministic: %s != %s", key1, key2)
	}
	if key1 == "" {
		t.Error("BuildListKey returned empty string")
	}
}

func TestBuildListKey_DifferentInputs(t *testing.T) {
	q1 := struct {
		Page     int
		PageSize int
	}{Page: 1, PageSize: 20}
	q2 := struct {
		Page     int
		PageSize int
	}{Page: 2, PageSize: 20}

	k1 := BuildListKey("staked", q1)
	k2 := BuildListKey("staked", q2)
	if k1 == k2 {
		t.Errorf("different queries should produce different keys, both got %s", k1)
	}
}

func TestBuildListKey_DifferentEventTypes(t *testing.T) {
	query := struct {
		Page int
	}{Page: 1}

	k1 := BuildListKey("staked", query)
	k2 := BuildListKey("withdrawn", query)
	if k1 == k2 {
		t.Errorf("different event types should produce different keys, both got %s", k1)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/cache/ -v`
Expected: FAIL — `internal/cache` 包不存在

- [ ] **Step 3: 实现缓存工具**

```go
// internal/cache/cache.go
package cache

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Get 从 Redis 读取并反序列化缓存值。命中返回 (value, true, nil)。
func Get[T any](ctx context.Context, rdb *redis.Client, key string) (T, bool, error) {
	var zero T
	data, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("cache get %s: %w", key, err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return zero, false, fmt.Errorf("cache unmarshal %s: %w", key, err)
	}

	return result, true, nil
}

// Set 序列化并存储值到 Redis，带 TTL。
func Set(ctx context.Context, rdb *redis.Client, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal %s: %w", key, err)
	}

	if err := rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("cache set %s: %w", key, err)
	}

	return nil
}

// DeleteByPrefix 通过 SCAN + DEL 删除匹配前缀的所有 key。
func DeleteByPrefix(ctx context.Context, rdb *redis.Client, prefix string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("cache scan %s: %w", prefix, err)
		}

		if len(keys) > 0 {
			if err := rdb.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("cache del %s: %w", prefix, err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// BuildListKey 根据事件类型和查询参数生成缓存 key。
// 格式："{eventType}:list:{md5(json)}"
func BuildListKey(eventType string, query any) string {
	data, err := json.Marshal(query)
	if err != nil {
		log.Printf("cache: failed to marshal query for key building: %v", err)
		return fmt.Sprintf("%s:list:error", eventType)
	}
	hash := md5.Sum(data)
	return fmt.Sprintf("%s:list:%x", eventType, hash)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/cache/ -v`
Expected: PASS — 3 个测试全部通过

- [ ] **Step 5: 提交**

```bash
git add internal/cache/cache.go internal/cache/cache_test.go
git commit -m "feat(cache): 新增通用缓存工具 Get/Set/DeleteByPrefix/BuildListKey"
```

---

### Task 2: 更新配置 — 新增 CacheTTL

**Files:**
- Modify: `internal/config/config.go`
- Modify: `config.yaml`

- [ ] **Step 1: 修改配置结构体**

在 `internal/config/config.go` 的 `Redis` 结构体中新增 `CacheTTL` 字段：

```go
type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	CacheTTL int    `mapstructure:"cache_ttl"` // 缓存过期时间（秒），默认 60
}
```

- [ ] **Step 2: 修改 config.yaml**

在 `redis` 配置段新增 `cache_ttl`：

```yaml
redis:
  addr: localhost:6379
  password:
  db: 0
  cache_ttl: 60
```

- [ ] **Step 3: 确认编译通过**

Run: `go build ./...`
Expected: 编译成功

- [ ] **Step 4: 提交**

```bash
git add internal/config/config.go config.yaml
git commit -m "feat(config): Redis 配置新增 cache_ttl 字段"
```

---

### Task 3: 改造 StakedEventService — List 加缓存

**Files:**
- Modify: `internal/service/staked_event_service.go`

- [ ] **Step 1: 改造 StakedEventService**

将 `internal/service/staked_event_service.go` 改为：

```go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type StakedEventListResult struct {
	Items    []models.StakedEvent `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

type StakedEventService struct {
	repo     *repository.StakedEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewStakedEventService(repo *repository.StakedEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*StakedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create staked event service: repository is nil")
	}

	return &StakedEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *StakedEventService) GetByID(ctx context.Context, id int64) (*models.StakedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *StakedEventService) List(ctx context.Context, query repository.StakedEventQuery) (*StakedEventListResult, error) {
	key := cache.BuildListKey("staked", query)

	// 尝试读缓存
	if result, ok, err := cache.Get[StakedEventListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get staked list: %v", err)
	}

	// 查数据库
	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &StakedEventListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	// 写缓存
	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set staked list: %v", err)
	}

	return result, nil
}
```

- [ ] **Step 2: 确认编译通过**

Run: `go build ./...`
Expected: 编译失败 — `router.go` 中 `NewStakedEventService` 调用参数不匹配（预期行为，后续 Task 修复）

- [ ] **Step 3: 提交**

```bash
git add internal/service/staked_event_service.go
git commit -m "feat(service): StakedEventService List 方法加入 Redis 缓存"
```

---

### Task 4: 改造其余 4 个 Service — List 加缓存

**Files:**
- Modify: `internal/service/reward_claimed_event_service.go`
- Modify: `internal/service/withdrawn_event_service.go`
- Modify: `internal/service/min_stake_amount_updated_event_service.go`
- Modify: `internal/service/reward_rate_updated_event_service.go`

- [ ] **Step 1: 改造 RewardClaimedEventService**

将 `internal/service/reward_claimed_event_service.go` 改为：

```go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type RewardClaimedListResult struct {
	Items    []models.RewardClaimedEvent `json:"items"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
}

type RewardClaimedEventService struct {
	repo     *repository.RewardClaimedEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewRewardClaimedEventService(repo *repository.RewardClaimedEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*RewardClaimedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward claimed event service: repository is nil")
	}

	return &RewardClaimedEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *RewardClaimedEventService) GetByID(ctx context.Context, id int64) (*models.RewardClaimedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RewardClaimedEventService) List(ctx context.Context, query repository.RewardClaimedEventQuery) (*RewardClaimedListResult, error) {
	key := cache.BuildListKey("reward-claimed", query)

	if result, ok, err := cache.Get[RewardClaimedListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get reward-claimed list: %v", err)
	}

	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &RewardClaimedListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set reward-claimed list: %v", err)
	}

	return result, nil
}
```

- [ ] **Step 2: 改造 WithdrawnEventService**

将 `internal/service/withdrawn_event_service.go` 改为：

```go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type WithdrawnListResult struct {
	Items    []models.WithdrawnEvent `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

type WithdrawnEventService struct {
	repo     *repository.WithdrawnEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewWithdrawnEventService(repo *repository.WithdrawnEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*WithdrawnEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create withdrawn event service: repository is nil")
	}

	return &WithdrawnEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *WithdrawnEventService) GetByID(ctx context.Context, id int64) (*models.WithdrawnEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *WithdrawnEventService) List(ctx context.Context, query repository.WithdrawnEventQuery) (*WithdrawnListResult, error) {
	key := cache.BuildListKey("withdrawn", query)

	if result, ok, err := cache.Get[WithdrawnListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get withdrawn list: %v", err)
	}

	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &WithdrawnListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set withdrawn list: %v", err)
	}

	return result, nil
}
```

- [ ] **Step 3: 改造 MinStakeAmountUpdatedEventService**

将 `internal/service/min_stake_amount_updated_event_service.go` 改为：

```go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type MinStakeAmountUpdatedListResult struct {
	Items    []models.MinStakeAmountUpdatedEvent `json:"items"`
	Total    int64                               `json:"total"`
	Page     int                                 `json:"page"`
	PageSize int                                 `json:"page_size"`
}

type MinStakeAmountUpdatedEventService struct {
	repo     *repository.MinStakeAmountUpdatedEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewMinStakeAmountUpdatedEventService(repo *repository.MinStakeAmountUpdatedEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*MinStakeAmountUpdatedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create min stake amount updated event service: repository is nil")
	}

	return &MinStakeAmountUpdatedEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *MinStakeAmountUpdatedEventService) GetByID(ctx context.Context, id int64) (*models.MinStakeAmountUpdatedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *MinStakeAmountUpdatedEventService) List(ctx context.Context, query repository.MinStakeAmountUpdatedEventQuery) (*MinStakeAmountUpdatedListResult, error) {
	key := cache.BuildListKey("min-stake-amount-updated", query)

	if result, ok, err := cache.Get[MinStakeAmountUpdatedListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get min-stake-amount-updated list: %v", err)
	}

	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &MinStakeAmountUpdatedListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set min-stake-amount-updated list: %v", err)
	}

	return result, nil
}
```

- [ ] **Step 4: 改造 RewardRateUpdatedEventService**

将 `internal/service/reward_rate_updated_event_service.go` 改为：

```go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type RewardRateUpdatedListResult struct {
	Items    []models.RewardRateUpdatedEvent `json:"items"`
	Total    int64                           `json:"total"`
	Page     int                             `json:"page"`
	PageSize int                             `json:"page_size"`
}

type RewardRateUpdatedEventService struct {
	repo     *repository.RewardRateUpdatedEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewRewardRateUpdatedEventService(repo *repository.RewardRateUpdatedEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*RewardRateUpdatedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward rate updated event service: repository is nil")
	}

	return &RewardRateUpdatedEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *RewardRateUpdatedEventService) GetByID(ctx context.Context, id int64) (*models.RewardRateUpdatedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RewardRateUpdatedEventService) List(ctx context.Context, query repository.RewardRateUpdatedEventQuery) (*RewardRateUpdatedListResult, error) {
	key := cache.BuildListKey("reward-rate-updated", query)

	if result, ok, err := cache.Get[RewardRateUpdatedListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get reward-rate-updated list: %v", err)
	}

	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &RewardRateUpdatedListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set reward-rate-updated list: %v", err)
	}

	return result, nil
}
```

- [ ] **Step 5: 确认编译通过**

Run: `go build ./...`
Expected: 编译失败 — router.go 和 main.go 中构造函数参数不匹配（预期行为，后续 Task 修复）

- [ ] **Step 6: 提交**

```bash
git add internal/service/
git commit -m "feat(service): 所有 EventService List 方法加入 Redis 缓存"
```

---

### Task 5: 改造 Listener Handler — 写入后删缓存

**Files:**
- Modify: `internal/listener/staked_event_handler.go`
- Modify: `internal/listener/reward_claimed_event_handler.go`
- Modify: `internal/listener/withdrawn_event_handler.go`
- Modify: `internal/listener/min_stake_amount_updated_event_handler.go`
- Modify: `internal/listener/reward_rate_updated_event_handler.go`

- [ ] **Step 1: 改造 StakedEventLogHandler**

将 `internal/listener/staked_event_handler.go` 改为：

```go
package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"

	pkgabi "test-stake-backend/internal/abi"
	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
)

const stakedEventName = "Staked"

type StakedEventLogHandler struct {
	repo          *repository.StakedEventRepository
	rdb           *redis.Client
	contractABI   abi.ABI
	stakedEventID common.Hash
}

func NewStakedEventLogHandler(repo *repository.StakedEventRepository, rdb *redis.Client) (*StakedEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create staked event handler: repository is nil")
	}

	contractABI, err := pkgabi.LoadStakeABI()
	if err != nil {
		return nil, err
	}
	stakedEvent, ok := contractABI.Events[stakedEventName]
	if !ok {
		return nil, fmt.Errorf("create staked event handler: Staked event not found in ABI")
	}

	return &StakedEventLogHandler{
		repo:          repo,
		rdb:           rdb,
		contractABI:   contractABI,
		stakedEventID: stakedEvent.ID,
	}, nil
}

func (h *StakedEventLogHandler) EventName() string {
	return stakedEventName
}

func (h *StakedEventLogHandler) EventID() common.Hash {
	return h.stakedEventID
}

func (h *StakedEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	// 写入成功后清除列表缓存
	if err := cache.DeleteByPrefix(ctx, h.rdb, "staked:list:"); err != nil {
		log.Printf("cache delete staked list prefix: %v", err)
	}

	log.Printf("staked event inserted: tx=%s index=%d user=%s amount=%s", event.TxHash, event.LogIndex, event.User, event.Amount)
	return nil
}

func (h *StakedEventLogHandler) parseLog(eventLog types.Log) (*models.StakedEvent, error) {
	if len(eventLog.Topics) < 2 {
		return nil, fmt.Errorf("Staked log topics length = %d, want at least 2", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.stakedEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		Amount *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, stakedEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack Staked log data: %w", err)
	}
	if unpacked.Amount == nil {
		return nil, fmt.Errorf("unpack Staked log data: amount is nil")
	}

	return &models.StakedEvent{
		ContractAddress: eventLog.Address.Hex(),
		User:            common.BytesToAddress(eventLog.Topics[1].Bytes()).Hex(),
		Amount:          unpacked.Amount.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}
```

- [ ] **Step 2: 改造 RewardClaimedEventLogHandler**

将 `internal/listener/reward_claimed_event_handler.go` 改为：

```go
package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"

	pkgabi "test-stake-backend/internal/abi"
	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
)

const rewardClaimedEventName = "RewardClaimed"

type RewardClaimedEventLogHandler struct {
	repo                 *repository.RewardClaimedEventRepository
	rdb                  *redis.Client
	contractABI          abi.ABI
	rewardClaimedEventID common.Hash
}

func NewRewardClaimedEventLogHandler(repo *repository.RewardClaimedEventRepository, rdb *redis.Client) (*RewardClaimedEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward claimed event handler: repository is nil")
	}

	contractABI, err := pkgabi.LoadStakeABI()
	if err != nil {
		return nil, err
	}
	rewardClaimedEvent, ok := contractABI.Events[rewardClaimedEventName]
	if !ok {
		return nil, fmt.Errorf("create reward claimed event handler: RewardClaimed event not found in ABI")
	}

	return &RewardClaimedEventLogHandler{
		repo:                 repo,
		rdb:                  rdb,
		contractABI:          contractABI,
		rewardClaimedEventID: rewardClaimedEvent.ID,
	}, nil
}

func (h *RewardClaimedEventLogHandler) EventName() string {
	return rewardClaimedEventName
}

func (h *RewardClaimedEventLogHandler) EventID() common.Hash {
	return h.rewardClaimedEventID
}

func (h *RewardClaimedEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	if err := cache.DeleteByPrefix(ctx, h.rdb, "reward-claimed:list:"); err != nil {
		log.Printf("cache delete reward-claimed list prefix: %v", err)
	}

	log.Printf("reward claimed event inserted: tx=%s index=%d user=%s amount=%s", event.TxHash, event.LogIndex, event.User, event.Amount)
	return nil
}

func (h *RewardClaimedEventLogHandler) parseLog(eventLog types.Log) (*models.RewardClaimedEvent, error) {
	if len(eventLog.Topics) < 2 {
		return nil, fmt.Errorf("RewardClaimed log topics length = %d, want at least 2", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.rewardClaimedEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		Amount *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, rewardClaimedEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack RewardClaimed log data: %w", err)
	}
	if unpacked.Amount == nil {
		return nil, fmt.Errorf("unpack RewardClaimed log data: amount is nil")
	}

	return &models.RewardClaimedEvent{
		ContractAddress: eventLog.Address.Hex(),
		User:            common.BytesToAddress(eventLog.Topics[1].Bytes()).Hex(),
		Amount:          unpacked.Amount.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}
```

- [ ] **Step 3: 改造 WithdrawnEventLogHandler**

将 `internal/listener/withdrawn_event_handler.go` 改为：

```go
package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"

	pkgabi "test-stake-backend/internal/abi"
	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
)

const withdrawnEventName = "Withdrawn"

type WithdrawnEventLogHandler struct {
	repo             *repository.WithdrawnEventRepository
	rdb              *redis.Client
	contractABI      abi.ABI
	withdrawnEventID common.Hash
}

func NewWithdrawnEventLogHandler(repo *repository.WithdrawnEventRepository, rdb *redis.Client) (*WithdrawnEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create withdrawn event handler: repository is nil")
	}

	contractABI, err := pkgabi.LoadStakeABI()
	if err != nil {
		return nil, err
	}
	withdrawnEvent, ok := contractABI.Events[withdrawnEventName]
	if !ok {
		return nil, fmt.Errorf("create withdrawn event handler: Withdrawn event not found in ABI")
	}

	return &WithdrawnEventLogHandler{
		repo:             repo,
		rdb:              rdb,
		contractABI:      contractABI,
		withdrawnEventID: withdrawnEvent.ID,
	}, nil
}

func (h *WithdrawnEventLogHandler) EventName() string {
	return withdrawnEventName
}

func (h *WithdrawnEventLogHandler) EventID() common.Hash {
	return h.withdrawnEventID
}

func (h *WithdrawnEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	if err := cache.DeleteByPrefix(ctx, h.rdb, "withdrawn:list:"); err != nil {
		log.Printf("cache delete withdrawn list prefix: %v", err)
	}

	log.Printf("withdrawn event inserted: tx=%s index=%d user=%s amount=%s", event.TxHash, event.LogIndex, event.User, event.Amount)
	return nil
}

func (h *WithdrawnEventLogHandler) parseLog(eventLog types.Log) (*models.WithdrawnEvent, error) {
	if len(eventLog.Topics) < 2 {
		return nil, fmt.Errorf("Withdrawn log topics length = %d, want at least 2", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.withdrawnEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		Amount *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, withdrawnEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack Withdrawn log data: %w", err)
	}
	if unpacked.Amount == nil {
		return nil, fmt.Errorf("unpack Withdrawn log data: amount is nil")
	}

	return &models.WithdrawnEvent{
		ContractAddress: eventLog.Address.Hex(),
		User:            common.BytesToAddress(eventLog.Topics[1].Bytes()).Hex(),
		Amount:          unpacked.Amount.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}
```

- [ ] **Step 4: 改造 MinStakeAmountUpdatedEventLogHandler**

将 `internal/listener/min_stake_amount_updated_event_handler.go` 改为：

```go
package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"

	pkgabi "test-stake-backend/internal/abi"
	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
)

const minStakeAmountUpdatedEventName = "MinStakeAmountUpdated"

type MinStakeAmountUpdatedEventLogHandler struct {
	repo                         *repository.MinStakeAmountUpdatedEventRepository
	rdb                          *redis.Client
	contractABI                  abi.ABI
	minStakeAmountUpdatedEventID common.Hash
}

func NewMinStakeAmountUpdatedEventLogHandler(repo *repository.MinStakeAmountUpdatedEventRepository, rdb *redis.Client) (*MinStakeAmountUpdatedEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create min stake amount updated event handler: repository is nil")
	}

	contractABI, err := pkgabi.LoadStakeABI()
	if err != nil {
		return nil, err
	}
	minStakeAmountUpdatedEvent, ok := contractABI.Events[minStakeAmountUpdatedEventName]
	if !ok {
		return nil, fmt.Errorf("create min stake amount updated event handler: MinStakeAmountUpdated event not found in ABI")
	}

	return &MinStakeAmountUpdatedEventLogHandler{
		repo:                         repo,
		rdb:                          rdb,
		contractABI:                  contractABI,
		minStakeAmountUpdatedEventID: minStakeAmountUpdatedEvent.ID,
	}, nil
}

func (h *MinStakeAmountUpdatedEventLogHandler) EventName() string {
	return minStakeAmountUpdatedEventName
}

func (h *MinStakeAmountUpdatedEventLogHandler) EventID() common.Hash {
	return h.minStakeAmountUpdatedEventID
}

func (h *MinStakeAmountUpdatedEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	if err := cache.DeleteByPrefix(ctx, h.rdb, "min-stake-amount-updated:list:"); err != nil {
		log.Printf("cache delete min-stake-amount-updated list prefix: %v", err)
	}

	log.Printf("min stake amount updated event inserted: tx=%s index=%d old_amount=%s new_amount=%s", event.TxHash, event.LogIndex, event.OldAmount, event.NewAmount)
	return nil
}

func (h *MinStakeAmountUpdatedEventLogHandler) parseLog(eventLog types.Log) (*models.MinStakeAmountUpdatedEvent, error) {
	if len(eventLog.Topics) < 1 {
		return nil, fmt.Errorf("MinStakeAmountUpdated log topics length = %d, want at least 1", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.minStakeAmountUpdatedEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		OldAmount *big.Int
		NewAmount *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, minStakeAmountUpdatedEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack MinStakeAmountUpdated log data: %w", err)
	}
	if unpacked.OldAmount == nil || unpacked.NewAmount == nil {
		return nil, fmt.Errorf("unpack MinStakeAmountUpdated log data: amount is nil")
	}

	return &models.MinStakeAmountUpdatedEvent{
		ContractAddress: eventLog.Address.Hex(),
		OldAmount:       unpacked.OldAmount.String(),
		NewAmount:       unpacked.NewAmount.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}
```

- [ ] **Step 5: 改造 RewardRateUpdatedEventLogHandler**

将 `internal/listener/reward_rate_updated_event_handler.go` 改为：

```go
package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"

	pkgabi "test-stake-backend/internal/abi"
	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
)

const rewardRateUpdatedEventName = "RewardRateUpdated"

type RewardRateUpdatedEventLogHandler struct {
	repo                    *repository.RewardRateUpdatedEventRepository
	rdb                     *redis.Client
	contractABI             abi.ABI
	rewardRateUpdatedEventID common.Hash
}

func NewRewardRateUpdatedEventLogHandler(repo *repository.RewardRateUpdatedEventRepository, rdb *redis.Client) (*RewardRateUpdatedEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward rate updated event handler: repository is nil")
	}

	contractABI, err := pkgabi.LoadStakeABI()
	if err != nil {
		return nil, err
	}
	rewardRateUpdatedEvent, ok := contractABI.Events[rewardRateUpdatedEventName]
	if !ok {
		return nil, fmt.Errorf("create reward rate updated event handler: RewardRateUpdated event not found in ABI")
	}

	return &RewardRateUpdatedEventLogHandler{
		repo:                     repo,
		rdb:                      rdb,
		contractABI:              contractABI,
		rewardRateUpdatedEventID: rewardRateUpdatedEvent.ID,
	}, nil
}

func (h *RewardRateUpdatedEventLogHandler) EventName() string {
	return rewardRateUpdatedEventName
}

func (h *RewardRateUpdatedEventLogHandler) EventID() common.Hash {
	return h.rewardRateUpdatedEventID
}

func (h *RewardRateUpdatedEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	if err := cache.DeleteByPrefix(ctx, h.rdb, "reward-rate-updated:list:"); err != nil {
		log.Printf("cache delete reward-rate-updated list prefix: %v", err)
	}

	log.Printf("reward rate updated event inserted: tx=%s index=%d old_rate=%s new_rate=%s", event.TxHash, event.LogIndex, event.OldRate, event.NewRate)
	return nil
}

func (h *RewardRateUpdatedEventLogHandler) parseLog(eventLog types.Log) (*models.RewardRateUpdatedEvent, error) {
	if len(eventLog.Topics) < 1 {
		return nil, fmt.Errorf("RewardRateUpdated log topics length = %d, want at least 1", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.rewardRateUpdatedEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		OldRate *big.Int
		NewRate *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, rewardRateUpdatedEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack RewardRateUpdated log data: %w", err)
	}
	if unpacked.OldRate == nil || unpacked.NewRate == nil {
		return nil, fmt.Errorf("unpack RewardRateUpdated log data: rate is nil")
	}

	return &models.RewardRateUpdatedEvent{
		ContractAddress: eventLog.Address.Hex(),
		OldRate:         unpacked.OldRate.String(),
		NewRate:         unpacked.NewRate.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}
```

- [ ] **Step 6: 确认编译通过**

Run: `go build ./...`
Expected: 编译失败 — main.go 中 handler 构造函数参数不匹配（预期行为，下一个 Task 修复）

- [ ] **Step 7: 提交**

```bash
git add internal/listener/
git commit -m "feat(listener): 所有 handler 写入事件后主动删除列表缓存"
```

---

### Task 6: 更新依赖注入 — router.go 和 main.go

**Files:**
- Modify: `internal/api/router.go`
- Modify: `main.go`

- [ ] **Step 1: 改造 router.go**

将 `internal/api/router.go` 改为：

```go
package api

import (
	"fmt"
	"net/http"
	"time"

	"test-stake-backend/internal/repository"
	"test-stake-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, rdb *redis.Client, cacheTTL time.Duration) error {
	r.GET("/health", healthHandler)

	// StakedEvent
	stakedEventRepo, err := repository.NewStakedEventRepository(db)
	if err != nil {
		return fmt.Errorf("register staked event repository: %w", err)
	}
	stakedEventService, err := service.NewStakedEventService(stakedEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register staked event service: %w", err)
	}
	NewStakedEventHandler(stakedEventService).Register(r)

	// RewardClaimedEvent
	rewardClaimedEventRepo, err := repository.NewRewardClaimedEventRepository(db)
	if err != nil {
		return fmt.Errorf("register reward claimed event repository: %w", err)
	}
	rewardClaimedEventService, err := service.NewRewardClaimedEventService(rewardClaimedEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register reward claimed event service: %w", err)
	}
	NewRewardClaimedEventHandler(rewardClaimedEventService).Register(r)

	// WithdrawnEvent
	withdrawnEventRepo, err := repository.NewWithdrawnEventRepository(db)
	if err != nil {
		return fmt.Errorf("register withdrawn event repository: %w", err)
	}
	withdrawnEventService, err := service.NewWithdrawnEventService(withdrawnEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register withdrawn event service: %w", err)
	}
	NewWithdrawnEventHandler(withdrawnEventService).Register(r)

	// MinStakeAmountUpdatedEvent
	minStakeAmountUpdatedEventRepo, err := repository.NewMinStakeAmountUpdatedEventRepository(db)
	if err != nil {
		return fmt.Errorf("register min stake amount updated event repository: %w", err)
	}
	minStakeAmountUpdatedEventService, err := service.NewMinStakeAmountUpdatedEventService(minStakeAmountUpdatedEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register min stake amount updated event service: %w", err)
	}
	NewMinStakeAmountUpdatedEventHandler(minStakeAmountUpdatedEventService).Register(r)

	// RewardRateUpdatedEvent
	rewardRateUpdatedEventRepo, err := repository.NewRewardRateUpdatedEventRepository(db)
	if err != nil {
		return fmt.Errorf("register reward rate updated event repository: %w", err)
	}
	rewardRateUpdatedEventService, err := service.NewRewardRateUpdatedEventService(rewardRateUpdatedEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register reward rate updated event service: %w", err)
	}
	NewRewardRateUpdatedEventHandler(rewardRateUpdatedEventService).Register(r)

	return nil
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
```

- [ ] **Step 2: 改造 main.go**

在 `main.go` 中：
1. 计算 `cacheTTL`（从配置读取，默认 60 秒）
2. `api.RegisterRoutes` 调用增加 `redisClient` 和 `cacheTTL` 参数
3. 每个 listener handler 构造函数增加 `redisClient` 参数

关键改动部分：

```go
// 计算缓存 TTL
cacheTTL := time.Duration(cfg.Redis.CacheTTL) * time.Second
if cacheTTL <= 0 {
	cacheTTL = 60 * time.Second
}

// 注册事件处理器（传入 redisClient）
for _, newHandler := range []func() (listener.ContractEventHandler, error){
	func() (listener.ContractEventHandler, error) {
		return listener.NewStakedEventLogHandler(stakedEventRepo, redisClient)
	},
	func() (listener.ContractEventHandler, error) {
		return listener.NewRewardClaimedEventLogHandler(rewardClaimedEventRepo, redisClient)
	},
	func() (listener.ContractEventHandler, error) {
		return listener.NewWithdrawnEventLogHandler(withdrawnEventRepo, redisClient)
	},
	func() (listener.ContractEventHandler, error) {
		return listener.NewMinStakeAmountUpdatedEventLogHandler(minStakeAmountUpdatedEventRepo, redisClient)
	},
	func() (listener.ContractEventHandler, error) {
		return listener.NewRewardRateUpdatedEventLogHandler(rewardRateUpdatedEventRepo, redisClient)
	},
} {
	// ... 保持不变
}

// 注册路由（传入 redisClient 和 cacheTTL）
if err := api.RegisterRoutes(r, db, redisClient, cacheTTL); err != nil {
	log.Fatalf("Failed to register routes: %v", err)
}
```

- [ ] **Step 3: 确认编译通过**

Run: `go build ./...`
Expected: 编译成功

- [ ] **Step 4: 运行全部测试**

Run: `go test ./...`
Expected: 通过

- [ ] **Step 5: 提交**

```bash
git add internal/api/router.go main.go
git commit -m "feat: 完成 Redis 缓存依赖注入，串联所有组件"
```
