# 资产与标签 P1 波次开发计划(2026-09-06)

> 依据:docs/design/asset-tag-research-mature-designs.md 八缺口差距矩阵;社区最佳实践检索(4 路调研)
> 结论回填见 §六。执行纪律:每项完成即门禁验证+commit(worktree feat/asset-tag-p1),
> 按项 ff-only 合并回 main,波次结束清理 worktree 与分支。迁移占号:000185/000186/000187
> (占号前已 fetch gitea 并核对全分支无撞号,2026-09-06 06:18)。

## T1 装机/拆机资产联动(修 G1 主漂移源)

设计要点:
1. asset.PGStore 新增窄接口方法(quadlink 侧声明 AssetStateSink,资产域隐式实现,
   跨域不直触对方表):MarkDeployed(assetID,addressID,workerID,workerName) 与
   MarkReleased(assetID)。
2. 幂等语义:UPDATE 带 status 闸门(DEPLOYED 仅从 IN_STOCK/MAINTENANCE 进入,
   SCRAPPED 终态不可拉回;IN_STOCK 仅从 DEPLOYED 退回),RowsAffected=0 即幂等重放,
   不重复落轨迹行;地址名快照由资产域自查 addresses.name。
3. 联动点:VerifyScan 正常 MATCH、幂等重放 MATCH(重试自愈)、重装复用 relink(旧资产
   MarkReleased+新资产 MarkDeployed)、UnbindRequireScan 成功后 MarkReleased。
4. 失败语义(R1 修订:同库强一致,弃 best-effort):联动失败 slog.Error「[quadlink] ASSET LINKAGE FAILED」附载荷上下文,
   不阻断扫码主流程(成熟系统同款取舍:业务开通优先,台账漂移由 T4 巡检兜底)。
5. 迁移 000185 存量补账:LINKED 链路且资产 IN_STOCK → DEPLOYED+绑地址+补轨迹行
   (DISTINCT ON 防重;MAINTENANCE/SCRAPPED 不动,留给巡检人工甄别)。
6. 装配:internal/app/wiring_aaa_infra.go 资产店构造后 UseAssetSink 注入四码店。

涉及文件:internal/domain/asset/pg_write.go(+MarkDeployed/MarkReleased),
internal/domain/quadlink/{quadlink.go,pg_scan.go}(+AssetStateSink/UseAssetSink/联动点),
internal/app/wiring_aaa_infra.go,migrations/000185_*,两域单测。
验收命令:go build ./... && go vet ./internal/domain/... && go test ./internal/domain/quadlink/... ./internal/domain/asset/... && gofmt -l internal/domain | wc -l(=0)

## T2 标签回收闭环+绑定事件流+报废路径(修 G5/G8+G1 的 SCRAPPED 缺口)

设计要点:
1. 迁移 000186:tag_events(id,tag_id,action,asset_id,actor_account_id,detail,created_at;
   索引 (tag_id,created_at DESC)/(action));action 枚举 BIND/UNBIND/RECYCLE。
2. asset.PGStore 新增:UnbindTag(tagID,expectedAssetID,actor,detail)——校验当前绑定
   关系一致后置 bound_asset_id=NULL+status=UNBOUND,同事务写 UNBIND 事件;
   CreateTag/CreateAsset 双绑成功时同事务写 BIND 事件(事件流从源头闭环)。
3. ScrapAsset(assetID,actor,reason):状态置 SCRAPPED(任意非终态可进,终态幂等),
   落轨迹行;若标签仍绑则强制解绑并写 RECYCLE 事件——对齐 ITIL 退役联动释放。
4. 新端点(menu:asset 组):POST /assets/{assetId}/scrap{reason};POST /tags/{tagId}/unbind
   {expectedAssetId?,reason?};openapi+路由注册+两生成器重跑(routes/perms --check 过门禁)。
5. 事件写入失败路径留 [asset] TAG EVENT FAILED 可 grep 日志。

涉及文件:migrations/000186_*,internal/domain/asset/{asset.go,pg_write.go,pg_events? 新文件},
internal/httpapi/admin/{asset.go,asset_handlers.go},api/openapi/admin/asset.yaml,fields.md。
验收命令:go build ./... && go test ./internal/domain/asset/... ./internal/httpapi/admin/... && node scripts/gen-bossctl-routes.mjs --check && go run ./scripts/genrouteperms --check

## T3 型号字典 asset_models(修 G6 前半)

设计要点:
1. 迁移 000187:asset_models(id,vendor,model,category,spec,status ENABLED/DISABLED,
   created_at;UNIQUE(vendor,model))+assets.model_id 可空引用;按存量 type 播种
   (vendor='(存量未登记)',model=存量 type 值)并回填 model_id——自由文本渐进收敛,
   '光猫'与'ONU'是否合并留业务裁定(字典层先并存)。
2. CreateAsset 支持 ModelID:校验存在;Type 空时取 model.category 回填(存量列保留为展示冗余)。
3. 新端点:GET /asset-models(目录下拉)/POST /asset-models(建档);停用不删除(引用计数由
   model_id 引用关系天然承载)。
4. fields.md §4.1 同步 model_id 与字典说明。

涉及文件:migrations/000187_*,internal/domain/asset/{asset.go,pg_write.go,pg.go},
internal/httpapi/admin/{asset.go,asset_handlers.go},api/openapi/admin/asset.yaml,fields.md。
验收命令:go build ./... && go test ./internal/domain/asset/... ./internal/httpapi/admin/... && node scripts/gen-bossctl-routes.mjs --check && go run ./scripts/genrouteperms --check

