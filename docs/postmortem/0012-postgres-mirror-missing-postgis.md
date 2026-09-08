# 0012: postgres 私有镜像缺 postgis,地址创建 58P01 内部错误

- 日期: 2026-09-09
- 等级: P2(核心写路径部分不可用,只读与多数写路径正常)
- 发现人: 负责人(W4/W5 验收复跑,create address 连续失败)

## 现象

- W4/W5 验收脚本在 POST /addresses 一步失败:HTTP 200 信封内 code=50000 内部错误。
- 服务端日志: user: create address: ERROR: could not access file `$libdir/postgis-3`: No such file or directory (SQLSTATE 58P01)。
- W2/W6/W8 与 T2 验收同期全绿——只有触发空间函数的路径(地址创建)挂。

## 根因链

1. 架构评审发现 2.4 已裁定:迁移用 PostGIS 类型,postgres 必须用 postgis/postgis:16-3.4(infra.yml 权威定义)。
2. docker-compose.102.yml(私有仓库镜像全量钉版)却把 postgres 覆盖成 192.168.0.102:5000/boss/postgres:16——该镜像 2026-07-07 构建,是纯 postgres:16,无 postgis。种子镜像时漏掉了 postgis 要求。
3. 覆盖文件与权威文件并存时,override 的 image 赢。2026-09-08 12:23 CST postgres 容器因 infra 栈重建被以纯镜像重新拉起;数据卷 pgdata 里的扩展元数据仍在(pg_extension 有 postgis),于是常规读写全部正常。
4. postgis 的 .so 是懒加载:不调用空间函数就不会碰 $libdir。事故因此潜伏近一天,直到首个地址创建(触发空间函数)才炸。

## 影响

- 地址创建不可用,依赖地址的验收/业务流(W4 许可、W5 投资测算)阻塞;其余路径无感。
- 无数据丢失:卷未动,纯镜像文件层问题。

## 修复

- deployments/docker-compose.102.yml postgres 镜像改回 postgis/postgis:16-3.4(宿主机已预载,与 infra.yml 对齐)。
- 容器以正确镜像重建(命名卷 pgdata 不动),postgis_full_version() 恢复,地址创建恢复,W4/W5 验收复跑通过。

## 教训

1. 镜像覆盖必须逐字段对齐权威定义的能力前提:钉私有镜像省的是网,丢的是能力;种子镜像时按 infra.yml 的裁定逐项核对(本例 postgis/ltree)。
2. 懒加载扩展的故障是潜伏的:扩展元数据在库、函数文件不在容器,查 pg_extension 完全正常——验收必须覆盖真正调用扩展函数的路径(地址创建正是 postgis 的最小探针)。
3. infra 栈重建(宿主机操作/磁盘清理/重启)会让从未显式修过的错误配置首次生效——变更后应跑一条全链冒烟(含空间路径)而非只看容器 healthy。
