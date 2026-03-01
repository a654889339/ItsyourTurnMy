# 腾讯云部署指南

本文档介绍如何将财务系统后端部署到腾讯云。

## 部署方式

### 方式一: 腾讯云轻量应用服务器 (Docker Compose)

适合小型项目或测试环境。

#### 1. 准备工作

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh
sudo systemctl enable docker
sudo systemctl start docker

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

#### 2. 部署应用

```bash
# 克隆代码
git clone <your-repo-url>
cd backend

# 设置环境变量 (可选)
export JWT_SECRET="your-strong-jwt-secret"

# 使用 SQLite (简单部署)
docker-compose up -d

# 或使用 MySQL
export MYSQL_PASSWORD="your-mysql-password"
docker-compose -f docker-compose.mysql.yaml up -d
```

#### 3. 验证部署

```bash
# 检查容器状态
docker-compose ps

# 检查健康状态
curl http://localhost:8080/health
```

---

### 方式二: 腾讯云容器服务 TKE (Kubernetes)

适合生产环境，支持高可用和自动扩缩容。

#### 1. 前置条件

- 已创建 TKE 集群
- 已配置 kubectl 连接到集群
- 已开通腾讯云容器镜像服务 (TCR)

#### 2. 推送镜像

```bash
# 登录腾讯云容器镜像服务
docker login ccr.ccs.tencentyun.com -u <用户名>

# 构建并推送镜像
docker build -t ccr.ccs.tencentyun.com/<命名空间>/finance-backend:v1.0.0 .
docker push ccr.ccs.tencentyun.com/<命名空间>/finance-backend:v1.0.0
```

#### 3. 配置 Secrets

```bash
# 创建命名空间
kubectl apply -f deploy/k8s/namespace.yaml

# 修改 secret.yaml 中的配置 (base64 编码)
# 生成 base64: echo -n 'your-value' | base64
kubectl apply -f deploy/k8s/secret.yaml
```

#### 4. 部署应用

```bash
# 使用 Kustomize 一键部署
kubectl apply -k deploy/k8s/

# 或逐个部署
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
kubectl apply -f deploy/k8s/ingress.yaml
kubectl apply -f deploy/k8s/hpa.yaml
```

#### 5. 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n finance-system

# 查看服务
kubectl get svc -n finance-system

# 查看日志
kubectl logs -f -l app=finance -n finance-system
```

---

## 配置说明

### config.yaml 配置项

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.http_port` | HTTP 端口 | 8080 |
| `server.grpc_port` | gRPC 端口 | 9090 |
| `server.mode` | 运行模式 | release |
| `database.driver` | 数据库类型 | sqlite |
| `jwt.secret` | JWT 密钥 | - |
| `jwt.expire_hours` | Token 有效期 | 24 |

### 环境变量

所有配置都可以通过环境变量覆盖：

| 环境变量 | 说明 |
|----------|------|
| `HTTP_PORT` | HTTP 端口 |
| `DB_DRIVER` | 数据库类型 (sqlite/mysql) |
| `MYSQL_HOST` | MySQL 主机 |
| `MYSQL_PORT` | MySQL 端口 |
| `MYSQL_USER` | MySQL 用户名 |
| `MYSQL_PASSWORD` | MySQL 密码 |
| `MYSQL_DATABASE` | MySQL 数据库名 |
| `JWT_SECRET` | JWT 密钥 |
| `LOG_LEVEL` | 日志级别 |

---

## 生产环境建议

### 数据库

- 使用腾讯云 TencentDB for MySQL
- 开启自动备份
- 配置读写分离 (高并发场景)

### 安全

- 修改默认 JWT 密钥
- 配置 HTTPS (使用腾讯云 SSL 证书)
- 限制 CORS 来源
- 启用腾讯云 WAF

### 监控

- 配置腾讯云 CLS 日志服务
- 配置腾讯云 Prometheus 监控
- 设置告警规则

### 高可用

- 部署多副本 (replicas >= 2)
- 配置 HPA 自动扩缩容
- 使用腾讯云 CLB 负载均衡

---

## 常用命令

```bash
# 构建镜像
./deploy/deploy.sh build

# 推送镜像
./deploy/deploy.sh push

# Docker Compose 部署
./deploy/deploy.sh compose

# K8s 部署
./deploy/deploy.sh deploy

# 更新部署
./deploy/deploy.sh update

# 回滚
./deploy/deploy.sh rollback

# 查看状态
./deploy/deploy.sh status

# 查看日志
./deploy/deploy.sh logs
```

---

## 故障排查

### 容器无法启动

```bash
# 查看容器日志
docker logs finance-backend

# K8s 查看事件
kubectl describe pod -l app=finance -n finance-system
```

### 数据库连接失败

1. 检查数据库地址和端口
2. 检查用户名密码
3. 检查网络安全组规则

### 健康检查失败

```bash
# 测试健康检查接口
curl http://localhost:8080/health
```
