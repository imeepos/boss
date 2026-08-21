# UI 生成反馈台账

## A. 设计稿质量（视觉一致性 / 保真不足）

- 2025-08-20 profile-v1：不合格（C-）。致命：8 区块强行一屏 → 行高 24-26dp/按钮 31dp 远低于 48dp 触控标准；
  卡片间距 4-5dp、页边距 8-10dp 无呼吸感；Tag 三种样式混用无语义；图标两组风格拼接；平台归属混乱
  （Android 挖孔+iOS 状态栏、无手势条/安全区）；退出登录高饱和通栏红抢过业务入口；辅助文字 10-12sp 不可读。
  根因：生成 prompt 要求"一屏展示全部区块"，未写移动端硬约束（允许滚动/最小触控/边距/安全区），
  gpt-image-2 只能靠全面缩水满足"塞进一屏"。
  规则：移动端 prompt 必须写明——页面可垂直滚动不要求一屏、列表行最小 48dp、按钮最小 44dp、
  页边距 16dp、卡片间距 8-12dp、含状态栏与底部手势条、单一图标库、Tag 样式与语义一一对应、
  破坏性操作用白底红字描边而非实心通栏。
  再犯标记：无（首犯，已提炼为预设 M5 元模板）

## B. spec 完备性（提示词 / 注意事项遗漏）

- 2025-08-20 profile-v1：spec 本身完备（区块/状态/数据契约齐全），但它约束的是"实现"而非"设计稿"——
  spec 里写了行高 44dp、间距 4 的倍数，生成图的 prompt 却没用这些值。
  根因：spec 的设计 token 表是"事后规格"，没有回灌到生图 prompt。
  规则：先生成 prompt 后写 spec 时，spec 的 token 表必须与 prompt 的 Style/Colors 段同源；
  已有 spec 再生图时，把 token 表逐项翻译进 prompt。

## D. 用户端 4 tab 全量规范(2025-08-21 建立)
- 落地:`designs/USER-APP-SPEC.md`(12 节),覆盖首页/服务/账单/我的 + 共享组件 + token + 12 环节映射。
- 关键事实源优先级:代码真值(`ui/theme/Color.kt` + `ui/Theme.kt::Palette` + `ui/Widgets.kt`) > 本规范 > 旧 UI-SPEC.md。
- 主色冲突:`UI-SPEC.md` 写 `#086CF5`,代码真值 `#007AFF`(BrandBlue);新页面以本规范为准,
  与旧 UI-SPEC 冲突时一律写"以 USER-APP-SPEC 为准"。
- 页边距:旧 UI-SPEC 写 16dp,代码 AppCard.outer=14dp;**14dp 是项目既有约定,禁止改为 16dp**,
  否则 4 tab 出现 4 种页边距撕裂。
- 退出登录:FEEDBACK A 区已定为红线(白底卡 + 居中红字 + 小图标),ProfileMenu.kt 严格遵循。
  规则:任何"账号与设置"类页面的破坏性操作一律沿用本样式,禁止实心通栏红按钮。
- 12 环节映射:步骤条 4 节点命名全 App 固定("提交订单 · 受理成功 · 上门安装 · 完成"),
  任何新页面/新稿复用,分屏稿 prompt 必须逐字写死步骤标签。
- 未编造清单:Product 无 originalPrice/features、Order 列表无 createdAt、Address 无类型、
  Profile 无头像 URL —— 这几条写进 USER-APP-SPEC §11,新页面禁止补字段,缺数据时空态或省略。

### D2. 第二轮对齐(2025-08-21)—— 代码为准全面校正
- **触发**:用户明确指示"代码和 UI 规范不一致的地方以代码为准"。重新 grep 38 个 page + 6 个 ui 源文件,
  整理出 14 项不一致,全部回填 USER-APP-SPEC。
