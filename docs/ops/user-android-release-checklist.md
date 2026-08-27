# user Android 首发发版 checklist（D-1 签字项）

> 来源：2026-08-27 上线计划会议五、D-1（发版 checklist 签字）与 D-5 发布基建。
> 规则：**P0/P1 零放行，质量负责人独享停发权**；每一项有具体验证命令与预期结果，签字=逐项跑通。
> 积分/弱网/打磨等执行进度见 `meeting-minutes/2026-08-27-user-android-two-week-launch.md`。

## 签字项

| # | 项目 | 验证命令 / 预期 | 状态 |
|---|------|-----------------|------|
| 1 | 签名验证 | `scripts/user-android-release-sign.sh`：apksigner verify 通过 + release 指纹 ≠ debug 指纹；指纹见 adopted note 出包留档 | 本轮已跑通 |
| 2 | 版本号对齐 | `app/build.gradle.kts` versionCode/versionName；单调递增表见 adopted note（首发 2/0.1.0）；UpdateApi 自更新判定依赖同表 | 已定档 |
| 3 | cleartext 收敛 | Manifest 无 `usesCleartextTraffic`，`res/xml/network_security_config.xml` 仅白名单 102 内网/联调/模拟器回环；`aapt dump xmltree app-release.apk AndroidManifest.xml` 核对 | 已实现+物证（release 包：usesCleartextTraffic 0 处、networkSecurityConfig=@0x7f140002、无 ACCESS_BACKGROUND_LOCATION） |
| 4 | 权限实测 | 真机(Android 13+)冷启弹 POST_NOTIFICATIONS；拒绝/允许两条路径不崩溃；定位权限按场景触发（确认无 ACCESS_BACKGROUND_LOCATION） | 模拟器已实测，真机待测 |
| 4b | 脱敏验收 | 页面展示一律 `phoneMasked/idNoMasked`；明文手机号仅拨号盘 ACTION_DIAL（联系师傅设计意图），不落屏幕文案 | 审计通过 |
| 5 | UpdateApi 全链路 | `GET /client/latest` 出新版本 → App 启动弹升级提示 → 下载安装（后端版本元数据端点由后端 D-7 补齐；未就绪时回滚仅剩 adb 人工） | 后端待交付 |
| 6 | 多语言+实名回归 | 语言切换失败保留本地高亮；实名三态(未提交/审核中/驳回)+驳回重新提交入口走查 | 已核验 |
| 7 | 出包记录归档 | release 证书 SHA-256 指纹 + 版本 + 构建时间落 adopted note（本次:`1C:AC:B3:E1:03:CE:87:6E:9E:22:C9:FD:C8:D2:08:7C:ED:BF:45:58:A4:10:92:F8:EC:25:02:16:00:EE:F2:DA`） | 已归档 |
| 8 | 关键路径回归 | `connectedDebugAndroidTest` 全绿（当前 8/8）+ 6 条关键路径手测（登录真码→产品→下单→支付→订单/账单→实名） | 自动化部分已绿 |

## D10 稳定性基线（2026-08-27 实测）

- **冷启 <3s**：干净模拟器(swiftshader 软渲染)3 次 `am start -W -S` TotalTime = 1229/1196/1179ms，均值 ≈1.2s；首启含通知权限弹窗场景 2277ms——均 <3s 红线，真机更快。测量注意：权限弹窗会顶替 topResumedActivity 导致 `am start -W` 报 0，须先 `pm grant ... POST_NOTIFICATIONS` 再测。
- **无 ANR**：connected 套件整跑无崩溃/ANR；连测用 AVD `test_device`（无头 `-no-window -gpu swiftshader_indirect`）。
- 压测 P95<1.5s / 监控告警 5xx：后端侧（102 环境 D-3 压测，明远）。

## 佳宁走查量化抽样（2026-08-27 实测,AVD 360dp 基线）

