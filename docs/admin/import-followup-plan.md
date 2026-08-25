# 批量导入后续改进计划

> 2026-09-03｜基于第一阶段审计、Go/React/Postgres 现状和社区实践制定。

## 1. 社区实践结论

- 导入向导应遵循“选择文件 → 映射/预览 → 逐行校验 → 明确确认 → 提交 → 结果追踪”的状态链，避免上传即写入。参考 [Universal Data Import Wizard](https://www.c-sharpcorner.com/article/building-a-universal-data-import-wizard-mapping-columns-preview-validation/) 和 [Safe bulk imports](https://appmaster.io/blog/safe-bulk-imports-preview-validate-commit)。
- 文件上传必须限制扩展名、MIME、大小，服务端重新校验内容，不信任客户端文件名；参考 [OWASP File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html)。
- 批处理要有幂等键、唯一约束和可重试语义；客户端预检只能改善体验，不能替代数据库约束。参考 [Google Cloud Storage idempotency discussion](https://github.com/googleapis/java-storage/pull/2905)。

## 2. 目标方案

### P0：职责收敛与真实验收

1. 地址和 Geo 导入入口只保留在各自业务页。
2. 数据导入中心只渲染任务列表，补充实体、操作人、时间范围和结果状态筛选。
3. 使用带前缀的临时数据在 102 做真实验证：成功、重复、非法、依赖缺失、权限不足、失败行导出、刷新和任务记录，验证后清理并巡检。

### P1：统一任务契约（已完成）

1. ~~`import_tasks` 增加 `total/skipped` 统计字段~~ 已完成（迁移 000145）。
2. ~~统一 `imported/failed/skipped` 口径~~ 已完成。
3. ~~任务记录登记失败不能静默~~ 已完成（用户可见提示 + 三语言文案）。
4. ~~增加筛选接口契约~~ 已完成（kind/operator/from/to + 纯函数测试）。

### P1：后端可靠性（大部分已完成）

1. ~~地址导入改为事务批处理~~ 已完成（单事务 + 23505→409）。
2. ~~统一各实体唯一冲突及 FK 错误映射~~ 已完成（isUniqueViolation 按 SQLSTATE 判定 + customer.ErrDuplicate）。
3. ~~关键自然键由数据库约束保障~~ 已核查确认：全部实体已有库级唯一约束（见 import-batch-audit.md"已收敛事项"），无需新增迁移。
4. 产品 effective_at 缺省落零值问题：客户手机号/ODN 复合键约束已确认无缺；effective_at 待产品域专项处理。

### P2：质量与安全

1. 增加 BatchImportEntry、TaskList、筛选和 ODN tab 映射测试。
2. 增加真实 API 冒烟和可清理 E2E 测试，禁止 mock 替代真实业务验证。
3. 按 OWASP 校验文件类型、大小、内容和失败响应，不信任客户端文件元数据。
4. 第二批页面逐页形成“永久排除/专用流程/待后端接口”裁定。

## 3. 执行批次与门禁

- 批次 A：文档与任务契约设计，单独 commit。
- 批次 B：前端入口与导入中心职责收敛，typecheck + test + build 后 commit。
- 批次 C：任务记录模型和筛选接口，Go 单测/契约测试后 commit。
- 批次 D：后端事务、错误映射和约束修复，Go 全量门禁后 commit。
- 批次 E：102 真实验证、数据清理、最终全量门禁后 commit。

每批均在独立 worktree 完成，先同步 `main`，再推送 feature，主树只做 `ff-only` 合并；合并成功后才删除 worktree 和分支。

## 4. 当前明确不做

- 不在本阶段为第二批页面直接增加普通导入。
- 不通过客户端导入绕过订单状态机、审核、实名、财务和税务流程。
- 不以 mock 数据替代 102 真实 API。
- 不在未核查存量重复数据前新增唯一约束迁移。
