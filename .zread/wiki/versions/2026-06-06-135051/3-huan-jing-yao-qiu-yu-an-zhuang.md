本文档详细介绍了 test-stake-backend 项目的运行环境要求、依赖项配置和安装步骤。作为以太坊质押合约事件监听后端服务，本项目需要特定的基础设施支持，包括 Go 运行环境、数据库服务、缓存系统和以太坊节点连接。

## 环境要求

### 系统要求

**Go 运行环境**：项目基于 Go 语言开发，需要 Go 1.26 或更高版本。从 `go.mod` 文件可以看到当前使用的是 Go 1.26.3 版本，建议使用相同或更新的版本以确保兼容性。

**操作系统**：支持 Windows、macOS 和 Linux 操作系统，推荐使用 Linux 进行生产环境部署。

### 基础设施要求

**数据库服务**：项目使用 MySQL 作为主要数据存储，需要 MySQL 5.7 或更高版本。配置文件中显示数据库默认使用 3306 端口，字符集为 utf8mb4，支持完整的 Unicode 字符集。

**缓存系统**：项目使用 Redis 进行缓存优化，提高查询性能。配置文件显示 Redis 默认使用 6379 端口，支持密码认证和数据库选择。

**以太坊节点**：项目需要连接以太坊节点进行事件监听，支持 RPC 和 WebSocket 两种连接方式。从配置文件可以看到需要配置 `rpc_url` 和 `ws_url`，推荐使用 Infura 等专业节点服务。

## 核心依赖项

### 主要框架和库

项目依赖以下核心组件，这些依赖项在 `go.mod` 文件中明确定义：

**以太坊交互**：`github.com/ethereum/go-ethereum v1.17.3` - 以太坊官方 Go 客户端，用于与智能合约交互和事件监听。

**HTTP 框架**：`github.com/gin-gonic/gin v1.12.0` - 高性能的 HTTP Web 框架，用于构建 REST API 接口。

**数据库 ORM**：`gorm.io/gorm v1.31.1` 和 `gorm.io/driver/mysql v1.6.0` - ORM 框架和 MySQL 驱动，简化数据库操作。

**缓存客户端**：`github.com/redis/go-redis/v9 v9.20.0` - Redis 官方 Go 客户端，用于缓存管理。

**配置管理**：`github.com/spf13/viper v1.21.0` - 灵活的配置管理库，支持多种配置格式。

### 依赖项版本说明

从 `go.mod` 文件可以看到，项目使用了精确的版本控制策略。主要依赖项都指定了具体的版本号，确保构建的一致性和可重复性。所有依赖项都遵循 Go 模块管理规范，通过 `go.sum` 文件进行完整性校验。

## 安装步骤

### 1. 获取项目代码

首先需要将项目代码克隆到本地环境：

```bash
git clone <repository-url>
cd test-stake-backend
```

### 2. 安装 Go 依赖项

使用 Go 模块管理器下载所有依赖项：

```bash
go mod download
```

此命令会根据 `go.mod` 文件自动下载所有必需的依赖项到本地模块缓存。如果网络环境受限，可以配置 Go 模块代理：

```bash
export GOPROXY=https://goproxy.cn,direct
```

### 3. 配置环境文件

项目使用 YAML 格式的配置文件，需要从示例文件创建实际配置：

```bash
cp config.yaml.sample config.yaml
```

配置文件包含以下主要部分：

**服务器配置**：设置 HTTP 服务监听的主机地址、端口和运行模式（debug/test/release）。

**数据库配置**：配置 MySQL 连接参数，包括主机地址、端口、用户名、密码和数据库名称。

**Redis 配置**：设置 Redis 连接地址、密码、数据库索引和缓存过期时间。

**以太坊配置**：配置以太坊节点连接信息，包括 RPC 和 WebSocket URL、质押合约地址和起始区块号。

### 4. 数据库准备

项目使用 GORM 的自动迁移功能，在启动时会自动创建数据库表。但需要确保：

1. MySQL 服务已启动并可访问
2. 配置的数据库用户具有创建表的权限
3. 数据库名称在 MySQL 中已存在（如不存在需手动创建）