- **关键修正**(USER-APP-SPEC §11.5):
  1. **登录/注册/实名/找回**用私有 RN token(`RN.primary #086CF5` ≠ 全局 `#007AFF`),非设计失误,是有意"门面页/操作页"色系分离;新增 §2.3、§5.1、§6.0 四页区块清单。
  2. **首页未读红点** `#FF5252`(代码) ≠ `Palette.err #FF3B30`(规范原写);不是 bug,是首页特定强化色。
  3. **"在网"徽章**用 Green500 实色 8dp 圆角胶囊(非 Tag 浅底规范);仅限此一处例外。
  4. **找回密码错误色**用 `Palette.err`(非认证流 `#FF2D2F`);局部差异,沿用原值。
  5. **登录页 AuthSegment selected 色** 用全局 `Palette.primary #007AFF`(非 RN.primary);与同 App 其他分段控件一致。
  6. **字号体系**:原规范写 5 档(12/14/16/20/22),实测 13 档(11/12/12.5/13/14/15/16/17/18/20/22/24/30),新增 §3.1 实测表。
  7. **字重体系**:原 3 档(W500/W600/Bold),实测 4 档(Normal/W500/W600/Bold)。
- **规则**(沉淀):
  - **任何新页面/新设计稿,先 grep `mobile/user/android/app/src/main/java/com/ymm/boss/user/ui/theme/Color.kt`
    与 `page/RealNamePage.kt::RN` 两个文件确认色值,再下笔。**
  - **出现"代码 vs 规范"冲突,默认改规范(USER-APP-SPEC)而非改代码,除非用户明确要求重构**。
  - **不允许设计师/agent 用"统一"为名,抹平差异(如把登录页渐变"统一"成首页渐变、把未读红点"统一"成 err 红)**;
    设计师必须尊重代码真实存在的设计决策,除非有评审记录支持改动。
- **再犯标记**:无(首轮审查建立,未生成新稿即校准)。

### D3. 第三轮核对(2025-08-21)—— 页边距"单值"是错的,实为两套上下文
- **触发**:用户要求了解用户端 UI 设计规范,逐项回读代码核对 USER-APP-SPEC 时发现硬伤。
- **错误**:原 §4 写"滚动区左右边距 = 14dp"并加了 ⚠️ 强调"禁止改 16dp,否则出现 4 种页边距"。
  实际代码 `ui/PinnedGradientPage.kt:43` 明写 `sideMargin = 16.dp`,首页/我的滚动区就是 16dp;
  且这两页的卡片会 override `outer` 去掉水平内边距(`ProfilePage.kt:160`),不存在 14+16 叠加。
- **真值**:页边距是**两套上下文**——pinned 骨架页(首页/我的)= 16dp 由滚动区给;
  非 pinned 页(服务/账单/详情)= 14dp 由 `AppCard.outer` 给。全仓实测 14dp 24 处、16dp 11 处,并存且各有归属。
- **根因**:第二轮审查只 grep 了 `AppCard` 默认值,没读 `PinnedHeaderSpec`,把一个上下文的值当成了全局唯一值;
  更糟的是给错误值配了一句语气很强的"禁止"警告,把错误固化成了红线。
- **规则**(沉淀):
  1. **写"全局唯一值"型断言前,必须确认没有第二个来源**——几何参数尤其容易有"容器给"和"组件给"两条路径,
     只 grep 组件默认值会漏掉容器。至少 grep 一次 `object *Spec`/骨架文件再下结论。
  2. **语气强度必须与验证强度匹配**:没有逐调用点核对过的值,不许写"禁止/必须";
     错误结论配强语气会让后续 agent 不敢质疑,危害大于结论本身错。
  3. 规范里凡出现"某某= 单个数值"且该数值在代码中出现次数 >1 处不同值时,一律改写成上下文表格。
- **再犯标记**:1 次(D2 轮已犯"只查一个来源就下全局断言",本轮是同类根因的第 2 次体现)。


## 方法论（蒸馏自上述条目）

- 移动端设计稿生成硬约束已固化为预设 ui-proto skill 的 M5 元模板（见该 skill）。

