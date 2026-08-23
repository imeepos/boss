# 累犯台账(重复犯错计数)

> 每次 notes.md 反思后同步更新:同一坑再犯就 +1 并追加日期;新坑从 1 起登记。
> ≥2 次的条目必须同步登到 SKILL.md 顶部"高频红线"区。只增不改,纠正用新条目。

| 坑 | 次数 | 发生日期 | 后果 |
|---|---|---|---|
| bash 默认 cwd 是主树而非 worktree,python/sed 批量编辑跑错树 | 1 | 2026-08-22(改了主树 check-contract-sync/routes.go;另一次 FileNotFound 才发现) | worktree 内每条 bash 显式带 workdir,heredoc 开头 pwd 自检 |
| 非交互 rebase 的 reword sed 按行号命中错误 pick,把 main 侧提交贴了自己的 message | 1 | 2026-08-22(开放平台 M1 分支,三轮返工) | sed 按 hash 前缀匹配不按行号;改完 git log --graph 验证 |
| 用原生 `<select>` 写下拉(option 弹层无法随主题定制) | 2 | 2026-08-18(顶栏语言切换), 2026-08-18(geo 分页 size changer+国家筛选) | 两次被用户点名"奇怪",返工 1 轮 |
| edit 前不用 read 工具看文件 / 凭上轮记忆拼 old_string | 5 | 2026-08-18(cat 后 edit 被拒), 2026-08-18(sed 后 edit 被拒), 2026-08-18(UserMenu old_string not found) | 每次浪费一轮重试 |; 2026-08-20(并行会话改文件,edit 连续 file changed since read); 2026-08-21(data-relations.md 用 bash sed 读后 4 个 edit 全被拒,重读一轮); 2026-08-25(notify_test.go python 改后 edit 拒,locale 凭记忆 edit 拒) |
| 会话边界残留未提交修改被误判为本会话引入的回归 | 1 | 2026-08-21(套餐详情页遇 OrderPage.kt 已存在 4 处编译错误) | 浪费一轮排查后才定位是前一会话遗留 |
| 没有验证动作就在总结里声称"已适配/已验证"(未 grep 令牌就说主题自适应) | 3 | 2026-08-18(geo 下拉 --shell-bg 不存在), 2026-08-19(geo 下拉/URL 状态只跑 build/test 未做真实 DOM 点击断言), 2026-08-19(用户中心表单主题只做门禁/HMR未做双主题真实页面验证) | 静默失败差点漏过,用户追问才暴露 |
| 文字字形(▾/✓)当图标 | 1 | 2026-08-18(geo 下拉箭头) | 被用户点名"太小",返工 1 轮 |
| 不查 antd 一手规范就自造组件结构 | 1 | 2026-08-18(初版分页只有上一页/下一页) | 返工 1 轮 |
| i18n 加 key 只改 locale 漏改 types.ts | 1 | 2026-08-18(UserMenu profile key) | tsc TS2353 抓住,险些 |
| 任务完成后忘记 git commit(工作区留脏) | 6 | 2026-08-18(geo 分页组件遗留到下任务), 2026-08-18(geo 多语言收尾发现交织改动), 2026-08-18(URL 状态+分页+下拉全会话未提交,用户点名), 2026-08-19(geo 批量导入完成未提交,用户点名), 2026-08-19(PSGC 迁移验证完直接总结,000041 未提交,反思第 0 步才补), 2026-08-21(DOM 验证耗时期间改动被并行会话扫进混合提交 2c34af5,消息不合规且无法干净拆分) | 用户需手动提醒,改动长期悬空 |
| 开工前不检索 skill 既有教训,重新发明已有正解 | 1 | 2026-08-19(brew 装 PG 10 分钟超时,lesson 28 早有"pgx 直连远端库") | 白等一轮,差点真装本地 PG |
| Compose 大段重写后漏 import / 使用不存在的 FontWeight 枚举 | 1 | 2026-08-20(用户首页视觉重写) | 首次 compileDebugKotlin 失败,修复后通过 |
| edit 的 new_string 与 old_string 范围不对称(顺手带函数头/只删换行的 no-op) | 2 | 2026-08-19(pg.go ListAddresses 头部重复), 2026-08-19(address/index.tsx 两行并一行致 TS1005) | 各废一轮 build+定位 |
| pgx 严格参数校验:占位符少于传参数直接 unused argument | 1 | 2026-08-19(e2e 清理六参数喂 $1 语句,整批 DELETE 全灭) | 废一轮全量验证 |
| t.Cleanup 里用池而资源用 defer 释放,清理跑在池关闭后静默失败 | 1 | 2026-08-19(e2e cleanup 在 defer pool.Close 之后,仅 -v 日志可见) | 废一轮,残留假象误导排查 |
| 按精确后缀清理测试数据,子测试自造独立后缀漏删 | 1 | 2026-08-19(W8 用 orderNo6() 另起后缀,w8 树永远清不掉) | 废一轮,靠 psql 残留计数才暴露 |
| 模型不支持图像输入却尝试读图分析 | 1 | 2026-08-18(使用 read_image 读取截图失败) | 浪费时间尝试不支持的功能,需改用代码分析 |
| 浏览器自动化工具缺失时未提前检查 | 1 | 2026-08-18(尝试使用 Playwright/Puppeteer 失败) | 应先检查环境依赖再选择工具 |
| 浏览器操作测试误用 playwright 代替 cdp-capture.mjs | 2 | 2026-08-18(本次会话,用 playwright 而非 cdp-capture 做浏览器调试), 2026-08-18(用户再次强调,升级至高频红线 #2) | 用户两轮点名纠正:cdp-capture 含 console 报错+失败请求响应体+网络采集,是调试排查首选;playwright 仅用于项目 E2E 自动化脚本 |
| 承诺"会保存/已记录"但当场不落盘,被用户连催 | 1 | 2026-08-19(test-accounts.json 只口头答应,连催 4 轮才真正 write) | 浪费 4 轮,用户失去耐心;凡承诺保存必须当场 write+ls |
| 机制未证明就把复现 bug 判为"环境怪象"放走 | 1 | 2026-08-20(Compose 尾随lambda连环push误判模拟器input怪象,用户真机复现才修) | 空耗多轮理论推演,用户二次报障 |
| 并行agent共享工作区,修复验证后未立即commit被回退 | 1 | 2026-08-20(僵尸subagent三次git checkout掉未提交修复+覆盖安装旧APK) | 已验证的修复反复"失效",排查方向被带偏 |
| 登录/业务响应按mock平铺结构解析,切真实后端未核对信封 | 1 | 2026-08-20(worker/user双端token从顶层取,真实在data.token,全端401) | 双端全部业务接口401 |
| 外部并行修改文件后未重新Read就edit("file changed since read") | 1 | 2026-08-20(ProfileScreen被僵尸进程回退后edit被拒) | 废一轮重读;与高频红线#1同源,计数并入其教训 |
| 端口被陈旧进程 IPv4/IPv6 双绑导致"假 404/假路由缺失"(lsof 不在默认 PATH 需用 /usr/sbin 全路径) | 1 | 2026-08-19(后端冒烟:orphan ./server-new 占 127.0.0.1:18080,curl 打偏) | 空耗多轮误判自己路由没注册 |
| 测试 fake 桩未完整实现 Go 接口全部方法(go vet 报缺方法) | 1 | 2026-08-19(fakeTaxStub 只写 ListInvoices、fakeUserData 缺 ListUserVerifyRecords 等) | 编译期逐个撞,多轮修正 |
| Go 单测 `:=` 单值赋给返回多值的 helper 编译错 | 1 | 2026-08-19(signCustomerToken 返回 (string,error),`tok :=` 报 mismatch) | 一轮编译错误 |
| 本机启动 boss 服务做冒烟(测试服务器只有 102 一台,提交后自动部署,本机配置低) | 1 | 2026-08-19(本地 go run /tmp/boss-new 冒烟,撞端口双绑假 404,用户明令"尽量不要本机启动") | 浪费多轮;应等 102 自动部署后用部署地址验证 |
| git add 提交前不查暂存区,并行会话已 stage 的文件被扫进提交 | 1 | 2026-05-25(user端布局修复混入 gen-er-drawio 等 5 文件,soft reset 重来) | 提交污染,需拆分返工;add 前先 git diff --cached 复核 |
| 共享模拟器做 UI 验证不先确认前台包名 | 1 | 2026-05-25(worker 僵尸进程抢前台,dump 到 worker 页面误当 user 页) | 差点据错误页面下结论;dump 前必查 topResumedActivity |
| 开发新页面前计划内置 mock 样例数据而非直连 102 真实服务 | 2 | 2026-08-20(设计稿规格 L 节自带"mock 数据"要求,照单全收计划造假,用户点名"不要使用mock 我在self-evolving已经警告过多次"), 2026-08-20此前多轮(用户口头警告) | 用户明确反复警告;规格文档里写 mock 也不执行,一律 curl 102 真实端点取数 |
| 2026-08-20 | Compose Box offset 只移视觉留死间隙(以为上移了,下张卡多 32dp) | 1 |
| 2026-08-20 | adb 只见模拟器就称"已装到手机",用户实体机没收到新包 | 1 |
| 2026-08-20 | 验证码 5 分钟过期当成登录链路故障排查 | 1 |
| 本机自启服务对接而非 102 部署 | 1 | 2026-08-20 | 上级叮嘱明确对接 102:28080;部署走 push→CI,本地起服务用户会点名纠正 |
| 2026-08-20 | UI 空间词(内圆角/交点)不问清就连猜 3 次,每轮真机验证成本 | 1 |
| 2026-08-20 | build FAILED 后 && 链上 adb install 仍装旧包报 Success | 1 |
| 2026-08-20 | 首帧测量为 0 导致负 padding 闪退 | 1 |
| 2026-08-20 | insets 被 statusBarsPadding 消费,sibling scrim 高度取 0 | 1 |
| 2026-08-20 | 改动未及时 commit 被并行会话裹走 | 1 |

| 2026-08-20 | 多会话共享仓库,pull 后未先 build,被并行提交的坏代码(ea91bdf getHeight)阻塞 | 1 |
| Android 构建安装脚本只覆盖 user，未检查 PATH adb 和多设备选择 | 1 | 2026-08-21 | 直接运行失败或误装模拟器；应检查 SDK adb 并显式指定实体机 serial |
| 2026-08-20 | 工具调用被打断后 build/install/commit 悬空,未先 git status 核对 | 1 |
| Java 环境未先检查就运行 Gradle | 1 | 2026-08-21 | 首次 compile 失败，设置 JAVA_HOME 后通过 |
| 用户报告 UI 异常,未先排除客户端陈旧(HMR/缓存/看错地址)就深挖代码 | 2 | 2026-08-20(smsconfig 图标"hover 才出现",多轮 DOM/令牌/部署排查后用户硬刷新即好), 2026-08-23(apiKey 列表绑定账号 #undefined,fix b11c607 已合 main 且 bundle 含 subjectCell,用户截图仍 #undefined,核对 nginx cache-control 才发现 max-age=31536000 immutable + index.html 未 no-cache 致 chunk hash 错位) | 空耗多轮排查一个不存在的 bug;**修完代码必须先 curl 远端 bundle 验证新代码文本已在内,再交付用户**;已升级高频红线 #2a |
| 使用 docs 里记录的辅助脚本前未验证其可用性(dev-token.mjs 已失效 404) | 1 | 2026-08-20(免登录脚本登录路径为旧 /auth/login) | 废一轮,应先 curl 验证端点再引用 |
| 自写临时 CDP 脚本做浏览器调试而非用 cdp-capture.mjs | 1 | 2026-08-20(元素级 clip 截图自写 icon-check.mjs,被中断未跑成) | 与高频红线#2精神冲突;确需元素级截图应先扩展 cdp-capture 而非另起炉灶 |
| 跑 docker prune 前未核对仅本地标签镜像与宿主 registry 登录态 | 1 | 2026-08-20(docker-clean.sh 删掉 deploy-runner 镜像,CI 断链;叠加重建出 600 迁移文件致 boss-server 重启 13 次) | prune 前先圈关键镜像,清理后必验 CI 链路 |
| 组件内部 padding 吃掉外部归零 modifier,造成 no-op 假修复 | 1 | 2026-08-21(worker 首页 OverviewCard 顶距,传 padding(top=0) 无效,真机确认后才真修) | 假修复一轮+用户二次点名 |
| 4 | 2026-08-21 | 真机 reboot 前未查 secure keyguard,PIN 锁死设备(1 次) |
| 改 handler 未先 grep OpenAPI schemas.yaml 看完整字段,导致 schema 列出 12 字段 handler 只返 6 字段,前端 optString 静默吞空变"沉默 bug" | 1 | 2025-08-21(师傅工单详情 customerName/phone/address/product/finishedAt 等 12 字段全空白,FEEDBACK E 区记录) | schema/handler 漂移无失败信号;改 handler 前必 grep schema,list/detail 必须共用联表根 |
| 任务开始前不看 `git status -uall`,带别人的 dirty diff 进提交 | 1 | 2026-08-21(user 端 dev-mode 临改被误判为我的范围而动 import) | 险些污染提交;应先划定"我的工作面",别人的 lint/缺 import 只提示 owner,不顺手改 |
| 本地 macOS 与 102 部署文件是独立副本(非 git 同步),本地 edit 不影响服务器 | 1 | 2026-08-21(MinIO compose 文件改完本地,docker compose 在 102 上仍跑老版本;反复 "Recreated" 但配置不变才发现) | 浪费 10+ 轮;改部署文件必须先 ssh+cat 确认服务器实际版本,或先 scp 同步再 edit |
| docker compose 顶层 `secrets:` 块在 UBI Micro 镜像容器内 `/run/secrets/` 不存在 | 1 | 2026-08-21(MinIO 镜像 RELEASE.2025-09-07,`docker compose config` 渲染正常但 `docker inspect` 见 Mounts 无 secrets) | 弃用 docker secrets,改 long-syntax bind mount;UBI Micro 镜像基础 docker secrets 兼容性未经验证前默认走 bind mount |
| gin 同 prefix 下两个 RouterGroup 注册同路径不同中间件,panic "already registered" | 1 | 2026-08-21(pub 和 uauth 都挂 `/debug/sms-code`,gin tree 路径节点冲突) | 同 prefix 下单一路由注册;场景差异用可选鉴权中间件在 handler 内分支 |
| build.gradle.kts 默认端口与真实服务不一致,装完 APK 连不上才发现 | 1 | 2026-08-21(默认8080,102服务器28080,用户登录到验证码发送失败) | 开工前先 grep 默认端口,不一致先修再开发;或用 `-PbossBaseUrl=` 覆盖 |
| 未完工改动被并行会话卷进混合提交(失去独立revert性) | 2 | 2026-08-21(fe30e60 卷入附件后端+i18n,7d0d984 卷入组件前端); 2026-08-25(popover 统一样式 4 文件被卷进 b31a217 feat(push)/41a0125 feat(worker),已验证内容在 HEAD 但不可独立 revert) | 提交纪律被破坏,只能记录在案;CDP 验证与 commit 之间的窗口越长越危险,改完立刻提交 |
| 并行agent把进行中的半成品连同无关文件提交成一条巨石 commit | 2 | 2026-08-20(僵尸subagent回退), 2026-08-21(fe30e60 混入 OrderPage.kt) | 提交污染难 revert;开工与收尾各 git status 一次,提交前 diff --cached 复核 |
| 新服务依赖可写目录但部署物(compose/Dockerfile)未随代码提交,102 装配 nil | 1 | 2026-08-21(backup 卷 permission denied,两轮部署) | 功能代码与部署物必须同批提交 |
| subagent "failed" 通知后未用 list_agents 复核,僵尸 agent 仍在并行写文件,与主线产出重复/冲突 | 1 | 2026-08-21(6 个重构 subagent 全报 failed,实际 3 个仍 running,残留 sections/ 等重复文件混入 stash) | 收到 failed 通知先 list_agents 核实;提交前 git status 出现非预期未跟踪文件必须查来源 |
| commit message 里写未经测量的数字(行数"370->202",实际 179) | 1 | 2026-08-21(profile 拆分,rebase reword 补救) | 写结论数字前先 wc -l 实测;此为红线#6 的数字版 |
| 列表页前端用统一 `Row.id` 取主键,后端 SQL AS 出来的实际主键名因表而异(addonId/couponId/denomId/customerId/faqId/guideId/id),ID 列显示 undefined 且 toggle/disable URL 拼出 `xxx/undefined/...` 死链 | 1 | 2026-08-22(用户端配置页 /bss/userdata 7 个 tab 全军覆没) | 写列表组件前先 `SELECT ... AS "..."` 列出后端实际别名,Tab 定义需自带 idKey;tabs.ts 必须有单测锁住映射 |
| worktree commit 完成后未先 ff-merge 就清理,commit 随 worktree 删除被 GC(无 remote 备份) | 3 | 2026-08-22(intel 板块可视化 c1 /gis/points 290 行 commit 在 worktree 删除时丢失); 2026-08-22(下一期 A2 maps/tile-source.ts 91 行 commit 28b2ec1 丢失,原因变种:并行会话推新 commit → ff-merge 失败 → 误把 worktree 与分支一起删 → GC); 2026-08-22(/intel/report 视域 NPE 修复 10d34c4 差点丢:worktree cp 改后 push 成功但 merge 失败,worktree remove + branch -d 同时执行触发 GC;幸亏远端 push 已存在且主工作区有修改残留) | worktree 收尾顺序:① push origin ② 立刻 merge --ff-only ③ worktree remove ④ branch -d ⑤ push --delete;每步独立可逆,绝不跳步。变种防御:**ff-merge 失败时不要删 worktree**——先 rebase worktree 到新 main 再 merge;只有 push 成功后且 merge 失败时才允许清理。**绝不允许 worktree remove 与 branch -d 同一指令组合执行**——失败后回滚无补救 |
| 并行会话推 commit 后,我方 worktree ff-merge 失败,误以为"合并失败=工作丢失" | 4 | 2026-08-22(A2 丢失的直接诱因); 2026-08-22(A4 i18n 7 个 key + A5 GIS UI 一起丢,因习惯性"merge 失败就清掉"); 2026-08-22(/intel/report 视域 NPE 修复差点丢 commit 10d34c4,主工作区有残留兜底); 本任务全程共触发 4 次 | ff-merge 失败时**严禁 worktree remove + branch -d**;唯一允许操作:git rebase main 在 worktree,重试 ff-merge;已落 4 次,已升级红线 #9 |
| 误闯并行会话的 worktree 并编辑其未提交文件(2026-08-22,用户点名) | 1 | 2026-08-22 |
| 带暂存主工作建临时验证分支,git commit(-am)把暂存文件卷进临时提交,删分支时差点丢主工作(靠 dangling commit 找回) | 2 | 2026-08-22(门禁D项验证,同会话连犯两次) | 验证分支只 add 明确 pathspec;commit 前必 git status 核对暂存清单;删验证分支前确认主工作文件仍在工作树 |
| git mv 误在主 workdir 执行(违反"禁止主分支修改") | 1 | 2026-08-22(invite_reward 让号) | 多 worktree 并行时,每条命令显式确认 workdir 参数指向自己的 worktree,不依赖默认值 |
| 跨年度规划未先读取契约和既有路线图 | 1 | 2026-08-26 | 容易把待建域、历史阶段和现状能力混写；修复是先读 terms/domain-map/fields 与既有路线文档，再写年度和季度出口 |
| 年度计划拆季度时未统一目标、交付物、验收结构 | 1 | 2026-08-26 | 季度内容难以比较和验收；修复是固定四段结构，并区分规划 Q1-Q4 与自然季度 |
| 季度范围未核对能力域的已建/待建边界 | 1 | 2026-08-26 | 远期计划会隐式承诺未建域；修复是先查 domain-map，再逐季写明确不做项 |
| 智能化计划未按数据、人工兜底和复制依赖排序 | 1 | 2026-08-26 | AI/预测维护容易变成无事实源、无审计的口号；修复是数据底座→辅助建议→审批执行→多区域复制逐季推进 |
| 审计中将局部代码或页面误判为季度目标完成 | 2 | 2026-08-27(初次差距审计将 user 端点/移动端局部页面描述得过于接近 PORT 完成), 2026-08-27(PORT/openplat 专项复核后修正) | 交付结论失真；修复是代码、契约、迁移、自动化测试、验收报告和真实环境证据至少交叉核对，区分部分闭环与已验收 |

| 怀疑外部环境异常但未先在本地主树跑最小 reproduce 用例就归因 | 1 | 2026-08-22(102 创建开放应用 42200,本地 reproduce pass → 二进制漂移) | 先本地二分代码 bug vs 部署漂移,再做归因 |
