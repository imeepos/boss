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

### P1：统一任务契约

1. 在 `import_tasks` 增加 `skipped`、`status`、`total` 等必要统计字段，保留 `detail` 扩展字段。
2. 统一 `imported/failed/skipped` 口径：重复预检计 skipped，实际请求失败计 failed，成功创建计 imported。
3. 任务记录登记失败不能静默；至少在 UI 显示“业务数据已处理但任务记录登记失败”。
4. 增加筛选接口契约和任务详情展示。

### P1：后端可靠性

1. 地址导入改为事务边界明确的批处理，重复和依赖错误映射为业务错误。
2. 统一法人、部门、岗位、产品、客户和 ODN 的唯一冲突及 FK 错误映射。
3. 关键自然键由数据库约束保障，前端预检只作为友好提示。
4. 复核产品生效时间、客户手机号和 ODN 复合键等契约后再新增迁移。

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
