# Notes

> 2026-08-24 盘点压缩：原 919 行逐任务反思已去重提炼。
> 唯一性经验归入下方"跨任务提炼"；逐任务细节已由 references/(lessons/known-issues/red-lines/techniques + knowledge 索引)承接。
> 之后仍按 SKILL.md 流程：一次任务一段，只增不改；累计 5 轮以上再做一次去重盘点。

## 跨任务提炼（按复发频次排序）

### 累犯TOP（5次以上）
1. **并行会话/僵尸 subagent 共享工作区**（≥8 次）：开工先 `git status -uall` 划界；提交用显式 pathspec 或先 `git diff --cached --stat` 核对；验证通过立即 commit（提交是防并行走失的唯一硬保障）；commit 失败/空提交先 `git log --oneline -5` 查是否被并行吞并；别人的 WIP 不碰不代提交；pull 后先 build 确认基线绿。僵尸 subagent 会中断后仍写盘/擅自 commit，interrupt + send_message 强制检查点，超 ~3 轮无落盘就打断。
2. **PATH/JAVA_HOME 环境事实**（≥6 次）：brew 工具（go/docker/graphviz/pdftotext/lsof）全在 /opt/homebrew/bin（lsof 用 /usr/sbin/lsof）；每个 bash 调用先 export PATH。gradle 报"无 Java Runtime"先查 `~/.gradle/daemon/*/daemon-*.out.log` 的 javaHome=（openjdk@17 在 Cellar 下，/usr/libexec/java_home 是 stub 会误导）。
3. **edit/read 红线**（≥5 次）：edit 前必须 read 工具（bash cat/sed 输出不算观察）；外部/并行改动后重 read；old_string 与 new_string 范围严格对称（不顺手增删行）；tab 缩进敏感，拼前确认层级；shell 批量改完立即 read 再 edit。
4. **未验证不声称已验证**（≥5 次）：总结里"已适配/已验证"必须有对应动作支撑；grep BUILD SUCCESSFUL 再 adb install（build 失败链上 install 仍报 Success）；adb devices -l 核对 serial 是用户手里的实体机；门禁输出计数为 0 也显示 OK 是假阴性（contract-sync 扫旧目录）；healthz ok 只证明"有容器活着"；部署生效用 docker exec strings 二进制符号或镜像 tag 验证。
5. **任务完成 = 门禁通过 + 已提交**（≥4 次）：收尾门禁 = typecheck/test/build + commit + `git status --short` 干净；git status 有非本任务文件不代提交。用户明确说"不要 commit"时让位于用户指令。

### 后端 Go / PG（复发 2-4 次）
- pgx：nullable 列 SELECT 必须 COALESCE 包裹才能 Scan 到值类型；占位符从 $1 连续编号（42P18 先查编号）；参数个数不匹配报 unused argument；简单协议 []byte→jsonb 会 22P02，5xx 要 ssh 看 boss-server 日志实锤 SQLSTATE。
- 迁移：新增前 `ls migrations/*.up.sql | tail -5` 拿真实最大编号（不信截断列表，grep 代码注释交叉验证）；BEGIN/COMMIT 必须成对（漏 COMMIT = "已记录未生效"脏状态，核实后 delete schema_migrations 记录重放）；新文件进 Docker 镜像前 chmod 644（write 工具默认 600，容器 app 用户读不了崩循环）；403 先比 schema_migrations 与 migrations/ 目录（sysadmin 无隐式全权，漏迁移是首要嫌疑）。
- 测试：给 Go 接口加方法同 commit 补全所有 fake 桩（go vet 报错清单即桩清单）；t.Cleanup 与 defer 不混用（LIFO）；清理目标按命名模式匹配不信精确后缀；签 token 必须在 t.Setenv 密钥之后；验证闭环 = 真库跑测试 + psql 按前缀计数残留为 0。
- 42200 一律先读后端请求 struct（整数 id 不传字符串、字段类型不猜）；置备类接口 FK 用前面创建响应返回的真实 id，不凭 fixture 猜；双表关联语义先 grep 两端写入代码确认谁写关联字段。
- 环境：测试服务器只有 102（192.168.0.102:28080），提交后 CI 自动部署，不本机起服务；真库 DSN 在 configs/config.example.yaml（25432），本机无 psql 用 /tmp 临时 go + pgx 直连。

### 前端 web/admin（复发 2-4 次）
- i18n 是类型闭环：加 key 同步 types.ts + zh-CN/en-US/ms-MY 三份 locale，改完 tsc --noEmit；三份 locale 可脚本一次改。
- CSS 令牌：引用每个 var(--x) 前 grep 确认已定义（引用不存在的令牌会静默 fallback 白底）；新组件自带 [data-theme] 双令牌块；颜色问题用 getComputedStyle 程序化断言。
- 布局静默失败：动 width/padding/flex 后必须目视/截图验证（tsc 零覆盖）；首次写全局样式就上 box-sizing:border-box；撑满父级不写 width:100%+padding。button/input/select 不继承 color，深色表面显式写。
- 禁原生 select 做枚举切换器（图标按钮 + 自定义下拉浮层）；图标一律描边 SVG，禁文字字形。
- URL 状态双轨：URL 只做初始值，事件改本地 state + setter 双写 URL，URL 后续变化不回灌；用 useSearchParams + {replace:true} 不造轮子。
- React.lazy 对 named export 必须映射 { default: module.X }，先跑 typecheck。
- 页面任务先输出信息架构再编码；"两页视觉一致"直接提组件化，不对比修补。

