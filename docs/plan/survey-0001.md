# 全库 Survey #1（2026-08-18）

> 方法：04-simplification/04——按域分治、逐候选分类、批量提案。本 survey 是节律起点（生产期每 2-4 周一次）。
> 结论先行：**无空壳域、无死代码强候选**；候选集中在"文档滞后"与"两处轻量超额"。3 天库龄，删减面本来就小——本次价值在于建立台账,下次有账可比。

## 分类

| # | 候选 | 分类 | 裁决/动作 |
|---|---|---|---|
| S1 | `internal/domain/ai`、`apikey`、`geo`、`report`、`worker` 未列入 README 目录树 | 文档滞后 | KEEP+修：README 目录树补 5 域（已执行） |
| S2 | `server-ts/`（TS 实体镜像）与 `web/admin` 并存 | 双物嫌疑 | KEEP：职责已裁定（note 2026-08-17），server-ts 不部署不连库；W2 门禁将其对账机械化后价值更高 |
| S3 | 根目录 `需求提示词-*.md`、`技术栈方案-*.md`（原始输入） | 归档候选 | KEEP 暂缓：9 阶段未走完，仍是需求权威；阶段 9 后迁 docs/archive/ |
| S4 | `internal/domain/asset/pg.go` 316 行、`user/pg.go` 309 行 | 超 300 红线 | 已进 check-contract-sync baseline 豁免，整改随下次域内改动顺手拆（读侧/写侧分文件） |
| S5 | `internal/domain/gis`、`analytics` 等阶段 8/9 域已有实现 | 提前生长 | KEEP：均有 pg 实现+集成测试,不是空壳；阶段推进时按 seam 模板继续 |
| S6 | 仓库根 `boss-entities-er.*`（drawio 三格式） | 位置 | 暂缓：docs/contract/ 内已有 data-model 系列,二选一在下次文档清理时定,登记在案 |

## 无强候选的证据

- 17 个 domain 目录全部有非 doc 实现（最少 apikey 3 文件）。
- TODO/FIXME 存量 3 个，无"标记比代码活得久"。
- 无 revert 矿可挖（0 revert），无孤儿测试。

## 下次 survey 入口（2-4 周后）

1. 对照本台账逐项复核 S3/S6；
2. web/admin 页面数 vs admin.yaml 路径数增长曲线；
3. baseline 豁免清单是否清零（S4）。
