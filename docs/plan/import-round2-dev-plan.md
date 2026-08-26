# 批量导入第二轮开发计划

> 2026-09-03｜基于后台逐页审计、102 真实验证记录和本轮社区实践检索制定。

## 1. 目标与边界

本轮只修复已确认的导入可靠性、可见性和契约漂移问题，不把“需要专用流程”的页面改造成普通导入。普通实体继续调用既有创建端点；地址与 Geo 保持现有后端批量接口。

## 2. 社区实践与定制方案

| 实践 | 来源 | 本项目落地 |
|---|---|---|
| 自然键、数据库唯一约束和幂等键共同防重；重试必须返回同一逻辑结果 | [Designing Idempotent Bulk Import Pipelines](https://dev.to/hammadxcm/designing-idempotent-bulk-import-pipelines-e164-vin-and-the-rest-1man) | 为导入任务增加可选 `clientKey`，数据库部分唯一索引，重复登记采用 upsert；保留无 key 的历史调用兼容性 |
| staging/preview/validate/commit 分离，避免上传即写入 | [Staging tables vs direct imports](https://appmaster.io/blog/staging-tables-vs-direct-imports-csv-excel-uploads) | 保持当前“文件→解析预览→确认→提交”链路；预检失败显式提示，不把客户端预检当数据库约束 |
| 导入权限按数据域细分，入口和服务端都门禁 | [Granular Access Control for Data Imports](https://help.keka.com/hc/en-us/articles/42624377443345) | 端点权限同时用于前端入口和后端执行；地址继续使用 `menu:importer`，Geo 使用 `menu:geo` |
| OpenAPI 作为版本化契约，并在 CI 进行实现漂移检测 | [OpenAPI drift gate example](https://github.com/lumose-health/GlycemicGPT/commit/40145de920f67fa3dac76dd584976781c40035d7) | 补齐已确认的 required/status/limit 描述和审核请求体；不在本轮重构全站 response envelope |

## 3. 执行批次

每批均在此 feature worktree 完成，测试通过后立即独立提交。

### 批次 1：ODN 模板样例

修正 ODN 设备模板中的非法样例编码 `ODN-DEV-001` 为符合规范的 `OLT001`，运行 importer 前端相关测试和类型检查。

### 批次 2：预检失败可见性

为实体列表预检和动态行数参数读取增加可见的非阻断警告，区分“预检未完成”和“默认上限生效”；补充三语言文案与测试。数据库唯一约束仍是最终防线。

### 批次 3：任务中心刷新

任务列表增加窗口重新获得焦点/页面恢复可见时刷新，保留手动刷新；避免引入高频轮询。

### 批次 4：任务登记幂等

新增迁移 000146 与 `client_key` 部分唯一索引；扩展任务登记契约和存储层，带 key 走 upsert，无 key 兼容旧地址/Geo；实体导入每次运行生成 key。补充存储/HTTP/前端回归测试。

### 批次 5：契约治理

补齐师傅审核通过请求体、CMS `limit` 的真实边界说明、配置模板字段和状态 enum。对模板非法 status 增加服务端拒绝和回归测试，防止静默归一化掩盖错误。

### 批次 6：权限裁定与专用流程设计

记录 `menu:importer`/`menu:address` 权限边界裁定；输出 CMS、资产置备、员工审核、区域树、师傅实名、模板版本流程设计清单。只形成后续开发输入，不直接增加普通导入。

## 4. 门禁与验证标准

- 前端批次：`pnpm typecheck && pnpm test -- --run && pnpm build`。
- Go 批次：`/opt/homebrew/bin/go test ./internal/httpapi/admin/...`，涉及领域时增加对应包测试。
- 迁移/契约批次：执行项目 `make check` 或等价 contract sync 门禁。
- 合并后将 feature 部署到 102，再用真实接口复验任务登记幂等、权限、重复、失败和列表刷新；未部署前只标记“代码已完成”。
- 测试数据使用带前缀的临时数据，验证完成后删除并按前缀巡检为 0。

## 5. 明确不在本轮直接实现

- 不新增 CMS、资产/标签/端口、员工、师傅、区域、模板的普通平面导入。
- 不绕过订单、派单、实名、审核、财务或税务流程。
- 不统一全站 response envelope，不在无业务裁定时改变已有权限语义。
- 不将 102 未部署结果写成已验证。
