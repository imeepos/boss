#!/bin/bash
# BOSS 生产集群一键部署脚本
# 用法:./scripts/deploy-cluster.sh [node-1|node-2|node-3|all]

set -e

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

# 节点配置
NODE1_IP="192.168.0.102"
NODE2_IP="192.168.0.103"
NODE3_IP="192.168.0.104"

# 检查 Docker 是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker 服务未启动，请启动 Docker"
        exit 1
    fi
    
    log_info "Docker 检查通过"
}

# 创建数据目录
create_data_dirs() {
    log_info "创建数据目录..."
    sudo mkdir -p /data/boss/{pgdata,miniodata,vmdata,etcd_data,prom_data,grafana_data,loki_data,redis-data}
    sudo chown -R $USER:$USER /data/boss
    log_info "数据目录创建完成"
}

# 部署基础设施 (node-1)
deploy_infra() {
    log_info "部署基础设施 (node-1)..."
    docker compose -f docker-compose.102.yml up -d
    
    log_info "等待基础设施就绪..."
    sleep 30
    
    # 检查服务状态
    docker compose -f docker-compose.102.yml ps
    log_info "基础设施部署完成"
}

# 部署 PostgreSQL 从节点 (node-2, node-3)
deploy_pg_replica() {
    log_info "部署 PostgreSQL 从节点..."
    
    # 复制配置文件
    cp docker-compose.pg-replica.yml /tmp/
    cp pg-replica-init.sh /tmp/
    chmod +x /tmp/pg-replica-init.sh
    
    # 在从节点运行
    ssh $NODE2_IP "cd /opt/boss && docker compose -f docker-compose.pg-replica.yml up -d"
    ssh $NODE3_IP "cd /opt/boss && docker compose -f docker-compose.pg-replica.yml up -d"
    
    log_info "PostgreSQL 从节点部署完成"
}

# 部署 Redis 哨兵 (所有节点)
deploy_redis_sentinel() {
    log_info "部署 Redis 哨兵..."
    
    # node-1: 主节点
    cp redis-master.conf /tmp/
    docker compose -f docker-compose.redis-sentinel.yml up -d
    
    # node-2, node-3: 从节点
    scp redis-replica.conf $NODE2_IP:/opt/boss/
    scp redis-replica.conf $NODE3_IP:/opt/boss/
    scp sentinel.conf $NODE2_IP:/opt/boss/
    scp sentinel.conf $NODE3_IP:/opt/boss/
    
    ssh $NODE2_IP "cd /opt/boss && docker compose -f docker-compose.redis-sentinel.yml up -d"
    ssh $NODE3_IP "cd /opt/boss && docker compose -f docker-compose.redis-sentinel.yml up -d"
    
    log_info "Redis 哨兵部署完成"
}

# 部署应用 (所有节点)
deploy_app() {
    log_info "部署应用服务..."
    
    # 复制配置文件到所有节点
    scp docker-compose.102.app.yml $NODE2_IP:/opt/boss/
    scp docker-compose.102.app.yml $NODE3_IP:/opt/boss/
    scp app.env $NODE2_IP:/opt/boss/
    scp app.env $NODE3_IP:/opt/boss/
    
    # 部署应用
    docker compose -f docker-compose.102.app.yml up -d
    ssh $NODE2_IP "cd /opt/boss && docker compose -f docker-compose.102.app.yml up -d"
    ssh $NODE3_IP "cd /opt/boss && docker compose -f docker-compose.102.app.yml up -d"
    
    log_info "应用服务部署完成"
}

# 部署扩展组件
deploy_extensions() {
    log_info "部署扩展组件..."
    
    # 复制配置文件到所有节点
    scp docker-compose.102.extend.yml $NODE2_IP:/opt/boss/
    scp docker-compose.102.extend.yml $NODE3_IP:/opt/boss/
    
    # 部署扩展组件
    docker compose -f docker-compose.102.extend.yml up -d
    ssh $NODE2_IP "cd /opt/boss && docker compose -f docker-compose.102.extend.yml up -d"
    ssh $NODE3_IP "cd /opt/boss && docker compose -f docker-compose.102.extend.yml up -d"
    
    log_info "扩展组件部署完成"
}

