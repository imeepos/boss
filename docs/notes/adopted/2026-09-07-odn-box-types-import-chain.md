# ODN 六类箱体设备类型字典扩展与资源链导入裁定

- 日期:2026-09-07
- 场景:P-INFRA-1 W3(ODN 网络资源表批量导入),需求源 docs/ODN网络资源表模板.xlsx。
- 裁定口径:《Suniway ODN 基础设施资源编码规范》V1.0(docs/pdfs,2.2 编码类型/2.4 箱内部件/3.1 同址扩容)。

## 决策

1. 现实层级 ODF/OCC/ODB/OBD/SDB/SBD 六类箱体设备落 **odn_device 核心链路设备字典**(000081 既定,迁移 000209 扩展),不落 odn_facility。其中 ODF/OCC/ODB/SDB 为规范 2.2 既有核心链路类型;**OBD(一级分光器)按规范 2.4 箱内部件规则归属 ODB**;**SBD(二级分光器)规范 V1.0 未列,按 2.4 同构裁定新增(归属 SDB)**。设备码统一 3 字母前缀+3 位编号+同址扩容后缀 -N(3.1,-1 非法)。
2. 导入域(模板批量导入)箱体设备允许无城市:模板不带城市码,红线"不猜填"禁止虚构 PRV/城市;prv_code/city_prefix 对箱体设备放宽可空,SNW/OLT/PRT/TBP 仍强制城市(显式 CHECK,原 NOT NULL 语义等价保留);城市设备与导入域设备经部分唯一索引 uq_odn_device_box(code WHERE prv_code IS NULL)分域,存量数据零改动。
3. 资源链 23 列落 odn_resource_chain 单表(迁移 000210),23 列归一化指纹唯一去重;导入时链上箱体编码按说明页层级逐级取/建导入域设备(OCC/ODF 顶层,ODB←OCC、OBD←ODB、SDB←ODB、SBD←SDB);机房/OLT/ODF 的站点前缀引用(如 SITE001_ODF001_A)非规范设备格式,按文本引用承载不展开。
4. 枚举映射(枚举 sheet 为校验字典):资源状态 规划→PLANNED、已安装/已测试→IN_BUILD、在用→IN_SERVICE、已报废→RETIRED、**留空→PLANNED(保守,绝不当作已安装/在网)**;端口状态→端口四态 IDLE/RESERVED/USED/DISABLED;敷设方式 AERIAL/UNDERGROUND/SUBMARINE/MICROTRENCH/INDOOR;ROW 状态 NOT_STARTED/PENDING/APPROVED/EXPIRED/NA;PECE 状态 PENDING_SIGN/SIGNED/STAMPED/NA;分光比 1:2~1:128 枚举,总分光比=一级×二级。

## why

- 六类箱体是《基础设施资源编码规范》2.2 的核心链路节点,000081 已为其建好设备字典与归属链(ErrBadHierarchy);odn_facility 是《地理空间编码规范》的网格/顺序型基础设施,六类箱体与网格分区无语义交集。
- 不猜填红线 vs 展开完整性:链上引用的资源必须存在(验收硬规则),展开须落真实资源行;无城市却强制城市会逼出虚构数据,故以"分域唯一索引"承载导入域。
- SBD 是模板现实层级(SDB 箱内二级分光)而规范未列,按规范自身原则(容量预置 3 位编号、编码即信息、箱内部件归属域)同构扩展,保持规则纯净(规范 1.3 原则 3)。

## 放弃了什么(被否决项)

- 扩展 odn_facility 六类 kind 并放宽其 NOT NULL:会松动五类地理空间设施(P/MH/TW/CLS/TBX)语义,与"既有五类语义与存量数据不变"冲突。
- OBD/SBD 按规范 2.4 物理标签 2 位数字(01~99)箱内后缀编码:与模板现实(OBD001/SBD001)冲突,系统侧统一 3 位设备码;物理标签规格属现场施工口径,不在服务端强校验。
- 仅存链表不展开:孤儿/半链巡检("链上引用的资源必须存在")失去落点。
- 全量建模光缆/纤芯:规范 5.1 "--" 仅设计图纸不入库,纤芯编号作链引用列,不入 odn_cable_segment/odn_fiber。

## 关联

- 迁移 000209(odn_device 字典扩展)/000210(odn_resource_chain);契约 terms.md §4、fields.md §1.5.12;domain-map.md ODN 行。
- 实现内部:internal/domain/odn/chain*.go、pg_chain*.go;路由 internal/httpapi/admin/odn_resource_import.go(menu:odn)。
- W4 ROW 路权与 PECE 许可工作流以 row_status/pece_status 列为数据基础。
