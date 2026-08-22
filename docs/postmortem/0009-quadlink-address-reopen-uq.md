# 0009 - 同地址重开单环节5撞唯一索引(quad_links)

- 日期: 2026-08-23
- 发现场景: Q1 主链路 102 真实环境验收,客服代客为同地址第二位客户下单
- 影响范围: 任何地址存在历史 quad_links 行(拆机释放/未扫码放弃)后,该地址
  所有新订单 charge(自动段5-8)必 50000,主链路中断

## 症状

```
POST /api/admin/v1/orders/ORD-20260823-000445/charge → 50000
automation: applyTag: order: applyTag create quad link:
quadlink: create link: ERROR: duplicate key value violates
unique constraint "uq_quad_links_address" (SQLSTATE 23505)
```

## 根因

- 000086 给 asset/port/address 三列建了"非空即唯一"的部分唯一索引(不看 status)。
- 000097 以"000086 非空唯一已完全覆盖 000056 的 *_active 索引"为由删掉
  active 形态索引——该冗余论只在"每码永远至多一行"时成立。
- 000088 起 customer 1:N、拆机 UnbindRequireScan 保留 UNLINKED 行留痕,
  同码出现"历史 UNLINKED + 新预绑定"多行成为正常生命周期,
  非空唯一把已释放行也计入,同地址/端口/资产重开必撞。

## 修复

- migration 000110:三码唯一索引收窄为 `status IN ('LINKED','CONFLICT')`
  (活跃链路唯一),与 000097 自述契约"至多一条非空活跃链路"对齐。
- 回归: `internal/app/e2e_quadlink_reopen_test.go`
  (同地址两单各自 applyTag 成功,两条 UNLINKED 并存,端口不重叠)。

## 教训

- "索引 A 被索引 B 覆盖"的裁定必须枚举 B 的谓词语义差异(此处 status 过滤),
  不能只看"都更严格/更宽松"的方向。
- 唯一约束变更要跑"生命周期重放"用例(开单→释放→重开),不只是单次正向流。
