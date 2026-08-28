// Package procurement 采购-库存域(增量挂靠,不开阶段 10)。
//
// 职责边界:
//   - 供应商/采购单/采购明细/到货入库的增删改查;
//   - 入库确认 = 同事务(建 asset_batches 批次→逐台建 assets IN_STOCK→回填入库单);
//
// 跨域规则(防 domain-map §4 规则 2 越界):
//   - procurement 域不直连 internal/domain/asset 实现;
//   - 反向:asset 包通过 ProcurementService.CreateReceipt 调过来,事务边界在 procurement 内闭合;
//   - 库存查询走实时聚合(assets WHERE status='IN_STOCK' AND batch_id=?),不物化视图。
//
// 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md §决策 1。
package procurement