- 2025-08-20 profile v4-v7（4 轮）：即使有 UI-SPEC token + UI-PARADIGM 范式 + 优秀单屏参考图，
  gpt-image-2 输出的位图始终无法达到 dp 级规范——行高稳定落在 24-34dp（要求 48dp）、图标家族
  必然混杂（线性+面性+不同线宽）、辅助文字必然偏小。v6/v7 在结构/色彩/层级等定性维度已合格
  （色彩克制/悬浮卡/白底红字退出/三级表面全通过）。
  根因：位图生成模型没有精确尺寸概念，prompt 里写 48dp/56dp 它按"视觉比例感"渲染；参考图
  重绘（edit 端点）同样不能保尺寸。
  规则：①设计稿验收分两层——定性层（结构/色彩/层级/组件形态）用识图判定，dp 层（行高/间距/字号）
  不作为位图验收门槛，由 spec.md 承担精确值；②像素级还原用实现后 --diff 验收，不要求位图达标；
  ③单屏参考图优于多屏拼图（拼图稀释风格信号，v6 教训）。
  再犯标记：行高不足——v2/v4/v5/v6/v7 共 5 次，确认为模型能力边界，不再重试dp级修复。

### D4. 消息中心 SegmentBar 5 tab 溢出（2025-08-21 真机回归）
- **触发**：消息中心页 redesign 后装机回归，5 个 PillTab（全部/账单缴费/余额预警/故障公告/优惠活动）
  在 360dp 屏宽放不下，最后一个"优惠活动"被 Row 强制挤成每个字一行（竖排 4 个字）。
- **根因**：`Row(fillMaxWidth + spacedBy(8.dp))` 中 5 个 PillTab + spacing 总宽 ≈ 432dp > 360dp 屏宽，
  Compose Row 不允许子项超出可用空间，最后一个 PillTab 被压缩宽度 → Text 默认单行 → 逐字折行。
  同样写法在 `BillsPage`（3 tab 放得下）/`UsagePage`（3 tab 放得下）未触发，所以原设计稿未写横滑约定。
- **修法**：Row 外层套 `horizontalScroll(rememberScrollState())`，溢出 tab 可横滑到位，
  这是项目内尚未统一的"超 4 项胶囊必须可横滑"约定。
- **规则**（沉淀）：
  1. **新增页面用 PillTab 列表时，tab 数 × 平均文字宽 + 内边距 > 屏宽 - 32dp 边距时，
     一律加 horizontalScroll**——不要假定 Row 会自动折行或截断，Compose 默认行为是挤压导致竖排。
  2. **写设计稿 prompt 时，若 tab 数 ≥4，显式声明"分类胶囊可横向滚动"**，避免位图模型按"全在一屏"理解。
  3. **真机回归要算屏宽**——gpt-image-2 输出的设计稿画布 1024×1536，屏幕物理宽 360dp / 540dp / 1080dp 不一，
     位图看着"塞得下"不代表物理机能塞下。
- **再犯标记**：无（首犯）。

### D5. 套餐详情页"对比为空"——后端硬编码假数据(2025-08-21 真机回归)
- **触发**：用户装机看 product detail 页，发现"套餐对比"卡片显示"暂无可比套餐"EmptyState，
  设计稿承诺的 2-3 个同类对比行完全没渲染。
- **根因**：`internal/httpapi/user/trade.go::portalProductDetail` 把 `specs` 硬编码成 1 行
  (`{"label":"带宽","value":p.Bandwidth}`)、`compare` 硬编码成 `[]any{}` 空数组——前端永远拿不到对比。
  同样地 `description` 字段未返回,前端 `Notice("—")` 占位。
  **前端开发完了,但后端是空壳**——这种"前后端分头写"的盲区,前一任务只校验了编译+render 冒烟,
  没真发请求看响应。
- **修法**:把详情 endpoint 改成从 `ProductOffer` 真实字段派生 specs/description/compare,
  DB 未建模的字段(合约月数/安装费/设备/适用范围)按 category 规则文案占位。
  compare = 同 category 下最多 3 个 PUBLISHED 同类产品(排除自己)。
  新增 `TestPortal_ProductDetail` 注入 3 宽带 + 1 fusion + 1 草稿,
  验证:6 项 specs 全有值 / description 非空 / compare 2 个(排除自己+排除 fusion)/
  草稿产品 404。
