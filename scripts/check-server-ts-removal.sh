#!/bin/bash

# 检查 server-ts 移除后的文档一致性
echo "=== 检查 server-ts 移除后的文档一致性 ==="

# 1. 检查是否还有 server-ts 目录引用
echo "1. 检查 server-ts 目录引用..."
if grep -r "server-ts/" --include="*.md" . | grep -v "已移除" | grep -v "不再维护" | grep -v "已迁移"; then
    echo "❌ 发现未更新的 server-ts/ 引用"
else
    echo "✅ server-ts/ 引用已清理"
fi

# 2. 检查是否还有 TypeORM 引用
echo "2. 检查 TypeORM 引用..."
if grep -r "TypeORM" --include="*.md" . | grep -v "已移除" | grep -v "不再维护" | grep -v "已迁移"; then
    echo "❌ 发现未更新的 TypeORM 引用"
else
    echo "✅ TypeORM 引用已清理"
fi

# 3. 检查是否还有 TypeScript 实体引用
echo "3. 检查 TypeScript 实体引用..."
if grep -r "TypeScript 实体" --include="*.md" . | grep -v "已移除" | grep -v "不再维护" | grep -v "已迁移"; then
    echo "❌ 发现未更新的 TypeScript 实体引用"
else
    echo "✅ TypeScript 实体引用已清理"
fi

# 4. 检查关键文档是否已更新
echo "4. 检查关键文档更新..."
required_files=(
    "README.md"
    "docs/contract/fields.md"
    "docs/contract/alignment-audit.md"
    "docs/plan/admin-system-plan.md"
    "docs/plan/survey-0001.md"
    "docs/plan/admin-a0-plan.md"
    "docs/plan/admin-contract-audit-prompt.md"
    "docs/notes/adopted/2026-08-17-server-ts-entity-mirror.md"
    "docs/notes/adopted/2026-08-21-customer-code-in-quadlink.md"
    "docs/notes/README.md"
    ".agents/skills/self-evolving/notes.md"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file 存在"
    else
        echo "❌ $file 不存在"
    fi
done

# 5. 检查决策记录是否完整
echo "5. 检查决策记录..."
if [ -f "docs/notes/adopted/2026-08-21-remove-server-ts.md" ]; then
    echo "✅ 决策记录文档存在"
else
    echo "❌ 决策记录文档缺失"
fi

if [ -f "docs/notes/2026-08-21-documentation-update-summary.md" ]; then
    echo "✅ 文档更新总结存在"
else
    echo "❌ 文档更新总结缺失"
fi

echo ""
echo "=== 检查完成 ==="