# 位置/地址统一改造落地口径(2026-09-01)

> 范围:工单坐标快照、地址 geom 写入、逆地理最近邻、区域子树匹配、接单半径闸门。
> 契约同步:fields.md §1.5/§9.6a、terms.md §4(adopted 2026-09-01 各行)。

## 背景

涉位置数据有七套表示(geo_subdivision 行政区划 / addresses 地址树 / regions 经营树 /
user_addresses 地址簿 / worker_locations 师傅位置 / dispatch_tickets 打卡 / odn 码表),
此前双向转化多处断供:addresses.geom 有列无写、工单无坐标使 radiusKm 从未生效、
自动派单路径 region_id=0 使区域闸门全放行、坐标无法归属到地址节点。

## 裁定

1. **坐标是唯一空间锚点,树是组织方式。** 一切"位置"可表达为坐标,"地址"是树上节点;
   正向 = 节点写 geom(`PUT /addresses/{id}/geom`),逆向 = KNN 最近邻
   (`GET /addresses/nearest`,ST_DWithin 信度上限 + `ORDER BY geom <-> point` 单次索引扫描)。
2. **工单坐标派单时刻物化(site_lat/site_lng,000174)。** 快照口径,不随地址树漂移,
   同 orders.price_snapshot;与 000164 arrive_lat/lng(师傅侧事实)互不混用。
   同事务解析 region_id ← orders.region_path 最近祖先或自身,失败降级 NULL 不阻断派单。
3. **区域匹配升级为子树语义(祖先或自身)。** SQL `ltree path <@`,批量口
   (MatchedRegionIDs)供任务池防 N+1;师傅区域 0=不限、工单区域 0=放行的历史口径保留。
   对齐 H3 式"粗区域先行、距离兜底"分层派单社区共识。
4. **半径闸门启用但前提缺失一律跳过。** radiusKm>0 + 工单快照 + 师傅最新位置三者齐备才校验;
   无法判定 ≠ 超距,不误拦;位置查询失败按拒绝留痕。
5. **regions 与 addresses 保持解耦(000002 ADR reaffirm)。** 经营树是组织概念,不并入地理树;
   「地址→经营区域」走 addresses.region_id 硬关联(fields.md §1.5),不建新映射表。

## 放弃了什么

- **H3/geoindex 六边形索引**:当前规模(省级树+楼栋地址)用 ltree+geography 足够,
  引入 H3 需要双写与一致性维护,收益不成比例;子树匹配已在 SQL 层实现同构语义。
- **引入外部地理编码(Nominatim/商业 API)**:逆地理只做"坐标→库内最近节点"而非
  "坐标→完整门牌",不引外部依赖、无配额与合规面;后续若要门牌级可再挂转换服务。
- **自动派单打分(距离加权抢单池排序)**:本轮只把坐标/区域两条事实链修通;
  打分排序是策略层,待派单量上来后另行裁定,避免无数据支撑的参数调优。

## 已知边界(非缺陷)

- 存量 addresses.geom 为空、user_addresses.address_path 历史空串:靠治理工作台
  (needsReview/unlinked 队列)与 admin 选点逐步回填,不做批量猜测式回填。
- 工单坐标快照此前落库的历史工单为 NULL:半径闸门对其跳过,行为同旧版。