- **规则**(沉淀):
  1. **写"详情/对比/列表"类页面前,必须先发真请求看后端实际响应**——不能看 schema / 看代码默认值就假设有数据。
  2. **后端硬编码空数组/空字符串/硬编码占位值,前端拿到的就是空壳**——发现 empty data 必须顺着数据链路反查后端,
     不能在前端加 fallback 文案了事(掩盖问题,用户更困惑)。
  3. **新增 endpoint 必须配套测试验证派生逻辑**——`TestPortal_ProductDetail` 这一条堵的就是"硬编码占位永远跑过编译"。
- **再犯标记**:无(首犯,新类型)。

## B. spec 完备性（提示词 / 注意事项遗漏）追加

- 2025-08-20 profile v2-v7（用户反馈）：生图 prompt 写得越细效果越差。我把 dp 数值、行高、
  每个组件的微观规则全塞进 prompt（60+ 行），生成结果死板且仍达不到数值要求。
  根因：混淆了"生成约束"和"验收标准"——dp 级规则应该用于识图验收和 spec，不该塞进生图 prompt；
  过度约束剥夺了 gpt-image-2 自身的设计能力，适得其反。
  规则：生图 prompt 只写四层——①用途与平台 ②页面元素与功能清单 ③核心风格关键词
  ④规范 token（标准色/字体，少量hex）。控制在 ~150 词以内；禁止 dp/sp 数值、禁止逐组件微观规则；
  M5/M6 清单只作验收和 spec 素材，prompt 里最多保留"可滚动/平台特征"一句话。

## C. 系列多屏一致性

- 2025-08-21 login-register-states v1（用户反馈）：三等分拼屏把注册（低频）抬到与登录（高频）同级，
  违背使用频率层级——高频路径占显眼位与主 CTA，低频操作降为文字链接。
  根因：prompt 只按"状态枚举"排屏，没按"使用频率×视觉权重"分配版面。
  规则：多状态拼屏先做频率分层——高频状态等分占屏，低频状态收敛为入口级元素
  （小字链接/次级按钮），并在 prompt 里写明其"视觉层级最低，不抢主操作注意力"。
  再犯标记：无

- 2025-08-20 realname 单屏三张（step1/2/3）：各自合格，但三张的步骤进度条流程命名和 Stepper 组件样式互不相同
  （基本信息-人脸识别-完成 / 证件上传-人脸识别 / 填写信息-身份验证-审核状态）。
  根因：分次生成时未把"统一步骤标签文案 + 复用同一 Stepper"写成硬约束，gpt-image-2 每次独立发挥。
  规则：①系列多屏稿优先"一张图多屏拼版"（realname-flow-states 一稿四屏命名完全统一，验证有效）；
  ②必须分屏生成时，prompt 逐字写死步骤标签（如 填写信息-证件上传-审核状态），并声明
  stepper component identical across screens, only current step differs。
  再犯标记：无

- 2025-08-20 profile-v8（用户反馈）：prompt 第一句要先声明场景——"设计高保真UI设计稿 + 场景
  （手机/平板/PC）"，再跟具体要求；且要明确"全屏预览稿"：整张画布就是屏幕本身，
  不要设备边框、画板标注、页面标题水印等任何额外说明。
  根因：模板没有开篇定场景，模型可能自行加画板装饰/标注文字。
  规则：prompt 固定首句结构：[高保真UI设计稿，{手机|平板|PC}端{页面名}，全屏预览——整张图即屏幕，
  无边框无标注无额外文字]，随后 Elements/Style/Tokens/Content 四层。

## D. 跨端视觉对齐