### Android/Compose（复发 2-4 次）
- 尾随 lambda 陷阱：自定义组件回调参数放最后、其后不放默认值参数；多插槽组件显式写 `right = {}`；新组件 grep 所有 `Cell(...) {` 确认插槽语义。
- Modifier 叠加顺序会吃掉外部归零：组件内部写死的 padding 要覆盖必须加显式参数，不靠外部 modifier。
- 验证：真机点击先 uiautomator dump 取 bounds 不猜坐标；每轮 dump 前确认 topResumedActivity 是目标页；无图像模型时用 uiautomator dump + PIL 像素采样量化断言。
- base URL 出厂默认应是最常用端口（102:28080），不是占位值；开发前 grep BOSS_BASE_URL 确认 debug/release 一致。
- 内联块注释不能机械正则替换成 //（右花括号被吞），批量改完 grep `//.*}` 复核。
- 共享真机：动 device 状态前查 `dumpsys window policy | grep secure`（PIN 锁远程无法解锁）；装完用 dumpsys lastUpdateTime 确认没被覆盖。

### CDP/浏览器验证（复发 2-3 次）
- cdp-capture 每次运行全新 profile：localStorage 不跨运行持久，必须在同一次运行里"种存储 → location.href 跳转 → eval 断言"，且从 /login 起步再 replace。
- --eval 一律包 `(async()=>{...})() `（顶层 await 报 SyntaxError）；验证结果 console.log("VERIFY:"+JSON.stringify(v))（window 状态不进 logs）。
- 免登录优先直接注入 localStorage（boss.token + boss.servers），不走登录页表单，不信未验证的辅助脚本。
- 用户报 UI 异常第一动作：硬刷新 + 确认看的是 dev 还是 102 部署（DOM 断言正常而用户见异常 = 陈旧客户端）。
- DSH GUI 路由下 CDP 看到的是宿主 DOM，不是目标 Vite 应用。

### 流程/认知（复发 2-3 次）
- 归因前先做排除验证：工具超时别急着怪网络（去管道、加 -y、curl 目标 URL、查 registry）；故障归因在总结里写"如何确定的"（命令+输出）。
- "会保存/已记录"的承诺当场 write + ls 验证。
- 大文件下载 `for + curl -C -` 循环续传，完成判据是内容可 parse 不是 exit 0。
- 环境失败（磁盘满/无 JDK）与代码失败分开判断；macOS 正则替换用 python3 re（sed 的 \b 静默忽略、& 是整个匹配）。
- 猜测用户意图第 2 次被打回就用 ask_user_question，选项覆盖不同层级维度。
- 规格文档与用户裁定冲突时用户裁定优先；含 mock 字样的规格第一步 curl 102 真实端点。
- 模型不支持 read_image 先查 ~/.dsh/settings.yaml 的 input: [text,image] 配置（可能是配置漏了）；确实不支持时用视觉模型补位或列"请人类目测"清单。
- 契约门禁/部署验证：凡输出计数先看计数是否为 0/异常小再信 OK；连续 push 会并发 deploy 撞容器名，验完再 push。
- 102 部署链路单点：docker-registry(htpasswd)与 act_runner 认证；docker prune 前圈出"仅本地标签"镜像并确认 registry 有备份，清理后立刻验证 CI 全链路。

## 任务日志（一段一行，仅存该任务唯一性结论）

