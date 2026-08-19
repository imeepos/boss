# 决策记录（Agent Notes）

> 制度依据 dsh-codebase-wisdom 01-foundation/09 与 07-design/02：每个不可逆决策一篇 dated note，记 why 与放弃了什么。

## 规则

1. 任何不可逆/难逆决策（密钥入库、双模型并存、域分合、序列/ID 方案、内置数据方式），落地当天写一篇 `yyyy-mm-dd-topic-title.md` 进对应 lifecycle 目录。
2. note 结构：`# 标题` / `日期` / `决策` / `why` / `放弃了什么（被否决项）` / `关联`。不写实现细节，实现看代码。
3. 决策被推翻时**不删除**原文，在文中加 `> Amended yyyy-mm-dd: ...` 指向新 note（决策代谢：冻结 + cross-link + amend，不销毁）。
4. ADR（docs/ADR-*.md）仍保留给架构级大决策；note 记"日常但不可逆"的裁定。两者互链。
5. 评审 finding 的编号沿用现有体系：架构评审 `发现 N.M`（architecture-review.md），契约对账 `D/A/B/E/G#`（contract/alignment-audit.md），事后复盘 `docs/postmortem/000N-*`。

## 目录

- `adopted/` 已采纳并生效的决策
- `rejected/` 调研后否决的方案（记否决理由，防止重提）

## 索引

| 日期 | 决策 | 文件 |
|---|---|---|
| 2026-08-17 | order_no 改 DB 序列生成 | adopted/2026-08-17-order-no-db-sequence.md |
| 2026-08-17 | server-ts 作为 TS 实体镜像层保留 | adopted/2026-08-17-server-ts-entity-mirror.md |
| 2026-08-18 | app.env 固定密钥直接入库 | adopted/2026-08-18-app-env-in-repo.md |
| 2026-08-18 | geo 与 gis 分立两个域 | adopted/2026-08-18-geo-vs-gis-split.md |
| 2026-08-18 | 菲律宾行政区划以 migration 全量内置 | adopted/2026-08-18-psgc-builtin-migration.md |
| 2026-08-18 | 发票 ARN 发号采用行锁计数表（非 PG SEQUENCE） | adopted/2026-08-18-tax-invoice-arn-numbering.md |