- 真实链路 UI 驱动:adb input 驱动登录(dev 自动回填真码,验证码字段实测填入 6 位码)→ 首页真实数据(家庭宽带 100M/余额 92.00/进行中订单 ORD-20260821-000359)。
- 底栏四 tab 均分:首页[95,2283-160,2329] 服务[370] 账单[645] 我的[920](1080px 宽四等分,间距 275px≈91.7dp)✓。
- 关键入口行高:办套餐/查订单/缴费/报故障 187x45px(15dp 文本行,点击容器为 44dp IconTile 已代码核验)✓。
- 驱动要点:agreement 是行首 18dp 圆形(整行不可点),登录按钮前置 agreed 勾选——uiautomator 坐标驱动时先点圆。

### 扩页抽样(同日续测)

- **账单 tab(=订单列表,设计如此)** 真实数据:ORD-20260822-000363(合同收费/千兆宽带1000M/已取消)、ORD-20260821-000359(派单);筛选胶囊 全部/进行中/已完成/已取消 y293-356(63px≈21dp)均匀分布 ✓;订单卡行高 60px+ 由卡片整体 clickable 承担 ✓。
- **服务 tab** 真实数据:家庭宽带100M ¥99/月、测试套餐-500M畅享 ¥199/月(100M/500M 合约均入列);分类胶囊 宽带/5G/增值服务 y304-367 均匀(x=126/349/546)✓。
- **UpdateApi 升级判定实链路**:`/client/latest?versionCode=2` 与 `=1` 均返回标准信封且 updateAvailable=false——契约与解析器对齐,双门槛判定待后端发布记录(apprelease 域)注入后再验(非客户端缺口)。
- 观察项:底栏第三 tab 标签「账单」=订单列表页(OrdersPage),真正的账单列表(BillsPage)在 我的→我的账单;与既有设计一致,非回归。
- **我的 tab**(四 tab 抽样收口) 真实数据:头部 采购经理·王 / 139\*\*\*\*1234(脱敏✓) / 已实名;快捷入口 实名信息(已实名)/家庭地址/我的套餐(家庭宽带100M);菜单行 221-943 宽 63px 高、行距 169px 均匀,含「我的积分」(R1 新增入口实存)✓。
- **关键路径 E2E 已扩至 5 读腿**:登录真码→产品→实名(VERIFIED)→订单(items 非空)→账单(信封键),connected 11/11 全绿零跳过。

### P2 留档(可发项,首发后排期)

- 多语言 UI 翻译(zh/en/fil 三语文案落地,当前页面硬编码中文,语言切换仅存偏好——会议砍项)。
- 逐页深度打磨剩余(List 动画/过渡、下拉刷新炫技、图片懒加载优化——会议明确砍)。
- 积分兑换实际链路(待后端可兑换券模板列表端点,见 ISSUE.md)。
- https 公网迁移(见 user-android-https-migration.md,视产品用户群确认提级)。
- /push/device 推送注册(待后端 B 轨 D-3 落地,Android 降级兜底)。
- 下单→支付 destruct E2E(造单后 cancel 清理,待回收通道强化后再自动化)。

### 无复现崩溃/ANR 观测(多轮 walkthrough 全程)

- 十轮连续 connected(11/11)+ 真链路 UI 驱动(登录/首页/账单/服务/我的)期间 logcat 无 FATAL/ANR;
- 关键页面(登录/实名/积分/订单确认/支付结果)渲染与交互路径均无异常记录。

### 走查量化第二轮(同日续测,子页抽样)

- **积分页真实数据**:客户 213 无积分 → 余额 0 + 「赚积分/积分明细」空态正确渲染(R1 空态路径实证)+「积分兑换/建设中」占位入口 ✓。
- **UpdateApi 客户端路径实测**:设置→检查更新 → 真实 /client/latest(versionCode=2)→ 弹「当前版本 v0.1.0 已是最新。」✓(签字项 5 客户端半程;发版数据注入后同路径出升级弹窗)。
- **订单详情里程碑视觉证据(R2 映射修复实证)**:装维中订单 ORD-20260821-000359 步骤条当前节点=「上门安装」(3),未到节点 4「完成」不误显 ✓;进度 0/12、师傅电话脱敏 138\*\*\*\*1001 ✓。

## 回滚通道

- 服务端 versionCode 白名单渐进放量 + 服务端开关指向旧 APK（后端灰度基建）；客户端 UpdateApi 自更新为人工通道兜底。