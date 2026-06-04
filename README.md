# Stake Backend

以太坊质押合约事件监听后端服务。通过 WebSocket 连接以太坊节点，实时监听并回放质押合约事件，持久化到 MySQL，并提供 REST API 查询。

## 功能

- **实时监听** — WebSocket 订阅合约事件，处理 Staked、RewardClaimed、Withdrawn、MinStakeAmountUpdated、RewardRateUpdated 五类事件
- **历史回放** — 启动时自动从合约部署区块开始回放落后区块，支持断点续传
- **REST API** — 基于 Gin 框架，每类事件提供分页列表和按 ID 查询接口
- **优雅关机** — 支持 SIGINT/SIGTERM 信号，平滑关闭 HTTP 服务和事件监听

## 技术栈

| 组件 | 技术 |
|------|------|
| HTTP 框架 | Gin |
| ORM | GORM (MySQL) |
| 以太坊交互 | go-ethereum |
| 配置管理 | Viper |
| 缓存 | Redis |

## 快速开始

### 环境要求

- Go 1.26+
- MySQL 5.7+
- Redis
- 以太坊节点（支持 WebSocket，如 Infura）

### 安装

```bash
git clone <repo-url>
cd test-stake-backend
go mod download
```

### 配置

复制示例配置文件并按需修改：

```bash
cp config.yaml.sample config.yaml
```

配置项说明：

```yaml
server:
  host: localhost
  port: 8080
  mode: debug          # debug / test / release
database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: stake
redis:
  addr: localhost:6379
  password: ""
  db: 0
eth:
  rpc_url: https://sepolia.infura.io/v3/YOUR_KEY
  ws_url: wss://sepolia.infura.io/ws/v3/YOUR_KEY
  stake_address: 0x...  # 质押合约地址
  start_block: 10986812 # 合约部署区块号
```

> `start_block` 必须设置为合约部署时的区块号，用于历史事件回放。

### 运行

```bash
go build -o bin/server .
./bin/server
```

启动后会自动：
1. 连接 MySQL 并迁移表结构（表名前缀 `t_`）
2. 从 `start_block` 开始回放历史事件
3. 切换到 WebSocket 实时订阅
4. 启动 HTTP 服务

### API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/staked-events` | 质押事件列表 |
| GET | `/staked-events/:id` | 按 ID 查询质押事件 |
| GET | `/reward-claimed-events` | 奖励领取事件列表 |
| GET | `/reward-claimed-events/:id` | 按 ID 查询奖励领取事件 |
| GET | `/withdrawn-events` | 提取事件列表 |
| GET | `/withdrawn-events/:id` | 按 ID 查询提取事件 |
| GET | `/min-stake-amount-updated-events` | 最低质押金额变更事件列表 |
| GET | `/min-stake-amount-updated-events/:id` | 按 ID 查询 |
| GET | `/reward-rate-updated-events` | 奖励费率变更事件列表 |
| GET | `/reward-rate-updated-events/:id` | 按 ID 查询 |

**通用查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码，默认 1 |
| `page_size` | int | 每页条数，默认 20，最大 100 |
| `contract_address` | string | 合约地址过滤 |
| `tx_hash` | string | 交易哈希过滤 |
| `block_number_from` | uint64 | 起始区块号 |
| `block_number_to` | uint64 | 结束区块号 |

Staked/RewardClaimed/Withdrawn 事件额外支持 `user` 参数按用户地址过滤。

## 项目结构

```
.
├── main.go                          # 入口，初始化依赖并启动服务
├── config.yaml.sample               # 配置示例
├── internal/
│   ├── abi/                         # 合约 ABI（编译时嵌入）
│   │   ├── abi.go
│   │   └── Stake.abi.json
│   ├── api/                         # HTTP 接口层
│   │   ├── router.go                # 路由注册
│   │   ├── common.go                # 通用解析/响应工具
│   │   └── *_event_handler.go       # 各事件处理器
│   ├── config/                      # 配置加载
│   │   └── config.go
│   ├── listener/                    # 链上事件监听
│   │   ├── contract_event_listener.go  # 核心监听器（订阅 + 回放）
│   │   └── *_event_handler.go       # 各事件解析器
│   ├── models/                      # 数据模型
│   │   ├── contract.go              # 合约同步状态
│   │   └── event.go                 # 五类事件模型
│   ├── repository/                  # 数据访问层
│   │   ├── common.go                # 公共查询/校验逻辑
│   │   ├── contract_repository.go   # 合约同步状态管理
│   │   └── *_event_repository.go    # 各事件仓库
│   └── service/                     # 业务逻辑层
│       └── *_event_service.go
```

## 添加新事件类型

1. `internal/models/` — 添加事件模型
2. `internal/repository/` — 添加仓库（复用 `BaseQuery`、`isDuplicateKeyError`）
3. `internal/service/` — 添加服务
4. `internal/listener/` — 添加事件处理器，实现 `ContractEventHandler` 接口
5. `internal/api/` — 添加 HTTP 处理器
6. `main.go` — 注册仓库和事件处理器
7. `internal/api/router.go` — 注册路由
