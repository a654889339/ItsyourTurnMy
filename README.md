# 财务管理系统

基于 gRPC 框架实现的 H5 财务管理系统，采用前后端分离架构。

## 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: gRPC + HTTP REST API
- **数据库**: SQLite / MySQL
- **认证**: JWT Token

### 前端
- **框架**: Vue 3
- **构建工具**: Vite
- **状态管理**: Pinia
- **图表**: ECharts
- **HTTP 客户端**: Axios

## 项目结构

```
F:\MyGo\
├── backend/                    # 后端服务
│   ├── config/
│   │   ├── config.yaml        # 配置文件
│   │   └── config.go          # 配置加载
│   ├── database/
│   │   └── db.go              # 数据库操作
│   ├── model/
│   │   └── model.go           # 数据模型
│   ├── proto/
│   │   └── finance.proto      # gRPC 服务定义
│   ├── service/
│   │   ├── auth.go            # 认证服务
│   │   ├── account.go         # 账户服务
│   │   ├── transaction.go     # 交易服务
│   │   ├── category.go        # 分类服务
│   │   └── report.go          # 报表服务
│   ├── deploy/
│   │   ├── k8s/               # Kubernetes 部署配置
│   │   ├── deploy.sh          # 部署脚本
│   │   └── README.md          # 部署文档
│   ├── main.go                # 主入口
│   ├── go.mod
│   ├── Dockerfile
│   ├── docker-compose.yaml
│   └── docker-compose.mysql.yaml
│
└── frontend/                   # 前端应用
    ├── src/
    │   ├── api/
    │   │   └── index.js       # API 封装
    │   ├── router/
    │   │   └── index.js       # 路由配置
    │   ├── store/
    │   │   └── auth.js        # 状态管理
    │   ├── styles/
    │   │   └── main.css       # 全局样式
    │   ├── views/
    │   │   ├── Login.vue      # 登录页
    │   │   ├── Layout.vue     # 布局组件
    │   │   ├── Dashboard.vue  # 仪表盘
    │   │   ├── Accounts.vue   # 账户管理
    │   │   ├── Transactions.vue # 收支记录
    │   │   └── Reports.vue    # 统计报表
    │   ├── App.vue
    │   └── main.js
    ├── index.html
    ├── package.json
    └── vite.config.js
```

## 功能模块

| 模块 | 功能描述 |
|------|----------|
| 用户认证 | 注册、登录、JWT Token 验证 |
| 账户管理 | 创建/编辑/删除账户，支持现金、银行卡、信用卡、投资账户 |
| 收支记录 | 记录收入/支出，支持分类、筛选、分页 |
| 分类管理 | 自定义收支分类，预置默认分类 |
| 统计报表 | 月度收支统计、分类占比图表、每日趋势图 |

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- npm 或 yarn

### 后端启动

```bash
# 进入后端目录
cd backend

# 下载依赖
go mod tidy

# 启动服务 (默认端口 8080)
go run main.go

# 或指定配置文件
go run main.go -config ./config/config.yaml
```

### 前端启动

```bash
# 进入前端目录
cd frontend

# 安装依赖
npm install

# 启动开发服务器 (默认端口 5173)
npm run dev

# 构建生产版本
npm run build
```

### 访问应用

- 前端页面: http://localhost:5173
- 后端 API: http://localhost:8080/api/v1
- 健康检查: http://localhost:8080/health

## API 接口

### 认证接口

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/auth/register` | 用户注册 |
| POST | `/api/v1/auth/login` | 用户登录 |
| GET | `/api/v1/auth/me` | 获取当前用户信息 |

### 账户接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/accounts` | 获取账户列表 |
| POST | `/api/v1/accounts` | 创建账户 |
| GET | `/api/v1/accounts/:id` | 获取账户详情 |
| PUT | `/api/v1/accounts/:id` | 更新账户 |
| DELETE | `/api/v1/accounts/:id` | 删除账户 |

