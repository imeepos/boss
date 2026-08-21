# 2026-08-21 数据备份迁移:本地磁盘 gzip JSONL 归档 + 追加式恢复

## 裁定

运维用"数据备份迁移"功能(后台 /base/backup,menu:backup)落为:

1. **归档格式 = gzip JSONL(自描述 v1)**:首行 `{"v":1}`;每表一行头 `{"t":表名,"c":[列名],"u":[pg udt 名]}`,之后每行 `{"r":[行值]}`。
2. **存储 = 服务端本地磁盘目录**(env `BOSS_BACKUP_DIR`,默认 `data/backups` 相对工作目录),文件名 `backup-<任务id>-<UTC时间戳>.jsonl.gz`;下载经 `/backup/jobs/:id/file`(Bearer 鉴权)流式返回。
3. **恢复语义 = 只补不删**:逐行 `INSERT ... ON CONFLICT DO NOTHING`,冲突行静默跳过;不 TRUNCATE、不 UPDATE。定位是环境间配置/基础数据迁移,不是灾备点恢复。
4. **执行模型 = 进程内串行**(Service 互斥锁,同时至多一条任务),任务簿记入 `backup_jobs` 表(迁移 000095)。
5. 备份候选集 = public 普通表,**排除 backup_jobs 与 schema_migrations**。

## 为什么(Why)

- 不用 pg_dump/pg_restore:服务跑在 102 Docker 内,不保证容器内有 pg_dump 二进制,也不应把 shell 权限暴露给 web 进程;自研 SQL 流式导出零外部依赖、格式自描述可校验。
- 不入 MinIO:运维归档属临时产物,本地磁盘 + 明确目录已满足"下载/再导入"闭环;引入对象存储会耦合 MinIO 配置可用性(attachment 域的教训:MinIO 未配置时上传直接不可用)。
- 只补不删:恢复若做 TRUNCATE/替换,在任何有外键的库上都无法保证顺序与一致性,且误操作不可逆;追加导入天然幂等、可重放,符合"迁移基础数据"的主场景。
- 串行执行:备份走流式查询占用连接,恢复是写入密集;运维工具无需并发,串行 + busy 码(42300)最简单可靠。

## 放弃了什么

- pg_dump 物理快照(依赖容器内二进制,放弃)。
- MinIO 对象存储归档(依赖外部可用性,放弃;将来跨主机迁移需求出现再演进)。
- 恢复时删改既有数据的能力(不可逆,明确不做;如需重置环境走 DBA 流程)。
- 数组列仅支持元素全字符串(text[]);bytea 走 base64 往返;嵌套/混合数组与自定义复合类型不支持(按失败处理,报错可见)。

## 关联

- 契约:docs/contract/fields.md 1.5.6;迁移 000095。
- 域:internal/domain/backup(SYS 域运维工具)。