## T4 巡检扩展两查(T1 的漂移兜底)

设计要点(纯增量,无迁移):report/pg_patrol.go orphanChecks 追加:
1. 「quad_links.LINKED but asset not DEPLOYED」——活跃链路资产状态漂移;
2. 「asset SCRAPPED but tag still bound」——报废未回收标签。
复用既有 (count,sample ids) 输出形状与每日 cron/门禁通道。
验收命令:go build ./... && go test ./internal/domain/report/...

## 提交与合并纪律

- 每项一个独立可 revert 的 commit(feat 带测试;契约变更随项同步 fields.md);
  调研/计划文档与 adopted note 一个 docs commit。
- 每项 commit 后:push gitea 分支 → 主树 fetch → main 未分叉即 ff-only 合并 → 分叉则
  worktree 内 merge main 消化冲突重跑门禁再合。波次收尾:worktree remove + branch -d + 远端分支删除。
- 不可逆裁定当天过账:docs/notes/adopted/2026-09-06-asset-tag-p1-wave.md(联动失败语义/
  幂等轨迹口径/拆机回库存裁定/事件表 DDL/字典播种策略)。

## 暂缓项(P2,后续波次)

服务端筛选分页(G7)——R4 调研结论已可用于设计,独立波次做;SN/MAC/LOID 身份列(G6 后半);
持有台账闭环(G7 死表激活);EPC 格式校验;财务台账关联。

## 六、社区最佳实践检索结论(调研回填,2026-09-06 四路子代理源码级核实)
### R1 资源状态联动(Snipe-IT/iTop/GLPI/MS 模式,源码级核实)

- Snipe-IT checkout/checkin 是同库同步事务(DB::transaction+行锁复检防双借);checkin 默认不改状态标签。
- iTop/GLPI 原生不跨对象强绑,一致性交内建审计(OQL/规则)——共识:工单侧状态机+联动+事后审计。
- 一致性取舍:模块化单体同库默认同事务强一致;跨库才上 Saga/Outbox。
- 采纳:T1 联动改为**强一致**——VerifyScan/UnbindRequireScan 的写路径(链路翻转+扫码日志+资产联动)
  包进同一事务,联动失败回滚阻断 MATCH,留 [quadlink] ASSET LINKAGE FAILED 可 grep 日志;
  MarkDeployed 行锁(SELECT FOR UPDATE)复检:DEPLOYED 幂等 no-op,SCRAPPED 拒绝转人工
  (ErrAssetScrapped);MarkReleased 状态翻转与清地址同一条 UPDATE 原子完成。
  原方案的 best-effort+ALERT 弃用(R1 明示同库禁静默漂移,巡检只作兜底不替代)。
  引用:snipe-it AssetCheckoutController/Asset.php、MSFT domain-events/saga/outbox、NetBox reports。

### R2 事件流与回收(Snipe-IT action_logs/bk-cmdb cc_AuditLog/ITIL-ITAD)

- action 短字符串枚举列(两系统一致,不放字典表);查询维度展开列、差异进 JSONB;
  事件表 append-only,状态留主表;报废软回收禁硬删,先写事件后改状态。
- 采纳:T2 tag_events(id,event_id UUID,tag_id,asset_id,action,actor_account_id,detail,changed JSONB,
  created_at;索引 (tag_id,created_at DESC)/(asset_id,created_at DESC));
  简化裁定:不建 request_id/source 列——本系统事件仅在状态 UPDATE 命中行时写入,
  状态层幂等已天然防重放,事件级幂等键留到出现跨服务重投需求再加。
  引用:snipe-it migrations/Actionlog.php、bk-cmdb audit.go、viprasol PG event sourcing。

### R3 型号字典(Snipe-IT models+fieldsets/GLPI/NetBox DeviceType)

- NetBox:Manufacturer—DeviceType PROTECT 防误删,part_number 分列,UNIQUE 约束;
  Snipe-IT:fieldset 挂型号层,deprecated 软停用;规格收 jsonb 不做动态物理列。
- 存量迁移:规范化→DISTINCT 候选→映射表驱动回填,映射不到挂占位+待办,禁止程序猜测合并。
- 采纳:T3 asset_models(id,vendor,model,category,spec JSONB,part_number,is_active,
  UNIQUE(vendor,model,category));assets.type 保留为展示冗余,model_id 可空回填;
  存量按 DISTINCT type 播种(vendor='(存量未登记)'),'光猫'与'ONU'合并为 category='ONU'
  属业务裁定——裁定前种子并存不合并(映射表可一对多暂缓),MI-ONU 单列。
  引用:NetBox DeviceType、GLPI empty.sql、SO/DBA 迁移讨论三则。

### R4 巡检与列表分页(NetBox Reports/AIP-158/160)

- 巡检:注册表驱动每规则一条幂等只读 SQL,巡检禁写数据;四级严重度;明细行即抛
  「[patrol] FAIL rule=..」+汇总落库双通道——与本项目 orphanChecks 形状同构,T4 直接复用。
- 分页(P2-5 设计定稿):page/page_size(上限100)+排序白名单映射+过滤白名单,响应 {items,total,page,page_size},
  管理后台先 offset+total,量级/导出需求出现再上 keyset;antd Table 服务端模式。
  引用:NetBox reports/custom-scripts、AIP-158/160、gin-pagination。