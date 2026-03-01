#!/bin/bash
# 远程服务器初始化和部署脚本
# 在腾讯云服务器上执行

set -e

echo "=========================================="
echo "  财务系统 - 腾讯云服务器部署脚本"
echo "  服务器: 106.54.50.88"
echo "  实例: lhins-dz0yr098"
echo "=========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 1. 更新系统
log_info "更新系统软件包..."
sudo apt-get update -y

# 2. 安装 Docker
if ! command -v docker &> /dev/null; then
    log_info "安装 Docker..."
    curl -fsSL https://get.docker.com | sudo sh
    sudo systemctl enable docker
    sudo systemctl start docker
    sudo usermod -aG docker $USER
    log_info "Docker 安装完成"
else
    log_info "Docker 已安装: $(docker --version)"
fi

# 3. 安装 Docker Compose
if ! command -v docker-compose &> /dev/null; then
    log_info "安装 Docker Compose..."
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    log_info "Docker Compose 安装完成"
else
    log_info "Docker Compose 已安装: $(docker-compose --version)"
fi

# 4. 创建项目目录
PROJECT_DIR="/home/ubuntu/finance-system"
log_info "创建项目目录: $PROJECT_DIR"
mkdir -p $PROJECT_DIR
cd $PROJECT_DIR

# 5. 创建数据目录
mkdir -p data logs

log_info "=========================================="
log_info "环境初始化完成!"
log_info "项目目录: $PROJECT_DIR"
log_info "=========================================="