# 部署监控 (所有节点)
deploy_monitoring() {
    log_info "部署监控系统..."
    
    # 复制配置文件到所有节点
    scp docker-compose.monitoring.yml $NODE2_IP:/opt/boss/
    scp docker-compose.monitoring.yml $NODE3_IP:/opt/boss/
    scp -r observability/ $NODE2_IP:/opt/boss/
    scp -r observability/ $NODE3_IP:/opt/boss/
    
    # 部署监控
    docker compose -f docker-compose.monitoring.yml up -d
    ssh $NODE2_IP "cd /opt/boss && docker compose -f docker-compose.monitoring.yml up -d"
    ssh $NODE3_IP "cd /opt/boss && docker compose -f docker-compose.monitoring.yml up -d"
    
    log_info "监控系统部署完成"
}

# 部署负载均衡
deploy_nginx() {
    log_info "部署 Nginx 负载均衡..."
    
    # 创建 SSL 目录(如果不存在)
    mkdir -p nginx-ssl
    
    # 复制配置文件到所有节点
    scp docker-compose.nginx.yml $NODE2_IP:/opt/boss/
    scp docker-compose.nginx.yml $NODE3_IP:/opt/boss/
    scp nginx.conf $NODE2_IP:/opt/boss/
    scp nginx.conf $NODE3_IP:/opt/boss/
    scp -r nginx-ssl/ $NODE2_IP:/opt/boss/
    scp -r nginx-ssl/ $NODE3_IP:/opt/boss/
    
    # 部署 Nginx
    docker compose -f docker-compose.nginx.yml up -d
    ssh $NODE2_IP "cd /opt/boss && docker compose -f docker-compose.nginx.yml up -d"
    ssh $NODE3_IP "cd /opt/boss && docker compose -f docker-compose.nginx.yml up -d"
    
    log_info "Nginx 负载均衡部署完成"
}

# 部署备份服务
deploy_backup() {
    log_info "部署备份服务..."
    
    docker compose -f docker-compose.backup.yml up -d
    
    log_info "备份服务部署完成"
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    # 检查所有服务状态
    docker compose -f docker-compose.102.yml ps
    docker compose -f docker-compose.redis-sentinel.yml ps
    docker compose -f docker-compose.102.app.yml ps
    docker compose -f docker-compose.102.extend.yml ps
    docker compose -f docker-compose.monitoring.yml ps
    docker compose -f docker-compose.nginx.yml ps
    
    # 检查 boss-server 健康状态
    for i in {1..5}; do
        if curl -sf http://$NODE1_IP:28080/healthz > /dev/null; then
            log_info "node-1 boss-server 健康"
            break
        fi
        sleep 5
    done
    
    for i in {1..5}; do
        if curl -sf http://$NODE2_IP:28080/healthz > /dev/null; then
            log_info "node-2 boss-server 健康"
            break
        fi
        sleep 5
    done
    
    for i in {1..5}; do
        if curl -sf http://$NODE3_IP:28080/healthz > /dev/null; then
            log_info "node-3 boss-server 健康"
            break
        fi
        sleep 5
    done
    
    log_info "健康检查完成"
}

# 显示帮助
show_help() {
    echo "BOSS 生产集群部署脚本"
    echo ""
    echo "用法: $0 [command]"
    echo ""
    echo "命令:"
    echo "  all          部署所有组件"
    echo "  infra        部署基础设施 (node-1)"
    echo "  pg-replica   部署 PostgreSQL 从节点"
    echo "  redis        部署 Redis 哨兵"
    echo "  app          部署应用服务"
    echo "  extensions   部署扩展组件"
    echo "  monitoring   部署监控系统"
    echo "  nginx        部署负载均衡"
    echo "  backup       部署备份服务"
    echo "  health       执行健康检查"
    echo "  help         显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 all        # 部署所有组件"
    echo "  $0 infra      # 仅部署基础设施"
    echo "  $0 health     # 执行健康检查"
}

# 主函数
main() {
    local command=${1:-help}
    
    check_docker
    
    case $command in
        all)
            create_data_dirs
            deploy_infra
            deploy_pg_replica
            deploy_redis_sentinel
            deploy_app
            deploy_extensions
            deploy_monitoring
            deploy_nginx
            deploy_backup
            health_check
            ;;
        infra)
            create_data_dirs
            deploy_infra
            ;;
        pg-replica)
            deploy_pg_replica
            ;;
        redis)
            deploy_redis_sentinel
            ;;
        app)
            deploy_app
            ;;
        extensions)
            deploy_extensions
            ;;
        monitoring)
            deploy_monitoring
            ;;
        nginx)
            deploy_nginx
            ;;
        backup)
            deploy_backup
            ;;
        health)
            health_check
            ;;
        help|*)
            show_help
            ;;
    esac
}

main "$@"
