# AAA 管理二期收敛计划

能力域：`AAA`；Agent：`AG-05`；internal：`aaa`；阶段：`7`。

## 交付

1. `/lo-accounts`、`/cdrs`、`/auth-logs` 使用服务端 `page/pageSize` 分页，返回 `items/total/page/pageSize`。
2. `keyword/status/loid` 由后端执行；CDR 支持 billingStatus，认证日志支持 result。
3. 管理端查询使用账号 `DataScope` 的 `legalEntityId/regionScope` 约束；前端过滤不作为权限边界。
4. 认证账号展示 `billingMode`，AAA 状态和 CDR/认证结果统一走 i18n。
5. 旧 HTML 仅作历史视觉原型，标明真实 API 和未交付功能，避免详情/导出/补单承诺漂移。
6. 更新 OpenAPI 和列表响应 schema；不新增 NAS、在线会话、补单等未有真实数据源的页面。

## 验证

- Go AAA/admin 单测、vet、契约同步
- web admin typecheck/test/build
- 102 登录后真实 API：分页、筛选、total、数据范围、summary
- 102 Admin 页面真实 DOM/API/console 验证
