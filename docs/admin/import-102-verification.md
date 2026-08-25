# 批量导入 102 真实验证记录（批次 E）

> 2026-09-03｜环境 `http://192.168.0.102:28080`（CI 部署 main@57a3f090）｜账号 admin(sysadmin)

## 验证结果

| # | 场景 | 请求 | 结果 | 判定 |
|---|---|---|---|---|
| 1 | 任务登记含 total/skipped | POST /import-tasks `{kind:entity:department,total:5,imported:3,failed:1,skipped:1}` | `code:0 ok` | 通过 |
| 2 | 计数超 total 拒绝 | POST total=2 imported=3 | `code:42200`（新校验文案） | 通过 |
| 3 | 任务列表无筛选 | GET /import-tasks | `code:0` 4 条 | 通过 |
| 4 | kind 精确筛选 | ?kind=entity:department | `code:0` 1 条，统计字段 total/imported/failed/skipped 全回显 | 通过 |
| 5 | operator 模糊筛选 | ?operator=管理（ILIKE） | `code:0` 4 条 | 通过 |
| 6 | 组合筛选 | operator+kind+from | `code:0` 1 条 | 通过 |
| 7 | 不存在类型 | ?kind=entity:nope | `code:0` 空集 | 通过 |
| 8 | 实体导入重复→冲突 | POST /departments 同名两次 | 首次 `code:0 id=23`；重复 `code:40900 资源冲突` | 通过（SQLSTATE 统一映射生效） |
| 9 | 地址导入重复→冲突 | POST /addresses/import 同 path 两次 | 首次 `imported:1`；重复 `code:40900` | 通过（事务+ErrDuplicate 生效） |
| 10 | 列表刷新数据源 | GET /departments、/addresses?parentId=0 | 新建行可见 | 通过 |

## 验证中发现并修复的缺陷

1. **任务列表 SQL 多余右括号**（`LIMIT 200)`）→ 全部查询 500。修复：`fix/import-task-sql`（5cf2d527）。真库日志实锤 `syntax error at or near ")"`。
2. **空时间参数 22007**：PG 的 OR 不保证惰性求值，空串参与 `::timestamptz` 转换报 `invalid input syntax for type timestamp`。修复：空 from/to 以 NULL 参数下发 + pgxmock 回归（57a3f090）。

两处均为"单测走 fake 未触达真 SQL"的盲区，由本轮真实环境验证暴露并当场闭环。

## 数据清理与巡检

- 临时部门 `accvrf-import-round11`(id=23)：已 DELETE（code:0）
- 临时地址 `accvrfr11`(id=513)：已 DELETE（code:0)
- 验证产生的 import_tasks 记录 2 条：已 SQL 清理
- 巡检：`departments LIKE 'accvrf%'` = 0；`addresses path LIKE 'accvrf%'` = 0 —— 无残留

## 本轮追加修复验证状态

本轮代码修复已完成但尚未部署到 102：
- 地址/Geo 前端入口按 `menu:importer`/`menu:geo` 置灰，并在执行层二次阻断。
- 实体导入遇 401 中止后刷新已成功写入的宿主列表。
- 未知任务 kind 原样显示，避免静默误标地址。
- 地址/Geo 任务登记失败不再被忽略，登记失败时不发送完成通知。

以上变更需要部署后在 102 复验；本文件既有批次 E 验证仍对应部署版本 `main@57a3f090`。

## 结论

既有计划批次 A–E 已完成；既有 P0/P1 项经历史验证通过。本轮追加修复已在代码中完成，部署后验证项见上。
