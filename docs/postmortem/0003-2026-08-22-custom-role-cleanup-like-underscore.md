# 0003-2026-08-22 验收清理 SQL 的 LIKE 下划线通配误删内置角色

## 现象

自定义角色验收完成后,用临时 Go 程序直连 102 库清理验收数据,`DELETE FROM roles WHERE code LIKE 'custom_%'` 返回"删除 2 行"(预期 1 行)。约 30 分钟后核对发现内置 `customer` 角色消失。

## 机理

PostgreSQL `LIKE` 中 `_` 是单字符通配符:`'custom_%'` 匹配 `custom` + 任意一个字符 + 任意后缀,`customer`(custom+e+r)命中。临时脚本未转义(应为 `LIKE 'custom\_%'`),把无 FK 引用的内置 `customer` 角色顺带删除(`partner_*` 因被 api_key 逻辑语义依赖但无 DB 引用也属侥幸——实际只有 customer 恰好撞上模式)。

## 影响

- 102 库 `customer` 角色行丢失约 30 分钟;该角色无菜单权限、无 accounts 引用,用户端登录走独立表不受影响;期间新建环境不可比对的种子差异。
- 已原地恢复(`INSERT ... VALUES ('customer','客户',true) ON CONFLICT DO NOTHING`),恢复后 roles 总数 9,与迁移种子一致。

## 教训

1. 临时清理 SQL 的 LIKE 模式含 `_` 必须转义;更优做法是按主键/精确码等值删除,不用模式匹配。
2. 删除语句的 rowsAffected 与预期不符(2≠1)时必须立刻停下核对,而不是继续走完流程——当时输出里"roles deleted: 2"已经暴露异常,被"多删了一个未知自定义角色"的错误假设带过。
3. 验收数据优先走带引用检查的应用 API 删除;直连 DB 只做只读核对,写操作要双人核对模式。

## 关联

- feat: 自定义角色(migrations/000100/000101,docs/notes/adopted/2026-08-22-custom-roles.md)
