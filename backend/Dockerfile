# 构建阶段 - 使用 Debian-based 镜像避免 musl 兼容性问题
FROM golang:1.21-bookworm AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的构建工具
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# 设置 Go 模块代理（国内加速）
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go.mod
COPY go.mod ./

# 复制源代码
COPY . .

# 下载依赖并更新 go.sum
RUN go mod tidy

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -o finance-server .

# 运行阶段
FROM debian:bookworm-slim

# 安装运行时依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata libsqlite3-0 wget \
    && rm -rf /var/lib/apt/lists/*

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN groupadd -g 1000 finance && \
    useradd -u 1000 -g finance -s /bin/sh -m finance

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/finance-server .

# 复制配置文件
COPY config/config.yaml /etc/finance/config.yaml

# 创建数据目录
RUN mkdir -p /app/data && chown -R finance:finance /app

# 切换到非 root 用户
USER finance

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动命令
ENTRYPOINT ["./finance-server"]
CMD ["-config", "/etc/finance/config.yaml"]
