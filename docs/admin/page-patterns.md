# Admin 页面设计系统 · 四类页面模式规范(Q3)

> 上位规范:`docs/admin/design-spec.md`(App Shell 与 Design Token)。
> 本文件规定**页面级**的四类模式:列表页 / 表单页 / 详情页 / 流程页。
> 组件事实源:`web/admin/src/components/ui/*`(原子)与 `web/admin/src/components/business/*`(模式件)。

## 0. 采用规则(硬性)

1. **新页面一律用模式件**,禁止手写 `<table>`/`<form>`/自绘弹层/自绘分页。
2. 存量页面改动时同步收敛到模式件(顺手改,不单独立项)。
3. 采用率由 `node scripts/check-ds-adoption.js` 度量,回归即红(见 §5)。
4. 模式件不满足需求时**扩展模式件**,不在页面内分叉副本。

## 1. 列表页(最高频)

```
PageHead(标题+说明)
[筛选区: Select/Input + 查询按钮]        ← 条件多于 3 个收进 Drawer
DataTable(columns, rows, emptyText)
Pagination(...pagerTexts(ns))
[行操作: Dropdown(查看/编辑/删除)]      ← 危险操作走 ConfirmDialog
```

- 组件:`PageHead` + `DataTable` + `Pagination` + `pagerTexts` + `ConfirmDialog`。
- 状态列一律 `StatusTag`(语义色由组件统一,页面不自配色)。
- 列定义三列对齐以 `docs/contract/fields.md` 为准,列名不得自造。

**参考实现**:`pages/billing/payment/index.tsx`。

## 2. 表单页/表单弹层

```
Dialog(或 Drawer,字段 ≤3 用 Dialog,>3 或含说明文本用 Drawer)
  FormField(label, required, hint, error) × N
    Input / Select / RadioGroup / DatePicker / Textarea
  [提交 Button(loading)] [取消]
```

- 组件:`Dialog` + `FormField` + ui 原子件;提交中禁用并显 loading。
- 校验错误展示在 `FormField.error`,不用全局 alert。
- 新建/编辑共用同一表单组件,编辑态预填。

**参考实现**:`pages/base/storageconfig/index.tsx`、`pages/billing/billing/run-modal.tsx`。

## 3. 详情页

```
PageHead(实体名+编号)
TabBar(概览 | 关联单据 | 操作留痕)      ← 无多段内容时省略
  概览: DetailDrawer 的 k-v 结构展开为卡片网格(Card)
  关联单据: DataTable(只读)
```

- 组件:`DetailDrawer`(抽屉式轻详情)或 `Card` 网格(整页详情)。
- 只读 k-v 一律 `k: v or '—'`,空值显示 `—` 不显示空白。

**参考实现**:`pages/org/shared` 的 DetailDrawer 用法。

## 4. 流程页(状态机驱动的多步操作)

```
PageHead(流程名)
[状态横幅: StatusTag(当前态) + 允许的下一步动作按钮组]
Timeline / 步骤列表(每步: 时间 + 操作人 + 结果 + 留痕字段)
DataTable(异步任务清单: 状态/重试按钮)
```

- 动作按钮只渲染**当前态合法迁移**(非法迁移后端已拒,前端不摆)。
- 每次动作落审计;失败任务提供显式重试入口(参考 `stop-resume-tasks retry`)。
- 长流程禁用"一键全过",逐步操作逐步留痕。

**参考实现**:`pages/billing/stopsrv/index.tsx`(停复机任务+重试)。

## 5. 采用率守护

```bash
node scripts/check-ds-adoption.js     # 输出各模块采用率与低于阈值的清单
```

- 阈值:模块内 ≥80% 页面引用 `components/business` 或 `components/ui`。
- 棘轮位 70%(2026-08-23 org/bss 表单批量收敛后);当前真基线 bss 71%/org 89%/
  partner·provision 75%/base 83%,目标 80%。

## 6. 收敛存量清单(2026-08-23 审计基线)

> home(公开官网落地页)自有视觉体系,不适用本规范,已从审计豁免。

| 模块 | 未采用页数 | 优先级 |
|---|---|---|
| boss | 0(2026-08-23 收敛:message 四页+WorkerPicker) | 已完成 |
| bss | 2(user/userdata) | 高 |
| base | 4 | 中 |
| org | 2(staff 树) | 低 |
| bss | 3 | 中 |
| profile | 0(2026-08-23 收敛:AuditSection) | 已完成 |
| backup/provision | 2 | 低 |

收敛方式:页面改动时顺手迁移;不接受"纯样式大改"式巨石提交。
