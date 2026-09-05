# 资产与标签系统设计调研:业界成熟方案对照与打磨路线图

> 日期 2026-09-06 | 调研范围 TM Forum SID/TMF634-639、运营商 RFID(EPC Gen2)、FTTH ONU 资源管理、
> ITIL4/ISO19770 ITAM、云厂商标签治理(AWS/阿里/腾讯)、开源 CMDB(蓝鲸/Snipe-IT/NetBox)。
> 实证基线:102 生产库真数据 + internal/domain/asset 全量代码走读。配套裁定:
> adopted note 2026-09-06-asset-tag-quality-gate(P0 已随本提交落地)。

## 一、我们已有的底子(对照业界不落下风的部分)

| 能力 | 现状 | 业界对应 |
|---|---|---|
| 资产-标签双向绑定 | 应用层互填 + DB 部分唯一索引(000158) + 每日巡检,真库 0 孤儿 | Snipe-IT checkout/checkin 一致性;蓝鲸唯一校验 |
| 盘点差异闭环 | 建单冻结快照→扫码回填→逐条处置→全处置才关单(000156) | ITAM 盘点=任务单+差异工单闭环(业界标准形态) |
| 扫码强校验 | 环节9 EPC↔资产核对,不一致拒推进;拆机必扫码 | FTTH 预放装核对基线 |
| 状态轨迹表 | asset_lifecycles 快照行(历史不漂移) | SID 资源生命周期;ITIL 状态机审计 |
| 换新单状态机 | PENDING→DOING→DONE/FAILED,终态不可逆(000159) | ITAM 借用/维修/处置工单化 |

## 二、真库实证的八个缺口(2026-09-06,102 生产库)

数据规模:assets 206 / tags 208 / batches 193 / lifecycles 6 / assignments 0 / replacements 2 / quad_links 14。

| # | 缺口 | 证据 | 业界做法 |
|---|---|---|---|
| G1 | 装机/拆机全流程不联动资产台账:扫码 LINKED 只写 quad_links+scan_logs,资产停留 IN_STOCK;全系统无 SCRAPPED 写路径 | 40 台 DEPLOYED vs 14 条链路;lifecycles 6 行 | TMF639 资源库存是唯一事实源,业务开通原子更新资源状态;SID 资源-服务关系表 |
| G2 | 建档不留痕(206 资产仅 6 条轨迹) | lifecycles 6 行 | ITAM:生命周期首行=入账,每次流转留审计 |
| G3 | 枚举无 DB 闸门,type 自由文本已脏('光猫'/'ONU'/'MI-ONU'/''并存) | 4 种 type 值+空串状态行 | 云厂商标签写入端硬校验;蓝鲸字段校验在存储层兜底 |
| G4 | CreateAsset/CreateTag 非原子:主档落库后回填失败留半成品 | 000087 前历史孤儿 124 条即此类产物 | 事务原子性是库存系统底线 |
| G5 | 标签无解绑/回收闭环:换新/报废后标签死绑,无法复用 | 代码中无任何 bound_asset_id 置空路径 | RFID 实践:标签 ID 与资产档案解耦,换绑是常态操作且必须留事件流 |
| G6 | 资产无设备身份:无 SN/MAC/LOID 列,无型号字典 | assets 仅 type 字符串 | FTTH:SN 唯一键+认证四元组(SN/MAC/LOID/Password)核对;SID:serialNumber/manufacturer/model 规格列+EAV 特征 |
| G7 | 查询能力缺失:列表全量返回无过滤分页;asset_assignments 只有 GET 没有 POST(死表,0 行) | admin handler 直接 ListAssets | CMDB 分页+多维筛选是标配 |
| G8 | 绑定/解绑无事件流(仅 RecordAudit 操作日志,档案页不可见) | 无 tag 绑定事件表 | CMDB:绑定/解绑/转移建模为可审计事件流,支撑追责 |

## 三、业界成熟方案要点(四路调研综合)

### 3.1 电信 BSS:TM Forum SID 与资源库存(TMF634/TMF639)

- 目录与库存分离:ResourceSpecification(型号规格,含默认特征)与 Resource 实例分表,实例挂 specId;
  Oracle UIM 即按此实现并开放 REST。
- 通用列只放 SN/厂商/型号/状态等高频字段,扩展特征走 resourceCharacteristic 键值对(EAV);
  物理资源与逻辑资源(端口/账号)用 resourceRelationship 关系表绑定(relation_type+生效起止)。
- REST 按 @type 多态统一端点,避免每类设备一套接口。

### 3.2 运营商级 RFID 资产管理(EPC Gen2 / ISO 18000-6C)

- 标签 ID 与资产 ID 解耦:服务端存绑定关系,换标签不动资产档案;
  盘点=任务单→批量读取→与在库清单差集→盘盈/盘亏差异工单(我们的 stocktake 已是此形态)。
- EPC 遵循 GS1 TDS 编码(头+域管理器+对象类+序列号),入库格式校验防串码;
  金属密集场景预留有源标签(2.4G)+中继接入(广东移动基站实践)。

### 3.3 FTTH ONU 资源管理与认证四元组

- 三大运营商认证字段各异(LOID+密码 / Password+SN / LOID+SN / 绑 MAC),但共同点是:
  ONU 档案以 SN 为唯一键,认证四元组+PON 口位置+装机地址构成核对基线,任一不符拦截工单。
- 预放装闭环:装维 App 扫 SN→生成/回填认证四元组→OLT 白名单→注册回读 MAC 与在线状态核销;
  OLT 上报未注册 ONU 自动生成异常工单防私接。