- 2025-08-21 师傅端 vs 用户端 token 分裂（用户反馈"和用户端对齐"）：
  师傅端旧规范（`docs/worker/UI-SPEC.md` v1，2025-08-20）色板/圆角/阴影/按钮与用户端
  （`designs/UI-SPEC.md`）存在 6 处分裂：
  - 主蓝 `#1677FF`（Ant Design 蓝）vs 用户端 `#086CF5`（克制科技蓝）
  - 头部渐变 `135deg` 2 段 vs 用户端 `160deg` 3 段
  - 页背景 `#F5F6F8` vs `#F6F8FA`
  - 卡片圆角 12dp vs 10dp
  - 卡片阴影 有 vs 无
  - 主按钮 绿底 `#52C41A` vs 蓝底（"接单/确认/完成=绿色"约定 vs 用户端"主操作=蓝"约定）
  根因：师傅端草稿（`docs/worker/style.css`）和 Compose（`theme/Color.kt`）早期
  沿用 Ant Design 色板，未与用户端基准对齐；两 App 视觉分裂导致品牌不统一。
  规则：①**同公司 App 必须共享同一套 brand token**（主蓝/渐变/页底/分割线/状态色组），
  业务差异（按钮文案/Tab 名/快捷入口内容）允许独立；②对齐口径记入 `docs/<portal>/UI-SPEC.md`
  §"对齐锚点"段，附"对齐核对清单"表（主蓝/渐变/页底/圆角/阴影/按钮色/分割线/Tag色组）；
  ③两 App 间 Tab 名差异保留（如师傅端 3 Tab vs 用户端 4 Tab），但 TabBar 视觉（图标尺寸/选中色/
  高度/未读气泡）必须一致；④"业务语义不同就换主色"是反模式，应靠文案+状态徽章表达，而非按钮色。
  再犯标记：无（首犯，已落到 docs/worker/UI-SPEC.md §12 对齐口径表）

## E. 后端/契约漂移(2025-08-21)
- **触发**：师傅反馈"工单详情页看不到工单详细信息,应该能看到订单信息脱敏的"。
  实地勘察发现：前端按 `api/openapi/worker/schemas.yaml::TicketDetail` 读 12 个字段
  (customerName/customerPhoneMasked/address/product/splitterPort/preBindTag/
  scheduleSlot/faultTypeLabel/reportedAt/slaLeftMinutes/remoteDiagnosis/finishedAt),
  但 `internal/httpapi/worker/ticket.go::workerTicketDetailHandler` 实际只返
  6 字段(ticketNo/bizNo/status/statusLabel/stages/quad/riskCheck),其余由
  `optString` 静默吞空值,详情页工单头塌成空骨架。
- **根因**：schema 是契约、handler 是实现,二者漂移无失败信号——前端按 schema
  读、handler 按"今天写了多少就返多少"返,中间无一致性门禁。`optString` 默认
  返回空串而非抛错,把漂移变成"看起来页面活了但啥也没有"的沉默 bug。
- **教训**：
  1. **schema 必须有代码守门**：OpenAPI/JSON-Schema 是契约,build 时若 handler 返
     出的 gin.H keys 不覆盖 schema 必填字段,应当告警(可写 codegen
     校验或 lint);短期无工具时,改 handler 前必先 grep `schemas.yaml` 看
     该接口 schema 完整字段列表。
  2. **前端 `optString` 把漂移变沉默**：`JSONObject.optString(key)` 默认空串、
     `optInt(key, -1)` 默认 -1,前端读空时常无感("字段不存在"和"字段为空"
     同形)。应当在 frontend review 检查"读到的字段如果为空、是否影响关键
     UI 区块",关键 UI 字段空时应有占位/告警,而非"什么都不显示"。
  3. **同一接口的 list/detail 必须共享读模型根**：`ListTicketItems` 已联
     customers/addresses/orders,`workerTicketDetailHandler` 却只读
     `dispatch_tickets` + `Track`,二者各走各的——这种"列表读了、详情没读"
     的不对称是漂移的温床。规则:同一资源 list/detail 必须共用同一套
     联表 SQL(可以分两方法但底层 join 一致)。
- **修法**(2025-08-21 commit 81dfed1)：
  - `TicketItem` 增 `CustomerPhone/OfferName/FinishedAt` 有数据支撑字段 +
    7 个 OpenAPI 预留位(空值,前端按"空不渲染"处理);
  - 新增 `GetTicketItemByNo`,共用 `ListTicketItems` 联表根;
  - handler 出门走 `httpx.MaskPhone` 脱敏;
  - `portalTicketOf` 同步填 product/customerPhoneMasked,与详情对齐;
  - `pg_sub_test` 增 2 个用例锁联表列序 + ErrOrderNotFound 行为。
