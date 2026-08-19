# Postmortem 约定

> 依据 09-signals/02：事故记录随修复同提交，不事后补写、不排队。

## 规则

1. 触发条件：生产/102 环境事故、e2e 一次性没过的真 bug、部署失败链（如 compose 二进制损坏那次）、数据修复。纯代码笔误不立档。
2. 命名：`000N-yyyy-mm-dd-slug.md`，编号单调递增。
3. 内容五段：现象 / 根因（机理而非人责）/ 修复 / 防回归（测试或门禁引用）/ 是否需要 Amended 决策 note。
4. 与修复同一个提交进库（fix 提交携带 postmortem 文件）；"修完再补"是被禁止的反模式。

## 已知候选补档（存量，一次性）

- ci/deploy-102 compose 二进制损坏排查（08-18）
- app.env secret 注入两轮失败 → 固定密钥裁定（已记 note 2026-08-18-app-env-in-repo，无需重复）
- 42P18 占位符编号坑、PSGC 忘提交 +1（已沉淀 self-evolving notes，不重复立档）
