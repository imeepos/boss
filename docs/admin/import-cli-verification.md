# 批量导入 bossctl CLI 验证记录

> 2026-09-03｜工具 `bossctl`（API key 免登录，admin/sysadmin）｜环境 102

## 验证矩阵

| # | 场景 | CLI 命令要点 | 结果 |
|---|---|---|---|
| 1 | 法人 创建/重复 | POST /legal-entities | create ✅ / 重复 `40900` ✅（本轮修复后生效） |
| 2 | 部门 创建/重复 | POST /departments | create ✅ / `40900` ✅ |
| 3 | 岗位 非法码/创建/重复 | POST /posts | 大写码 `42200` ✅ / 合法码 create ✅ / `40900` ✅ |
| 4 | 产品 创建/重复/effectiveAt | POST /products（不传 effectiveAt） | create ✅ / `40900` ✅ / `effectiveAt=now`（非 0001 零值）✅ |
| 5 | 账号 创建/重复 | POST /accounts | id=516 ✅ / `40900` ✅ |
| 6 | ODN 网格 创建/重复 | POST /odn/grids?prvCode&cityPrefix | create ✅ / `40900` ✅ |
| 7 | 客户 FK 不存在 | POST /customers legalEntityId=99999 | 修复已合入（2def20f7），**待 CI 部署后复验** |
| 8 | 地址导入 成功/重复 | POST /addresses/import | `imported:1` ✅ / `40900` ✅ |
| 9 | 任务登记 合法/非法 | POST /import-tasks | `ok:true` ✅ / 计数超 total `42200` ✅ |
| 10 | 任务筛选 | GET /import-tasks?kind / ?operator | 命中正确，`total/imported/failed/skipped` 持久化回显 ✅ |
| 11 | 无认证 | 裸 curl（无凭证） | `401 missing bearer token` ✅（bossctl 回退缓存 token 属预期） |
| 12 | 占用防护 | DELETE /departments（有账号挂靠） | `40900` 拒删 ✅ |

## 本轮 CLI 测试发现并修复的缺陷

1. **法人重复 → 500**：`CreateLegalEntity/UpdateLegalEntity` 未接 `isUniqueViolation`。修复 `0cf9830b`，部署后复验 `40900`。
2. **客户 FK 不存在 → 500**：`customer.ErrForeignKeyViolation` 未注册 httpx 映射。修复 `2def20f7`（含岗位模板非法码 `POST-IMP-01`→`post_imp_01`），待部署复验。
3. **岗位导入模板样例非法**：样例码大写+连字符，违反后端 `^[a-z][a-z0-9_]+$` 校验，模板自身样例行会被 422 拒绝。已随 `2def20f7` 修正。

## 数据清理与巡检

- API 删除：岗位 29、地址 515
- SQL 删除：账号 clivrf_imp、部门、法人 LE-ACCVRFA、产品、ODN 网格 91、任务记录 1 条
- 残留巡检（8 类实体合计）：**0**

## 结论

批量导入 12 个页面对应的创建端点在 CLI 直连下全部通过：重复一律 409、参数非法一律 422、未认证一律 401；任务契约（total/skipped 持久化+筛选）验证通过。客户 FK 映射待下一次部署后复验（预期 42200）。