- 2026-08-21 师傅端 token 失效：认证客户端统一把 HTTP/信封错误交一个回调，Compose 根路由注册后 reset 到 Login，避免各页面重复处理。
- 2026-08-21 订单预占按钮：预占按钮无显式 type 在表单里隐式提交；stage 2 预占必须先开资源核查抽屉，核查通过才可预占（未核查返 42200）。
- 2026-08-21 登录/注册/找回对齐设计稿：perl -pi 批量改后 edit 报 file changed（外部改≠已观察）。
- 2026-08-21 worker/user 体检续修：接口不接收文件却提示上传成功 = 假上传，先核对 API 是否真正接收文件字节。
- 2026-08-20 用户首页 Compose：FontWeight.Regular 不存在；保留既有单测断言的默认文案，不改变契约语义。
- 2026-08-18 web/admin shell：React 受控输入自动填充第一轮就用 native setter + input 事件。
- 2026-08-18 语言切换重构：浮层复用现有 token；补 aria-haspopup/listbox；多下拉共用一套 CSS 骨架选择器。
- 2026-08-18 顶栏系列：i18n key 漏改 types.ts 报 TS2353；侧栏激活项仅目标在可视区外才 smooth 滚动。
- 2026-08-18 登录凭证口径：401 先核对三处口令来源（账号哈希/BOSS_ADMIN_PASSWORD/冒烟默认值）；bootstrap ON CONFLICT DO NOTHING 不覆盖既有账号，历史账号直接 UPDATE 哈希。
- 2026-08-19 CORS：前端移除代理后必须同步确认服务端 CORS 中间件；白名单按"本机源不限端口"设计。
- 2026-08-18 /base/geo 403：schema_migrations.version 是 TEXT；geo 域删除 = 软删除 is_active=false。
- 2026-08-18 geo 全栈 + CI：env_file 不入库会炸无人值守 CI，固定密钥（JWT）配置一次永不轮换，绝不随机兜底；无 docker/psql 时用 sqlglot 拦 SQL 语法错。
- 2026-08-18 geo 页 Pro 化：抽屉 onClose（关）与 onChanged（刷新）必须分开；写完 JSX 通读一遍（tsc 拦不住语义废话）。
- 2026-08-18 图标补齐："缺图标"先分资产缺（ls 资源目录）还是渲染缺（grep 渲染点）；遮罩图标拷现有 SVG 规格，颜色由 currentColor 决定。
- 2026-08-18 gitea secret 被驳：内网 homelab + 用户要简单 → 直接提 app.env 入库选项说清风险让用户选，不默认上标准流程。
- 2026-08-18 geo 多语言：新组件第一版所有文案走 t.*、颜色走 token，后补成本远高于首写；多语言任务全量扫裸字符串，不按点名范围窄化。
- 2026-08-18 控件无法交互：先确认目标 app 实际运行入口，DSH GUI 的 DOM 不能证明业务页状态；弹层 z-index 要覆盖 Drawer。
- 2026-08-19 PSGC 数据：数据集先看新旧口径标志（ARMM vs BARMM）对照官方数字；迁移可回滚用"单事务 down→up 回环"验证（stripTx 后外层起事务）。
- 2026-08-18 subdivisions 500：Go 后端第一轮 grep 就限定 internal/，不搜已移除的层。
- 2026-08-19 e2e 自清理：先跑 pg_constraint 依赖图照拓扑序排语句。
- 2026-08-19 用户中心：先信息架构再编码；个人中心用设置工作区不堆卡片。
- 2026-08-18 分页下拉 4 连纠：静默失败 + 总结说没验证过的假话是最严重模式；写新组件前 grep lessons 相关关键词。
- 2026-08-19 bossctl：flag not defined 第一时间看 -h；契约 A 门禁先读 collectSpecPaths 源码确认匹配机制（$ref 行不递归子文件）。
- 2026-08-19 开网全流程：先建资产再以 boundAssetId + status BOUND 创建标签（CreateAsset 不回写 bound_asset_id）；客户凭证 = API key(subjectType=customer)，sign 完立即写 identities.json 落盘。
- 2026-08-19 Android 工程初始化：wrapper 生成失败直接 unzip gradle 发行版 jar 取 gradle-wrapper.jar。
- 2026-08-19 tailwind 重构：多会话共享仓库绝不裸 git add + commit，一律 pathspec commit。
- 2026-08-19 三端 API 前缀：跨包搬文件后方法不能定义在外部类型上，换函数后 grep 调用点复核归零。
- 2026-08-20 subagent 并行：清理 untracked 绝不用 rm -rf 目录（误删 tracked 文件），用 git clean -nd 预览。
- 2026-08-20 师傅端三连 bug：UI"理论上不可能"的行为先加 Log.d(Throwable) 插桩拿 ground truth。
- 2026-08-19 假 404：`/usr/sbin/lsof -nP -iTCP:<port> -sTCP:LISTEN` 查 IPv4/IPv6 双绑，curl 命中错进程。
- 2026-08-20 user-home：规格与用户裁定冲突用户优先；gradlew 管道 tail 吞退出码，"build 通过"是假的。
- 2026 设计稿提示词：模板加"特殊视觉元素转写"必填节；示例只给维度提示，数值必须量取（示例数字不得照抄）；模板副本从主模板同步生成，不留手填版本。
- 2025-08-20 ui-proto/gpt-image：cordis 沙箱禁 Node timers，先查沙箱 API；!!js 标签 scalar-only 逐项标记；multipart 字段名先用最小请求探（文档写 image[] 实际要 image）；自反馈机制要落到"文件 + 强制时序"才有效；视觉模型产可执行产物的关键是 system 写死输出骨架。
- 2026-08-20 真机联调：验证码 5 分钟有效期，发码前先告知、过期直接重发不排查。
- 2026-08-22 阿里云短信：E.164 归一化显式 + 前缀必须命中支持区号，裸号默认区号只对无前缀输入生效；先写归一化边界用例再写实现。
- 2026-08-21 auth-config：dev-token.mjs 之类辅助脚本会失效（API 前缀已废），直接 curl 换 token 更稳。
- 2026-08-20 「我的」页圆角：UI 空间描述歧义（"内圆角/交点"）第 2 次猜错就 ask_user_question 覆盖层级维度；clip 裁剪配不透明底色；build FAILED 后链上 install 仍报 Success。
- 2026-08 user-android JDK：找 JDK 先看 gradle daemon 日志 javaHome=。
- 2026-08-20 短信配置菜单图标：docs 免登录脚本与选择器会过期，免登录直接注入 localStorage。
- 2026-08-20 账单 tab：102 docker-registry htpasswd 无明文需新增 ci 用户；先读 Dockerfile USER 与源文件权限再触发 CI。
- 2026-08-20 docker prune 事故：镜像内迁移文件 600 + app 用户 = 崩溃循环；宿主 docker login 一次（凭据在 102:~/boss/deploy-image/dotdocker/config.json）；容器崩溃 docker run --rm --entrypoint sh 直接验镜像内权限。
- 2026-08-20 worker 页面：busyDays 是计数非日期集合，页面字段以实际 handler 返回为准，缺失字段明确回退。
- 2026-08-20 二级页面 5+1 subagent：多 agent 撞半编辑态文件等 1 分钟重试不改别人文件；跨文件模式问题（尾随 lambda）分工过细没人兜底，复查 agent 全局 grep 一次全抓。
- 2026-08-23 Stripe：handler 在 BindBody 前先查 PayGateway.Get，未配 BOSS_STRIPE_API_KEY 即 42200——先读 handler 校验顺序再怀疑请求体。
- 2026-08-23 OpenAPI 对账：门禁"0 条路由全部有契约"却 OK = 形同虚设，凡计数先看是否为 0。
- 2025-XX 覆盖率：不可达分支用包级 var 注入缝，不为覆盖率改生产代码；先可编译再谈覆盖率。
- 2026-08-21 打包安装：脚本只覆盖 user 且 PATH 无 adb，用 SDK 内 adb；两台设备时明确指定实体机 serial，pm path + lastUpdateTime 核验。
- 2026-08-21 worker 首页：Modifier.padding(top=0.dp) 传参被内部 4dp 覆盖 = no-op 假修复，改完真机确认数值不靠代码推断。
- 2026-08-21 实名认证：uiautomator 点击无效第一反应怀疑按钮 enabled=false 门控条件。
- 2026-08-21 工作台核对：数据核对用"同源交叉验证"（原始接口拉回 python 重算再对照 dashboard）。
- 2026-08-20 ODN 系列：NOT NULL DEFAULT '' 列配 NULLIF($n,'') 必触发 23502；>3 次补丁脚本就整文件重写；集成测试开头清理残留 + -count=2 验证幂等；共享 i18n 文件不能直接 add，从 HEAD 生成最小 patch 再 cached apply；gofmt -l 覆盖全部触碰包。
- 2026-04-11 ConfirmDialog：告警页真实路由是 /alarm/alarm（菜单分组前缀），先 grep menu.def.ts 拿真实 path。
- 2026-08-24 ResourcePicker：长中文 commit message 用临时文件 + git commit -F（heredoc 在 bash3.2 报 bad substitution）；i18n 键落三语言后 grep 锚点行验证落点。
- 2026-08-24 师傅接单：Setenv 放进签发 helper 首行；edit 前 cat -et 确认缩进层级。
- 2026-08-21 dev-mode 验证码：gin 同一路径双 RouterGroup 注册会 panic，单一端点 + 可选鉴权中间件分支更清晰；dev 端点 404 时客户端静默降级不提示。
- 2026-08-21 套餐详情：Composable 子组件拿 CoroutineScope 是反模式，scope 留给最近一层；色值争议用 gpt-image-analyze 自动比对。

