# PP2-W1 boss 域体验走查审计(Phase A · 只读)

> 会话 PP2-W1 · 2026-09-07 · 走查对象:102 部署前端 http://192.168.0.102:5180(main@011bacd8 部署版)
> 方法:cdp-admin-capture.mjs 实截 + DOM 断言 + console/网络采集;截图与日志留 /tmp/pp2-w1/(不入库)。
> 范围:菜单 16 页(order/dispatch/dismantle/callback/complaint/feedback/service-metrics/install-board/worker/worker-reg/worker-ops/site/site-cats/knowledge/release/message)含下单抽屉、地址链、指派/转派、师傅 5 类对话框、下发抽屉、审批弹层、编辑器页等 32 个源文件全部交互面。
> 走查基线:16 页全部在真实数据上打开,DOM 断言采集到按钮清单/表格行数/输入框占位;console 0 报错、网络 0 失败(16 页 logs 全绿)。

## 一、总评

| 维度 | 事实 | 结论 |
|---|---|---|
| 卫生红线 | native select=0(16 页 DOM 实测)、alert=0、32 文件最大 258 行 | PP1 成果保持,本轮不回退 |
| toast 反馈 | sonner 仅 4/32 文件(worker/index、worker/TeamDialogs、message/NoticesTab、message/WorkerMessagesTab) | 成功反馈是全域最大缺口(P0) |
| 失败原因保真 | 至少 8 处 catch 后丢弃 e.message 换固定文案 | P0:失败原因不可见=不可处置 |
| 错误呈现模式 | 7 页「错误横幅替换整张表」 | P0 模式缺陷:一次动作失败把列表藏掉 |
| Dropdown 值契约 | ≥9 处把显示文案当 value 传入,触发器回退 ariaLabel | P1 簇:当前选中值不可见(102 实证「状态状态」) |
| 表格可读性 | 9 处列表裸露 #内部编号 | P1:关联对象应显可读名 |
| API 能力对齐 | complaint 3 个路由未暴露;worker-reg 状态维度未暴露;release 状态机前端不设防 | P1 |

## 二、逐页走查表

「步数」= 高频操作从页面到达可提交形态的最少点击数。

