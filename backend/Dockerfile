# 构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的构建工具（包括 sqlite-dev 用于静态编译）
RUN apk add --no-cache gcc musl-dev sqlite-dev

# 设置 Go 模块代理（国内加速）
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go.mod
COPY go.mod ./

# 复制源代码
COPY . .

# 下载依赖并更新 go.sum
RUN go mod tidy

# 构建应用（动态链接，避免 SQLite 静态编译问题）
RUN CGO_ENABLED=1 GOOS=linux go build -o finance-server .

# 运行阶段
FROM alpine:3.19

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata sqlite-libs

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 finance && \
    adduser -u 1000 -G finance -s /bin/sh -D finance

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