## 2026-08-21 表结构对账与 ER 图同步

- 坑:用 bash sed 读 data-relations.md 后直接 edit,4 个编辑全被拒(第 5 次犯红线#1);排查脚本规格时凭 grep 记忆写 worker_replace_logs 的"UQ ticket_no+epc",核对 DDL 后才改掉,险些把臆造约束写进 ER 图。
- skill 有预警:红线#1 原文就写了 cat/sed 不算已读。
- 重来:凡是要 edit 的文件,一律先 read 工具;ER 规格里每条 UQ/FK 注记必须回 grep 对应 DDL 再落笔。

## 2026-08-2x importer 页重构(文件上传+预览+契约修复)
- 哪个坑浪费最多时间:V8 新版 JSON.parse 报文不再含 "position N",行号定位单测失败;另 read_image 本环境只回元数据不能目测。
- skill 有没有提前警告我:red-lines #7(不假设图像输入)命中;V8 报文格式无沉淀。
- 重来一次:先 node -e 验证目标运行时报文格式再写解析;目测类验证直接声明"请人类目测"。
- 新经验已喂:lessons #74(V8 JSON 报文)、#75(git 暂存区并行遗留按 pathspec 提交)、techniques(eval 内 location.href 导航模式)。

## 2026-08-21 全仓时间/时区审计
- 哪个坑浪费最多时间：dashboard.go 注释断言"DB 时间戳按 UTC 扫描"，与 pgx v5 实测(回扫进程本地时区)矛盾——差点照注释下结论。
- skill 有没有提前警告我：没有；时区三重巧合(会话/DSN/容器)无沉淀。
- 重来一次：涉及时区的结论一律先连库实测，不信代码注释。
- 新经验已喂：lessons #76/#77，审计报告 docs/review/time-timezone-audit.md。

## 2026-08-21 importer Excel 导入任务
- 哪个坑浪费最多时间:门禁通过后先跑长链路验证(E2E+CDP 双主题),期间并行会话把我的 8 个文件连同它自己的 notify 路由扫进同一个混合提交 f919acc;git status 突然"干净"导致一轮恐慌排查。
- skill 有没有预警:部分预警(lessons 有"git add 前查暂存区"与"并行会话覆盖未提交修改"),但没有"门禁绿后先 commit 再验证"的明确指令。
- 重来一次:门禁(typecheck/test/build)一绿立即 commit,再做 E2E/截图等耗时验证,验证发现问题的修复走第二个提交。

## 2026-08-21 E12-E16 修复轮

- 坑:凭直觉把 factSnap 当 8 列拼 $1-$11,pgxmock 单测静默通过,真库集成测试才抓到 insufficient arguments。
- skill 有部分预警:lessons #39/#57 讲过占位符编号,但没讲"const 拼列先数列数";已补 lessons 新条。
- 重来:拼含常量的 SQL 前先数列;含 SQL 的改动必须有一条真库集成验证。

## 2026-08-21 业务时区裁定与落地
- 哪个坑浪费最多时间:make check 红时一度以为是自己的回归,基线 worktree 对照后确认 16 项失败全部为并行会话遗留。
- skill 有没有提前警告我:techniques #26 只讲了编译阻塞,没讲门禁红盘的基线对照法。
- 重来一次:门禁红的第一动作就是基线对照,再决定修不修。
- 新经验已喂:techniques(基线 worktree 对照法)。

## 2026-08-21 102 boss-server 崩溃循环排查
- 哪个坑浪费最多时间:ssh 单引号里嵌 heredoc 传 SQL,docker exec 静默没执行(退出码 0 无输出),差点误判已修复;靠"验证实际状态"才发现。
- skill 有没有提前警告:红线 6(没验证动作不许声称已修)拦住了我,先查了实际分区分布才发现没执行。
- 重来一次:远程执行 SQL 一律本地写文件 → scp → docker cp → psql -f,绝不走 ssh 内嵌 heredoc。

## 2026-08-25 后台提醒中心(notify 域)全栈落地
- 哪个坑浪费最多时间:i18n 三份 locale 插 key 时,凭"pageUnit 结尾"猜段落,把 notif 块插进了 company 段而不是 message 段,三份全错,返工一轮才发现;另有 python 脚本改完文件后凭旧记忆 edit,报 file changed since read。
- skill 有没有提前警告我:有——高频红线#1 正是"edit 前必须 read",又犯了;段落锚点问题 lessons 里没有对应条目(新教训)。
- 重来一次:locale 插入前先 grep "message: {" 拿行号,用"下一段段名"做唯一锚点,不看尾部 key 形状;任何脚本改文件后立即重新 read。

## 2026-08-21 官网首页 /home 任务
- 坑:edit 的 old_string 以 `<Route` 这类高频重复片段做锚,匹配到了相邻的 ucenter 路由而非目标路由,改完才发现(红线#4 变体:锚点不唯一)。修法:多行 old_string 必须包含目标独有上下文(如 path 属性行),改完立刻重读确认。
- 坑:根路径 "/" 原本被 AuthGuard 整包住,index 子路由里的分流组件永远执行不到(未登录先被踢 /login)。守卫区改无路径布局路由 + 顶层独立 "/" 分流路由解决。
- 并行会话半成品(pages/backup)让全仓 typecheck 一度挂掉;等对方自愈后门禁通过,commit 精确 add 6 个文件避开污染。

## 2026-08-25 push-config 落地(并行 agent 共存会话)

- 最大的坑:共享工作区有另一个并行 agent 同步开发(backup 功能),我的新文件(jpush.go)被其中途重构、wiring.go 被连环改写、我的半成品被对方打包进两个巨石提交。教训:大粒度 write 后立刻 build 验证;提交前必须重新 diff 确认哪些是自己的产物;不要基于记忆断言文件内容。
- 新坑(重试已成功的 edit 导致双重插入):一次消息里发了两次同样的 edit,第一次成功第二次把 Push 装配块插了两遍,靠 grep 发现。重试前先确认上次是否已生效。
- skill 提前预警了:menu.def 新增项必须补 items/<key>.svg(docs/boss-admin-web.md 记录),这次靠它躲过无图标坑;红线#3(禁原生 select)第一次写就踩了,靠红线记忆当场改 Dropdown。
- UI 验证新姿势:AuthGuard 需要 boss.servers JSON 含 id 字段且 boss.server.active 指向该 id;vite 无代理禁用,API 直连绝对地址;DOM 断言用 --eval "JSON.stringify({path,text})" 比截图更硬。

## 2026-08-21 附件管理组件(web/admin 前后端)
- 最耗时的坑:并行 agent 共享工作区。兄弟会话先改坏 App.tsx(引用未建的 ./pages/backup)致 vite 500、又停掉 5199 dev server、最后把我的全部改动卷进它的混合提交(fe30e60/7d0d984,不可独立 revert)。
- skill 提前警告过吗:警告过并行 agent 风险,但"改动被兄弟会话代为混合提交"是新变体;cdp 首个 eval 在 about:blank 执行抛 SecurityError 的竞态 skill 未提。
- 重来一次:早 10 分钟用 git diff --name-only 快照自己的文件清单;dev server 自己起而非蹭兄弟的;cdp 用"eval1 注入+跳转合并、eval2 轮询"结构一次成功。

## 2026-08-21 修顶栏导航首次点击全页闪烁
- 最大坑:验证时只注入 token 不够,`boss.servers`/`boss.server.active` 也必须注入,否则 apiBaseUrl 走相对路径 404 → AuthGuard 登出跳登录页,断言根本没跑在目标页面上。skill 里其实写了要注入三个 key,下次照单全收。
- 自我预警:足够,累犯台账"无验证声称已修复"红线这次提前规避了(真实 DOM 断言 sameNode)。
- 重来一次:提交前 git status 发现 TopBar.tsx/OrderPage.kt 是并行会话遗留,只 add 自己的两个文件——这条已在台账,执行了。

## 2026-08-25 修顶栏用户头像下拉样式错乱
- 最大坑:`min-w-45` 是 Tailwind v4 写法,本项目 v3.4 静默丢弃,菜单塌到 71px 文字竖排;同文件 w-45/w-27 同病。CDP 断言 min-width=0px 一锤定音。
- 自我预警:lessons #12"CSS 引用不存在令牌静默 fallback"同族,但类名层面没有对应条目,已补 lessons 77。
- 重来一次:遇到"样式错乱"先量 getComputedStyle,不猜 CSS;并 grep 全仓同族类名评估范围,范围外的不混提交。

## 2026-08-21 数据备份迁移功能(backup 域)
- 最浪费时间的坑:并行会话把我进行中的半成品(还混入无关 OrderPage.kt)直接提交成 fe30e60 巨石 commit;以及 102 容器命名卷 root 属主导致服务装配 nil,两轮部署才修好。
- skill 有没有提前警告:并行会话问题有(recidivism 已登记过同源坑);Docker 卷属主坑无。
- 重来一次:开工即 git status 分辨并行改动;凡新增服务依赖可写目录,部署 compose/Dockerfile 与代码同一提交落地,并在镜像里预建目录 chown。

## 2026-08-25 顶栏下拉统一 antd 圆角卡片样式
- 最大坑:CDP 双主题验证完到 git commit 之间隔了一次 pnpm 门禁,期间并行会话把我 4 个已 stage 文件卷进它的 feat(push)/feat(worker) 提交。内容已验证且在 HEAD,但失去独立 revert 性——累犯台账该条第 2 次。
- 自我预警:lesson 74 明确预警过,这次验证环节多、窗口拉长,还是中招。改完应先 commit 再做长门禁。
- 重来一次:样式类改动验证成本高时,先 commit 一次"CID 验证过的中间态",门禁后如需修正再补 commit。

## 2026-08-21 导入中心接入附件选择器
- 最耗时的坑:并行会话两次抢提交(b31a217 卷走本轮全部改动,连"立即提交"的窗口都没抢过);CI 部署延迟 ~1 分钟,404 轮询即可,勿误判未部署。
- skill 提前警告过吗:卷提交已有 lessons 74,但"status 查完到 add 之间几秒内又被卷"仍无解——只能缩短改动到提交的时距,改完一块立刻提交。
- 重来一次:UI 断言里按钮文案先 grep locale 实际值再写正则(本次 /开始导入/ 不匹配"导入",白断言一个字段);验证样例数据必须先读目标 schema(preview.ts ADDR_FIELDS),不符合 schema 会误判为选择器故障。

## 2026-08-21 官网首页顶栏导航闪烁
- 坑:用户说"顶部导航"先入为主修了后台 AdminLayout,实际指官网首页 /home——公开页与后台壳层是两套路由,C TA 跨顶层路由首载 lazy chunk 仍会整页闪 Loading。同一根因两个发病位置。
- 重来一次:接到"XX 页面闪烁"先确认用户说的是哪个页面/哪条路由路径,再定位 Suspense 边界。

## 2026-08-21 代码规整 codereview(6 文件拆分)
- 最耗时的坑:6 个并行 subagent 全部收到 "failed" 通知,实际 3 个还在跑,把重复拆分文件写进工作区;git stash -u 时被卷进又 pop 出来,差点污染提交。
- skill 提前警告过吗:recidivism 有"并行agent把半成品卷进提交",但没有"failed 通知不可信"这一条。
- 重来一次:收到 subagent failed 通知先 list_agents 复实况;发现非预期未跟踪文件先查 mtime 与来源再删;另外 commit message 里的行数必须实测(本次 202/179 写错,rebase 补救)。
- 有效手法:拆分前先看同目录既有模式(logic.ts/styles.ts/wiring_*.go),新文件名对齐既有命名;门禁(check-contract-sync)自身就是超标受害者,拆完 C 项即绿。

## 2026-08-21 admin 包 3 文件路由 handler 拆分
- 最耗时的坑:工作区里同时有另一个并行 session 在做同包其他文件(billing/dispatch/tax/userdata_more 等)的同类拆分;中途并行 session 反复改 `billing.go`/`dispatch.go`/`userdata_more.go` 等,本轮 `git stash pop` / `mv *_handlers.go /tmp/` / `git checkout HEAD -- ...` 来回 4-5 趟,差点把别人半成品冲掉。
- skill 提前警告过吗:无。本任务与并行 session 同目录不同文件,但都操作 admin 包,go build 错误会互相掩盖。
- 重来一次:接到"只动这 3 个文件(+同包新建文件)"的任务,先 `git status --short` 备份当前 dirty state(快照一份到 /tmp),验证门禁时只把无关 WIP `mv` 到 /tmp,**不要** `git checkout HEAD --` 别人的 WIP——git restore 后并行 session 又写回来就会重新冲突;最稳的是:复制无关 WIP 文件到 /tmp 单独验证,验证完原样 `mv` 回工作区,不动 git。
- 有效手法:参考既成模式 `internal/httpapi/worker/ticket_action*.go`(扁平路由表 + 同包 *_handlers.go 拆分);具名 handler 用工厂签名 `func xxx(a *app.Application) gin.HandlerFunc` 与 W1 模板对齐;新文件 ≤300 行约束下,worker 18 个 handler 拆 3 个文件(主档/fact+event/notice+message),org 14 个 handler 1 个文件(292 行,主题连贯)。

## 2026-08-21 admin 路由 handler 批量拆分(3 subagent 并行) + 契约补登
- 最耗时的坑:三方互踩——(1)某 agent 用 git checkout/stash "清理"工作区,回滚了父会话未提交的 yaml 契约修改;(2)父会话自己用 git stash 诊断测试失败,又差点吞掉并行 agent 的半成品(agent 只能重写)。并行协作中任何一方执行改变工作区的 git 写操作都会互相毁灭。
- 重来一次:并行 agent 在场时,父会话绝不用 stash 诊断(改用临时 worktree 或等 agent 收尾);未提交的重要改动立即 commit 保护;给 agent 的禁令要写明"不得执行任何改变工作区的 git 写操作,只读 git status/diff/log 无妨"。
- 纠正:A agent 写入 red-lines.md 的"连 git status 也不许跑"过宽——真正的红线是 git 写操作(checkout/restore/stash/clean/commit/reset),只读命令无害。
- 有效手法:补契约时先查定义是否已存在于域 yaml(本次 /customers/{id}、push/device 都只是聚合 ref 缺失,2 行搞定);checker 失败项先 stash-free 地 git show HEAD: 对比判定"预存在"还是 WIP 引入(agent 的预存在判断是错的)。

## 2026-08-25 worker 端 JPush 集成(worktree 会话)

- 最大坑:gitea/main 本身编译破损(并行方提交了 10 处 FieldLabel 调用但定义文件漏 add,另有缺 import/误限定引用)。教训:worktree 基线编译失败时先 stash 自己的改动验基线,别急着怀疑自己。
- stash -u + reset --hard + 事后 pop 丢了部分改动:重放比救回快,所有小改动在对话里有据可查。
- JPush 5.7.0 AAR 的 manifest 自带 ${JPUSH_APPKEY}/${JPUSH_CHANNEL} 占位符,app 侧只要 manifestPlaceholders,自己写 meta-data 反而 merge 冲突。
- write 工具对刚被 git clean 掉的路径报 "file no longer exists":改用 bash heredoc 落盘。
- JAVA:JDK17 在 /opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home;worktree 需手动拷 local.properties。

## 2026-08-25 user/worker 域 handler 函数体拆分(worktree 会话)

- 任务:重构 internal/httpapi/{user,worker} 共 11 个文件;按"超 60 行函数"拆 + 注册函数收敛为扁平表;零行为变更。
- 用户清单里的"111x3/107x3/75x2"是函数总行数,不是单函数 >60 — 实际最长单 handler 49 行;按"handler 函数聚合视图 + helper 提取"思路统一拆到 30 行内。
- 关键技巧:
  - 同包新建 `*_handlers.go`,原文件留路由表 + 共享类型/视图辅助,跨文件共享无需 export。
  - 错误短路 + gin.H{} 的 helper 用 `(result, ok)` 双返回值;helper 失败已 respondErr,handler 一句 `if !ok { return }` 续写。
  - type alias 兜底:返回 `billing.Invoice` 实类型,绝不引入接口抽象(初版用 `invoiceLike` 接口反而破坏调用点)。
- 禁改动 git 写操作(任务硬规则)+ 共享工作区并行会话:全程 `git status --short` 自查,build 错先想"是不是并行 agent 改了 admin 域",别浪费时间排查自己的 user/worker 文件。
- 旧文件加注释 + 提取 helper 后行数膨胀超出 300 上限(trade.go 305, asset.go 320, profile.go 334):必须再拆第二轮;判断标准 = 文件末尾 `awk '/^func /' | wc -l` 与 `wc -l` 同时看。

## 2026-08-21 ISSUE.md 12环节缺口 测试复现+修复

- 哪个坑浪费最多时间:e2e 集成套件自 5c5f2a7(归属按地址推导)起就一直红(LegalEntityID 硬编码撞平台兜底主体、共享地址撞 quad_link 唯一索引、capPub nil panic、grpc seed 二插工单),先跑基线才发现红的不止我的用例,连环修了 6 处测试债。
- skill 有没有提前警告:部分——红线第 6 条"未验证不声称"促使我先跑基线;但"改测试前先确认基线颜色"没有明示。
- 重来一次:动任何集成测试前,先在干净 checkout 上跑一遍同 run 的测试,红的先分类"环境漂移/测试债/我的改动"再动手。

## 2026-08-21 ISSUE.md 第2批(rollback/渠道/dev-token)

- 哪个坑浪费最多时间:给 OrderService 接口加方法后,三处测试 fake(admin/user/worker)各自实现接口而漏补新方法,build 连环红三轮才补齐。
- 重来一次:改领域接口前先 grep 所有实现方(含 *_test.go 的 fake),一次性列全再动手。

## 2026-08-21 修复经营区域页 /regions 404
- 最大坑:registerRegionRoutes 从 org.go 拆出后漏在 registerAdminDomainRoutes 挂载,handler/路由文件都在、单测全绿(因为没测路由装配),线上恒 404。拆分路由文件时"定义-挂载"两步分离是静默失败点,回归测试必须走完整 Register 装配。
- skill 提前预警了吗:boss-admin-web.md 口令段落已纠正过 admin/admin123,本次会话首轮摘要仍引用了过期口令说法导致首次 curl 401——摘要前应再核对 skill 文档最新版本。
- 重来一次:定位 404 时第一时间 grep "路由函数名" 的调用点(不只是定义),拆分类 bug 5 分钟可定位。

## 2026-08-21 官网首页视觉重设计(worktree 流程)
- 坑:明知红线#7"不假设模型支持图像输入",仍先试了 read_image——返回元数据无画面,白费一步;应直接上 --eval DOM 断言。
- skill 有提前警告:是(红线#7),未遵守。
- 重来一次:截图后直接写 VERIFY 断言链(结构/文案/主题/报错四类),不做读图尝试。
- worktree+立即 commit+合并清理流程本次顺畅,无返工。

## 2026-08-25 官网首页页脚全幅铺满 + 语言选择框深色主题适配
- 最大坑:用户反馈"底部有边距",Footer 组件已有 px-4 md:px-[60px] 边距,需要去掉改为全幅铺满;同时 Dropdown 组件在深色背景下样式不协调。
- skill 有提前警告:是(lessons 有"CSS 引用不存在令牌静默 fallback"),但本次问题是显式设置的边距而非令牌缺失。
- 重来一次:用户说"全屏"时,检查组件是否有 padding/margin 限制;深色背景下的表单控件需要显式设置 borderColor/backgroundColor/color。
- 新经验:Footer 全幅铺满用 mx-auto max-w-7xl 居中限宽而非 px 边距;深色背景下的 Dropdown 需要特殊样式覆盖。

## 2026-08-26 remote-desktop trackC: audio + RoomList + 双浏览器 e2e

- 哪个坑浪费最多时间:playwright config 默认 baseURL/webServer.port=5173,与同机 BOSS 应用的 dev server 撞端口,所有现有 e2e 跑出来都挂在"BOSS 首页"而不是 remote-desktop,浪费一轮排查才意识到是端口串台。修正:playwright.config 改读 PW_PORT 环境变量,本地 `CI=1 PW_PORT=5291 pnpm exec playwright test` 才跑通。
- skill 有没有提前警告:有——recidivism #30「端口被陈旧进程双绑」描述了类似现象,但偏重 102 server 端的孤儿进程,没明示并行 worktree vite dev server 同端口的事故。
- 重来一次:开工第一个 bash 先 `curl http://localhost:5173/ | head -3` + `curl http://localhost:5173/src/App.tsx | grep -m1 refreshReg` 验证 baseURL 路径打的是不是目标仓库;不是立刻排查端口冲突,而不是去看截图找组件问题。
- peer.ts addLocalStream 已经天然 iterate 所有 tracks (line 76),audio 不用改 — 实现前先读现状避免重复改。
- playwright `.mjs` helper 文件不能写 TS-style `import type {Page}`;要么改名 .ts,要么用 `import('@playwright/test').Page` 在 JSDoc 里写。

## 2026-08-22 招商入驻域全链路(web/admin + Go 后端)

- 哪个坑浪费最多时间:102 部署 CI 三连坑——pnpm@latest 升级致 admin-web 镜像构建失败(任务 2287)、compose up 容器滞留 Created 误以为没部署、CORS 白名单让 token 注入直连 28080 全 404。三者都不在代码里,全靠翻 gitea actions 日志(zstd 解压)+ docker ps -a + 网络日志定位。
- skill 有没有提前警告我:部分——"对接 102/别本机起服务"避免了本地冒烟弯路;但部署流水线的 Created 态和 CORS 白名单无记录,已补 ISSUE.md + lessons 77/78。
- 重来一次:push 后第一时间看 gitea actions 日志而不是反复 curl 轮询;新增可空列时一开始就 COALESCE 全套(坑 74),httpx 错误映射随域错误同提交(坑 75)。

## 2026-08-22 内网 102:5180 经 138 公网暴露

- 哪个坑浪费最多时间:authorized_keys 选项语法——`permitremoteopen` 根本不是 authorized_keys 选项(client/Match 专用),正确的是 `permitlisten`,但该 Ubuntu sshd 8.9 对 permitlisten 报 "bad key options";最终退到 `restrict,port-forwarding`。期间 sed 占位词 TUNNELTEST 又被 sshd 当未知选项,连带排查。
- skill 有没有提前警告我:没有,references 里完全没 138/公网隧道知识;ssh config 里其实已有 public-box 线索(`~/.ssh/config` grep 138 一发命中),但没有"先查 ssh config 再问用户"的经验条目。
- 重来一次:排查 authorized_keys 拒绝直接 `sudo /usr/sbin/sshd -d -p <ufw放行的临时端口>` 抓 "bad key options" 一行定位,不盲猜选项名;选项从最小集(restrict,port-forwarding)起步再加严。

## 2026-08-22 入驻申请分步表单 + 消息中心通知

- 哪个坑浪费最多时间:CDP 冒烟三次误判"步骤跳转失败"——两次是我自造的信用码不足 18 位、一次漏填第2步必填联系人,校验全部在正确拦截;另发现 cdp-capture 每次运行全新 profile,localStorage 跨运行不共享。
- skill 有没有提前警告我:React 原生 setter 填表(坑 79)有预警,直接复用一次通过;profile 不共享无记录。
- 重来一次:冒烟数据先按校验规则自检(长度/格式逐项数一遍);需要 localStorage 状态的断言全部放同一次调用的多 --eval 里完成。

## 2026-08-2x 组织架构与人员页(/org/staff 前后端)

- 哪个坑浪费最多时间:push origin 不存在(remote 叫 gitea)与 102 口令漂移(Boss-admin-2026 → 实测 admin123)各废一轮;其余顺利——先读 skill 知识库再动手,复用 AccountForm/DeptForm/PostForm/ConfirmDialog 零返工。
- skill 有没有提前警告我:大部分有(级联下拉/禁 select/门禁流程/CDP 验证);remote 名与 102 实际口令两处事实漂移无记录,已修 docs/boss-admin-web.md。
- 重来一次:push 前先 `git remote -v`;连 102 先用 devseed 口令 admin123 试,40100 再翻 app.env。
