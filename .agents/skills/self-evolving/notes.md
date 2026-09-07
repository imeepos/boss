# Notes

## 2026-09-07 pickers 三处改造(C1 BLOCKED/C2 SimplePicker/C3 DialogPicker 首接入)

- 最耗时坑:CDP 交互断言里 option 用 el.click() 三轮无效,最后读 Dropdown 源码发现选择绑在 onMouseDown——而 techniques.md **已有这条**(CDP 断言自研 Dropdown 条目),开工前没 grep 该文件,白烧三轮采集。教训:skill 第 6 节『开工前 grep 关键词』必须真的做,场景词=组件名+『断言/click』。
- 次坑:bash 单引号包 --eval JS 参数,JS 内部单引号字符串把它截断,SyntaxError 不像代码错像工具坏;改双引号包参数(JS 内全用单引号字符串、避开 $/反引号/双引号)一次过。同红线 11 族。
- 小坑:①read 并行 3 个大文件时返回 undefined lines,改串行恢复;②glob 在大目录(api 根)30s 超时两次,bash ls/find 秒回——目录枚举优先 bash。
- C2 验证遇 102 bills 空表,学会先 curl API 证空再判『选择器坏』,避免误修;并用先例页(billing/billing,同一 Dropdown 基座+真实数据)佐证选中/回显链路。

## 2026-09-07 102 全量测试数据清理(主会话直做,无 worktree)

- 最耗时坑:四件事叠一起。①ssh+psql 内联 -c 叠引号 3 连炸(红线9a已犯14次,换「本地写 SQL 文件 + ssh stdin 管道」一次过,该模式应默认化);②pg_stat_user_tables 估算全失真(assets 估219实566、bills 估0实4),差点按估算判「空表跳过」;③DELETE 脚本手工排 FK 序三连反序回滚(invoices→bills、coupon_redemptions→coupons、procurement_receipts→asset_batches),且 del_orders 临时表建了却漏写 orders 本体 DELETE,靠复核 count 抓出;④17:16 有并行终端用 admin@192.168.0.15 导入 346 个 OWPAL/OWTAC 客户壳+资产,落在我确认范围之后——停下来问用户,确认为造数后才纳入。
- skill 有没有预警:红线9a 管了引号但没管「批量删除应从 pg_constraint 生成拓扑序」「行数盘点禁信 pg_stat」;备份-单事务-巡检门禁三件套(backup-102.sh/ON_ERROR_STOP/db-patrol-gate)全部按既有约定走,三次回滚+一次漏删都零损失,机制红利明显。
- 重来一次:大清理标准动线=count(*) 全量盘点 → pg_constraint 拉 FK 图生成拓扑序删除脚本 → backup-102.sh → 本地 SQL 文件+ssh stdin 单事务 ON_ERROR_STOP → 复核+db-patrol-gate;盘点后新出现的并行造数必须扩范围前问用户。

## 2026-09-07 P-INFRA-1 负责人立项(feat/plan-infra-buildout)

- 最耗时坑:--no-checkout worktree 半检出状态直接 commit——index 为空,git status 满屏 D 误当常态,add+commit 产出「删全仓只留 docs」的提交 eb0b6aa;幸运未推送未并 main,reset --hard main 后 cherry-pick 重做无损。同轮 ff-only 合并撞并行 kaihu 线推进 main,按协议回 worktree reset --hard main + cherry-pick 后一次过。
- skill 有没有预警:红线 9/13 管「ff 失败禁删 worktree」「长命令后台跑」,没管「--no-checkout 的 index 陷阱」;DSH 零参工具(get_goal/session_link_list)不传参报 binding arguments must be lossless JSON,须显式传 {}。均已登记 lessons/recidivism。
- 重来一次:worktree 提交前必跑 git status --porcelain 判「整树 D」;commit 后必 git diff --name-only HEAD~1 HEAD 核对范围;零参工具默念「空对象也要传」。
- 立项产物:docs/plan/infra-buildout-plan.md(缺口五项/最佳实践引用/W1-W2 派发与迁移号预分配/合并顺序);W1(承包商结算)、W2(网格投资测算)两个专属会话已派发。注意:台账 #12 裁定 archiveSession 在本环境禁用——「工作完归档会话」须向用户报备 GUI 手工归档,不得调该工具。
## 2026-09-07 PP2-U6 跨域业务流冒烟(feat/pp2-u6-smoke)

- 零踩坑轮。W4 沉淀的两条直接复用:收尾清理直接 --force 一次过(W4 的 Directory not
  empty 教训);发现清单写明证据等级。部署新鲜度验证用「构建产物特征串」(在产物 js
  里 grep 本波合并才有的代码文本,savePartial/不在 options 值域各 1 命中)比猜 hash 可靠。
- 走查载体选择:发现 /bss/onboarding 本身就是五环节一站式工作台后,流程 A 的主断言
  从「逐页走」改为「一站走通+各页在位」,操作步数维度直接给出最优证据。
- 只读纪律的边界:确认弹窗打开后一律取消,真实写操作(保存参数/停用/删任务)不触发,
  toast 路径以代码+产线用法佐证并如实登记——宁可证据弱一档,不污染业务数据。

## 2026-09-07 PP2-W4 base+辅助页 Phase A/B(feat/pp2-w4-base-aux)

- 最耗时坑:无。本轮最大收获是「审计误报的勘误机制」:Phase A 用 grep -c | head -1
  做每目录计数,只取了第一个命中文件的行数,把 address(已有 ErrorBanner)与
  realname-review(已有服务端分页)误判为缺口;Phase B 逐文件复核时两条 P2 撤销。
  grep -c 输出是多文件多行,head -1 只代表第一个文件——目录级结论必须逐文件或全量统计。