### 3.4 ITIL4/ISO19770 ITAM

- 状态机固定(在库/在用/借用/维修/退役/报废),每次流转留审计;财务台账(原值/折旧)与实物台账
  (位置/责任人/状态)同 ID 关联、分口径报表、定期盘点对账。
- 退役联动释放关联资源(我们的对应物:SCRAPPED 时回收标签、清理四码链路)。

### 3.5 云厂商标签治理(AWS/阿里/腾讯)

- 写入端硬校验(键值长度/字符集/配额 20~50);系统前缀保留(aws:/acs:,对应我们可用 sys:)。
- 治理=策略声明+不合规报表+定时修复,写入强拦只对关键资源;分账标签须显式激活并记生效时刻。
- 标签继承在归属变更时复制快照留来源标记,不做运行时动态推算;改标签权限独立于资源操作权限并全量审计。

### 3.6 开源 CMDB(蓝鲸/Snipe-IT/NetBox)

- 结构化属性(产权/SN/局站)走模型字段+唯一校验+格式校验;多维圈选才用多对多标签,二者不混用。
- Snipe-IT:自定义字段集(Fieldset)挂在型号层,正则/必填在写入端强制;checkout/checkin 全程留痕。
- 标签 schema:tags(id,name,namespace)+关联表复合主键+双侧索引;命名空间 ns:key=value 防跨域冲突。

## 四、差距矩阵 → 打磨路线图

### P0 数据质量闸门(本波已落地,commit 见 adopted note)

1. 000184:status/type DB CHECK + 存量脏行清洗(G3,G4 的存储侧)。
2. CreateAsset/CreateTag 事务化 + 建档即落轨迹首行(G2,G4)。

### P1 功能闭环(高价值,建议下一波)

1. **装机/拆机资产联动**(G1):环节9 VerifyScan MATCH 时资产→DEPLOYED+绑地址+落轨迹;
   拆机 UNLINKED 时→IN_STOCK+清地址+落轨迹;新增 SCRAPPED 终态路径(退役处置入口)。
   验收:走通 12 环节订单后 SQL 断言 assets.status=DEPLOYED 且 lifecycles 有 DEPLOYED 行;
   巡检(LINKED 链路 vs 资产状态)差集为 0。
2. **标签回收闭环 + 事件流**(G5,G8):UnbindTag API(置 bound_asset_id=NULL+status=UNBOUND+事件行);
   SCRAPPED/换新 DONE 时自动回收旧标签;tag_events(tag_id,action,old_asset,new_asset,actor,at)。
   验收:换新完成后旧标签 status=UNBOUND 且 events 有 RECYCLE 行;拆机后标签可重绑他资产。
3. **型号字典 asset_models**(G6 前半):models(vendor,model,category,spec fields 原型),
   assets.model_id;存量 type 归一迁移('光猫'并入 ONU 或建档保留);admin 维护页。
   验收:新建资产必选型号;页面类型列读字典;存量 type 全部命中字典。
4. **巡检扩展**(G1 兜底):每日巡检加'LINKED 链路但 asset 非 DEPLOYED'与'SCRAPPED 资产仍绑标签'两查。
   验收:构造漂移数据后巡检输出含该行,ALERT 日志可 grep。

### P2 运营增强

5. 列表服务端筛选分页(assetCode/tagNo/epc/status/region/address),前端去掉全量本地过滤(G7)。
6. SN/MAC/LOID 身份列+唯一索引,换新核对升级为四元组核对(G6 后半,对齐 3.3)。
7. 持有台账闭环:POST /assets/assignments(领用/归还,EffectiveTo 回填)+死表激活(G7)。
8. EPC 格式校验(GS1 TDS 简化:结构 regex+长度),防串码(3.2)。
9. (远期)财务台账关联:采购单价→assets 成本列,折旧口径报表(3.4)。

## 五、引用来源

电信 BSS/资源库存:
- https://docs.oracle.com/en/industries/communications/uim/8.0/rest-api/resource-inventory.html
- https://www.tmforum.org/open-digital-architecture/open-apis/resource-inventory-management-api-TMF639/v5.0
- https://www.tmforum.org/open-digital-architecture/open-apis/resource-catalog-management-api-TMF634/v4.1
- https://www.tmforum.org/Browsable_HTML_SID_R20.0/content/_3E3F0EC000E93DDFC04301E2-content.html
RFID/EPC:
- https://ref.gs1.org/standards/tds/1.8.0/
- http://www.gongkong.com/article/202210/101748.html(广东移动基站 RFID)
- https://chainway.net/Cases/Info/234(铁塔 UHF 盘点)
FTTH/ONU 认证:
- https://www.chinadsl.net/forum.php?mod=viewthread&action=printable&tid=178219(三网认证字段对照)
- http://i.mqrouter.com/help/itms-fast-book/(ITMS/TR069 注册模板)
ITAM/ITIL4:
- https://www.manageengine.cn/products/service-desk/help/adminguide/alc-through-po-and-barcode.html
- https://www.givainc.com/blog/it-asset-management-best-practices/
云厂商标签:
- https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html
- https://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_policies_tag-policies.html
- https://help.aliyun.com/zh/resource-management/tag/user-guide/use-tags-to-control-access-to-resources-1
- https://www.tencentcloud.com/zh/document/product/1113/58501
CMDB:
- https://github.com/TencentBlueKing/bk-cmdb
- https://snipe-it.readme.io/docs/custom-fields
- https://docs.netbox.dev/en/stable/models/extras/tag/