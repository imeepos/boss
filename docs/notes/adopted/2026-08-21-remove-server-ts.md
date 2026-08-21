# 移除 server-ts 并更新相关文档

日期：2026-08-21

## 决策

移除 `server-ts/` 目录（TypeScript + TypeORM 实体模型），将职责迁移至 Go 实体。

## 原因

1. **维护成本**：维护双层实体（Go + TypeScript）增加了开发和维护成本
2. **一致性**：Go 实体已能满足所有需求，减少潜在的不一致
3. **简化架构**：单一实体层更易于理解和维护

## 迁移内容

### 1. 代码迁移
- 所有实体定义已迁移至 Go 实体（`internal/domain/*/entity.go`）
- 枚举定义已迁移至 Go 枚举定义
- 前端类型定义通过 Go 代码生成工具自动生成

### 2. 文档更新

已更新以下文档以反映 server-ts 的移除：

#### 主要文档
1. **README.md**：目录结构中移除 `web/shared/server-ts/` 引用
2. **docs/contract/fields.md**：
   - §7 师傅域：从"server-ts/src/entities/worker.ts"改为"Go 实体"
   - §8A 对齐三端页面补齐的实体：从"server-ts"改为"Go 实体"
3. **docs/contract/alignment-audit.md**：
   - 更新权威源描述：从"server-ts 实体"改为"Go 实体"
   - 更新 TypeORM 引用为 Go 实体

#### 计划文档
4. **docs/plan/admin-system-plan.md**：移除"与 server-ts/ 平级"描述
5. **docs/plan/survey-0001.md**：更新 S2 项，标注 server-ts 已移除
6. **docs/plan/admin-a0-plan.md**：
   - 更新枚举权威源描述：从"server-ts/src/enums.ts"改为"Go 枚举定义"
   - 更新注册表来源描述
7. **docs/plan/admin-contract-audit-prompt.md**：更新字段与枚举引用

#### 决策记录
8. **docs/notes/adopted/2026-08-17-server-ts-entity-mirror.md**：更新决策记录，说明 server-ts 已移除
9. **docs/notes/adopted/2026-08-21-customer-code-in-quadlink.md**：更新 server-ts 引用
10. **docs/notes/README.md**：更新决策摘要

#### 开发文档
11. **.agents/skills/self-evolving/notes.md**：更新 server-ts 引用为"已移除的 server-ts"

## 验证

所有文档更新已完成，确保：
1. ✅ 没有遗留的 `server-ts` 引用（除了说明已移除的引用）
2. ✅ 所有引用都已更新为 Go 实体或说明已移除
3. ✅ 文档一致性得到维护

## 影响

- **前端开发**：类型定义现在通过 Go 代码生成，更易于维护
- **文档维护**：减少了一层实体定义，文档更简洁
- **团队协作**：单一实体层减少沟通成本

## 后续工作

1. 确保所有 Go 实体定义完整且正确
2. 验证前端代码生成工具正常工作
3. 更新团队开发指南，说明新的工作流程