### 交易接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/transactions` | 获取交易列表 |
| POST | `/api/v1/transactions` | 创建交易记录 |
| GET | `/api/v1/transactions/:id` | 获取交易详情 |
| PUT | `/api/v1/transactions/:id` | 更新交易记录 |
| DELETE | `/api/v1/transactions/:id` | 删除交易记录 |

### 分类接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/categories` | 获取分类列表 |
| POST | `/api/v1/categories` | 创建分类 |
| PUT | `/api/v1/categories/:id` | 更新分类 |
| DELETE | `/api/v1/categories/:id` | 删除分类 |

### 报表接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/reports/stats` | 获取统计概览 |
| GET | `/api/v1/reports/monthly` | 获取月度报表 |

## 配置说明

### 配置文件 (config.yaml)

```yaml
server:
  http_port: 8080        # HTTP 端口
  grpc_port: 9090        # gRPC 端口
  mode: release          # 运行模式: debug/release

database:
  driver: sqlite         # 数据库类型: sqlite/mysql
  sqlite_path: ./data/finance.db

jwt:
  secret: your-secret    # JWT 密钥 (生产环境请修改)
  expire_hours: 24       # Token 有效期
```

### 环境变量

所有配置项都可通过环境变量覆盖：

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `HTTP_PORT` | HTTP 端口 | 8080 |
| `DB_DRIVER` | 数据库类型 | sqlite |
| `SQLITE_PATH` | SQLite 文件路径 | ./data/finance.db |
| `MYSQL_HOST` | MySQL 主机 | localhost |
| `MYSQL_PORT` | MySQL 端口 | 3306 |
| `MYSQL_USER` | MySQL 用户名 | root |
| `MYSQL_PASSWORD` | MySQL 密码 | - |
| `MYSQL_DATABASE` | MySQL 数据库名 | finance |
| `JWT_SECRET` | JWT 密钥 | - |
| `LOG_LEVEL` | 日志级别 | info |

## 部署指南

### Docker 部署 (单机)

```bash
cd backend

# SQLite 版本
docker-compose up -d

# MySQL 版本
docker-compose -f docker-compose.mysql.yaml up -d
```

### 腾讯云 TKE 部署

```bash
# 1. 构建并推送镜像
docker build -t ccr.ccs.tencentyun.com/<命名空间>/finance-backend:v1 .
docker push ccr.ccs.tencentyun.com/<命名空间>/finance-backend:v1

# 2. 部署到 Kubernetes
kubectl apply -k backend/deploy/k8s/

# 3. 查看部署状态
kubectl get pods -n finance-system
```

详细部署文档请参考: [backend/deploy/README.md](backend/deploy/README.md)

## 开发指南

### 添加新的 API

1. 在 `proto/finance.proto` 中定义 gRPC 服务
2. 在 `service/` 目录下实现业务逻辑
3. 在 `main.go` 中注册路由
4. 在前端 `api/index.js` 中添加 API 调用

### 数据库迁移

数据库表会在首次启动时自动创建，表结构定义在 `database/db.go` 中。

### 前端开发

```bash
cd frontend

# 启动开发服务器 (支持热更新)
npm run dev

# 代码检查
npm run lint

# 构建生产版本
npm run build
```

## 常见问题

### Q: 如何修改数据库?

修改 `config/config.yaml` 中的 `database.driver` 配置:
- `sqlite`: 使用 SQLite (默认)
- `mysql`: 使用 MySQL

### Q: 如何修改端口?

1. 修改配置文件中的 `server.http_port`
2. 或设置环境变量 `HTTP_PORT`

### Q: 忘记密码怎么办?

目前需要直接操作数据库重置密码，后续版本将添加密码重置功能。

### Q: 前端如何连接不同的后端地址?

修改 `frontend/vite.config.js` 中的 `proxy` 配置:

```javascript
server: {
  proxy: {
    '/api': {
      target: 'http://your-backend-host:8080',
      changeOrigin: true
    }
  }
}
```

## 版本历史

- **v1.0.0** - 初始版本
  - 用户认证 (注册/登录)
  - 账户管理
  - 收支记录
  - 分类管理
  - 统计报表
  - 腾讯云部署支持

## 许可证

MIT License
