# 文档更新总结：移除 server-ts

日期：2026-08-21

## 背景

根据项目架构调整，已移除 `server-ts/` 目录（TypeScript + TypeORM 实体模型），将职责迁移至 Go 实体。为保持文档一致性，已更新所有相关文档。

## 更新的文档列表

### 1. 核心文档
1. **README.md** (第 88-92 行)
   - 移除目录结构中 `web/shared/server-ts/` 的描述

2. **docs/contract/fields.md**
   - 第 426-428 行：§7 师傅域描述从 "server-ts/src/entities/worker.ts" 改为 "Go 实体"
   - 第 534-536 行：§8A 对齐三端页面补齐的实体从 "server-ts" 改为 "Go 实体"

3. **docs/contract/alignment-audit.md**
   - 第 3 行：权威源描述从 "server-ts 实体" 改为 "Go 实体"
   - 第 75 行：TypeORM 引用改为 "Go 实体"

### 2. 计划文档
4. **docs/plan/admin-system-plan.md** (第 42 行)
   - 移除 "与 server-ts/ 平级" 描述

5. **docs/plan/survey-0001.md** (第 11 行)
   - 更新 S2 项：标注 server-ts 已移除

6. **docs/plan/admin-a0-plan.md**
   - 第 19 行：枚举权威源从 "server-ts/src/enums.ts" 改为 "Go 枚举定义"
   - 第 54 行：注册表来源描述更新

7. **docs/plan/admin-contract-audit-prompt.md** (第 17 行)
   - 字段与枚举引用从 "server-ts/src/enums.ts" 改为 "Go 枚举定义"

### 3. 决策记录
8. **docs/notes/adopted/2026-08-17-server-ts-entity-mirror.md**
   - 整个文档重写，说明 server-ts 已移除

9. **docs/notes/adopted/2026-08-21-customer-code-in-quadlink.md** (第 53-54 行)
   - 更新 server-ts 引用为已移除状态

10. **docs/notes/README.md**
    - 第 23 行：更新决策摘要
    - 新增第 2026-08-21 条目

### 4. 开发文档
11. **.agents/skills/self-evolving/notes.md**
    - 第 395 行：更新 server-ts 引用为 "已移除的 server-ts"
    - 第 402 行：更新建议说明

## 更新原则

1. **准确性**：所有更新都准确反映 server-ts 已移除的事实
2. **一致性**：确保所有文档使用统一的术语和描述
3. **可追溯性**：保留决策记录，说明移除原因和迁移方案

## 验证结果

✅ 所有 server-ts 引用已更新或移除
✅ 所有 TypeScript 相关引用已更新
✅ 文档一致性得到维护
✅ 决策记录完整

## 后续工作

1. **代码验证**：确保所有 Go 实体定义完整且正确
2. **前端集成**：验证前端代码生成工具正常工作
3. **团队通知**：更新团队开发指南，说明新的工作流程
4. **CI/CD 验证**：确保构建和部署流程正常

## 影响评估

### 正面影响
- ✅ 简化架构，减少维护成本
- ✅ 消除双层实体的潜在不一致
- ✅ 文档更简洁清晰

### 需要关注
- ⚠️ 确保前端类型生成工具稳定可靠
- ⚠️ 团队成员需要适应新的工作流程
- ⚠️ 监控是否有遗漏的 TypeScript 引用

## 总结

本次文档更新已全面完成，所有相关文档都已更新以反映 server-ts 的移除。文档现在准确反映了项目当前的架构状态，为团队提供了清晰的技术参考。