- skill 有没有预警:红线 14(键名手滑/漏 description)本轮犯了 2 次(bash 缺 description、
  description" 多引号),都是连发快节奏下发生;发车前默念必填键应成为肌肉记忆。
- 重来一次:审计类任务先写「证据采集脚本」再下结论;发现清单每条标注证据等级
  (源码行号/现场断言/推断),Phase B 勘误就有据可依。
- 额外收获:负责人中场指令引用的 scripts/accept/pp2-gate.sh 在切分支后进的 main,
  worktree 里没有——先 merge main 反向同步再找基建,是并行波次的常规操作。

## 2026-09-07 PP2-W0 选择器基座(U0,feat/pp2-w0-pickers)

- 最耗时坑:CDP 断言用 document.querySelector('aside') 当抽屉锚点,但页面有导航 aside + 抽屉 aside 两个,查错子树导致「错误行不存在/面板没打开」的假象,连烧 4 轮才用全文档 [role=alert] 计数定性;同段 discovered Esc 冒泡被 Radix Drawer dismiss 层接走、整只抽屉被关——一个真 UX bug 藏在假象后面。
- skill 有没有预警:红线 22(反斜杠转义引号)预警了 printf 方案会炸,改 write 工具落提交信息文件一次过;红线 11(三引号/裸反引号)在 pickerCore 追加时仍手滑写了一次 python 三引号,parse error 立刻定位但白耗一轮——起草含代码体字符串时,先扫一遍内容里有没有反引号/${/三引号再发车。
- 重来一次:走查断言脚本第一版就「role+aria-label 唯一锚定」,绝不用裸标签;fetch 拦截造障要在目标请求发生之前装好(先造障再开面板);vitest 全量挂在资源上时,先隔离复跑 + main 基线对照再定性,不急着改代码。
- 额外收获:走查不只是验收——Esc 冒泡缺陷就是走查断言「expandedAfterEsc 应为 false」揪出来的,断言脚本写成契约的机械翻译最值钱。

## 2026-09-07 报告中心(intel/report)打磨轮

- 最耗时坑:vite dev/preview 在新 worktree cwd 静默挂起(零输出不绑端口),烧了约 15 分钟才绕道主树 cwd 起静态服务器;skill 无预警,已记 known-issues。
- skill 有没有预警:红线 4(edit 不对称)与红线 14(缺 new_string)都预警了但仍然犯——两次都发生在赶进度连发编辑的节奏里;必填键默念应在起草时执行,不是发车时。
- 重来一次:开工先批量 read 全部待改 worktree 文件,编辑全部发车后再统一跑门禁;CDP 驱动 Dropdown 先读组件源码确认事件绑定(onMouseDown),能省两轮试错。
- 额外收获:测试断言失败揪出一个真 a11y 缺陷(aria-label 撞名)——自动化验证与可达性共用同一份「唯一语义标签」前提。

## 2026-09-06 Interim 累计口径修正(AAA-A4,feat/aaa-a4-interim)

- 哪个坑浪费了最多时间?三次自伤全是红线 4 变体:重写 import 块漏抄既有 fmt 行(vet 红)、接口注释第二行漏 tab(gofmt 红)、修 tab 时又给原本有 tab 的第一行多加一个(二次红)。均在一次 gofmt/vet 验证轮内定位,约 3 分钟,零扩散。
- skill 有没有提前警告?红线 4(对称性+改完立刻验证)与红线 1(编辑前先读)全程应验,按约定验证所以当场逮住;红线 11/14 的行数组+charCode 构造一次过,无炸程序。
- 重来一次怎么做?imports/注释这类局部重写,new_string 逐行与旧块对齐:既有行只能原样保留或原样搬移,新增行先想好缩进层级;改完第一动作 gofmt -l + go vet(本轮顺序正确,未带病提交)。
- 本轮正解沉淀:Interim 覆盖语义用单条 SQL CASE 取大 + RETURNING 已存值驱动告警,SQL 提常量供测试 QuoteMeta 精确匹配,实现与断言零漂移;pgxmock 层做『精确值断言』的标准姿势。

## 2026-09-06 采购单详情白屏(采购域,fix/purchase-detail-white-screen)

- 哪个坑浪费了最多时间?线上部署验证:轮询用「与上次 hash 不同」当信号,撞上并行部署 hash 翻转(Dv1VgrCR 与 DCXW1E89 互相换位),10 秒假阳性,误在旧包上白验一轮还把旧包崩溃误判成「修复无效」。
- skill 有没有提前警告?红线 14(发车前必检必填键)已预警,本轮仍漏一次 bash description 整程序 rejected;红线 7(无图模型)在案,仍在 Promise.all 批里发 read_image。两处台账如实 +1。
- 重来一次怎么做?①部署完成判定=排除全部已知旧 hash + 新 hash 上行为断言(点详情)双确认,hash 相等性永远不可靠;②发车前逐调用默念必填键(edit: old+new,bash: command+description);③本模型截图一律 cdp --eval DOM 断言,read_image 不进任何批。

## 2026-09-06 类型归一+巡检扩展(P4-B,feat/p4-b)

- 哪个坑浪费了最多时间?(1)e2e 脚本 cleanup_data 的 echo 进 stdout,seed_data 被命令替换捕获后 batchId 变成多行脏值,造数直奔服务端自动编码全漏 cleanup 前缀——两坑叠加一次跑出真实造数泄漏(102 上四个孤儿资产+FK 卡批次删除),手工清场后才修;教训:e2e 造数必须显式带业务侧唯一编码前缀,不能指望回收模式兜住服务端生成编码。(2)main 在会话中途前进(P4-A 合入),make check 与 merge 并发差点互踩——先 job_kill 再合并,合并后重跑全套。
- skill 有没有提前警告?红线 11(${ 未转义炸模板串,改行数组 join 一次过)、红线 1(file changed since read,重读即过)、红线 14(description 键手滑带引号整程序 parse error 两次)全部应验;响应码口径 notes 里已有「Respond 恒 200」教训,但本轮裁定走 respondBadRequest 字面 400 形态(与 sort 白名单同型),两形态并存,断言前先 grep 确认目标路径真实形态。
- 重来一次怎么做?①并行共享资源(102 DB)的造数/清理脚本,首次实跑后必立刻 SQL 直查残留再继续;②分支开工后定期 git fetch 对比 main 前进,发现被超尽早反向同步;③对「写路径校验」类需求,先查测试夹具里已有的方言取值(本轮 OLT 藏在 pg_model_test.go),白名单少一个合法值=红一片存量测试。

## 2026-09-06 报废三要素确认(P3-F,feat/p3-f)

- 哪个坑浪费了最多时间?run_code 单引号 JS 串内联 bash JSON 体,转义引号被 JS 解码成裸引号,e2e 脚本 14 行 JSON 载荷全 mangled——bash -n 对引号重排照样过,靠 sed 抽查才逮住(累犯 #24)。另 e2e 造数编码 A-ACC2- 逃离 cleanup 的 A-ACC-% 回收模式,残留四类在 102 实测复现(累犯 #25),手工清场+改尾缀重跑闭环。
- skill 有没有提前警告?红线 1(pg.go sed 后 edit 被拒,台账 #1 +1)、11 变体再次应验;renderToStaticMarkup 下 Radix Dialog 渲染空、useT 必须在 LocaleProvider 内层组件调,属前端测试新坑,组件拆出 ScrapRefBlock/ScrapFields 导出直测一次过;TZ=Asia/Shanghai 假红已在台账,直接显式带 TZ 零排查成本。
- 重来一次怎么做?①bash 脚本含 JSON 的行,写入后必 grep 转义符字面,bash -n 不算验证;②新增造数类别先核对 acceptance-cleanup.sh 的 DELETE 模式清单,编码用模式内前缀+尾缀;③React 组件测试优先拆内层纯组件导出直测,Dialog 壳只测挂载;④本库 Respond 恒 HTTP 200,422 语义=业务码 42200,e2e 负例断业务码。

> 2026-08-24 盘点压缩：原 919 行逐任务反思已去重提炼。

## 2026-09-06 装机联动端到端实测(P2-T3)

- 哪个坑浪费了最多时间?run_code 内联内容三连:(1)bash 行内含单引号塞 JS 单引号串被内容 ' 截断(L2 行漏闭合逗号);(2)tools.edit 多行 new_string 用双引号串塞裸换行,Expected ',' got '#';(3)edit 后用预计算索引连续 splice,前一步改了尺寸后索引错位,文件碎片化到不可修补,只能整体重写。前两条并累犯 #11(13 次),第三条新登记。
- skill 有没有提前警告?红线 11 已在案但只写了反引号/${,没写'内容单引号'与'裸换行'两个变体,本次补全;13 行两段式 worktree/后台长命令全部命中规避。环境类坑无预警:并行会话在验收跑分中途重部署 boss-server(容器 Up 1 minute),S7/patrol 空响应全由它起——验收前后应查 docker ps uptime。
- 重来一次怎么做?大文件编辑一律:整体重写 or 每编辑一步重读重算锚点,绝不批量预计算索引;JS 串构造 bash 内容统一用行数组+join,含单引号的行先转义。环境发现三件(bindTagEvent json bug/TL1 端点漂移/addresses.label 拒连字符)当天进 known-issues+ISSUE.md。
## 2026-09-05 资产台账 admin CRUD(P2-W1-T1)

- 哪个坑浪费了最多时间?run_code 引号边界两连:(1)Go 源码行内双引号("+g.table+")放进 JS 双引号串,内容引号截断串边界报 g is not defined,排查一轮;(2)多行 bash(含 heredoc commit message)塞 JS 双引号串直接 Expected ',' got ident——多行命令必须模板体,含双引号行必须 JS 单引号串。已并累犯 #11。
- skill 有没有提前警告?红线 11/13/14 都在案(两段式 worktree/后台长命令/发车前参数自检全部命中规避);pgxmock「无 WithArgs 即期望 0 参」是新坑,锁行查询漏 WithArgs 八个测试连红,pgxmock 报错文案 expected 0 but got 1 arguments 指向清晰,一轮修。
- 重来一次会怎么做?①openapi yaml 内联 flow map 的 description 先想特殊字符({} , :)该不该引号,写前查既有行风格;②域测试先跑单包再进 make check,别让全量门禁当第一道试金石;③本次 worktree 两段式+后台 reset+兄弟目录核对全部一次过,流程红线吃透了。
## 2026-09-06 资产/标签成熟方案调研+数据质量闸门

## 2026-09-06 P1 波次(联动/回收/字典/巡检)+ 部署排障

- 哪个坑浪费了最多时间?两块:(1)生成含 Go struct tag 的文件时裸反引号/转义连环炸(本轮 parse error 三连,recidivism 11 再+1)——正解是 ` 占位 + U() 替换器,严禁内联拼接;(2)CI 部署静默停摆排障耗 40 分钟——runner 日志无错误行、真相在 gitea 库 action_task.log_filename 指向的 zstd 日志,且 boss 与 gitea 是两个 postgres 实例别连错;根因=deploy-runner 镜像被 prune,凭据/二进制重建即失。
- skill 有没有提前警告?红线 9a/11/14 都在案且本轮各中一次(heredoc 叠引号/裸反引号/内层漏 description)——说明「知道」不等于「发车前自检」,复杂生成物应先用 |占位|+替换器成型再发车。
- 重来一次会怎么做?①凡整文件生成,先列操作清单并全部 ` 化;②pgxmock 锚点用无特殊字符短片段(已沉淀 techniques);③CI 静默失败先查 action_task 日志而非反复空推;④部署排障中误伤 postgres 连接池时,重启容器是最短恢复路径(有卷无损,服务自愈)。

- 哪个坑浪费了最多时间?两处小坑各费一轮:(1)对 worktree 副本直接 write 被拒——主树读过不等于 worktree 读过,read/write 的已观察状态都按绝对路径跟踪;(2)102 迁移 watcher 在 ssh 外层命令里叠引号当场语法错 exit 2(红线 9a 第 11 次)。主链路(查库实证→差距矩阵→迁移真库预演→事务化→11 处 mock 同步→合并推送)一次通过。
- skill 有没有提前警告?都有:红线 1 与 9a 均在案,属于「知道但执行时图省事」。红线 14 的发车前必填键自检应扩展为「引号/heredoc 形态」自检。
- 重来一次会怎么做?worktree 轮开工先列「文件操作清单」,对每个将 write/edit 的 worktree 绝对路径一次性批量 read;远端 SQL 先在 heredoc 里成型、验证语法,再嵌入轮询/验收脚本。
> 唯一性经验归入下方"跨任务提炼"；逐任务细节已由 references/(lessons/known-issues/red-lines/techniques + knowledge 索引)承接。
> 之后仍按 SKILL.md 流程：一次任务一段，只增不改；累计 5 轮以上再做一次去重盘点。

## 2026-08-25 实时位置上报闭环

- 哪个坑浪费了最多时间？迁移号只检查了当前可见分支，门禁才发现 `feat/odn-gis-link` 已占用 000142/000143；随后按跨未合并分支让号到 000144，并重新跑门禁。
- 这个 skill 有没有提前警告我？有，AGENTS.md 与后端经验明确要求逐分支检查迁移号；首次检查命令错误地只覆盖了部分 refs，说明必须直接用 `git for-each-ref` + `git ls-tree` 并验证门禁。
- 重来一次我会怎么做？创建迁移前先 `git fetch --prune`，遍历本地和远端所有 refs 的 migrations 文件，再创建；Android 构建前先检查 Gradle wrapper 缓存/网络，失败时明确记录未验证而不声称 APK 构建成功。

## 2026-09-03 后台批量导入页面审计

- 哪个坑浪费了最多时间？主分支在 feature 审计期间被并行会话推进，首次 `ff-only` 合并失败；按协议回到 feature 同步 `gitea/main` 后再推送并快进合并，未丢提交。另一个实际环境坑是默认 `go` 不在 PATH，改用 `/opt/homebrew/bin/go` 后测试通过。
- 这个 skill 有没有提前警告我？有：worktree 合并失败不得删除、必须在 feature 侧同步主分支；环境工具链必须先检查。前端权限/追踪修复也遵循先 read 再 edit，门禁通过后分批 commit。
- 重来一次我会怎么做？开工时同时记录远端 main 基线并在合并前主动 fetch/merge，避免先尝试必然失败的 ff-only；Go/Pnpm 门禁统一使用绝对工具路径或显式 PATH。审计报告应把“普通导入能力”与“逐行调用创建端点”严格区分，并把所有菜单路径和未部署状态一并列出。

## 2026-09-03 导入幂等项目与 provision jsonb 二次根因

- 哪个坑浪费了最多时间？模板创建 500 的根因链条两层：先以为只是缺 `::jsonb` cast，实际 102 部署后仍报 `SQLSTATE 22P02`——pgx v5 对 `[]byte` 参数默认按 **bytea** 编码，即使 SQL 有 `$4::jsonb` cast 也失败；必须传 `string`（text 编码，`text::jsonb` 合法）。这与 import_task.go 中 `$7::jsonb + string` 的成功先例一致，但首版只改了 cast 没改参数类型。
- 这个 skill 有没有提前警告我？有：真实验证优先、失败命令要查真因、pgx 参数类型要按真库行为判断；但"pgx 按 bytea 编码 []byte + jsonb cast"这个具体行为之前没沉淀，且 pgxmock 无法模拟真实类型编码（只在 102 暴露）。
- 重来一次我会怎么做？jsonb 参数一律传 `string` 并参考同仓已验证的 import_task 先例；首版修复后就立即在真实环境复验而不是只跑单测；部署验证用轮询镜像 sha 判断最新代码上线，而不是反复猜。

## 2026-08-30 内存态业务数据审计与审计同步落库

- 哪个坑浪费了最多时间？首次运行 Go 测试时默认 PATH 没有 go，实际 go/gofmt 在 `/opt/homebrew/bin`；另外主分支被并行提交推进，首次 ff-only 合并失败。
- 这个 skill 有没有提前警告我？有：工具链必须先检查、ff-only 失败必须回 feature 同步 main，不能删除工作树；本次按协议同步后成功合并并清理。
- 重来一次我会怎么做？开工时固定 `PATH=/opt/homebrew/bin:$PATH`，并在第一次合并前先 fetch/核对远端 main。审计先区分生产 PG 装配与测试内存替身，再只修复会造成业务留痕丢失的生产异步队列；限流桶和渠道注册表属于架构约束，先显式记录其单实例/启动期边界，避免把瞬态状态误改成伪持久化。

## 2026-09-03 批量导入第二轮可靠性修复

- 哪个坑浪费了最多时间？幂等键初版把 legacy 空键写成空字符串，触发部分唯一索引，后续通过真实 SQL 语义检查发现必须转换为 NULL；另有一次在仓库根目录运行 pnpm 导致误判，改到 web/admin 后恢复。
- 这个 skill 有没有提前警告我？有：先读文件再 edit、失败命令要核查真实原因、工具路径需先确认；但“可选幂等键+唯一索引”还应在设计时显式验证空值语义和 legacy 调用兼容性。
- 重来一次我会怎么做？迁移设计同时写入 NULL/非 NULL 两条 pgxmock 回归，先跑定向测试再进行全量门禁；每个命令固定正确 workdir 和绝对工具路径，合并前先同步远端 main。


## 2026-08-26 Gitea runner 并发运维

- 哪个坑浪费了最多时间？没有真正代码坑；主要风险是不能在现有部署任务运行时重启 runner。先检查任务容器，等待 task 3017 完成后才修改 capacity 并重启，避免中断线上部署。
- 这个 skill 有没有提前警告我？有：实施经验要求先查 102 实际部署副本，不能只改仓库文件；红线要求部署验证不能只看 healthz。实际用 runner 日志、任务容器和镜像/容器状态确认了安全窗口与并行效果。
- 重来一次我会怎么做？先记录活动 task 和 runner config，再做可回滚备份；对会操作共享 Docker 的 deploy workflow 先加 concurrency，再在无活动任务窗口重启 runner，最后用两个不同 label 的实际任务确认 capacity=2。
## 2026-09-03 102 验收共享锁

- 哪个坑浪费了最多时间？首次读取把 TL1 脚本误认为在 scripts/ops/ 下，实际路径是 scripts/verify-tl1-e2e.sh；通过 glob 复核后改正。
- 这个 skill 有没有提前警告我？有：真实路径必须用 glob/read 查证、worktree 命令必须显式指定目录；本次按 worktree 开发并核对分支后提交。
- 重来一次我会怎么做？开工先用 glob 精确列出三个目标脚本，再设计共享 helper；轻量自测只触碰 /tmp，不执行完整 E2E，并在总结中区分已验证的静态/锁自测与未执行的 102 查询。

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

## 2026-09-03 批量导入入口下沉

- 哪个坑浪费了最多时间？并行会话在同一 feature worktree 交错提交/修改，导致地址页与主分支版本冲突，合并后还出现一次地址树 UI 被过度压缩的风险；通过恢复主分支完整文件、只保留入口替换并重新门禁纠正。
- 这个 skill 有没有提前警告我？有：并行会话基线、编辑前 Read、提交后门禁和不得误闯其他 worktree；但本轮仍需在共享 worktree 中先中断并行代理，再做冲突核对。
- 重来一次我会怎么做？共享 worktree 只允许一个写入者；并行代理只做只读审计，所有页面修改在单一 feature worktree 顺序完成；每次 merge 后立即对关键页面做 `git diff <baseline>` 和真实 DOM/CDP 断言，避免用构建通过代替行为验证。

## 2026-09-03 批量导入后续改进

- 哪个坑浪费了最多时间？计划执行中发现主分支实际包含并行会话新提交，导致最初预期的地址/Geo入口版本和 `BaseImportEntry` 命名发生漂移；通过重新读取主树、单独 worktree、类型检查定位并修正 Geo 引用。
- 这个 skill 有没有提前警告我？有：先检查 worktree/分支、编辑前 Read、每项完成即 commit、门禁后再 ff-only；本轮最终按该流程完成。
- 重来一次我会怎么做？先把社区实践和当前事实写入计划 commit，再只选择一个最小可验收批次（本轮是导入中心职责收敛），避免在没有任务契约迁移设计前同时改后端模型。

## 2026-09-03 导入任务统计契约

- 哪个坑浪费了最多时间？新增服务接口参数后，Go 调用链（地址、Geo、fake、测试）必须全部同步；前端 worktree 初始没有 node_modules，先安装依赖后门禁才有可信结果。
- 这个 skill 有没有提前警告我？有：接口变更先 grep 全仓调用点、Go fake 同步、前端依赖检查、迁移号跨分支检查和每批门禁。
- 重来一次我会怎么做？先更新 service 接口和所有调用点，再改迁移与前端；先运行 `go test` 获取编译桩清单，再运行前端 `pnpm install --ignore-workspace` 后 typecheck/test/build。

## 2026-09-03 地址导入可靠性

- 哪个坑浪费了最多时间？将逐行地址导入改为事务后，pgxmock 原有测试只期待 Exec/Query，没有 Begin/Commit，导致门禁失败；同步补齐事务期望后恢复全量通过。
- 这个 skill 有没有提前警告我？有：事务改动必须同步测试桩、先用 Go 测试暴露调用契约，错误码复用既有 `ErrDuplicate` 映射。
- 重来一次我会怎么做？先在测试中增加 Begin/Commit/回滚场景，再切生产实现；同时补一个失败后确认回滚的断言，避免只验证成功路径。

## 2026-09-03 导入登记失败反馈

- 哪个坑浪费了最多时间？任务登记函数原本是 fire-and-forget，改为 async 后必须在 401 中止路径和正常完成路径都 await，否则仍会出现状态竞态和未提示。
- 这个 skill 有没有提前警告我？有：关键失败禁止静默、接口/文案三语言同步、每批 typecheck/test/build 和与最新 main 反向同步。
- 重来一次我会怎么做？先抽出可测试的 `registerTask` 纯回调或注入登记函数，再写组件测试断言失败提示；当前先以现有页面门禁和统一错误文案确保行为落地。

## 2026-09-03 导入任务筛选

- 哪个坑浪费了最多时间？后端筛选接口扩展后，前端日期范围必须明确使用 UTC 半开区间，避免结束日期遗漏当天数据；同时工作区命令必须始终在正确子目录运行。
- 这个 skill 有没有提前警告我？有：时间字段契约、服务接口调用链和每批同步门禁要求；本轮通过 `from >=`、`to <` 半开区间实现并验证。
- 重来一次我会怎么做？先抽出查询参数纯函数测试 kind/operator/from/to，再接 UI，减少直接改 JSX 后才发现缺少筛选契约的往返。

## 2026-09-03 导入筛选测试

- 哪个坑浪费了最多时间？直接把筛选 URL 拼接写在组件里无法稳定单测，且工具栏曾重复渲染刷新按钮；抽出 `buildTaskQuery` 后同时修复重复按钮并覆盖编码和 UTC 日期边界。
- 这个 skill 有没有提前警告我？有：UI 状态与 URL 约定、纯函数优先、真实 DOM 验证和类型/测试/构建门禁。
- 重来一次我会怎么做？筛选功能第一提交就先设计查询参数纯函数和测试，再接控件，避免先写页面后补契约。

## 2026-09-03 唯一冲突错误映射

- 哪个坑浪费了最多时间？`isUniqueViolation` 写死了约束名包含 "username"，导致部门/岗位/法人/客户/产品的唯一冲突全部落到 500；去掉约束名过滤后一行修复，但残留的 "strings" 导入又造成编译失败。
- 这个 skill 有没有提前警告我？有：改共享判定函数先 grep 全部调用点、每次改动后立即编译测试、错误码复用既有映射。
- 重来一次我会怎么做？错误分类判定一律基于 SQLSTATE 而非约束名字符串；改完立刻 `go build` 清理无用导入，再补跨约束名回归表驱动测试。

## 2026-09-03 唯一约束核查

- 哪个坑浪费了最多时间？此前子代理审计断言"客户手机号无 DB UNIQUE"，本轮直查迁移文件发现 000043 早已建 `uq_customers_app_login_phone`——审计结论过期未复核差点催生一次多余迁移。
- 这个 skill 有没有提前警告我？有："先查库、再接口复核"、不核查存量前不新增唯一约束迁移。
- 重来一次我会怎么做？任何"缺约束/孤儿"结论先 grep 迁移文件+直查权威表再落文档；文档审计清单每收敛一项就当场勾掉，防止陈旧结论被后续轮次当作待办重复执行。

## 2026-09-03 导入入口组件测试

- 哪个坑浪费了最多时间？worktree 内首次 pnpm install --silent 静默失败导致测试没跑就退出 1；去掉 --silent 看到真实输出后才发现是依赖未装。
- 这个 skill 有没有提前警告我？有：worktree 内需真实安装依赖、测试通过才算门禁；本轮补齐了 BatchImportEntry 渲染/权限提示/自定义文案五条断言。
- 重来一次我会怎么做？新 worktree 门禁命令固定为 install（非静默）→ typecheck → test → build 四步串，静默参数只在确认环境就绪后使用。

## 2026-09-03 导入 102 真实验证（批次 E）

- 哪个坑浪费了最多时间？两个后端 bug 都是"单测 fake 未触达真 SQL"的盲区：SQL 多余右括号、空串参数 ::timestamptz 22007（PG 的 OR 不惰性求值）；只有打真实环境才暴露。
- 这个 skill 有没有提前警告我？有：禁止 mock 替代真实验证、healthz 200 只证明容器活着、CI 部署要看镜像 tag 而非健康检查。
- 重来一次我会怎么做？新增/修改 SQL 的批次在合并前就用 102 直连或部署环境跑一次真查询；pgxmock 测试断言参数形态（nil vs 空串）防 22007 类回归。

## 2026-09-03 bossctl CLI 验证导入

- 哪个坑浪费了最多时间？CLI 系统性测试暴露三个漏网缺陷（法人 dup 500、客户 FK 500、岗位模板样例自违法）——单测+curl 抽查不如"实体×(create/dup/fk/invalid)"矩阵逐个打真实端点；bossctl 成功响应打印裸 data（非信封）导致解析脚本连续 KeyError。
- 这个 skill 有没有提前警告我？有：bossctl-cli skill 的职责分工与免登录方式；self-evolving 的"真实端点逐个验证"红线。
- 重来一次我会怎么做？错误映射类改动合并前跑 CLI 全量矩阵而非只测改到的实体；bossctl 解析先看裸输出再写取值路径。

## 任务日志（一段一行，仅存该任务唯一性结论）

- 2026-08-29 侧栏层级间距：不要用 `first:` 给二级项加首项间距，因为分组标题也是 section 首个子节点；用包裹二级列表的容器统一增加顶部间距，并只在展开态增加缩进。
- 2026-08-29 侧栏分组折叠回归：`showItems` 不能用 active 路由强制展开，否则点击子菜单后折叠按钮状态与实际内容矛盾；应始终尊重 folded 状态，并在 worktree 合并前完成 typecheck/test/build。
- 2026-08-24 工作台订单状态跳转：仪表盘统计行用 navigate 携带 `status`，订单列表用 useQueryState/useQueryInt 做首次 URL 初始化并在本地交互时双写 URL；worktree 合并前若 main 已推进，必须回 feature merge main 后重新门禁再 ff-merge。
- 2026-08-23 性能与 SLO 优化：Go 门禁因 PATH 未含 Homebrew 工具和磁盘不足失败；下次先检查 `command -v go gofmt` 与 `df -h`，并在报告中将环境阻塞与代码失败分开。
- 2026-08-24 渠道 API key 模板：给已有 Service 接口增加参数会波及 fake 实现与测试；必须先 grep 全仓调用点并同步测试桩，且 Go 工具可能不在 PATH，要先使用 /opt/homebrew/bin。
- 2026-08-24 渠道佣金台账：渠道结算必须按 partner account 的 legal_entity_id 做数据隔离，结算 UPDATE 同时校验企业归属与 ACCRUED 状态；迁移号要在合并后重新 rebase，避免并行分支造成 ff 失败。
- 2026-08-24 渠道订单接口：不要复制直营状态机；伙伴订单 handler 只负责伙伴企业身份和审计，订单创建必须复用 `OrderService.Submit`，由地址继续推导归属并保留 12 环节语义。
- 2026-08-24 渠道审计报表：审计报表优先聚合既有 `audit_logs`，通过订单归属 `legal_entity_id` 做租户隔离；新增权限必须同步角色绑定、契约字段和回滚迁移。
- 2026-08-24 渠道基础反作弊：订单风控检查必须在伙伴企业隔离后、OrderService.Submit 前执行；命中限额或客户冷却要写 `partner_order.blocked` 审计，且允许服务为空以兼容旧装配。
- 2026-08-24 风控契约收口：风控参数用 `biz_params` 默认值迁移并映射资源繁忙错误；服务端 `PartnerOrder` 标记必须是 `json:"-"`，避免客户端伪造伙伴来源。
- 2026-08-24 伙伴风控原子化：仅在 handler 预查计数不足以抗并发；伙伴 `Submit` 必须在同一事务内锁定 `legal_entities`、校验客户/产品租户归属、计数后插入订单和 stage 日志。
- 2026-08-24 伙伴区域权限：复用 accounts.region_scope LTREE，不新增重复权限表；设置区域时必须校验 regions 属于当前 legal_entity，订单归属仍以地址推导为权威。
- 2026-08-24 区域权限接入：仅提供 region-scope 写接口不形成有效隔离；伙伴员工和订单列表必须在 SQL 中使用同一账号 `region_scope` LTREE 子树谓词过滤。
- 2026-08-24 佣金自动计提：佣金应在订单第 12 环节完成后触发，且只对 `AGENT` 渠道；金额从产品月费与订单预缴月数计算，使用幂等台账接口，失败不得阻断订单状态机。
- 2026-08-24 佣金比例热配置：跨域读取 `biz_params` 应通过最小接口注入订单域，解析失败或越界回退默认值；主流程不能依赖配置读取成功。
- 2026-08-24 渠道验收：先用 `go test ./internal/domain/order` 覆盖配置回退，再用 `BOSS_PG_TEST_DSN` 跑真实订单 12 环节与 requestId 幂等 E2E；两者通过才可宣称订单链路已验证。
- 2026-08-24 风控回归：pgxmock 风控测试应分别覆盖每日上限和客户冷却，断言 reason 码与租户查询参数；真实并发锁语义仍需 PostgreSQL 集成测试，不能由 mock 代替。
- 2026-08-24 真实锁验收：集成测试必须在完整 Git worktree 中执行；`BOSS_PG_TEST_DSN` 未设置时只能报告 skipped，设置 102 DSN 后 `TestPartnerTenantLockIntegration` PASS 才算锁语义有证据。
- 2026-08-24 伙伴台账回归：佣金写入测试应验证幂等 upsert 的真实参数，并验证结算零行时拒绝，避免只测列表读路径。
- 2026-08-24 受限 API key：模板码必须从 Lookup 传入认证上下文，Authz 先按模板权限集判定；不能只在签发接口保存 template_code 而继续让 key 继承完整 RBAC。
- 2026-08-24 API key 回归：异步 `Touch` 测试必须等待原子计数，模板权限测试同时断言允许 `menu:partner-orders` 与拒绝普通管理菜单。
- 2026-08-24 伙伴 HTTP 回归：直接调用 handler 比装配完整 RBAC 路由更稳定；测试 stub 要按拆分后的 Partner/Commission/Audit/Region 接口分别注入，避免 nil 权限服务 panic。
- 2026-08-24 真实 E2E 稳定性：共享测试库重跑时，模板编码必须同时满足唯一性和数据库 VARCHAR(32) 长度；优先使用纳秒尾部短码，避免全流程被无关重复键/长度错误阻断。
- 2026-08-24 伙伴审批 E2E：真实库审批测试必须清理 admin account、partner application、legal entity，并断言初始口令长度、APPROVED 状态和账号租户绑定。
- 2026-08-24 伙伴佣金 E2E：台账真实库测试若依赖 AGENT 订单，必须明确记录无订单时 skipped；不能把 skipped 误报成通过，需后续用伙伴订单种子补齐。
- 2026-08-24 佣金真实库种子：若测试库没有伙伴管理员，测试可创建带唯一用户名的临时 partner_admin 并清理；PostgreSQL NUMERIC 乘法必须显式 cast，避免 float8 参数触发 operator is not unique。
- 2026-08-24 Q2 验收矩阵：必须把每个交付项映射到实现文件、路由、自动化测试和明确状态；尚未串联的伙伴全旅程与前端 102 验收要单独列为缺口，不能用直营 E2E 代替。
- 2026-08-24 前端验收：伙伴页面存在不等于 102 页面已验收；必须先确认实际部署路由，根 URL 404 时记录为未验收，不用截图或 build 结果替代线上点击证据。
- 2026-08-24 102 路由探测：`/healthz=200` 只证明后端存活；`/`、`/login`、`/admin` 和 `/api/admin/v1/auth/login` 404 说明当前 URL 没有 Web Shell/API 路由，必须记录部署入口阻塞。
- 2026-08-24 102 入口纠正：`docker-compose.102.app.yml` 明确 admin-web 暴露 `:5180`、server 暴露 `:28080`；先探测编排文件再判定 28080 404，CDP 已在 5180 验证 `/login` 与 `/partner/apply`。
- 2026-08-24 102 管理员冒烟：先在同一次 CDP 会话写入 `boss.servers`/active，再用原生 setter 触发账号密码 input，点击 `button[type=submit]`；成功证据是 URL `/dashboard`、`boss.token` 和菜单 DOM，而非只看 HTTP 200。
- 2026-08-24 伙伴页面线上验收：管理员 token 访问 `/partner/home` 路由可达不等于伙伴企业 API 数据可验收；管理员角色不应伪装成 partner_admin，必须使用真实伙伴凭证或明确保留缺口。
- 2026-08-24 102 权限冒烟：管理员 token 调用伙伴企业 API 返回 403 是正确门禁证据；真实伙伴数据验收不能用管理员 token 替代，需使用审批返回的一次性 partner_admin 凭证。
- 2026-08-24 102 版本漂移：同一资源 GET 可 200 而新增 POST 返回 404；必须用源码路由注册与线上逐路由 curl 对照，确认镜像落后后停止伪造线上验收。
- 2026-08-24 102 部署恢复：push main 触发 deploy-102 后，先看 `docker inspect boss-server Created` 与日志路由注册，再重试 POST；部署完成后才可用审批返回的一次性凭证做伙伴 Web E2E。
- 本轮知识库：新后端路由合并到 main 不等于 102 已部署；真实 `/knowledge-articles` 返回非 JSON 404，必须报告为线上未验收，不能用本地测试替代部署证据。
- 本轮配置模板：新增迁移字段后，旧 102 镜像仍可能返回旧字段且 HTTP 200；必须用 GET 返回字段、PUT/状态/删除逐路由实测，不能把“列表可用”当作新能力已部署。
- 2026-08-24 渠道下单回归：handler 测试必须断言服务端设置 `PartnerOrder` 与伙伴 `LegalEntityID`，并单独验证风控拒绝时不会调用 OrderService。
- 2026-08-24 工作台统计卡跳转：统计卡必须由组件统一处理鼠标/键盘交互，目标列表页同时消费 URL 条件；今日订单需前后端共同支持时间条件，不能只改变前端地址。
- 2026-08-24 订单状态趋势：后端趋势接口返回按 terms.md 五种状态拆分的 series，前端图表通过 legend button 切换可见曲线；扩展接口时同步更新后端 contract test、Dashboard DTO 和三份 locale。
- 本轮 Q1 CS/AR：迁移号查到未合并分支已占 000117，必须让号到 000118；Go 工具不在 PATH（gofmt/go test 均未执行），收尾应明确区分代码门禁未运行与代码错误。
- 本轮指标增量：并行主树编辑事故再次发生，必须在每次写入前用绝对 worktree 路径确认目标，且测试桩改动要在特性树完成；核心门禁实际应显式使用 `/opt/homebrew/bin/go`，不能信任 PATH。
- 本轮前端看板：已有 i18n 类型结构和多语言 locale 不易用宽 old_string 精确替换，新增 UI 应优先复用已有列名/页面文案或先定位精确行，避免为少量标签扩大契约改动；前端 typecheck/test/build 三门禁均可直接验证数据接入。
- 本轮补齐页面：新增菜单 key 后必须同步 zh-CN/en-US/ms-MY 三份 locale，且 locale 脚本从仓库根目录执行；在 web/admin 子目录执行仓库相对路径脚本会静默找不到文件，导致键集测试失败。

- 2026-08-22 GIS 地图空白：真实 102 `/gis/points?level=1` 返回空 items 时，页面原先用条件渲染卸载 OpenLayers；地图底图也随之消失。地图容器必须独立渲染，空数据提示用 pointer-events-none 覆盖层，避免把“无点位”误处理成“无地图”。
- 2026-08-22 GIS 地图仍空白：OpenLayers `map.on()` 返回 EventsKey，不能传给 `map.un()` 当 listener；React StrictMode 清理 effect 时抛 `removeEventListener` 异常，地图随组件卸载。统一用 `unByKey()` 清理 OL 事件。

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


## 2026-09-07 W5 投资分析深化与分光比建模(执行会话)

- 哪个坑浪费了最多时间? read 工具按**字节**截断(非行数),把 2143 行的 fields.md 当"全量读到"整文件回写,静默截掉尾部 143 行;靠 wc -l 对账才发现,git checkout 恢复后改用 edit 锚点编辑。损失一轮+差点丢契约尾部。
- 这个 skill 有没有提前警告我? 没有——skill 只警告了"编辑前必须 read",没警告"read 的'全量'可能是被截断的全量"。已喂回红线 26。
- 重来一次我会怎么做? ①大文件一律 edit 锚点编辑,绝不 read 回写;②造数脚本动笔前先逐字段核对 yaml 请求形状与枚举字典(本次 grids 的 prv/city 是 query 参数、facility 的 gridCode 必填、label 必须小写、receipt confirm 必须带 items、链行枚举传中文标签——五个坑五轮重跑);③并行 main 上 ff-only 失败后严禁继续 && 链(本次 push 把别人未推的 commit 顺带推了,所幸无损);④新增路由后先跑 gen-bossctl-routes + genrouteperms 再进 make check(目录漂移是已知必现,不要等门禁红)。
