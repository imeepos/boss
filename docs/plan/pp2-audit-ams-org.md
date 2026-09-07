# PP2 W2 · ams+org 域体验走查(Phase A 只读,2026-09-07)

> 任务 U2/波次 W2:ams(asset/inventory/purchase/replace/stock/tag)+ org(apikey/company/datascope/department/menuperm/openplat/partner/post/region/staff)共 16 页(含抽屉子页 21 个)。
> 方法:源码逐文件细读(ams 34 文件/org 31 文件)+ 102 部署前端真实走查(http://192.168.0.102:5180,cdp-admin-capture 实截+DOM 断言+console/网络采集,16 页零 console 报错/零 4xx+);删除类操作逐一核对后端路由。
> 证据:截图与 logs 留 /tmp/pp2-w2/(不入库);审计规则=路线图第三节,纪律=第四节。本阶段未改任何代码。

## 一、总览

| 维度 | 实测 | 结论 |
|---|---|---|
| 红线 | native select=0、裸 alert=0、两域无 >300 行文件 | PP1 卫生成果保持 |
| toast(sonner)覆盖 | ams 14 文件/org **0 文件** | org 域反馈链整体缺失 |
| 成功无 toast 的写路径 | 25+ 处(明细见 P0-1) | 本波最大体验债 |
| DataTable 采纳 | ams/org 均 0,裸 table 26 文件 | 表格改造主战场 |
| 遗留选择器 | ResourcePicker 1 处(replace);自研 setTimeout toast div 1 处(replace) | W0 兼容层+统一 |
| 死代码 | ams/DangerOps.tsx(119 行,含影响面确认弹层)、ams/EventDrawer.tsx(79 行)零引用 | 或接线或删除 |
| 服务端分页 | asset/tag 有;stock/replace/purchase/inventory/org 全系=全量拉取+前端切片 | 数据量增长即劣化 |
| 102 实测数据量 | apikey 82 行无分页、inventory 16 页、purchase 23 单、openplat 9 应用 | apikey/partner 无分页已痛 |

## 二、逐页走查表

| 页面 | 高频操作步数 | 反馈链缺口(P0 视角) | 表单问题 | 表格问题 | 选择器 |
|---|---|---|---|---|---|
| /ams/asset 资产台账 | 新建/编辑/报废/轨迹/删除全部 1 击;型号/批次字典页头 1 击 | 删除失败走页顶 error 条(不可复制);批次/型号/标签下拉数据源加载失败静默为空数组(违可观测红线) | 无分组(7 字段平铺);SN/MAC/LOID 无格式提示;标签/批次下拉无搜索;类型筛选触发器误显「全部状态」(应为全部类型,实测两触发器同文案) | 入库批次=#batchId、位置=#addressId 裸内部编号;标签联表只拉前 200 条,超出静默显 —;轨迹抽屉「操作师傅/操作人」两列同源重复(workerName) | Dropdown ✓;服务端分页+q+状态/类型筛选 ✓(标杆页) |
| /ams/tag 电子标签 | 新建 1 击;禁用/启用/解绑/事件流行内 1 击 | 解绑/禁用失败走 error 条不可复制;**UnbindConfirmDialog(影响面+原因必填)死代码未接线**,解绑实为简单 confirm | band 频段自由文本无枚举/说明 | 绑定资产列裸 #assetId(应显资产编码) | status Dropdown ✓;服务端分页 ✓;解绑带 expectedAssetId 乐观锁 ✓ |
| /ams/stock 盘点 | 新建 1 击;差异处置=列表→明细→行处置 3 击 | 建单成功/diffHandle 成功/CONFIRM/FIX/ESCALATE 成功**均无 toast**;FIX/ESCALATE 缺备注先弹页级 error(就地提示弱) | 扫码「资产ID」自由数字文本**无资产搜索选定**;扫码状态裸枚举 IN_STOCK…未译;处置备注输入沉在表格底部与按钮分离(分组混乱) | 全量拉取+前端切片,无检索/无状态筛选;首列 #任务ID;handledBy 裸 #账号ID | 法人 Dropdown ✓ |
| /ams/replace 更换单 | 新建 1 击;派单 2 击(行→选人→确认);取消 1 击+确认 | 新建成功**无 toast**;成功提示用自研 setTimeout div(3 秒)非 sonner(机制不统一) | 原因无必填标识 | 设备列裸 #assetId;全量+前端分页,无检索/状态筛选;行操作与 API 对齐(PENDING 专属派单/取消,终态禁流 ✓) | ResourcePicker(可搜索,遗留待 W0);WorkerPicker 跨域复用 boss ✓ |
| /ams/purchase 采购单 | 新建/详情/提交/取消/入库确认行内 1 击 | **提交订单无 confirm 无 toast**(状态机推进);建单/入库确认成功**无 toast**;列表 totalAmount 直接 toFixed(2)(未走 fmtAmount,信封异常即白屏风险;detail 已 unwrap) | **创建/编辑「公司 ID」=type=number 裸 ID 输入且默认硬编码 1**(错实体风险);供应商下拉静默默认选第一个;明细行只增不可删;物料编码自由文本+硬编码英文占位 MI-ONU(未 i18n);入库确认明细**硬编码预填 ONU/1GE 且不回带订单明细** | 供应商/公司名人类可读 ✓;statusFilter 服务端过滤后前端再过滤(冗余);无 q 检索;全量+前端分页;入库单面板独立刷新无分页 | 供应商 Dropdown ✓;法人=裸数字输入(重点违例) |
| /ams/inventory 库存查询 | 只读 1 击 | 刷新无 busy 禁用;materialCode **每键击发请求无防抖** | — | 批次列裸 #批次 ID;**缺仓库/库位等关联信息列**;统计卡 grid-cols-3 只放 2 张(占位空) | 无 |
| /org/company 子公司 | 新建/编辑/详情/员工面板全 1 击 | 保存成功**无 toast**;行内按钮无 busy 禁用 | label+必填 ✓;税项属地/开票通道 SimplePicker ✓(枚举已译,域内标杆);员工面板手机号占位硬编码 13800000000(CN 格式,未 i18n) | 客户端检索+分页(小表可);无删除且后端无 DELETE ✓ 对齐 | SimplePicker ✓;员工录入角色 Dropdown ✓ |
| /org/staff 组织架构与人员 | 选节点→加成员 2 击;树操作按钮 **hover 才显**(触屏/键盘不可达,藏深) | 全部写操作(部门/岗位增删改/成员增改/启停)成功**无 toast**;成员启停**无 confirm**(对比员工面板有);load=Promise.all 单 catch,失败不可定位哪个接口 | 复用 Dept/Post/Account 表单(级联赋岗 ✓) | 成员列表检索+分页 ✓;accounts 全量拉取上限风险(>默认 limit 时成员计数失真,待 Phase B 核) | 树+面板 ✓ |
| /org/department 部门 | 新建/编辑/详情 1 击 | 保存成功**无 toast**;行内按钮无 busy | DeptForm ✓(label/必填/法人下拉/就地错误) | 检索+分页 ✓;详情抽屉裸 ID 行;**无删除入口**(后端有 DELETE /departments/:id 占用拒 40900,staff 树有)跨页能力不一致 | 法人 Dropdown ✓ |
| /org/post 岗位 | 同上 | 同上(保存无 toast) | PostForm ✓;角色多选=checkbox 芯片组(非 W0 MultiSelect);角色多时无搜索 | 同上;**无删除入口**(后端有 DELETE /posts/:id) | 部门 Dropdown ✓ |
| /org/region 经营区域 | 下钻 1 击;覆盖主体=行内下拉**选中即改库** | **覆盖主体 PUT coverage 无 confirm/无 toast/无 busy**,误选即生效;失败静默 error 条 | 无表单(区域为种子数据,后端无增删 API,页面无增删 ✓ 对齐) | 层级路径列裸 ltree 原文;无 busy/加载态;下钻+级别筛选+检索 ✓ | 覆盖主体行内 Dropdown 即提交(交互危险);区域选定靠下钻按钮非树 picker |
| /org/menuperm 菜单权限 | 新建角色 1 击;编辑/删除行内 1 击 | 角色建/改/删成功**无 toast**;行内按钮无 busy;删除 danger confirm ✓;403/409 错误映射语义化 ✓(域内最佳错误处理) | RoleForm ✓(模板下拉/分组勾选/已选计数);**权限码 50+ 无搜索过滤** | 矩阵+过滤+分页 ✓;error banner 渲染在 tbody td 内(结构怪);角色卡无分页(小表可) | Dropdown ✓ |
| /org/datascope 数据权限 | 只读 | refresh 按钮无 busy;**搜索语义分裂**:输入仅前端过滤,服务端 keyword 须手点刷新才生效 | —(只读) | 检索+分页 ✓;数据范围列显 LTREE 原文 | — |
| /org/partner 入驻审核 | 页签 1 击;通过 1 击;驳回 2 击(行内输入) | **approve 失败静默(无 else);reject 未捕获异常(无 try/catch)= P0**;通过无 confirm(结果弹窗含初始密码可视为确认) | 驳回意见行内输入 ✓ 必填 | 状态页签 URL 态 ✓;**无分页/无检索/无总数**;组件栈(Table/Badge/ToolbarButton)与域内裸 table 页不一致 | Badge/页签 ✓ |
| /org/apikey API Key | 签发 1 击;吊销 1 击+确认 | 吊销/签发成功**无 toast**(签发有 plainKey 一次性浮卡+复制 ✓);列表 load 无 busy;失败走 error 条 | KeyForm ✓(账号 Dropdown+用途名校验);绑定账号下拉全量无搜索,仅过滤 status=1 | **82 行实测无分页/无总数**;客户端 keyword 过滤;keyPrefix 掩码 ✓;账号列降级 type:#ref ✓ 留痕 | Dropdown ✓(待接搜索) |
| /org/openplat 开放平台 | 创建 1 击;查看/停用行内 1 击 | **停用/启用应用无 confirm**;停启/测试事件/重投/删订阅成功**无 toast**;删订阅 danger confirm ✓;secret 一次性浮卡+复制 ✓;事件目录加载失败显式提示 ✓(全域唯一做对的 catch) | AppForm ✓;SubForm=MultiSelect 带搜索 ✓;endpoint 无 URL 格式校验 | 投递列表**硬截断前 20 条无提示**;无分页;lastError/httpStatus 可读 ✓ | MultiSelect ✓(W0 形态标杆) |

## 三、修复清单(Phase B 执行序)

### P0 反馈链缺失/危险交互(路线图 §3.2)
1. **全域成功反馈补齐(sonner toast.success)**:org 域 25 文件 0 覆盖为最大块;ams 缺口=stock(建单/diffHandle/三处置)、purchase(建单/提交/入库确认)、replace(建单)。逐页清单:company 保存、EntityStaff 录入/重置/启停、department/post 保存、staff 全部树操作与成员启停、region 覆盖、menuperm 角色增删改、apikey 签发+吊销、openplat 停启/测试事件/重投/删订阅、partner 通过/驳回。
2. **partner approve 失败静默+reject 无异常捕获**:补 catch→toast.error(文案可复制);api/partner.ts 返回 falsy 分支显式报错。
3. **region 覆盖主体行内即改库**:改为显式确认(或选中后确认)+busy+toast;失败可复制。
4. **purchase 提交订单**:补 ConfirmDialog(DRAFT→SUBMITTED 推进)+toast.success。
5. **openplat 停用应用**:补 danger confirm(生产应用误点即断供)。
6. **staff 成员启停**:补 confirm(对齐 EntityStaffPanel 语义)。
7. **失败原因可复制**:两域 error 条统一支持选中复制/复制按钮(§3.2);asset 下拉数据源加载失败从静默空数组改为显式错误提示(可观测红线)。

### P1 表单/表格(路线图 §3.3/§3.4)
8. purchase 创建/编辑+SuppliersDrawer:法人 ID 数字输入→法人 SimplePicker/Dropdown(对齐 DeptForm);删除默认 1 硬编码;供应商默认选中改显式必选。
9. purchase 明细行补删除按钮;物料编码接搜索选择器(物料字典)或至少 i18n 占位;入库确认回带订单明细预填(替换硬编码 ONU/1GE)。
10. stock 扫码资产 ID→资产搜索选择器;扫码状态枚举译名;处置备注与行按钮就近分组(就地校验)。
11. 裸内部编号清理(降级规则=名称优先 #ID 兜底+留痕):asset #batchId/#addressId、tag/replace 绑定资产列显 assetCode、inventory 批次列、stock handledBy、EventDrawer actorAccountId、region path 列(加名称链)。
12. asset 轨迹抽屉「操作师傅/操作人」重复列:核对 lifecycleColumns 语义,补操作人数据或砍列。
13. inventory materialCode 300ms 防抖(对齐 asset/tag);评估补仓库关联列。
14. tag 标签联表 200 条上限:按页 tagId 反查或降级 #ID 留痕。
15. department/post 页删除入口与 staff 树二选一统一(后端 DELETE 均存在+占用拒);Phase B 裁定后在两页补齐 danger confirm+40900 文案,或文档化收敛到 staff 页。
16. apikey/partner 补分页+总数(partner 亦补检索);menuperm 权限勾选加搜索;openplat 投递截断显式提示总数。
17. datascope 搜索语义统一(输入即服务端查,或纯前端过滤,二选一)。
18. OrgTree hover-only 操作改常显(或 focus 可达),消藏深入口。

### P2 一致性(路线图 §3.5)
19. replace 自研 toast div→sonner;DangerOps/EventDrawer 死代码:解绑接线 UnbindConfirmDialog(影响面+原因)或删除,不许两存。
20. partner 组件栈与域内对齐方向确认(最小改动,不做视觉重设计)。
21. i18n 残留:EntityStaff 手机号占位 138…、purchase MI-ONU/RK-2026… 硬编码占位;asset 类型筛选「全部状态」文案改「全部类型」。
22. 详情抽屉裸 ID 行(department/post/menuperm)统一命名(如「记录 ID」)或移除。
23. post 角色多选、apikey 账号选择接 W0 MultiSelect/搜索选择器(W0 合并后零成本改造)。
24. 空态文案混用他页 ns(apikey 用 company.empty、多处 loading 当空态)清理;purchase 双重过滤去重+totalAmount 统一 fmtAmount。

## 四、选择器专项(对接 W0 基座)

| 现状 | 位置 | Phase B 处置 |
|---|---|---|
| native select / 裸 alert | 全域 0 | 保持,不回退 |
| 遗留 ResourcePicker 直用 | replace 1 处 | W0 兼容层生效后零改动受益,仅回归验证 |
| 自由文本 ID 输入(重点违例) | purchase 法人 ID、SuppliersDrawer 法人 ID、stock 扫码资产 ID | 换 SimplePicker/搜索选择器(P1-8/10) |
| 有下拉无搜索 | asset 批次/型号/标签、apikey 账号、post 部门/角色 | 接 W0 搜索选择器 |
| MultiSelect 标杆 | openplat SubForm 事件目录 | 域内推广参照 |
| 税项等枚举标杆 | company SimplePicker(税属/通道,三语已译) | 域内推广参照 |
| region 树选择 | 下钻按钮替代树;行内下拉即提交 | P0-3 治危险交互;树 picker 待 W0 供给评估,无则保持下钻+确认 |

## 五、删除类操作 × 后端路由核对(禁止强加/漏报双查)

| 操作 | 前端入口 | 后端路由(已核) | 语义 | 现状判定 |
|---|---|---|---|---|
| 删资产 | 行内(IN_STOCK 且非 SCRAPPED 显) | DELETE /assets/:assetId,守卫:仅入库且无标签绑定/流转引用,40900 列阻断项 | 真删 | danger confirm ✓,确认文案已述守卫 ✓;失败可复制性缺(P0-7) |
| 标签删除 | **无入口** | 后端无 DELETE /tags(仅 disable/enable/unbind) | — | 对齐 ✓,不缺不滥 |
| 采购单取消 | 行内(非 RECEIVED) | POST /procurement/orders/:id/cancel | 状态终止非删 | danger confirm ✓+toast ✓(域内标杆) |
| 入库单驳回 | 面板行内(DRAFT) | POST /procurement/receipts/:id/reject | 终态 | 原因必填 ✓+toast ✓ |
| 供应商停用 | 抽屉行内 | POST /procurement/suppliers/:id/disable | 可逆 | danger confirm ✓+toast ✓ |
| API Key 吊销 | 行内 | DELETE /api-keys/:id | 吊销(真删) | danger confirm ✓;成功无 toast(P0-1) |
| 删角色 | 角色卡(自定义) | DELETE /roles/:roleId(内置拒改删/占用拒) | 真删 | danger confirm ✓+语义化错误 ✓;成功无 toast(P0-1) |
| 删部门/岗位 | staff 树(hover) | DELETE /departments/:deptId、/posts/:postId(占用 40900 拒) | 真删 | danger confirm ✓;department/post 独立页无入口(P1-15 统一) |
| 删 Webhook 订阅 | 应用详情 | DELETE /openplat/subscriptions/:id | 真删 | danger confirm ✓;成功无 toast(P0-1) |
| 删区域/法人 | 无入口 | 后端无对应 DELETE | — | 对齐 ✓ |

结论:未发现「只读事实记录被强加删除按钮」;真删除均有 danger confirm 且后端有占用守卫;漏报项为零;缺口集中在「成功无反馈」与 region 覆盖/openplat 停用两处无 confirm 的写操作。

## 六、Phase B 验收口径(预告对齐)

- 门禁:make web-admin-check 全绿(TZ=Asia/Shanghai);i18n 三语键 append-only 独立小提交;单文件 ≤300 行。
- 逐页对照路线图第三节 6 条规则;102 真实走查复验:cdp 实截+DOM 断言+console/网络零报错;证据落 docs/acceptance/2026-09-07-pp2-w2-ams-org.md。
- 前置:merge main(含 W0 选择器基座)后开工;本分支 feat/pp2-w2-ams-org 仅含本审计文档。
