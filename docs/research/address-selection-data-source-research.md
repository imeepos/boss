# 家庭地址管理：地理位置添加方案与下拉数据源调研

> 基线：git main@596eff1a（最新决策 note 2026-09-04-user-android-address-locator）
> 作用域：mobile/user/android 地址簿 + 后端地址树数据链路
> 结论先行：推荐「后端权威树驱动级联 + 小区搜索兜底」，数据源分两层维护——行政区划层从官方编码表同步，小区/楼栋层随资源覆盖由 admin importer 持续导入。不引入地图 SDK。

## 一、业界四种主流方案

### 1. 行政区划级联选择（省市区三级联动）

- 交互：每级一个滚轮/列表，选父级才加载子级；数据可打包进客户端（离线）或由服务端按 parentCode 拉取。
- 覆盖深度：官方编码表只到「省-市-县区」（民政部口径）或「省-市-县区-乡镇街道」（统计局口径），**到不了小区/楼栋**。
- 典型实现：Android citypickerview/citypicker 滚轮组件；前端 npm 数据包（china-division、province-city-china，基于民政部最新数据，2026 版仍活跃维护）。
- 适用：收货地址的「行政区划段」。对宽带装机只解决前 50%——装维要的是小区+楼栋+端口。

### 2. POI 关键字搜索 / 自动补全

- 交互：一个搜索框，输入关键字实时出候选（小区/楼盘/地标），选中后回填结构化地址，门牌号手输。
- 数据源：图商持续维护（高德输入提示 + POI 搜索、腾讯 WebService、Google Places Autocomplete）。
  - 中国境内：测绘法约束下必须用国内图商；高德个人开发者有免费日配额，企业认证更高。
  - 海外（如菲律宾市场）：Google Places，按次计费，需 Key + 合规声明。
- 优点：UX 最好，数据新，能搜到「阳光小区 3 期」这种粒度。
- 缺点：返回的是**图商 POI id**，与运营商资源（端口/覆盖）无关——用户选了「能搜到的小区」不代表「能装宽带」；Key/配额/合规三件套是持续成本。

### 3. 地图选点 + 逆地理编码（GPS / 拖图钉）

- 交互：用户拖图钉或点「当前位置」，SDK 逆地理编码出结构化地址，用户微调。
- 精度：一般到楼栋级就衰减，适合外卖/打车「大概在哪」；对装机地址不够权威。
- 本仓库现状即此模式的降级版：`AddressEditorSheet` 拿 WGS84 坐标把 `GPS: N, E` 字符串塞进 door 字段（2026-09-04 决策记录），定位为**辅助参考**，不是权威数据源。

### 4. 运营商覆盖地址树（资源驱动，TMF673 Service Qualification）

- 交互：用户沿 市→区→街道→小区→楼栋 逐级选择（或搜索直达），树由运营商自维护，节点与 OLT/端口资源绑定；选完即知「能不能装」——这就是 TMF673 服务资格（Service Qualification）模式，国内电信/移动「覆盖查询」同构。
- 数据源：**没有公开数据集**，靠运营商自建：资源勘察录入、装维工单回填、渠道批量导入。
- 迁移 000038 头注已引用 TMF673/GeoNames 调研结论，本仓库的 `addresses` ltree（ADR-002：path 唯一权威，L1市→L5楼栋）就是按这个模式设计的。
- 适用：宽带/固网 ISP 的标准答案——订单 12 环节的「资源核查」（checkResource）本来就要按地址节点找端口。

## 二、下拉数据源从哪来（按层拆）

| 层级 | 官方数据源 | 更新节奏 | 获取方式 |
|---|---|---|---|
| 国家 | ISO 3166-1（alpha-2 码） | ISO 不定期发布 | 本仓库 `geo_country` 已落库（102 上 5 国） |
| 一级行政区（省/大区） | ISO 3166-2；CN 另有民政部《行政区划代码》；PH 为 PSA 的 PSGC | CN 年度（国务院批复后）；PH 季度 | 本仓库 `geo_subdivision` 已落库（102 上 43,775 条，其中 PH 43,769 = PSGC 全量） |
| 市/县区/街道（L2-L3） | CN：国家统计局统计用区划代码（年度，到村级）；PH：PSGC 层级 | 年度/季度 | 官方 CSV/JSON 下载 → 批量导入 |
| 小区/楼盘（L4） | **无官方数据**。来源三选：图商 POI / 运营商勘察自建 / 用户提交审核入库 | 资源覆盖扩展时 | 本仓库 admin `POST /addresses/import`（rows: path+name） |
| 楼栋/单元（L5） | 无。运营商勘察或物业资料 | 装维建档时 | 同上 importer；或装维工单回填 |

结论：**行政区划层（L1-L3）是「同步」问题，小区楼栋层（L4-L5）是「运营」问题**，两类数据源的维护方式完全不同。

## 三、数据怎么维护（业界通法 → 本仓库对应物）

| 维护机制 | 业界通法 | 本仓库现状 |
|---|---|---|
| 批量导入 | CSV/Excel → 校验 → 幂等 upsert，导入任务留痕 | 已有：admin `/addresses/import` + import-tasks 菜单（幂等裁定 2026-09-03） |
| 防误删 | 被订单/资源引用的节点禁删，只允许改名/停用 | 已有：DELETE 有子节点/引用时拒绝 |
| 路径稳定 | 地址码/path 不随改名变化 | 已有：ADR-002 ltree path 唯一权威 + UNIQUE 约束 |
| 官方层更新 | 年度跟随发布重新导入（全量 upsert） | 缺：无定时/脚本化的年度同步流程 |
| 小区层增长 | 覆盖勘察/装维回填，随资源开通入库 | 缺：无运营流程；102 树基本为空 |
| 权限管控 | 只有资源管理员能改地址库 | 已有：menu:address / menu:importer 权限码 |

