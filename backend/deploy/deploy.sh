#!/bin/bash
# 腾讯云部署脚本

set -e

# 配置变量
IMAGE_REGISTRY="ccr.ccs.tencentyun.com"
IMAGE_NAMESPACE="finance"
IMAGE_NAME="finance-backend"
IMAGE_TAG="${IMAGE_TAG:-latest}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 构建 Docker 镜像
build_image() {
    log_info "构建 Docker 镜像..."
    docker build -t ${IMAGE_REGISTRY}/${IMAGE_NAMESPACE}/${IMAGE_NAME}:${IMAGE_TAG} .
}

# 推送镜像到腾讯云容器镜像服务
push_image() {
    log_info "推送镜像到腾讯云..."

    # 登录腾讯云镜像仓库
    if [ -z "$TCR_USERNAME" ] || [ -z "$TCR_PASSWORD" ]; then
        log_warn "未设置 TCR_USERNAME 和 TCR_PASSWORD 环境变量"
        log_info "请手动登录: docker login ${IMAGE_REGISTRY}"
    else
        echo "$TCR_PASSWORD" | docker login ${IMAGE_REGISTRY} -u "$TCR_USERNAME" --password-stdin
    fi

    docker push ${IMAGE_REGISTRY}/${IMAGE_NAMESPACE}/${IMAGE_NAME}:${IMAGE_TAG}
    log_info "镜像推送完成"
}

# Docker Compose 部署 (单机)
deploy_compose() {
    log_info "使用 Docker Compose 部署..."

    if [ "$1" == "mysql" ]; then
        docker-compose -f docker-compose.mysql.yaml up -d
    else
        docker-compose up -d
    fi

    log_info "Docker Compose 部署完成"
}

# Kubernetes 部署 (TKE)
deploy_k8s() {
    log_info "部署到腾讯云 TKE..."

    # 检查 kubectl 配置
    if ! kubectl cluster-info &> /dev/null; then
        log_error "kubectl 未配置或无法连接到集群"
        exit 1
    fi

    # 使用 Kustomize 部署
    kubectl apply -k deploy/k8s/

    # 等待部署完成
    log_info "等待部署完成..."
    kubectl rollout status deployment/finance-backend -n finance-system

    log_info "Kubernetes 部署完成"
}

# 更新部署
update_k8s() {
    log_info "更新 Kubernetes 部署..."

    # 更新镜像
    kubectl set image deployment/finance-backend \
        finance-backend=${IMAGE_REGISTRY}/${IMAGE_NAMESPACE}/${IMAGE_NAME}:${IMAGE_TAG} \
        -n finance-system

    # 等待部署完成
    kubectl rollout status deployment/finance-backend -n finance-system

    log_info "更新完成"
}

# 回滚部署
rollback_k8s() {
    log_info "回滚 Kubernetes 部署..."
    kubectl rollout undo deployment/finance-backend -n finance-system
    kubectl rollout status deployment/finance-backend -n finance-system
    log_info "回滚完成"
}

# 查看状态
status() {
    log_info "部署状态:"
    kubectl get pods,svc,ingress -n finance-system
}

# 查看日志
logs() {
    kubectl logs -f -l app=finance,component=backend -n finance-system --tail=100
}

# 帮助信息
show_help() {
    echo "腾讯云部署脚本"
    echo ""
    echo "用法: $0 <命令> [选项]"
    echo ""
    echo "命令:"
    echo "  build           构建 Docker 镜像"
    echo "  push            推送镜像到腾讯云"
    echo "  compose [mysql] Docker Compose 部署 (单机)"
    echo "  deploy          部署到 TKE (Kubernetes)"
    echo "  update          更新 TKE 部署"
    echo "  rollback        回滚 TKE 部署"
    echo "  status          查看部署状态"
    echo "  logs            查看应用日志"
    echo ""
    echo "环境变量:"
    echo "  IMAGE_TAG       镜像标签 (默认: latest)"
    echo "  TCR_USERNAME    腾讯云镜像仓库用户名"
    echo "  TCR_PASSWORD    腾讯云镜像仓库密码"
}

# 主函数
case "$1" in
    build)
        build_image
        ;;
    push)
        push_image
        ;;
    compose)
        deploy_compose "$2"
        ;;
    deploy)
        deploy_k8s
        ;;
    update)
        update_k8s
        ;;
    rollback)
        rollback_k8s
        ;;
    status)
        status
        ;;
    logs)
        logs
        ;;
    *)
        show_help
        ;;
esac
