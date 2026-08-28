# admin 3 个新页面主题/视觉一致性修复（2026-08-28）

> 决策：docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md
> 工作分支：fix/admin-pages-theme
> 后端对接：http://192.168.0.102:28080

## 缺陷

1. **多主题适配完全失效**：三个新页面（`/ams/purchase`、`/ams/inventory`、`/boss/install-board`）
   使用了不存在的 Tailwind 假类名（`bg-shell-bg-card`、`text-shell-fg-muted`、`border-shell-divider`、
   `bg-brand-primary`、`text-status-danger`、`text-status-success`），`tailwind.config.*` 中
   **根本没定义这些令牌**。切换亮/暗主题时颜色完全不变（修复前截图已证实 6 张全是亮色无变化）。
2. **视觉风格不一致**：包成大卡片、表格行 h-11 + hover 高亮、input h-8 + 标准边框、错误框淡红底
   等都缺失，与 provision/stock 风格差异明显。

## 修复

| 组件 | 修复前（错） | 修复后（对） |
|---|---|---|
| 大卡片壳 | 无 | `rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]` |
| 统计卡 | `rounded-lg border-shell-divider` | `StatCard` 既有组件（零造轮子） |
| 表格头 | `text-shell-fg-muted` | `var(--shell-group-title)` + `bg-[var(--shell-menu-hover-bg)]` |
| 表格行 | 无 hover | `hover:bg-[var(--shell-menu-hover-bg)]` + `h-11` + `border-[var(--shell-side-border)]` |
| 空表行 | 裸 tr | `TableStateRow` 既有组件 |
| input | `bg-shell-bg-card border-shell-divider` | `bg-[var(--shell-input-bg)] border-[var(--shell-input-border)] text-[var(--shell-content-text)] focus:border-[var(--color-border-focus)]` |
| 错误框 | `text-status-danger`（不存在） | `text-[var(--color-danger)]` + `bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)]` |
| 主操作按钮 | `bg-brand-primary`（不存在） | `bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]` |
| 状态枚举 | 硬编码中文 | `t.common.statusTags[domain.value]` 自动从 i18n 读 |
| Dropdown | 无 placeholder option | 加 `{ value: '', label: d.colSupplier }` |

## 真实环境视觉证据（102 admin 容器）

`docs/notes/admin-pages-theme-evidence/` 6 张截图（2 主题 × 3 页）：
- `shot-purchase-light.png` 采购单亮色
- `shot-purchase-dark.png` 采购单暗色
- `shot-inventory-light.png` 库存查询亮色（156 批次/164 在库）
- `shot-inventory-dark.png` 库存查询暗色
- `shot-board-light.png` 施工看板亮色（错误框淡红底"网关错误(HTTP 404)"演示样式正确）
- `shot-board-dark.png` 施工看板暗色

## 已知 follow-up（样式工作之外）

1. `/boss/install-board` 用 `/dispatch_tickets` 路径，admin 端实际为 `/dispatch/pool`（需 workerId）
   → 需要后端补全 `GET /admin/dispatch_tickets`（列全部工单）OR 前端用其他端点
2. worker 端 install_logs POST 路由待补（已在 `2026-08-28-procurement-install-gis-e2e-evidence.md` 跟踪）

## 门禁

- typecheck: 0 错
- test: 60 文件 332 测试 全过（含 menu.def 84 + StatusTag 三语完备 + i18n 键集一致）
- build: 成功（dist 产出，0 幽灵令牌）
- 真实环境：6 截图 视觉验证通过，主题切换正确，错误展示样式正确