## 四、本仓库现状诊断（102 实测）

**已具备（后端，全部验证过）：**

- `addresses` ltree 表：L1-L5 + path 唯一 + level CHECK + geo 锚点（country/admin 码只挂根节点，`v_addresses_geo` 视图继承）。
- admin 端点全家桶：`GET /addresses?parentId=`（层级树）、`POST /addresses/import`、`PUT /addresses/{id}`、`DELETE`（有引用拒删）、`PUT /addresses/{id}/geo`、`GET /addresses/search?q=`（**命中节点+祖先链，懒加载树展开用**——语义就是为级联/搜索准备的）。
- geo 基础域：`geo_country`/`geo_subdivision`/i18n 译名/邮编正则（000038），102 上 PSGC 全量 43,769 条已在库。
- admin web 树管理页 + 节点抽屉 + geo 锚点抽屉。
- 用户端 CRUD：`/api/user/v1/addresses` list/create/update/delete（portal_address.go）。

**缺口（问题全在这里）：**

1. **102 的 addresses 树是空的**：41 个根节点全是 E2E 测试垃圾 + 1 个 L2 节点。权威树没数据，任何「从树选地址」都无从谈起。
2. **用户端契约没有树查询端点**：misc.yaml `/addresses` 只有本人地址簿 CRUD；App 下拉没有任何服务端数据源可用（2026-09-04 决策被迫降级为 GPS+历史小区，根因即此）。
3. **user_addresses 与树零关联**：`addr_code` 是自由文本（社区名），detail 是 community+" "+building+" "+door 拼接串；选了树节点也存不下结构化引用。
4. **geo_subdivision 与 addresses 树无桥接**：PSGC 全量已在库，但 L1-L3 没有物化进树，行政区划层白白重复手工录。
5. GPS 坐标串占位 door 字段，等契约字段就位后替换。

## 五、推荐方案：树驱动级联 + 小区搜索兜底

与 2026-09-04 决策（不引地图 SDK）兼容：数据源全在自有后端，App 零新增第三方依赖。

### 数据链路（三段）

1. **L1-L3 物化**：一次性（ thereafter 年度）从 `geo_subdivision` 生成树节点进 importer：path 取区划码小写（如 `ph.ncr.quezon_city`），name 取 i18n 中文名。PH 数据已齐，零采购成本。
2. **L4-L5 运营导入**：资源覆盖到哪录到哪——admin `/addresses/import` 收 CSV（模板列：path,name），随覆盖扩展持续导。这是 ISP 的「商品上架」动作，不是技术问题。
3. **官方层年度同步**：CN 民政部/统计局、PH PSGC 发布后重新导 geo 域再物化，importer 幂等保证可重复执行。

### 契约变更（user.yaml，前置条件）

- 新增 `GET /address-tree?parentId=`：返回子节点（id/name/level/hasChildren），登录即可，响应可缓存。
- 新增 `GET /address-tree/search?q=`：语义对齐 admin `/addresses/search`（命中+祖先链），小区层数据量大，搜索比逐级翻页好用。
- `AddressInfo` 扩展：`addressPath`（ltree 字符串）+ 保留 community/building/door 作展示冗余；user_addresses 加 `address_path` 列（不强 FK，树节点允许改名重整）。

### App 端交互（替换现有自由文本三件套）

- 市→区→街道：三个连续底部弹层列表（ModalBottomSheet + LazyColumn，懒加载 children），沿用现有组件风格，不引滚轮库。
- 小区：搜索框 + 服务端搜索（debounce 300ms），命中直接带出祖先链回显。
- 楼栋/门牌：继续手输（L5 数据通常不全，强选反而卡死用户）。
- GPS「使用当前位置」保留为辅助：定位后可提示「距最近小区节点 Xm」或仅作展示参考，不作为权威字段。

### 落地顺序（供后续任务拆分）

1. 后端：user.yaml 契约 + `GET /address-tree` 两端点 + user_addresses.address_path（迁移）。
2. 后端：L1-L3 物化脚本（geo_subdivision → importer），102 验证树有真数据。
3. 运营：L4-L5 样例导入（马尼拉演示小区），全链路可演示。
4. App：AddressEditorSheet 改级联+搜索（注意该文件已 300 行顶红线，需先拆文件）。
5. App：GPS 辅助定位降级为展示/参考。

## 参考来源

- [cn-division（基于民政部数据的中国行政区划 npm 包，2026 版）](https://github.com/kk-418/cn-division)
- [citypicker（Android 省市区滚轮组件）](https://github.com/chancelau/citypicker)
- [Google Places Autocomplete Address Form（国际地址表单官方范式）](https://developers.google.com/maps/documentation/javascript/examples/places-autocomplete-addressform)
- [高德开放平台流量限制说明（配额分层）](https://developer.amap.com/api/javascript-api-v2/flowlevel)
- 本仓库：ADR-002（ltree path 权威）、migrations/000038（ISO 3166/UN M49/CLDR/GeoNames/TMF673 调研引用）、000040（geo 锚点）、docs/notes/adopted/2026-09-04-user-android-address-locator.md（GPS+历史小区决策）