表名使用 `t_` 前缀，采用单数名词命名规范，例如 `t_staked_event`、`t_contract` 等。

### 5. 构建项目

使用 Go 编译器构建可执行文件：

```bash
go build -o bin/server .
```

构建过程会：
1. 编译所有 Go 源代码
2. 嵌入 ABI 文件（`internal/abi/Stake.abi.json`）
3. 生成平台特定的可执行文件

构建完成后，可执行文件会保存在 `bin` 目录下。

## 配置详解

### 配置文件结构

配置文件采用层次化的 YAML 格式，主要包含四个配置块：

**server 配置块**：
- `host`: 服务器监听地址，默认为 `localhost`
- `port`: 监听端口，默认为 `8080`
- `mode`: 运行模式，支持 `debug`、`test` 和 `release`

**database 配置块**：
- `host`: MySQL 服务器地址
- `port`: MySQL 端口，默认为 `3306`
- `username`: 数据库用户名
- `password`: 数据库密码
- `dbname`: 数据库名称

**redis 配置块**：
- `addr`: Redis 服务器地址，格式为 `host:port`
- `password`: Redis 认证密码
- `db`: Redis 数据库索引
- `cache_ttl`: 缓存过期时间（秒），默认为 60

**eth 配置块**：
- `rpc_url`: 以太坊 RPC 节点 URL
- `ws_url`: 以太坊 WebSocket 节点 URL
- `stake_address`: 质押合约地址
- `start_block`: 历史事件回放的起始区块号，必须设置为合约部署时的区块号

### 配置验证

配置文件加载过程在 `internal/config/config.go` 中实现，使用 Viper 库进行解析。系统会在启动时验证配置文件的完整性和格式正确性。如果配置文件缺失或格式错误，系统会输出错误信息并退出。

## 验证安装

### 构建验证

安装完成后，可以通过以下命令验证构建是否成功：

```bash
go build -o bin/server .
```

如果构建成功，会在 `bin` 目录下生成可执行文件。

### 依赖检查

使用以下命令检查依赖项是否完整：

```bash
go mod verify
```

此命令会验证所有依赖项的完整性和一致性。

### 配置文件验证

确保 `config.yaml` 文件存在且格式正确：

```bash
go run main.go -config-check
```

或者直接运行服务，观察启动日志中的配置加载信息。

## 常见问题解决

### Go 版本不兼容

如果遇到 Go 版本不兼容问题，请检查 `go.mod` 文件中的版本要求，并升级 Go 到指定版本。可以通过以下命令检查当前 Go 版本：

```bash
go version
```

### 依赖下载失败

如果遇到依赖下载问题，可以尝试：

1. 配置 Go 模块代理
2. 清理模块缓存：`go clean -modcache`
3. 使用 `go mod tidy` 重新整理依赖

### 数据库连接失败

如果数据库连接失败，请检查：

1. MySQL 服务是否运行
2. 配置文件中的数据库参数是否正确
3. 用户权限是否足够
4. 防火墙设置是否允许连接

### 以太坊节点连接问题

如果无法连接以太坊节点，请检查：

1. RPC 和 WebSocket URL 是否正确
2. API 密钥是否有效
3. 网络连接是否正常

## 下一步

完成环境配置和安装后，您可以参考以下文档继续学习：

1. **[配置文件详解](4-pei-zhi-wen-jian-xiang-jie)** - 深入了解每个配置项的详细说明和最佳实践
2. **[启动与运行服务](5-qi-dong-yu-yun-xing-fu-wu)** - 学习如何启动服务并验证运行状态
3. **[API接口测试](6-apijie-kou-ce-shi)** - 了解如何测试和使用项目提供的 REST API 接口

## 源码参考

本节内容基于以下源代码文件编写：

Sources: [go.mod](go.mod#L1-L83)
Sources: [config.yaml.sample](config.yaml.sample#L1-L22)
Sources: [README.md](README.md#L22-L37)
Sources: [main.go](main.go#L1-L175)
Sources: [internal/config/config.go](internal/config/config.go#L1-L60)