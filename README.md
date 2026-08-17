# BOSS 装维全流程服务端

基于《需求提示词-服务端-9阶段拆分.md》与《技术栈方案-一步到位.md》的 Go 模块化单体(monorepo),
按 9 个阶段渐进交付,后续可按域拆分为微服务,不换栈。

## 技术栈

| 层 | 选型 |
|---|---|
| 语言 | Go 1.22+ |
| Web/RPC | Gin(REST)+ gRPC + Protobuf |
| 编排 | Temporal(订单 11 环节状态机) |
| OLTP | PostgreSQL 16(ltree 地址层级、按月分区审计) |
| 缓存/锁 | Redis(redsync,端口预占互斥) |
| 消息 | Kafka(话单、状态变更、GIS 联动、配置下发) |
| 时序 | VictoriaMetrics(OLT 指标) |
| OLAP | Apache Doris(阶段 8/9) |
| 对象存储 | MinIO |
| 配置中心 | Nacos / etcd |
| 迁移 | golang-migrate |
| 观测 | OpenTelemetry + Prometheus + Loki |

## 目录结构

```
boss/
├── cmd/                        # 可执行入口(一个目录一个部署物)
│   ├── server/                 # 业务模块化单体:阶段1-6 的全部业务域
│   ├── aaa/                    # RADIUS AAA(阶段7,独立部署)
│   ├── collector/              # OLT SNMP 采集 + Trap 接收(阶段7)
│   ├── provisioner/            # 配置下发 worker(阶段7)
│   ├── gis/                    # GIS 联动服务(阶段8)
│   └── report/                 # 经营分析报告服务(阶段9)
├── internal/                   # 单体内部按域分包(边界=未来微服务边界)
│   ├── pkg/                    # 跨域共享基础设施(无业务语义)
│   │   ├── config/             # 配置加载(env/file/Nacos)
│   │   ├── database/           # PG(gorm/sqlc)、Redis、Kafka、MinIO 客户端
│   │   ├── auth/               # JWT 签发校验、RBAC 快照
│   │   ├── audit/              # 审计写入(异步、按月分区表)
│   │   ├── statemachine/       # 定义驱动的状态机(state/event/guard)
│   │   ├── lock/               # redsync 分布式锁
│   │   ├── middleware/         # gin 中间件:鉴权、租期、trace、recover
│   │   └── server/             # gin engine、grpc server、优雅退出组装
│   ├── domain/                 # 业务域(只经接口依赖彼此,禁止横向 import 实现)
│   │   ├── user/               # 阶段1:账号、角色、权限、区域地址层级
│   │   ├── customer/           # 阶段2:客户档案、产品资费
│   │   ├── asset/              # 阶段3:资产台账、电子标签、生命周期、盘点
│   │   ├── resource/           # 阶段4:OLT/分光器/端口台账
│   │   ├── order/              # 阶段5:订单、11 环节、端口预占、派单
│   │   ├── billing/            # 阶段5:出账、缴费、欠费停复机
│   │   ├── quadlink/           # 阶段6:四码合一、扫码强制、对账
│   │   ├── aaa/                # 阶段7:LOID 认证、授权缓存、话单入口
│   │   ├── device/             # 阶段7:OLT 管理、指标、告警
│   │   ├── provision/          # 阶段7:模板渲染、下发重试队列
│   │   ├── gis/                # 阶段8:asset.changed 消费、实体同步
│   │   └── analytics/          # 阶段9:五大指标、热力图、报告
│   └── app/                    # 单体装配:wiring、路由注册、依赖注入
├── pkg/                        # 可被外部服务(aaa/collector)复用的公共库
│   └── apitypes/               # 跨服务 DTO/错误码
├── api/                        # 接口契约
│   ├── proto/                  # 服务间 gRPC 契约(quadlink/aaa/device)
│   └── openapi/                # 对外 REST OpenAPI 3(APISIX 路由依据)
├── migrations/                 # golang-migrate SQL(up/down 成对)
├── deployments/                # 部署物
│   ├── docker-compose.infra.yml # 本地全套基础设施(PG/Redis/Kafka/Nacos/...)
│   ├── docker/                 # 各服务 Dockerfile
│   └── helm/                   # K8s Helm chart
├── scripts/                    # dev/setup.sh、proto 生成、k6 压测
├── docs/                       # 阶段设计文档、ADR
├── Makefile
└── go.mod
```

## 快速开始

```bash
# 1. 启动本地基础设施
make infra-up

# 2. 执行数据库迁移
make migrate-up

# 3. 运行业务单体
make run
```

## 阶段映射

| 阶段 | 落点 |
|---|---|
| 1 基础平台 | internal/domain/user + internal/pkg/{auth,audit} |
| 2 客户与资费 | internal/domain/customer |
| 3 资产台账 | internal/domain/asset |
| 4 资源台账 | internal/domain/resource |
| 5 订单/计费 | internal/domain/{order,billing} + Temporal |
| 6 四码合一 | internal/domain/quadlink + api/proto |
| 7 业网融合 | cmd/{aaa,collector,provisioner} + internal/domain/{aaa,device,provision} |
| 8 数字孪生 GIS | cmd/gis + internal/domain/gis |
| 9 经营分析 | cmd/report + internal/domain/analytics + Doris |
