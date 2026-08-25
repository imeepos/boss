# 批量导入社区最佳实践摘要

本阶段方案参考以下公开实践：

- [AppMaster：Safe bulk imports](https://appmaster.io/blog/safe-bulk-imports-preview-validate-commit)：预览、校验、提交分离，任务绑定批次和幂等语义。
- [dev.to：CSV imports for 10,000 messy rows](https://dev.to/manukminasyan/how-i-rebuilt-csv-imports-to-handle-10000-messy-rows-without-breaking-26ng)：staging、显式状态机、分块处理、重复处理和失败行下载。
- [Bluecore：Import job and run statuses](https://help.bluecore.com/help/understand-import-job-and-run-statuses)：任务状态和运行状态分离，支持部分成功及重试。
- [CSVBox：Downloadable error reports](https://blog.csvbox.io/csv-error-reports/)：失败报告保留原始列、行号和可操作错误。
- [Stripe：Idempotent requests](https://docs.stripe.com/api/idempotent_requests)：幂等键重放返回一致结果。
- [OWASP：File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html)：服务端重新校验文件类型、大小和内容，不能信任客户端元数据。

## 落地原则

1. 预览、校验、提交共用同一套解析、字段校验和权限规则。
2. 任务使用明确状态机，结果区分成功、失败、跳过，并提供失败行下载与重跑。
3. 前端预检仅改善体验，数据库唯一约束和服务端幂等才是最终防线。
4. 任务记录需要操作人、批次、源文件摘要、结果计数和错误明细，登记失败不能静默。
5. 大数据量采用 staging 和分块处理；当前实体规模较小可继续逐行调用，但应优先补齐任务统计和错误映射。
6. 高风险域不得通过普通导入绕过审核、状态机、实名、财务或权限流程。