| 页面 | 高频操作/步数 | 反馈链缺口 | 表单问题 | 表格问题 | 选择器使用 |
|---|---|---|---|---|---|
| order 订单 | 代客下单 1 步;跟踪 2 步;推进(核查→预占→收费)2-3 步 | 成功无 toast;错误横幅替换整表;跟踪失败文案固定丢原因(L67) | 下单抽屉优(FormField+hint);fixedCustomer 显示 #ID 非姓名 | 分页 total 未按 created=today 过滤口径(L186) | CustomerPicker+搜索 Dropdown+ResourcePicker(地址) |
| dispatch 派单 | 指派/转派 2 步 | 成功无 toast;错误横幅替换整表 | 抽屉 label「*师傅 ID」实为选择器;必填星是文本前缀 | #orderId、改派 #id/#ticketId 裸编号;工单池无检索;status 仅 URL 参数无控件 | WorkerPicker(自研卡片式,优);筛选 ResourcePicker |
| dismantle 拆机 | 新建 1 步抽屉 3 项 | 成功无 toast;错误横幅替换整表 | label 叫「订单/资产/端口 ID」实为选择器;无必填标记 | #orderId/#assetId/#portId 裸编号;无状态筛选 | ResourcePicker 直用 ×3 |
| callback 回调 | 重试 2 步(有确认) | 成功无 toast;错误横幅替换整表 | - | #id/#orderId 裸编号;result 裸枚举英文非 StatusTag;无结果筛选 | - |
| complaint 报障 | 办结 2 步(有确认) | 成功无 toast;错误横幅替换整表 | - | #customerId/#orderId 裸编号;无报障描述列;无状态/类型筛选 | - |
| feedback 回访 | 复核 2 步(有确认) | 成功无 toast;错误横幅替换整表 | - | #id/#ticketId 裸编号;缺评价内容列(只见分) | - |
| service-metrics 指标 | 只读 | 无刷新按钮、无 loading 态 | - | - | - |
| install-board 看板 | 只读+刷新 | - | - | 无行操作无工单反查;注释称用 /install-logs 实未用 | - |
| worker 师傅 | 新增 1 步;设队长 1 步;调队 2 步;详情 1 步 | Form/ResetPwd/Regions/Transfer 成功无 toast;index 两处有 toast(优) | TeamForm 无 placeholder 无必填标记;所属公司 ResourcePicker | 班组/区域名降级 #id 静默(catch 无留痕) | Dropdown+MultiSelect+ResourcePicker;卡片操作复用 Dropdown 做菜单 |
| worker-reg 注册审核 | 通过/驳回 2 步 | 成功无 toast;错误横幅替换整表 | 审批 label「班组 ID/区域 ID」;40 条无分页 | 首列裸 id;仅 PENDING,API status=全量(APPROVED/REJECTED 不可见) | ResourcePicker ×2 |
| worker-ops 师傅端内容 | 发公告 1 步 | load 丢原因;maint/hall 无刷新按钮 | 复用 NoticesTab(同公告表单问题) | hall #orderId 裸编号 | 页签内联 style 裸色值 |
| site 官网内容 | 新建 2 步(跳编辑页);删 2 步(有确认) | 保存/删除成功无 toast;编辑回显失败仅红字 | 编辑器三个下拉显示占位不显当前值(见 §三-2) | 每页条数选择器 no-op(L112);无关键词检索 | Dropdown 值契约违例 ×3;AttachmentPicker(封面上传优) |
| site-cats 官网分类 | 新建/编辑 1 步表单内嵌;删 2 步 | 保存/删除成功无 toast | 启用下拉显示占位;表单无必填标记 | 无分页(量小可接受) | Dropdown 值契约违例(启用) |
| knowledge 知识库 | 新建/编辑 1 步内嵌;删 2 步 | 保存/删除成功无 toast;错误纯红字无底色 | 状态下拉显示占位;code/title 无 placeholder 无必填标记 | content 截断有度(优) | Dropdown 值契约违例(状态) |
| release 版本发布 | 上传 1 步抽屉;编辑 1 步 | 上传/保存成功无 toast;上传无进度态(大文件仅文字变化);错误横幅纯红字 | 「状态状态」实证:当前状态不可见;白名单=逗号分隔 ID 自由文本;notes 单行 input | 无分页无检索;灰度列「比例/白名单数」可读(优) | Dropdown 值契约违例 ×3(端筛选/上传端/编辑状态);状态机不设防 |
| message 消息中心 | 下发 1 步抽屉;发公告 1 步 | 下发/公告成功有 toast(优);load/下发失败丢原因;toggle 成功无 toast 无失败确认 | 下发「内容」单行 input 应 textarea;公告 category 自由文本应枚举;触发器文案与 label 粘连(级别INFO) | 师傅列 #workerId;DataTable 已用(优) | ResourcePicker(师傅筛选/下发);Dropdown 值传值正确(对照组) |

### 对话框/抽屉补充(102 DOM 实证)

- 下单抽屉:客户*/产品*/装机地址*/渠道*/缴费方式 五组 label;档案地址自动带出+钉选回显;地址链五级先搜后建;固定客户时显示 #ID 非姓名。
- 指派抽屉:WorkerPicker 卡片含工号/电话/班组/区域/评分/已接单;label「*师傅 ID(DT-...)」文案应叫「师傅」。
- 师傅对话框:新增(工号/姓名/手机号/装维队/服务区域/登录密码,区域 MultiSelect+hint 优);队伍(名称/编码/所属公司);区域(覆盖式保存+主区域约定);重置密码(有说明文案);详情抽屉(主档+芯片统计+工单/消息/绩效/评价分段折叠,优)。
- 队伍卡片操作下拉实测:编辑装维队/业绩统计/解散 三项。
- 消息下发抽屉:师傅/级别/标题/内容;级别下拉显示「INFO」原文未本地化。
- 注册审批弹层:班组 ID/区域 ID 两个选择器+驳回备注 textarea。

