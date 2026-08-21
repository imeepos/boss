# BOSS 生产集群部署方案（3 节点）

## 架构总览

```
                    ┌─────────────────────────────────────────────┐
                    │              外部访问                        │
                    │   boss.ymm.cn → Nginx (任意节点)             │
                    └──────────────┬──────────────────────────────┘
                                   │
          ┌────────────────────────┼────────────────────────┐
          │                        │                        │
    ┌─────┴─────┐           ┌─────┴─────┐           ┌─────┴─────┐
    │  node-1   │           │  node-2   │           │  node-3   │
    │ (主节点)  │           │ (应用节点)│           │ (应用节点)│
    │ 192.168.0.│           │ 192.168.0.│           │ 192.168.0.│
    │   102     │           │   103     │           │   104     │
    ├───────────┤           ├───────────┤           ├───────────┤
    │ PG 主     │           │ PG 从     │           │ PG 从     │
    │ Redis 主  │◄─────────►│ Redis 从  │◄─────────►│ Redis 从  │
    │ Kafka     │           │ Kafka     │           │ Kafka     │
    │ Nacos     │           │ Nacos     │           │ Nacos     │
    │ MinIO     │           │ MinIO     │           │ MinIO     │
    │ Temporal  │           │ Temporal  │           │ Temporal  │
    │ VM        │           │ VM        │           │ VM        │
    ├───────────┤           ├───────────┤           ├───────────┤
    │ boss-srv  │           │ boss-srv  │           │ boss-srv  │
    │ boss-rpt  │           │ boss-rpt  │           │ boss-rpt  │
    │ APISIX    │           │ APISIX    │           │ APISIX    │
    │ Prometheus│           │ Grafana   │           │ Jaeger    │
    │ Loki      │           │           │           │           │
    └───────────┘           └───────────┘           └───────────┘
```

## 节点配置

| 节点 | IP | 角色 | 推荐配置 |
|------|-----|------|----------|
| node-1 | 192.168.0.102 | 主节点（PG主/Redis主/Kafka/Nacos/MinIO/Temporal/VM） | 16C 32G 1T SSD |
| node-2 | 192.168.0.103 | 应用节点1（PG从/Redis从/应用副本/APISIX） | 16C 32G 1T SSD |
| node-3 | 192.168.0.104 | 应用节点2（PG从/Redis从/应用副本/监控） | 16C 32G 1T SSD |

## 端口规划

| 服务 | 端口 | 说明 |
|------|------|------|
| PostgreSQL | 25432 | 主数据库 |
| PostgreSQL 复制 | 25433 | 流复制端口 |
| Redis | 26379 | 缓存 |
| Redis Sentinel | 26380 | 哨兵 |
| Kafka | 29092 | 消息队列 |
| Nacos | 18848 | 配置中心 |
| MinIO | 29000/29001 | 对象存储 |
| Temporal | 17233 | 工作流 |
| VictoriaMetrics | 18428 | 时序数据库 |
| boss-server HTTP | 28080 | 业务 API |
| boss-server gRPC | 29090 | gRPC 服务 |
| APISIX | 29080/29180 | API 网关 |
| Prometheus | 19090 | 指标采集 |
| Grafana | 19300 | 监控面板 |
| Loki | 19310 | 日志 |
| Jaeger | 16686 | 链路追踪 |

## 部署顺序

### 1. 准备工作（所有节点）

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 安装 docker-compose
sudo apt-get install -y docker-compose-plugin

# 创建数据目录
sudo mkdir -p /data/boss/{pgdata,miniodata,vmdata,etcd_data,prom_data,grafana_data,loki_data}
sudo chown -R $USER:$USER /data/boss
```

### 2. 部署基础设施（node-1 先行）

```bash
cd /opt/boss
docker compose -f docker-compose.102.yml up -d
```

### 3. 部署 PG 从节点（node-2, node-3）

```bash
docker compose -f docker-compose.pg-replica.yml up -d
```

### 4. 部署 Redis 哨兵（所有节点）

```bash
docker compose -f docker-compose.redis-sentinel.yml up -d
```

### 5. 部署应用（所有节点）

```bash
docker compose -f docker-compose.102.app.yml up -d
```

### 6. 部署扩展组件

```bash
docker compose -f docker-compose.102.extend.yml up -d
```

## 数据安全策略

### PostgreSQL

- **流复制**：1 主 2 从，异步复制，RPO < 1 秒
- **每日全量备份**：pg_dumpall → MinIO，保留 30 天
- **WAL 归档**：连续归档到 MinIO，支持任意时间点恢复
- **监控**：复制延迟 > 10s 告警

### Redis

- **哨兵模式**：3 节点哨兵，自动故障转移
- **RDB 持久化**：每 15 分钟快照
- **AOF 持久化**：每秒 fsync
- **监控**：内存使用 > 80% 告警

### MinIO

- **纠删码**：4 节点模式，容忍 2 节点故障
- **版本控制**：关键桶开启版本控制
- **跨节点复制**：重要数据同步到备份节点

### Kafka

- **3 副本**：每个 topic 3 副本，ISR >= 2
- **日志保留**：7 天或 100GB
- **监控**：消费延迟 > 1000 条告警

## 故障恢复

### PostgreSQL 主节点故障

```bash
# 1. 提升从节点为主
docker exec -it pg-replica-1 pg_ctl promote

# 2. 修改应用配置指向新主
# 3. 重建故障节点为新从
```

### Redis 主节点故障

哨兵自动故障转移，无需人工干预。

### 应用节点故障

Nginx 自动摘除故障节点，其他节点继续服务。
