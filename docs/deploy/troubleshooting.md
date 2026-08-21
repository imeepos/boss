# BOSS 生产集群故障排查手册

## 常见问题

### 1. PostgreSQL 复制延迟

**症状**: Grafana 面板显示复制延迟 > 10 秒

**排查步骤**:
```bash
# 检查主节点 WAL 发送进程
docker exec -it boss-postgres psql -U boss -c "SELECT * FROM pg_stat_replication;"

# 检查从节点接收进程
docker exec -it boss-pg-replica psql -U boss -c "SELECT * FROM pg_stat_wal_receiver;"

# 检查网络延迟
ping 192.168.0.103
ping 192.168.0.104

# 检查磁盘 IO
iostat -x 1
```

**解决方案**:
1. 如果是网络问题:检查网络配置,优化网络延迟
2. 如果是磁盘 IO 问题:升级 SSD,调整 WAL 写入策略
3. 如果是配置问题:调整 `wal_sender_timeout` 和 `wal_receiver_timeout`

### 2. Redis 哨兵故障转移失败

**症状**:Redis 主节点宕机后,从节点未自动提升为主

**排查步骤**:
```bash
# 检查哨兵状态
docker exec -it boss-redis-sentinel redis-cli -p 26379 sentinel masters

# 检查哨兵日志
docker logs boss-redis-sentinel

# 检查从节点状态
docker exec -it boss-redis redis-cli info replication
```

**解决方案**:
1. 检查哨兵配置:确认 `quorum` 设置正确
2. 检查网络:确认哨兵之间可以通信
3. 手动故障转移:执行 `docker exec -it boss-redis-sentinel redis-cli -p 26379 sentinel failover boss-master`

### 3. Kafka 消费延迟

**症状**:Prometheus 告警 KafkaConsumerLag > 1000

**排查步骤**:
```bash
# 检查消费者组状态
docker exec -it boss-kafka kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group boss-consumer-group

# 检查 topic 分区状态
docker exec -it boss-kafka kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic boss-order-events

# 检查 broker 状态
docker exec -it boss-kafka kafka-broker-api-versions.sh --bootstrap-server localhost:9092
```

**解决方案**:
1. 增加消费者实例:提高消费并行度
2. 增加分区数:提高 topic 并行度
3. 优化消费逻辑:减少单条消息处理时间

### 4. boss-server 无法启动

**症状**:容器启动后立即退出,日志显示连接数据库失败

**排查步骤**:
```bash
# 查看容器日志
docker logs boss-server

# 检查数据库连接
docker exec -it boss-server wget -qO- http://localhost:8080/healthz

# 检查环境变量
docker exec -it boss-server env | grep BOSS_

# 检查网络
docker exec -it boss-server ping 192.168.0.102
```

**解决方案**:
1. 检查数据库配置:确认 `BOSS_PG_DSN` 配置正确
2. 检查网络:确认容器可以访问数据库
3. 检查数据库状态:确认数据库已启动并可访问

### 5. MinIO 访问失败

**症状**:应用无法上传文件,日志显示连接 MinIO 失败

**排查步骤**:
```bash
# 检查 MinIO 服务状态
docker logs boss-minio

# 检查 MinIO 访问
curl http://192.168.0.102:29000/minio/health/live

# 检查 MinIO 配置
docker exec -it boss-minio mc alias set boss http://localhost:9000 boss boss12345
```

**解决方案**:
1. 检查 MinIO 配置:确认访问密钥和秘密密钥正确
2. 检查网络:确认应用可以访问 MinIO
3. 检查存储空间:确认 MinIO 存储空间充足

### 6. Nginx 负载均衡不生效

**症状**:请求未均匀分配到各节点

**排查步骤**:
```bash
# 检查 Nginx 配置
docker exec -it boss-nginx nginx -t

# 检查上游服务器状态
docker exec -it boss-nginx curl http://localhost/nginx-health

# 检查 Nginx 日志
docker logs boss-nginx
```

**解决方案**:
1. 检查上游配置:确认上游服务器地址和端口正确
2. 检查健康检查:确认健康检查端点可用
3. 重新加载配置:执行 `docker exec -it boss-nginx nginx -s reload`

## 紧急恢复流程

### PostgreSQL 主节点故障

1. **立即操作**:
   ```bash
   # 提升从节点为主
   docker exec -it boss-pg-replica pg_ctl promote
   
   # 修改应用配置
   # 在所有节点修改 app.env,将 BOSS_PG_DSN 指向新主节点
   ```

2. **后续操作**:
   - 重建故障节点为新从节点
   - 检查数据一致性
   - 更新监控配置

### Redis 主节点故障

1. **自动故障转移**:哨兵会自动将从节点提升为主节点
2. **验证**:
   ```bash
   docker exec -it boss-redis-sentinel redis-cli -p 26379 sentinel masters
   ```
3. **如果自动转移失败**:手动执行故障转移

### 全部节点故障

1. **数据恢复**:
   ```bash
   # 从备份恢复 PostgreSQL
   docker exec -it boss-pg-backup pg_restore -U boss -d boss /backups/pg/boss_full_YYYYMMDD_HHMMSS.sql.gz
   
   # 从备份恢复 MinIO
   mc mirror boss/boss-backups/minio/ /data/minio/
   ```

2. **服务恢复**:
   - 按顺序启动基础设施
   - 启动应用服务
   - 验证服务状态

## 监控指标说明

### 关键指标

| 指标 | 说明 | 告警阈值 |
|------|------|----------|
| `up{job="boss-server"}` | boss-server 存活状态 | == 0 告警 |
| `pg_stat_activity_count` | PostgreSQL 连接数 | > 80 告警 |
| `pg_replication_lag` | PostgreSQL 复制延迟 | > 10s 告警 |
| `redis_memory_used_bytes` | Redis 内存使用量 | > 80% 告警 |
| `kafka_consumer_lag_sum` | Kafka 消费延迟 | > 1000 告警 |
| `node_cpu_seconds_total` | CPU 使用率 | > 80% 告警 |
| `node_memory_MemAvailable_bytes` | 可用内存 | < 15% 告警 |
| `node_filesystem_avail_bytes` | 可用磁盘空间 | < 15% 告警 |

### Grafana 面板

1. **BOSS 集群概览**:显示所有节点状态
2. **PostgreSQL 监控**:显示数据库连接数、复制延迟、查询性能
3. **Redis 监控**:显示内存使用、连接数、命中率
4. **Kafka 监控**:显示消费延迟、分区状态、Broker 状态
5. **节点监控**:显示 CPU、内存、磁盘、网络使用情况