## 三、跨页共性缺陷(含机理与实证)

1. 【P0】成功无反馈:16 页中 12 页的写操作成功后仅静默刷新。修复=统一 toast.success(文案带单号/对象名),i18n 三语 append-only。
2. 【P1·高】Dropdown 值契约违例:components/Dropdown.tsx L48 以 option.value 严格匹配 value,未命中回退 placeholder/ariaLabel(L66)。以下页面把显示文案当 value 传入,触发器永远显示占位、当前选中值不可见(102 DOM 实证:site 编辑器「语言语言/分类分类/状态状态」、release 编辑「状态状态」+端筛选显示「端」、knowledge 状态、site 列表三筛选、site-cats 启用):修复=页面改传 value;W0 可在 Dropdown 兼容 label 匹配兜底(基座层一次修,全域受益)。
3. 【P0】错误横幅替换整表:order/dispatch/dismantle/complaint/callback/feedback/worker/worker-reg 七页 error 时整表不渲染,行动作一次失败即失去上下文。修复=错误条移到表上方常驻位,表格保留;或统一走 toast.error。
4. 【P0】失败原因丢失(catch 丢弃 e.message):order 跟踪 L67;message load L51/下发 L91;NoticesTab L37/L58/L64;AdminNotifsTab L55;worker-ops L29/L33;license 激活 L34(激活码失败最需后端原因)。修复=一律保留 e.message。
5. 【P1】表格裸内部编号 9 处(见逐页表):修复=联表显可读名(订单号/客户名/资产编码),取不到降级 #ID 并 title 留痕。
6. 【P1】complaint 行操作与 API 不对齐:后端已有 POST /complaints/:ticketNo/status(OPEN→PROCESSING→CLOSED 受理流转)、/escalate、GET events 轨迹,UI 仅「办结」,PROCESSING 态 UI 不可达。修复=行内加「受理」;详情抽屉读事件轨迹(不改后端)。
7. 【P1】worker-reg 能力未暴露:GET /worker-registrations 空 status=全部,UI 写死 PENDING;通过/驳回历史不可见;40 条无分页。
8. 【P1】release 状态机前端不设防:terms §4 ROLLED_BACK 不可再投放,编辑抽屉仍给全部 4 态,保存才报错;白名单=逗号分隔 ID 自由文本。修复=按行状态禁用非法目标态;白名单走选择器(依赖 W0)。
9. 【P1】表单规范散点:必填标记/placeholder/辅助说明三缺(队伍表单、注册审批、拆机、知识库);message 下发内容应 textarea;公告 category 应受控枚举;label 文案带「ID」字样的实为选择器(拆机/注册/指派)应正名。
10. 【P2】一致性:dispatch/worker-ops 页签内联 style+裸色值(#1677ff/#666/#f0f0f0,暗色破相,与 PP1-C 同型,建议换 business/TabBar);表格样式手写长串 16 页复制(建议渐进迁 DataTable,message 已示范);Modal 外壳三套(Shell/自写 fixed/ConfirmDialog);license 硬编码 tailwind 色(bg-emerald-500/red-600);order 跟踪与详情抽屉把「取消」排主按钮位;message 级别枚举未本地化;order 检索需手点刷新才生效(无回车触发)。

## 四、删除/危险操作与后端路由核对(硬性纪律项)

| 操作 | 前端确认 | 后端路由 | 后端守卫 | 核对结论 |
|---|---|---|---|---|
| order 取消 | ConfirmDialog danger | POST /orders/:orderNo/cancel(order_workflow.go) | 工作流状态机 | 工作流动作非删除,合规 |
| 装维队解散 | 自定义确认弹层(destructive 钮) | DELETE /worker-groups/:id | SoftDeleteGroup 软删+在职成员守卫 ErrGroupNotEmpty(pg_team.go L45) | 有确认有关联守卫,合规 |
| 官网文章删除 | ConfirmDialog danger | DELETE /site-posts/:id | 无(硬删 DELETE FROM cms_posts) | 真删除有确认,合规;coverAttachment 随行删 |
| 官网分类删除 | ConfirmDialog danger;40900→可读文案 | DELETE /site-categories/:id | categoryInUse 被引用即拒(ErrCategoryInUse) | 有确认有关联守卫,合规(模范) |
| 知识库删除 | ConfirmDialog danger | DELETE /knowledge-articles/:id | 无(硬删;知识库无被引用关系) | 有确认,合规 |
| complaint 办结 / callback 重试 / feedback 复核 / notice 上下架 | 前三者有确认;上下架无确认(可逆) | 路由均在 | - | 合规;notice 上下架登记即可 |

未发现「只读记录被强加删除按钮」或「真删除无确认」情形。

## 五、Phase B 修复清单(负责人批准后执行)

P0(反馈链,先行)
- B1 全域 toast 补齐:12 页写操作 success/error 分支统一 toast;文案带单号;i18n 三语 append-only 独立小提交。
- B2 错误呈现收敛:七页「错误换整表」改表上方错误条+表格保留;失败 catch 一律保留后端原因(§三-4 清单逐处)。
- B3 license 激活失败展示后端 message(现状全吞)。

P1(表单/表格/能力对齐)
- B4 Dropdown 值契约:9 处改传 value;W0 兼容层评审 label 匹配兜底。
- B5 裸编号改可读名:9 处列表联名,降级 #ID+title 留痕。
- B6 complaint 受理按钮+详情事件轨迹抽屉(只调既有 API)。
- B7 worker-reg 状态页签(全部/待审)+分页+审批 label 正名。
- B8 release 状态机按行禁用非法态;白名单选择器(依赖 W0);上传进度态文案。
- B9 表单规范:必填星/placeholder/hint 补齐(队伍/注册审批/拆机/知识库/下单固定客户显姓名);下发内容 textarea;公告 category 受控枚举。

P2(一致性,余力则做)
- B10 dispatch/worker-ops 页签换 business TabBar 去内联裸色;表格渐进迁 DataTable(message 先例)。
- B11 order 检索回车触发;order 跟踪/详情取消钮降级 outline;license 颜色走令牌;message 级别枚举本地化;order 分页 total 口径;site 列表 onSize 假控件处理;service-metrics 补刷新。

## 六、Phase B 验收口径(对齐路线图 §三/§五与任务卡预告)

- 机械门禁:make web-admin-check(TZ=Asia/Shanghai)全绿;boss 域 grep 无原生 select/裸 alert(Phase A 基线已绿,见 /tmp/pp2-w1/web-admin-check-baseline.log)。
- 逐页路线图 §三 六条规则复核;102 真实走查复验:实截+DOM 断言(每页至少 1 条交互断言)+console/网络零报错。
- 证据落 docs/acceptance/2026-09-07-pp2-w1-boss.md;i18n 键 append-only 独立小提交;单文件 ≤300 行。

## 七、证据索引(/tmp/pp2-w1/,不入库)

- 16 页首屏:order.png...message.png + 对应 *.logs.json(console/网络采集,全零报错)。
- 交互面:order-create/track、dispatch-assign、worker-new/team/team-ops、message-send2、worker-reg-approve、dismantle-create、release-edit2、site-editor 共 13 张+logs。
- 采集脚本:capture-all.sh / capture-interact.sh / capture-interact2.sh;结果解析 scan_logs.py;基线门禁 web-admin-check-baseline.log。
