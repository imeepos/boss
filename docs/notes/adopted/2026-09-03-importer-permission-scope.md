# 导入权限边界裁定

> 日期：2026-09-03

## 决策

地址批量导入继续使用 `menu:importer`，Geo 批量导入使用 `menu:geo`；前端入口和执行层均按实际端点权限门禁。暂不把地址批量导入改为同时要求 `menu:address`。

## Why

`migrations/000003_stage1_menuperm.up.sql` 中 `menu:importer`、`menu:address` 是独立权限；现有角色绑定中 importer 由 sysadmin 全量继承，业务角色未默认获得该权限。地址导入是数据导入中心的跨域能力，后端路由已经以 `menu:importer` 作为唯一门禁，前端应与之保持一致。强行增加双权限会改变现有接口语义，并可能使已有自定义角色静默失去能力。

如果未来向非 sysadmin 角色开放 `menu:importer`，应在权限评审中重新评估地址导入是否需要 `menu:address` 双重门禁，并补充真实角色验证。

## 放弃了什么

本轮不直接修改后端地址权限为 `menu:importer + menu:address`，避免未经角色矩阵核准改变生产权限；也不把地址导入放宽为仅 `menu:address`。

## 关联

- `internal/httpapi/admin/address.go`
- `web/admin/src/pages/base/importer/BatchImportEntry.tsx`
- `docs/plan/import-round2-dev-plan.md`
