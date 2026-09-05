# 2026-09-06 资产标签 P1 波次裁定(联动/回收/字典/巡检)

> 对应计划:docs/design/asset-tag-p1-plan.md;调研:asset-tag-research-mature-designs.md。
> 本 note 过账波次内不可逆裁定;T4(巡检两查)与计划文档已先行合入(d192ae4e/4da4c901)。

## T1 装机/拆机资产联动(裁定:同库强一致)

1. **失败语义**:采纳 R1 调研结论(Snipe-IT checkout 同库事务+行锁先例),联动为强一致——
   VerifyScan/UnbindRequireScan 的写路径(链路翻转+扫码日志+资产状态)同事务,联动失败
   回滚阻断扫码,留 [quadlink] ASSET LINKAGE FAILED 日志;弃 best-effort 草案。原方案的
   「失败不阻断+巡检兜底」降级为巡检定位辅助而非一致性手段。
2. **幂等口径**:MarkDeployed 行锁复检后按当前态分派——DEPLOYED no-op(重放自愈)、
   SCRAPPED 拒绝(ErrAssetScrapped 转人工,不自动复活报废件)、其余(IN_STOCK/MAINTENANCE)
   翻转+轨迹;MarkReleased 仅 DEPLOYED→IN_STOCK 且清地址与翻转同 UPDATE 原子。
3. **拆机落库态**:回 IN_STOCK(可复用)而非 MAINTENANCE——故障件走既有换新流程入
   MAINTENANCE,拆机本身不隐含故障。
4. **接口形状**:quadlink 声明 AssetStateSink 窄接口(asset.ExecQuerier 事务面),asset.PGStore
   隐式实现,wiring UseAssetSink 注入;quadlink→asset 单向接口依赖,资产域不感知四码域。
5. **000185 存量补账**:LINKED 链路且资产 IN_STOCK → DEPLOYED+地址+轨迹(仅此一态,
   MAINTENANCE/SCRAPPED 不动);102 真库预演 would_fix=0(现网 LINKED 已一致),语义面向
   其他环境与未来回放。

## T2 标签回收闭环+事件流(预裁定,落地时生效)

1. tag_events 事件表 append-only(R2 调研:Snipe-IT action_logs/bk-cmdb cc_AuditLog 同款):
   action 短枚举 BIND/UNBIND/RECYCLE,差异 payload 进 changed JSONB;
   **不建** request_id/source 列——事件仅在状态 UPDATE 命中行时写入,状态层幂等已防重放。
2. 报废 ScrapAsset:先写事件后改状态,SCRAPPED 时强制解绑标签写 RECYCLE(软回收禁硬删)。
3. 新端点 POST /assets/{assetId}/scrap、POST /tags/{tagId}/unbind,menu:asset 组。

## T3 型号字典(预裁定)

1. asset_models(vendor,model,category,spec JSONB,part_number,is_active,UNIQUE(vendor,model,category));
   assets.model_id 可空外键,type 保留为展示冗余列(R3:NetBox PROTECT/分列惯例)。
2. 存量按 DISTINCT type 播种(vendor='(存量未登记)')并按精确匹配回填 model_id;
   **'光猫'与'ONU'合并为 category='ONU' 属业务裁定,裁定前种子并存**;MI-ONU 单列。

## 放弃的方案

- T1 best-effort+ALERT(初审稿):R1 源码级证据否定——同库静默漂移违成熟惯例。
- T1 直接 UPDATE 无行锁:并发双扫码/拆机可竞态,SFOR UPDATE 复检为 Snipe-IT 同款。
- 事件表全量 event_sourcing(request_id+source+聚合版本):当前无跨服务重投场景,从简。

## 验收

- T1:go build/vet + quadlink/asset 两包单测(含 8 个新联动用例)全绿;mock 序列与事务边界一致。
- 102 真库预演:000185 would_fix=0;巡检两查 SQL 只读执行通过(LINKED 漂移 5 条在案,即日告警)。
- T2/T3 落地时按各自 ledger 验收命令机械验收。