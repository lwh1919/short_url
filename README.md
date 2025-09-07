# 短链接服务 (Short URL Service)

一个高性能、分布式的短链接生成服务，采用微服务架构，支持高并发场景，提供创建短链接，重定向短链接接口。


## 🚀 技术栈

| 类别 | 技术 |
|------|------|
| **语言** | Go 1.24+ |
| **Web框架** | Gin + gRPC |
| **数据存储** | MySQL 8.0 + Redis |
| **服务发现** | etcd |
| **限流熔断** | 令牌桶 + Hystrix |
| **容器化** | Docker + Docker Compose |
| **反向代理** | Nginx |
| **日志监控** | Zap + 健康检查 |


## ✨ 核心功能

### 1. 防穿透架构
**校验码 + 分布式归档布隆过滤器**
- 无效短链接请求过滤，彻底解决缓存穿透
- 布隆过滤器支持增量更新无锁机制（atomic原子操作）
- 异步归档机制，低性能损耗

### 2. 水平分表存储
- **6位短码**支持568亿条记录存储
- 分表策略防止单表数据量过大
- 索引优化保证查询性能

### 3. 三级写入体系
**环形缓冲区 + 对象池 + 并发写入**
- 聚合单行插入为批量事务写入
- 内存环形缓冲区应对突发流量
- 对象池减轻GC压力，分片并行保证原子性

### 4. 五级读取体系
**请求合并 + 请求过滤 + LocalCache + Redis + MySQL**
- Singleflight合并相同请求，解决缓存击穿
- TTL组合策略防止缓存雪崩
- 多级缓存逐级降级，保障高可用

### 5. 三级限流体系
**Nginx IP级 + 分布式限流器 + Hystrix API级**
- 细粒度限流保护，兜底服务器性能
- 令牌桶算法 + Redis Lua脚本实现
- 自适应熔断机制（错误率阈值 + 检测时间窗口）

### 6. 数据生命周期管理
- **定时清理**: Cron执行过期数据清理（缓存+数据库双写）
- **布隆维护**: 异步归档过滤器，支持增量更新
- **一致性**: 缓存数据库最终一致性保证

### 7. 微服务架构
**Docker + k8s 容器化部署**
- nginx（负载均衡 + 反向代理）
- web（熔断降级 + 限流）
- etcd（服务注册 + 负载均衡）
- rpc（心跳检测 + 服务发现）

## 🏗️ 项目架构

```
short_url_rpc_study/
├── rpc/                    # gRPC服务端
│   ├── main.go            # RPC服务入口
│   ├── app.go             # RPC应用初始化
│   ├── wire.go            # 依赖注入配置
│   ├── wire_gen.go        # 自动生成的依赖注入代码
│   ├── grpc/              # gRPC实现
│   │   └── shortUrl.go    # gRPC服务实现
│   ├── service/           # 业务逻辑层
│   │   ├── shortUrl.go    # 短链接服务
│   │   └── type.go        # 类型定义
│   ├── repository/        # 数据访问层
│   │   ├── shortUrl.go    # 数据访问接口
│   │   ├── dao/           # 数据访问对象
│   │   ├── cache/         # 缓存层
│   │   └── type.go        # 数据类型定义
│   ├── job/               # 定时任务
│   │   ├── cleaner.go     # 清理任务
│   │   ├── cronJob.go     # 定时任务配置
│   │   └── type.go        # 任务类型定义
│   ├── config/            # RPC配置
│   │   └── config.template.yaml
│   └── ioc/               # 依赖注入初始化
│       ├── db.go          # 数据库初始化
│       ├── etcd.go        # etcd客户端初始化
│       ├── grpc.go        # gRPC服务初始化
│       ├── job.go         # 定时任务初始化
│       ├── logger.go      # 日志初始化
│       ├── redis.go       # Redis客户端初始化
│       ├── redisCache.go  # 缓存初始化
│       └── repo.go        # 仓库初始化
├── web/                   # Web服务层
│   ├── main.go           # Web服务入口
│   ├── wire.go           # 依赖注入配置
│   ├── wire_gen.go       # 自动生成的依赖注入代码
│   ├── config/           # Web配置
│   │   └── config.template.yaml
│   ├── routes/           # HTTP路由
│   │   ├── api.go        # API路由
│   │   ├── health.go     # 健康检查路由
│   │   ├── server.go     # 服务路由
│   │   └── type.go       # 路由类型定义
│   ├── middlewares/      # 中间件
│   │   ├── logger.go     # 日志中间件
│   │   ├── ratelimit.go  # 限流中间件
│   │   └── signature.go  # 签名验证中间件
│   ├── ioc/              # 依赖注入初始化
│   │   ├── etcd.go       # etcd客户端初始化
│   │   ├── hystrix.go    # 熔断器初始化
│   │   ├── logger.go     # 日志初始化
│   │   ├── ratelimit.go  # 限流器初始化
│   │   ├── redis.go      # Redis客户端初始化
│   │   ├── server.go     # 服务初始化
│   │   ├── shortUrl.go   # 短链接服务初始化
│   │   └── web.go        # Web服务初始化
│   ├── pkg/              # Web公共库
│   │   └── ratelimiter.go
│   └── static/          # 前端静态资源
│       ├── index.html
│       └── maintenance.html
├── pkg/                  # 公共库
│   ├── bloom/           # 布隆过滤器
│   │   ├── bloom.go     # 布隆过滤器实现
│   │   ├── encryptor.go # 加密器
│   │   ├── lua.go       # Lua脚本
│   │   ├── redis_client.go # Redis客户端
│   │   └── scripts/     # Lua脚本文件
│   ├── generator/       # 短链接生成器
│   │   ├── charset.go   # 字符集定义
│   │   ├── generator.go # 生成器实现
│   │   └── generator_test.go # 测试文件
│   ├── go-redis-tokenbuket/ # Redis令牌桶限流
│   │   ├── ratelimit.go # 限流实现
│   │   └── scripts/     # Lua脚本
│   ├── logfile/         # 日志文件处理
│   │   └── logfile.go
│   └── sign/            # 签名验证
│       ├── epay/        # 支付相关签名
│       └── type.go      # 签名类型定义
├── proto/               # Protocol Buffers定义
│   └── short_url/
│       ├── short_url.proto
│       └── v1/
├── scripts/             # 数据库初始化脚本
│   ├── etcd_data/       # etcd数据脚本
│   └── mysql/
│       └── init.sql     # MySQL初始化脚本
├── nginx/               # Nginx配置
│   └── nginx.conf       # Nginx配置文件
├── test/                # 测试文件
│   ├── nginx_limit_test.go
│   └── web_limit_test.go
├── go.mod               # Go模块文件
├── go.sum               # Go依赖校验文件
├── docker-compose.yaml  # 容器编排
└── .gitignore          # Git忽略文件
```

## 🛠️ 快速开始

### 环境要求
- Go 1.24+
- Docker & Docker Compose

### 1. 启动依赖服务
```bash
docker-compose up 
```

### 3. 启动RPC服务
```bash
cd rpc/
go run main.go --config config/config.template.yaml
```

### 4. 启动Web服务
```bash
cd web/
go run main.go --config config/config.template.yaml
```

### 5. 访问服务
- Web界面: http://localhost:8080
- API接口: http://localhost:8080/api/shorten
- 健康检查: http://localhost:8080/health