- **未修**(本任务范围外,留待后续单提交):`internal/httpapi/admin`
  的 `fakeDispatchOrder` stub 缺 `ClaimDispatchTicket`,导致
  dashboard_test/dispatch_test 编译失败——本次提交前已 broken,
  用 `git stash` 验证过非本修改引入,不在本 fix 范围。
- **再犯标记**：1 次。规则写入 `references/red-lines.md`：
  改 handler 前必 grep schema,同一资源 list/detail 必须共用联表根。

### E2. 7 字段实装(2025-08-21 commit 99e495c)
- **上轮**(81dfed1):7 字段从"handler 不返"修成"返空串/0",骨架不塌;
  **本轮**:从"空串"变成"真实数据"。
- **Migration 000087**:
  - `complaints` 加 `created_at`(DEFAULT now())/`remote_diagnosis`/`sla_deadline`;
  - `dispatch_tickets` 加 `schedule_slot`/`splitter_port`/`pre_bind_tag`;
  - DB 验证通过:`\d complaints` + `\d dispatch_tickets` 确认 6 列全到位。
- **JOIN 改造**:`ListTicketItems`/`GetTicketItemByNo` SQL LEFT JOIN complaints(cmp),
  取 `cmp.type`/`created_at`/`remote_diagnosis`/`sla_deadline`;
  `FaultTypeLabel` 由 Go 侧 `complaintTypeLabels` map 映射;
  `SlaLeftMinutes` 由 `computeSlaLeft()` 实时计算。
- **环节写入**:
  - 环节5 `ApplyTag`:端口预占后查 `ports+resources` 取 `port_code`+`resource.code`,
    拼接 `splitterPort` 写入 `dispatch_tickets.splitter_port`(新增逻辑);
  - `pre_bind_tag`/`schedule_slot`:暂空,需上游调度/仓库系统提供数据源。
- **新契约文档**:`docs/contract/complaint-type-map.md` 定义 complaints.type → 故障类型
  中文标签 + SLA 小时数映射表,新增 complaints.type 值时必回写。
- **冒烟**:102 部署后 bossctl 验证 13 字段全部有值(新装工单 complaint 字段空=正确,
  无 complaints 关联;splitterPort 空=存量工单非 ApplyTag 流程创建)。
- **102 修障**:部署发现 server 卡在 migration 000086(`uq_quad_links_customer` UNIQUE
  无法创建),根因是 quad_links 存在 customer_id=213 的两行重复(UNLINKED stale +
  LINKED current),手动 DELETE stale 行后重启成功。规则:CI 构建失败时,
  先查 docker logs 看哪条迁移卡住,再查对应表数据是否满足约束。

