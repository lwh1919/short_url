# 短链接服务 (Short URL Service)

一个高性能、分布式的短链接生成服务，采用微服务架构，支持高并发场景。

## 🏗️ 项目架构

```
short_url_rpc_study/
├── rpc/                    # gRPC服务端
│   ├── main.go            # RPC服务入口
│   ├── grpc/              # gRPC实现
│   ├── service/           # 业务逻辑层
│   ├── repository/        # 数据访问层
│   ├── job/              # 定时任务
│   └── config/           # RPC配置
├── web/                   # Web服务层
│   ├── main.go           # Web服务入口
│   ├── routes/           # HTTP路由
│   ├── middlewares/      # 中间件
│   └── static/          # 前端静态资源
├── pkg/                  # 公共库
│   ├── bloom/           # 布隆过滤器
│   ├── generator/       # 短链接生成器
│   └── sign/           # 签名验证
├── proto/               # Protocol Buffers定义
├── scripts/             # 数据库初始化脚本
├── nginx/               # Nginx配置
└── docker-compose.yaml  # 容器编排
```

## 🚀 技术栈

### 后端技术
- **语言**: Go 1.24+
- **Web框架**: Gin
- **RPC框架**: gRPC + Protocol Buffers
- **ORM**: GORM + MySQL 8.0
- **缓存**: Redis
- **服务发现**: etcd
- **限流**: 令牌桶算法 + Redis + lua
- **熔断**: Hystrix + Sentinel

### 数据存储
- **数据库**: MySQL 8.0 (支持分表)
- **缓存层**: 
  - Redis (分布式缓存)
  - 本地LRU缓存
  - 布隆过滤器 (防穿透)

### 部署与运维
- **容器化**: Docker + Docker Compose
- **反向代理**: Nginx
- **日志**: Zap + 结构化日志
- **监控**: 健康检查端点

## ✨ 核心功能

### 1. 短链接生成
- 基于哈希算法生成唯一短码

### 2. 高并发支持
- 分布式架构设计
- Redis三级缓存 + 请求合并
- 令牌桶全局限流保护 + nginx ip级限流
- 熔断降级机制
- 校验码 + bloom 拦截无效请求
- nginx 负载均衡 + etcd 服务发现 负载均衡
- 写入优化 环形缓存区+切片池+并发写入

### 3. 数据管理
- 定时清理过期短链接 与 布隆归档
- 数据库分表支持
- 缓存一致性保证
- 索引优化

### 4. 监控与运维
- 健康检查接口
- 服务状态监控
- 结构化日志记录

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
