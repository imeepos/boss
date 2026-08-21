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
