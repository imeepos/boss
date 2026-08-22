# 自定义角色:模板复用 + 全集替换,不做增量 diff

## 日期

2026-08-22

## 决策

1. 内置 7 角色加 `is_builtin` 标记只读（迁移 000100）；允许新建派生角色（code 后端生成 `custom_*`）。
2. "复用内置角色权限模型"在前端完成：`GET /role-details` 返回每角色权限码全集,新建抽屉选模板=整体复制其权限集为初始勾选,用户可继续单独增删。
3. 后端写接口只接收**权限码全集**,UpdateCustomRole 全量替换 role_permissions,不做增量 diff。

## why

- 模板复制放前端:后端无状态、接口语义简单("给我最终权集"),模板选择是纯 UI 交互,服务端复制逻辑会引入 templateRoleCode + permissionCodes 的优先级歧义。
- 全集替换 vs 增量 diff:角色编辑以勾选集为唯一事实源,客户端算差集在并发编辑/勾选状态不同步时容易错位;替换语义幂等、可重放。
- code 后端生成而非用户填写:角色码是系统标识,用户可读性无意义,还带来重名/格式校验负担。

## 放弃了什么

- 服务端模板复制端点(POST /roles?template=ops):被"前端复制"取代,少一个语义分支。
- 自定义角色允许用户自定 code:放弃,理由见上。
- 按权限逐条 PATCH 的增量接口:放弃,理由见上。
- 自定义角色影响数据权限:本轮不做,数据范围仍走账号级 region_scope(accounts 表),与角色正交。

## 关联

- 迁移 `migrations/000100_custom_roles.up.sql`
- 契约 `api/openapi/admin/org.yaml`(/permissions、/role-details、/roles/{roleId})、`docs/contract/fields.md` 1.2
- 前端 `web/admin/src/pages/org/menuperm/`(角色管理卡片)、`web/admin/src/router/role-menu.ts`(动态菜单)
