# ADR-001:模块化单体 + monorepo,而非一开始微服务

技术栈方案要求"模块化单体 → 可拆微服务"。9 阶段只有阶段7(aaa/collector/provisioner)
和阶段8/9(gis/report)需要独立部署物;其余域先在同一进程内以接口依赖协作。

## 约束
1. `internal/domain/*` 之间禁止直接 import 对方内部实现,只依赖在各自包内声明的接口;
   装配(internal/app)负责绑定实现 —— 保证未来按域拆分只改装配与传输层。
2. `internal/pkg` 只放无业务语义的基础设施;带业务语义的(状态机、审计)也保持可复用。
3. 高并发路径(端口预占)从第一天用 sqlc 手写 SQL + `SELECT ... FOR UPDATE` + DB 唯一约束,
   Redis 锁只做削峰(见技术栈方案 3.1)。
4. 契约先行:服务间 gRPC(api/proto)、对外 REST(api/openapi),网关 APISIX 按契约生成路由。

## 后续 ADR
- [ADR-002](ADR-002-address-path-authoritative.md):地址层级以 ltree `path` 为唯一权威。
- [ADR-003](ADR-003-statemachine-authoritative.md):状态机为唯一权威,Temporal 只做编排。