### F. 官网首页 redesign 落地(2025-08-21 完成)
- 设计稿：`designs/landing-page-v1.png`（PC admin 1536×1024 横版）+ `designs/landing-page-v1.spec.md`
- 风格：科技商务风，深藏青 `#0F1E3B` + 金色 `#D5A63A` 主调，浅蓝渐变 Hero 区配内联 SVG 网络装饰
- 7 个区块：sticky 顶栏（Logo+3 锚点+语言/主题切换+CTA）→ Hero（双栏）→ 数据条（3 KPI 金色）→ 核心能力（6 卡 3×2）→ 客户成功实践（3 卡绿色语义图标+metric）→ CTA 横幅（皇冠+深底）→ 深色页脚
- 实现拆分：home/index.tsx (68 行聚合) + sections.tsx (230 行 6 个子区块) + icons.tsx (33 行) + HeroNetworkDecoration.tsx (49 行 SVG 装饰)
- i18n 三语同步新增 14 字段：navSolutions/bookDemo/bookExclusive/learnMore/heroLine2/casesTitle/casesSubtitle/cases/ctaBannerTitle/ctaBannerSubtitle/viewDetail/footerCopyright + 重命名 3 个旧字段
- 新增 7 个 SVG 图标（order/asset/network/twin/5g/refine/ops），全部 24 viewBox/stroke 1.8/round
- Hero 标题修复：`grid-cols-[1.2fr_1fr]` + `whitespace-nowrap` 双 span 解决 mid-character 折行（"营"单独成行）
- 落地路径 2c34af5（被并行会话的混合 commit 携带，非独立提交）
- **教训沉淀**：
  1. 共享工作区并行会话 `git add -A` 会把所有人的未提交改动卷入同一 commit（landing-page redesign 与附件管理器/empty state refactor 混在 2c34af5 共 82 文件）。修法：开工前 `git status` 划定边界、提交前用 pathspec 限制 `git add <明确清单>`，被卷入时**总结里点名告知用户**而非默默接受。
  2. 共享 i18n 文件被并行会话争抢时出现"自己的 key 丢失"：执行 `git checkout HEAD -- file` + 重应用后必须 `git diff --stat` 验证自己加的 key 仍在；本次经历 3 次往返才稳住。
  3. Hero 类大标题若用 `<br/>` 强制换行 + 2 列网格，列宽不够时第一个 span 内的中文字符会单独折行。修法：`whitespace-nowrap` 包每个 span + 主网格调成 `[1.2fr_1fr]` 给文字更多宽度；通用规则：中文标题强制不折行要显式 `whitespace-nowrap`。
  4. 位图模型没有精确尺寸概念——设计稿里 48px 标题在 2 列网格的左列里必然溢出。spec 必须写明"hero 标题在 [md+] 用 whitespace-nowrap + 列比例 ≥ 1.2"，实现才能落实。

### A. 附件管理器 9 类文件识别 + 彩色徽章(2025-08-21 完成)
- 设计稿：`designs/attachment-manager-v1.png`（PC admin 1536×1024 横版）
- 核心改进：原 AttachmentManager 仅展示 contentType 文本，行内无视觉区分。新版本按 MIME + 扩展名归 9 类（image/audio/video/pdf/document/spreadsheet/archive/code/other），行首 28-32px 圆角浅底彩色徽章 + 左侧分类侧栏 + 顶部类型下拉 + 列表/网格视图切换。
- 颜色写入 `tokens.css` 的 `.afti-*` 类（[data-theme] 双套），避免 JS 读主题。
- 客户端按类型过滤（后端契约不动），badge 显示当前页内分布；分页器在过滤后重算 total。
- 双主题截图验证（亮/暗各一张）：对比度足够，色彩克制。
- i18n 三语同步新增 fileCategory/filterByType/filterScopeHint/downloadSelected/download/categoryBadgeTip。
- 单测 126 通过：classifyAttachment 覆盖 9 类 + 边界（MIME/扩展名/中文名/URL 带 query/.tar.gz 复合扩展）。
- 落地路径 2c34af5（被并行会话合并提交到 mixed commit）。
- **教训沉淀**：
  1. 共享工作区并行 agent 可能把"自己的文件"和"对方正在改的文件"合并提交（rule 79）。本次 AttachmentManager 重设计 + table empty states refactor + 官网首页 redesign 全混在 2c34af5 一次提交（82 files,1530+ 346-），违反"一次提交=一个可独立陈述的变更"原则。修法：开工前 `git status` 确认彼此边界，发现被卷进非自己任务范围的 commit 时**在总结里点名告知用户**，不默默接受。
  2. 共享 i18n 文件（types.ts + 三份 locale）被多个并行会话争抢时，会出现"自己写的 key 丢失"现象（typeScript 被 reset 后我自己加的 key 没了），必须**最后 git diff 验证自己的 key 仍存在**，否则从头补一次。
  3. `.afti-*` CSS 变量按 [data-theme] 双套是简单可靠的多主题配色方案：无需 JS 读 `data-theme`，只需 CSS 选择器覆盖；组件代码零分支。
  4. 网格/列表视图的 toggle 按钮 aria-label 必须用人类可读文本（"grid"），不能拼字符串（`'文件名 list'`）——前者稳定，后者对前端可控性差。
